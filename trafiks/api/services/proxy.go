package services

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/gin-gonic/gin"
	"gorm.io/datatypes"
	"gorm.io/gorm"

	"github.com/trafiks/trafiks/api/repository"
	"github.com/trafiks/trafiks/models"
	"github.com/trafiks/trafiks/pkg"
	"github.com/trafiks/trafiks/pkg/cache"
	"github.com/trafiks/trafiks/pkg/source"
)

// ProxyResponse represents the result of a proxy request
type ProxyResponse struct {
	Body           []byte
	StatusCode     int
	Headers        http.Header
	CacheHit       bool
	TargetURL      string
	RequestHeaders map[string]string
	Error          error
	ErrorMessage   string
}

// RequestLogData holds data for logging a request
type RequestLogData struct {
	Req            *http.Request
	ClientIP       string
	UserAgent      string
	Service        *models.Service
	Resp           *http.Response
	ResponseBody   []byte
	RequestBody    []byte
	StatusCode     int
	ResponseTime   int64
	CacheHit       bool
	TargetURL      string
	RequestHeaders map[string]string
}

// NewRequestLogData creates a new RequestLogData with sensible defaults
func NewRequestLogData(req *http.Request, clientIP, userAgent string, service *models.Service) *RequestLogData {
	return &RequestLogData{
		Req:            req,
		ClientIP:       clientIP,
		UserAgent:      userAgent,
		Service:        service,
		StatusCode:     http.StatusOK,
		ResponseTime:   0,
		CacheHit:       false,
		RequestHeaders: make(map[string]string),
	}
}

// Builder methods for RequestLogData
func (r *RequestLogData) WithResponse(resp *http.Response) *RequestLogData {
	r.Resp = resp
	return r
}

func (r *RequestLogData) WithResponseBody(body []byte) *RequestLogData {
	r.ResponseBody = body
	return r
}

func (r *RequestLogData) WithRequestBody(body []byte) *RequestLogData {
	r.RequestBody = body
	return r
}

func (r *RequestLogData) WithStatusCode(code int) *RequestLogData {
	r.StatusCode = code
	return r
}

func (r *RequestLogData) WithResponseTime(ms int64) *RequestLogData {
	r.ResponseTime = ms
	return r
}

func (r *RequestLogData) WithCacheHit(hit bool) *RequestLogData {
	r.CacheHit = hit
	return r
}

func (r *RequestLogData) WithTargetURL(url string) *RequestLogData {
	r.TargetURL = url
	return r
}

func (r *RequestLogData) WithRequestHeaders(headers map[string]string) *RequestLogData {
	if headers != nil {
		r.RequestHeaders = headers
	}
	return r
}

// ProxyService handles proxy logic
type ProxyService struct {
	logger         pkg.LoggerClient
	serviceRepo    repository.ServiceRepoClient
	projectRepo    repository.ProjectRepoClient
	requestLogRepo repository.ProxyRequestLogRepoClient
	cacheClient    cache.Cache
	baseURL        string
	tlsPort        string // TLS port for HTTPS redirects
	streamHub      pkg.MetricsStreamHubClient
	webhookService WebhookEventSender
	httpClient     *http.Client
	sourceManager  *source.ServiceSourceManager
	webhookRepo    repository.WebhookRepoClient
}

// NewProxyService creates a new proxy service
func NewProxyService(
	logger pkg.LoggerClient,
	serviceRepo repository.ServiceRepoClient,
	projectRepo repository.ProjectRepoClient,
	requestLogRepo repository.ProxyRequestLogRepoClient,
	cacheClient cache.Cache,
	baseURL string,
	tlsPort string,
	streamHub pkg.MetricsStreamHubClient,
	webhookService WebhookEventSender,
	sourceManager *source.ServiceSourceManager,
	webhookRepo repository.WebhookRepoClient,
) *ProxyService {
	return &ProxyService{
		logger:         logger,
		serviceRepo:    serviceRepo,
		projectRepo:    projectRepo,
		requestLogRepo: requestLogRepo,
		cacheClient:    cacheClient,
		baseURL:        baseURL,
		tlsPort:        tlsPort,
		streamHub:      streamHub,
		webhookService: webhookService,
		sourceManager:  sourceManager,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		webhookRepo: webhookRepo,
	}
}

func (p *ProxyService) SetTlsPort(tlsPort string) {
	p.tlsPort = tlsPort
}

func (p *ProxyService) GetTlsPort() string {
	return p.tlsPort
}

// ProxyRequest handles a proxy request and returns the response
func (p *ProxyService) ProxyRequest(req *http.Request, clientIP, userAgent string) *ProxyResponse {
	ctx := req.Context()

	startTime := time.Now()
	response := &ProxyResponse{
		Headers:        make(http.Header),
		RequestHeaders: make(map[string]string),
	}

	proxyURL := extractProxyURL(req)
	if proxyURL == "" {
		p.logger.Errorf("Could not extract proxy URL from request")
		response.Error = fmt.Errorf("invalid proxy URL")
		response.ErrorMessage = "Invalid proxy URL"
		response.StatusCode = http.StatusBadRequest
		return response
	}

	service, err := p.serviceRepo.Find(ctx, &models.Service{ProxyURL: proxyURL})
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			p.logger.Errorf("Service not found for proxy URL: %s", proxyURL)
			response.Error = fmt.Errorf("service not found")
			response.ErrorMessage = "Service not found"
			response.StatusCode = http.StatusNotFound
			return response
		}

		p.logger.Errorf("Error finding service: %v", err)
		response.Error = fmt.Errorf("failed to find service")
		response.ErrorMessage = "Failed to find service"
		response.StatusCode = http.StatusInternalServerError
		return response
	}

	serviceSource := service.Source

	service, err = p.sourceManager.Use(serviceSource).Get(ctx, proxyURL)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			p.logger.Errorf("Service not found for proxy URL: %s using source: %s", proxyURL, serviceSource)
			response.Error = fmt.Errorf("service not found")
			response.ErrorMessage = "Service not found"
			response.StatusCode = http.StatusNotFound
			return response
		}

		p.logger.Errorf("Error finding service with source %s: %v", serviceSource, err)
		response.Error = fmt.Errorf("failed to find service")
		response.ErrorMessage = "Failed to find service"
		response.StatusCode = http.StatusInternalServerError
		return response
	}

	if service.Project.ID != 0 && service.Project.IsActive != nil && !*service.Project.IsActive {
		p.logger.Warnf("Project %s is inactive, rejecting proxy request", service.Project.Slug)
		responseTime := time.Since(startTime).Milliseconds()
		go p.logRequest(NewRequestLogData(req, clientIP, userAgent, service).
			WithStatusCode(http.StatusForbidden).
			WithResponseTime(responseTime).
			WithRequestHeaders(response.RequestHeaders))
		response.Error = fmt.Errorf("project is inactive")
		response.ErrorMessage = "Project is inactive"
		response.StatusCode = http.StatusForbidden
		return response
	}

	serviceConfig, err := service.GetConfig()
	if err != nil {
		p.logger.Errorf("error parsing service config: %v", err)
		responseTime := time.Since(startTime).Milliseconds()
		go p.logRequest(NewRequestLogData(req, clientIP, userAgent, service).
			WithStatusCode(http.StatusInternalServerError).
			WithResponseTime(responseTime).
			WithRequestHeaders(response.RequestHeaders))
		response.Error = fmt.Errorf("failed to parse service config")
		response.ErrorMessage = "failed to parse service config"
		response.StatusCode = http.StatusInternalServerError
		return response
	}

	if service.Scheme == models.SchemeHTTPS {
		isHTTPS := req.TLS != nil
		tlsPort := p.tlsPort

		host := req.Host
		if idx := strings.Index(host, ":"); idx != -1 {
			host = host[:idx]
		}
		httpsURL := fmt.Sprintf("https://%s:%s%s", host, tlsPort, req.URL.RequestURI())

		if !isHTTPS {
			if serviceConfig.HTTPSRedirect != nil && *serviceConfig.HTTPSRedirect {
				p.logger.Infof("Service %s: Redirecting HTTP request to HTTPS", service.ProxyURL)
				responseTime := time.Since(startTime).Milliseconds()
				go p.logRequest(NewRequestLogData(req, clientIP, userAgent, service).
					WithStatusCode(http.StatusMovedPermanently).
					WithResponseTime(responseTime).
					WithTargetURL(httpsURL).
					WithRequestHeaders(response.RequestHeaders))
				response.StatusCode = http.StatusMovedPermanently
				response.Headers.Set("Location", httpsURL)
				return response
			}

			responseTime := time.Since(startTime).Milliseconds()
			go p.logRequest(NewRequestLogData(req, clientIP, userAgent, service).
				WithStatusCode(http.StatusForbidden).
				WithResponseTime(responseTime).
				WithTargetURL(httpsURL).
				WithRequestHeaders(response.RequestHeaders))
			response.Error = fmt.Errorf("https required")
			response.ErrorMessage = "HTTPS required for this service"
			response.StatusCode = http.StatusForbidden
			return response
		}
	}

	var bodyBytes []byte
	var bodyHash string

	if req.Body != nil {
		bodyBytes, err = io.ReadAll(req.Body)
		if err != nil {
			p.logger.Errorf("Error reading request body: %v", err)
			responseTime := time.Since(startTime).Milliseconds()
			go p.logRequest(NewRequestLogData(req, clientIP, userAgent, service).
				WithStatusCode(http.StatusBadRequest).
				WithResponseTime(responseTime).
				WithRequestHeaders(response.RequestHeaders))
			response.Error = fmt.Errorf("failed to read request body")
			response.ErrorMessage = "Failed to read request body"
			response.StatusCode = http.StatusBadRequest
			return response
		}
		req.Body = io.NopCloser(bytes.NewReader(bodyBytes))

		if len(bodyBytes) > 0 {
			hash := sha256.Sum256(bodyBytes)
			bodyHash = hex.EncodeToString(hash[:])
		}
	}

	path := req.URL.Path
	if path == "/" {
		path = ""
	}
	targetURL := service.TargetBackendURL + path
	if req.URL.RawQuery != "" {
		targetURL += "?" + req.URL.RawQuery
	}
	response.TargetURL = targetURL

	if len(serviceConfig.QueryParams.Remove) > 0 {
		targetURL = removeQueryParams(targetURL, serviceConfig.QueryParams.Remove)
		response.TargetURL = targetURL
	}

	cacheHit := false
	if service.CacheEnabled && req.Method == http.MethodGet {
		cacheKey := cache.CacheKey(req.Method, req.URL.Path, req.URL.RawQuery, bodyHash)

		if cached, err := p.cacheClient.Get(req.Context(), cacheKey); err == nil && cached != nil {
			p.logger.Infof("Cache hit for %s %s", req.Method, req.URL.Path)
			cacheHit = true

			response.CacheHit = true
			response.Body = cached
			response.StatusCode = http.StatusOK
			response.Headers.Set("Content-Type", "application/json")

			tempReq, _ := http.NewRequestWithContext(req.Context(), req.Method, targetURL, nil)
			applyHeaderModifications(tempReq, req, serviceConfig)
			for key, values := range tempReq.Header {
				if len(values) > 0 {
					response.RequestHeaders[key] = values[0]
				}
			}

			responseTime := time.Since(startTime).Milliseconds()
			go p.logRequest(NewRequestLogData(req, clientIP, userAgent, service).
				WithResponseBody(cached).
				WithRequestBody(bodyBytes).
				WithStatusCode(http.StatusOK).
				WithResponseTime(responseTime).
				WithCacheHit(true).
				WithTargetURL(targetURL).
				WithRequestHeaders(response.RequestHeaders))

			return response
		}
	}

	webhook, err := p.webhookRepo.Find(
		ctx, &models.Webhook{UserID: service.Project.UserID, IsActive: pkg.BoolPtr(true)})
	if err != nil {
		p.logger.Warnf("failed to fetch webhook %v", err)

		response.Error = fmt.Errorf("failed to fetch webhook")
		response.ErrorMessage = "Failed to fetch webhook"
		response.StatusCode = http.StatusInternalServerError
		return response
	}

	if !cacheHit && p.webhookService != nil {
		go p.webhookService.SendEvent(
			webhook.ID,
			EventCacheMiss,
			map[string]interface{}{
				"method":      req.Method,
				"path":        req.URL.Path,
				"project_uid": service.Project.UID,
			},
		)
	}

	p.logger.Infof("Proxying %s %s to %s service: %s", req.Method, req.URL.Path, targetURL, service.UID)

	var bodyReader io.Reader
	if len(bodyBytes) > 0 {
		bodyReader = bytes.NewReader(bodyBytes)
	}

	backendReq, err := http.NewRequestWithContext(req.Context(), req.Method, targetURL, bodyReader)
	if err != nil {
		p.logger.Errorf("Error creating request: %v", err)
		responseTime := time.Since(startTime).Milliseconds()
		go p.logRequest(NewRequestLogData(req, clientIP, userAgent, service).
			WithRequestBody(bodyBytes).
			WithStatusCode(http.StatusInternalServerError).
			WithResponseTime(responseTime).
			WithTargetURL(targetURL).
			WithRequestHeaders(response.RequestHeaders))
		response.Error = fmt.Errorf("failed to create request")
		response.ErrorMessage = "Failed to create request"
		response.StatusCode = http.StatusInternalServerError
		return response
	}

	applyHeaderModifications(backendReq, req, serviceConfig)

	for key, values := range backendReq.Header {
		if len(values) > 0 {
			response.RequestHeaders[key] = values[0]
		}
	}

	resp, err := p.httpClient.Do(backendReq)
	if err != nil {
		p.logger.Errorf("Error forwarding request: %v", err)
		responseTime := time.Since(startTime).Milliseconds()

		if p.webhookService != nil && service.Project.UserID != 0 {
			eventType := EventUpstreamUnreachable
			if isTimeoutError(err) {
				eventType = EventUpstreamTimeout
			}
			go p.webhookService.SendEvent(
				webhook.ID,
				eventType,
				map[string]interface{}{
					"method":      req.Method,
					"path":        req.URL.Path,
					"target_url":  targetURL,
					"error":       err.Error(),
					"error_type":  eventType,
					"project_uid": service.Project.UID,
				},
			)
		}

		go p.logRequest(NewRequestLogData(req, clientIP, userAgent, service).
			WithRequestBody(bodyBytes).
			WithStatusCode(http.StatusBadGateway).
			WithResponseTime(responseTime).
			WithTargetURL(targetURL).
			WithRequestHeaders(response.RequestHeaders))

		response.Error = fmt.Errorf("failed to connect to upstream: %v", err)
		response.ErrorMessage = fmt.Sprintf("Failed to connect to upstream: %v", err)
		response.StatusCode = http.StatusBadGateway
		return response
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		p.logger.Errorf("Error reading response body: %v", err)
		responseTime := time.Since(startTime).Milliseconds()
		go p.logRequest(NewRequestLogData(req, clientIP, userAgent, service).
			WithResponse(resp).
			WithRequestBody(bodyBytes).
			WithStatusCode(http.StatusInternalServerError).
			WithResponseTime(responseTime).
			WithTargetURL(targetURL).
			WithRequestHeaders(response.RequestHeaders))
		response.Error = fmt.Errorf("failed to read upstream response")
		response.ErrorMessage = "Failed to read upstream response"
		response.StatusCode = http.StatusInternalServerError
		return response
	}

	if service.CacheEnabled && req.Method == http.MethodGet && resp.StatusCode == http.StatusOK {
		cacheKey := cache.CacheKey(req.Method, req.URL.Path, req.URL.RawQuery, bodyHash)
		ttl := time.Duration(service.CacheTTL) * time.Second
		if err := p.cacheClient.Set(req.Context(), cacheKey, body, ttl); err != nil {
			p.logger.Warnf("Failed to cache response: %v", err)
		}
	}

	response.Body = body
	response.StatusCode = resp.StatusCode
	response.Headers = resp.Header

	// Trigger webhook for failed requests
	if resp.StatusCode >= 400 && p.webhookService != nil && service.Project.UserID != 0 {
		go p.webhookService.SendEvent(
			webhook.ID,
			EventRequestFailed,
			map[string]interface{}{
				"method":      req.Method,
				"path":        req.URL.Path,
				"status_code": resp.StatusCode,
				"target_url":  targetURL,
				"project_uid": service.Project.UID,
			},
		)
	}

	responseTime := time.Since(startTime).Milliseconds()
	go p.logRequest(NewRequestLogData(req, clientIP, userAgent, service).
		WithResponse(resp).
		WithResponseBody(body).
		WithRequestBody(bodyBytes).
		WithStatusCode(resp.StatusCode).
		WithResponseTime(responseTime).
		WithCacheHit(response.CacheHit).
		WithTargetURL(targetURL).
		WithRequestHeaders(response.RequestHeaders))

	return response
}

// ServiceProxyHandler handles dynamic proxy requests based on service configuration
func ServiceProxyHandler(
	logger pkg.LoggerClient,
	serviceRepo repository.ServiceRepoClient,
	projectRepo repository.ProjectRepoClient,
	requestLogRepo repository.ProxyRequestLogRepoClient,
	cacheClient cache.Cache,
	baseURL string,
	tlsPort string,
	streamHub pkg.MetricsStreamHubClient,
	webhookService WebhookEventSender,
	sourceManager *source.ServiceSourceManager,
	webhookRepo repository.WebhookRepoClient,
) gin.HandlerFunc {
	proxyService := NewProxyService(
		logger,
		serviceRepo,
		projectRepo,
		requestLogRepo,
		cacheClient,
		baseURL,
		tlsPort,
		streamHub,
		webhookService,
		sourceManager,
		webhookRepo,
	)

	return func(c *gin.Context) {
		response := proxyService.ProxyRequest(c.Request, c.ClientIP(), c.GetHeader("User-Agent"))
		if response.Error != nil {
			if response.StatusCode == http.StatusMovedPermanently || response.StatusCode == http.StatusFound {
				c.Redirect(response.StatusCode, response.Headers.Get("Location"))
				return
			}
			c.JSON(response.StatusCode, gin.H{"error": response.ErrorMessage})
			return
		}

		copyHeaders(c.Writer.Header(), response.Headers)
		c.Data(response.StatusCode, response.Headers.Get("Content-Type"), response.Body)
	}
}

func extractProxyURL(req *http.Request) string {
	host := req.Host
	if host == "" {
		host = req.Header.Get("Host")
	}

	if idx := strings.Index(host, ":"); idx != -1 {
		host = host[:idx]
	}

	return host
}

func applyHeaderModifications(req *http.Request, originalReq *http.Request, config *models.ServiceConfig) {
	for key, values := range originalReq.Header {
		lowerKey := strings.ToLower(key)
		if lowerKey == "host" ||
			lowerKey == "connection" ||
			lowerKey == "keep-alive" ||
			lowerKey == "transfer-encoding" ||
			lowerKey == "upgrade" {
			continue
		}
		req.Header[key] = values
	}

	if len(config.Headers.Remove) > 0 {
		for _, headerToRemove := range config.Headers.Remove {
			req.Header.Del(headerToRemove)
		}
	}

	if len(config.Headers.Add) > 0 {
		for key, value := range config.Headers.Add {
			req.Header.Set(key, value)
		}
	}
}

func removeQueryParams(targetURL string, paramsToRemove []string) string {
	u, err := url.Parse(targetURL)
	if err != nil {
		return targetURL
	}

	query := u.Query()
	for _, param := range paramsToRemove {
		query.Del(param)
	}

	u.RawQuery = query.Encode()
	return u.String()
}

func copyHeaders(dest, source http.Header) {
	for key, values := range source {
		lowerKey := strings.ToLower(key)
		if lowerKey == "connection" ||
			lowerKey == "keep-alive" ||
			lowerKey == "transfer-encoding" ||
			lowerKey == "upgrade" {
			continue
		}
		for _, value := range values {
			dest.Add(key, value)
		}
	}
}

// logRequest logs a request using the provided RequestLogData
func (s *ProxyService) logRequest(data *RequestLogData) {
	s.logRequestDetailed(
		data.Req,
		data.ClientIP,
		data.UserAgent,
		data.Service,
		data.Resp,
		data.ResponseBody,
		data.RequestBody,
		data.StatusCode,
		data.ResponseTime,
		data.CacheHit,
		data.TargetURL,
		data.RequestHeaders,
	)
}

func (s *ProxyService) logRequestDetailed(
	req *http.Request,
	clientIP string,
	userAgent string,
	service *models.Service,
	resp *http.Response,
	responseBody []byte,
	requestBody []byte,
	statusCode int,
	responseTime int64,
	cacheHit bool,
	targetURL string,
	actualRequestHeaders map[string]string,
) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	ipAddress := clientIP

	sensitiveHeaders := map[string]bool{
		"authorization": true,
		"cookie":        true,
		"set-cookie":    true,
	}

	responseHeaders := make(map[string]string)
	if resp != nil {
		for key, values := range resp.Header {
			lowerKey := strings.ToLower(key)
			if !sensitiveHeaders[lowerKey] && len(values) > 0 {
				responseHeaders[key] = values[0]
			}
		}
	}

	respHeaders := make(map[string]string)
	for key, value := range actualRequestHeaders {
		lowerKey := strings.ToLower(key)
		if !sensitiveHeaders[lowerKey] {
			respHeaders[key] = value
		}
	}

	requestHeadersJSON, _ := json.Marshal(respHeaders)
	responseHeadersJSON, _ := json.Marshal(responseHeaders)

	maxBodySize := 10 * 1024
	requestBodyStr := formatBodyForStorage(requestBody, maxBodySize, req.Header.Get("Content-Type"))

	responseContentType := ""
	if resp != nil {
		responseContentType = resp.Header.Get("Content-Type")
	}
	responseBodyStr := formatBodyForStorage(responseBody, maxBodySize, responseContentType)

	requestLog := &models.ProxyRequestLog{
		ServiceID:       service.ID,
		ProjectID:       service.ProjectID,
		Method:          req.Method,
		Path:            req.URL.Path,
		QueryString:     req.URL.RawQuery,
		TargetURL:       targetURL,
		RequestHeaders:  datatypes.JSON(requestHeadersJSON),
		RequestSize:     int64(len(requestBody)),
		RequestBody:     requestBodyStr,
		StatusCode:      statusCode,
		ResponseHeaders: datatypes.JSON(responseHeadersJSON),
		ResponseSize:    int64(len(responseBody)),
		ResponseBody:    responseBodyStr,
		ResponseTime:    responseTime,
		CacheHit:        cacheHit,
		IPAddress:       ipAddress,
		UserAgent:       userAgent,
		RequestedAt:     time.Now(),
	}

	if err := s.requestLogRepo.Create(ctx, requestLog); err != nil {
		s.logger.Errorf("Failed to log proxy request: %v", err)
		return
	}

	if s.streamHub != nil {
		isError := statusCode >= 400
		event := pkg.MetricsEvent{
			Type:      "request_logged",
			ProjectID: service.ProjectID,
			Data: map[string]interface{}{
				"method":        req.Method,
				"path":          req.URL.Path,
				"status_code":   statusCode,
				"response_time": responseTime,
				"cache_hit":     cacheHit,
				"is_error":      isError,
				"request_size":  len(requestBody),
				"response_size": len(responseBody),
			},
			Timestamp: time.Now(),
		}
		s.streamHub.Broadcast(service.ProjectID, event)
	}
}

func formatBodyForStorage(body []byte, maxSize int, contentType string) string {
	if len(body) == 0 {
		return ""
	}

	isBinary := isBinaryContentType(contentType)

	if !isBinary {
		isBinary = !utf8.Valid(body)
	}

	if isBinary {
		prefixSize := 256
		if len(body) < prefixSize {
			prefixSize = len(body)
		}

		prefix := body[:prefixSize]
		encoded := base64.StdEncoding.EncodeToString(prefix)

		if len(body) > prefixSize {
			return fmt.Sprintf("[BINARY_DATA:size=%d bytes,content_type=%s,prefix_base64=%s]...",
				len(body), contentType, encoded)
		}
		return fmt.Sprintf("[BINARY_DATA:size=%d bytes,content_type=%s,base64=%s]",
			len(body), contentType, encoded)
	}

	bodyToProcess := body
	truncated := false
	if len(bodyToProcess) > maxSize {
		bodyToProcess = bodyToProcess[:maxSize]
		truncated = true
	}

	bodyStr := string(bodyToProcess)
	if truncated {
		return bodyStr + "..."
	}
	return bodyStr
}

func isBinaryContentType(contentType string) bool {
	if contentType == "" {
		return false
	}

	contentType = strings.ToLower(strings.TrimSpace(contentType))

	binaryTypes := []string{
		"image/",
		"video/",
		"audio/",
		"application/octet-stream",
		"application/pdf",
		"application/zip",
		"application/x-zip-compressed",
		"application/gzip",
		"application/x-gzip",
		"application/x-tar",
		"application/x-compress",
		"application/x-compressed",
		"application/x-bzip2",
		"application/x-7z-compressed",
		"application/x-rar-compressed",
		"application/vnd.ms-excel",
		"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
		"application/vnd.ms-powerpoint",
		"application/vnd.openxmlformats-officedocument.presentationml.presentation",
		"application/vnd.ms-word",
		"application/vnd.openxmlformats-officedocument.wordprocessingml.document",
		"application/x-shockwave-flash",
		"application/x-font-ttf",
		"application/x-font-otf",
		"application/font-woff",
		"application/font-woff2",
	}

	for _, binaryType := range binaryTypes {
		if strings.HasPrefix(contentType, binaryType) {
			return true
		}
	}

	return false
}

// Helper function to check if error is a timeout
func isTimeoutError(err error) bool {
	if err == nil {
		return false
	}
	errStr := strings.ToLower(err.Error())
	return strings.Contains(errStr, "timeout") ||
		strings.Contains(errStr, "deadline exceeded")
}
