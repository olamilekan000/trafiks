package services

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/trafiks/trafiks/api/dto"
	"github.com/trafiks/trafiks/api/repository"
	"github.com/trafiks/trafiks/cfg"
	"github.com/trafiks/trafiks/models"
	"github.com/trafiks/trafiks/pkg"
	"github.com/trafiks/trafiks/pkg/cache"
	"github.com/trafiks/trafiks/pkg/source"
)

type ProxyRequestLogClient interface {
	ListRequestLogs(c *gin.Context) (interface{}, *pkg.RestErr)
	GetRequestLog(c *gin.Context) (interface{}, *pkg.RestErr)
	ReplayRequest(c *gin.Context) (interface{}, *pkg.RestErr)
}

type ProxyRequestLog struct {
	logger         pkg.LoggerClient
	requestLogRepo repository.ProxyRequestLogRepoClient
	projectRepo    repository.ProjectRepoClient
	serviceRepo    repository.ServiceRepoClient
	webhookRepo    repository.WebhookRepoClient
	restErr        pkg.RestErrClient
	proxyService   *ProxyService
	webhookService WebhookEventSender
}

func NewProxyRequestLog(
	logger pkg.LoggerClient,
	requestLogRepo repository.ProxyRequestLogRepoClient,
	projectRepo repository.ProjectRepoClient,
	serviceRepo repository.ServiceRepoClient,
	restErr pkg.RestErrClient,
	config *cfg.Config,
	redisClient cache.RedisClient,
	streamHub pkg.MetricsStreamHubClient,
	webhookService WebhookEventSender,
	sourceManager *source.ServiceSourceManager,
	webhookRepo repository.WebhookRepoClient,
) ProxyRequestLogClient {
	proxyService := NewProxyService(
		logger,
		serviceRepo,
		projectRepo,
		requestLogRepo,
		redisClient,
		config.AppBaseURL,
		config.TLSPort,
		streamHub,
		webhookService,
		sourceManager,
		webhookRepo,
	)
	return &ProxyRequestLog{
		logger:         logger,
		requestLogRepo: requestLogRepo,
		projectRepo:    projectRepo,
		serviceRepo:    serviceRepo,
		webhookRepo:    webhookRepo,
		restErr:        restErr,
		proxyService:   proxyService,
		webhookService: webhookService,
	}
}

func (s *ProxyRequestLog) ListRequestLogs(c *gin.Context) (interface{}, *pkg.RestErr) {
	ctx := c.Request.Context()

	user := GetUserFromContext(c)
	if user == nil {
		return nil, s.restErr.Unauthorized("unauthorized")
	}

	projectUID := c.Param("projectId")
	if projectUID == "" {
		return nil, s.restErr.BadRequest("project ID is required")
	}

	project, err := s.projectRepo.Find(ctx, &models.Project{
		UID:    projectUID,
		UserID: user.ID,
	})
	if err != nil {
		return nil, s.restErr.NotFound("project not found")
	}

	pagination := &dto.Pagination{
		Page:  1,
		Limit: 20,
	}

	if pageStr := c.Query("page"); pageStr != "" {
		if parsedPage, err := strconv.Atoi(pageStr); err == nil && parsedPage > 0 {
			pagination.Page = parsedPage
		}
	}

	if limitStr := c.Query("limit"); limitStr != "" {
		if parsedLimit, err := strconv.Atoi(limitStr); err == nil && parsedLimit > 0 {
			if parsedLimit > 100 {
				parsedLimit = 100
			}
			pagination.Limit = parsedLimit
		}
	}

	filters := &repository.RequestLogFilters{}

	if method := c.Query("method"); method != "" {
		filters.Method = method
	}

	if statusCodeStr := c.Query("status_code"); statusCodeStr != "" {
		if statusCode, err := strconv.Atoi(statusCodeStr); err == nil {
			filters.StatusCode = statusCode
		}
	}

	if startTimeStr := c.Query("start_time"); startTimeStr != "" {
		if startTime, err := time.Parse(time.RFC3339, startTimeStr); err == nil {
			filters.StartTime = &startTime
		}
	}

	if endTimeStr := c.Query("end_time"); endTimeStr != "" {
		if endTime, err := time.Parse(time.RFC3339, endTimeStr); err == nil {
			filters.EndTime = &endTime
		}
	}

	if cacheHitStr := c.Query("cache_hit"); cacheHitStr != "" {
		if cacheHit, err := strconv.ParseBool(cacheHitStr); err == nil {
			filters.CacheHit = &cacheHit
		}
	}

	if path := c.Query("path"); path != "" {
		filters.Path = path
	}

	logs, err := s.requestLogRepo.FindMany(ctx, &models.ProxyRequestLog{ProjectID: project.ID}, pagination, filters)
	if err != nil {
		s.logger.Errorf("failed to fetch request logs: %v", err)
		return nil, s.restErr.ServerError("failed to fetch request logs")
	}

	serviceIDs := make(map[uint]bool)
	for _, log := range logs {
		if log.ServiceID > 0 {
			serviceIDs[log.ServiceID] = true
		}
	}

	proxyURLMap := make(map[uint]string)
	for serviceID := range serviceIDs {
		service, err := s.serviceRepo.Find(ctx, &models.Service{ID: serviceID})
		if err == nil && service != nil {
			proxyURLMap[serviceID] = service.ProxyURL
		}
	}

	result := make([]gin.H, 0, len(logs))
	for _, log := range logs {
		var requestHeaders map[string]interface{}
		var responseHeaders map[string]interface{}

		if len(log.RequestHeaders) > 0 {
			_ = json.Unmarshal(log.RequestHeaders, &requestHeaders)
		}
		if len(log.ResponseHeaders) > 0 {
			_ = json.Unmarshal(log.ResponseHeaders, &responseHeaders)
		}

		proxyURL := proxyURLMap[log.ServiceID]

		result = append(result, gin.H{
			"uid":              log.UID,
			"method":           log.Method,
			"path":             log.Path,
			"query_string":     log.QueryString,
			"target_url":       log.TargetURL,
			"proxy_url":        proxyURL,
			"status_code":      log.StatusCode,
			"response_time":    log.ResponseTime,
			"cache_hit":        log.CacheHit,
			"request_size":     log.RequestSize,
			"response_size":    log.ResponseSize,
			"request_headers":  requestHeaders,
			"response_headers": responseHeaders,
			"request_body":     log.RequestBody,
			"response_body":    log.ResponseBody,
			"ip_address":       log.IPAddress,
			"user_agent":       log.UserAgent,
			"requested_at":     log.RequestedAt,
			"created_at":       log.CreatedAt,
		})
	}

	return gin.H{
		"request_logs": result,
		"pagination": gin.H{
			"page":        pagination.Page,
			"limit":       pagination.Limit,
			"total":       pagination.Total,
			"total_pages": (int(pagination.Total) + pagination.Limit - 1) / pagination.Limit,
		},
	}, nil
}

func (s *ProxyRequestLog) GetRequestLog(c *gin.Context) (interface{}, *pkg.RestErr) {
	ctx := c.Request.Context()

	user := GetUserFromContext(c)
	if user == nil {
		return nil, s.restErr.Unauthorized("unauthorized")
	}

	projectUID := c.Param("projectId")
	if projectUID == "" {
		return nil, s.restErr.BadRequest("project ID is required")
	}

	logUID := c.Param("logId")
	if logUID == "" {
		return nil, s.restErr.BadRequest("log ID is required")
	}

	project, err := s.projectRepo.Find(ctx, &models.Project{
		UID:    projectUID,
		UserID: user.ID,
	})
	if err != nil {
		return nil, s.restErr.NotFound("project not found")
	}

	requestLog, err := s.requestLogRepo.Find(ctx, &models.ProxyRequestLog{UID: logUID})
	if err != nil {
		return nil, s.restErr.NotFound("request log not found")
	}

	if requestLog.ProjectID != project.ID {
		return nil, s.restErr.NotFound("request log not found")
	}

	var proxyURL string
	service, err := s.serviceRepo.Find(ctx, &models.Service{ID: requestLog.ServiceID})
	if err == nil && service != nil {
		proxyURL = service.ProxyURL
	}

	var requestHeaders map[string]interface{}
	var responseHeaders map[string]interface{}

	if len(requestLog.RequestHeaders) > 0 {
		_ = json.Unmarshal(requestLog.RequestHeaders, &requestHeaders)
	}
	if len(requestLog.ResponseHeaders) > 0 {
		_ = json.Unmarshal(requestLog.ResponseHeaders, &responseHeaders)
	}

	return gin.H{
		"uid":              requestLog.UID,
		"method":           requestLog.Method,
		"path":             requestLog.Path,
		"query_string":     requestLog.QueryString,
		"target_url":       requestLog.TargetURL,
		"proxy_url":        proxyURL,
		"status_code":      requestLog.StatusCode,
		"response_time":    requestLog.ResponseTime,
		"cache_hit":        requestLog.CacheHit,
		"request_size":     requestLog.RequestSize,
		"response_size":    requestLog.ResponseSize,
		"request_headers":  requestHeaders,
		"response_headers": responseHeaders,
		"request_body":     requestLog.RequestBody,
		"response_body":    requestLog.ResponseBody,
		"ip_address":       requestLog.IPAddress,
		"user_agent":       requestLog.UserAgent,
		"requested_at":     requestLog.RequestedAt,
		"created_at":       requestLog.CreatedAt,
	}, nil
}

// ReplayRequest replays a logged request
func (s *ProxyRequestLog) ReplayRequest(c *gin.Context) (interface{}, *pkg.RestErr) {
	ctx := c.Request.Context()

	user := GetUserFromContext(c)
	if user == nil {
		return nil, s.restErr.Unauthorized("unauthorized")
	}

	projectUID := c.Param("projectId")
	if projectUID == "" {
		return nil, s.restErr.BadRequest("project ID is required")
	}

	logUID := c.Param("logId")
	if logUID == "" {
		return nil, s.restErr.BadRequest("log ID is required")
	}

	project, err := s.projectRepo.Find(ctx, &models.Project{
		UID:    projectUID,
		UserID: user.ID,
	})
	if err != nil {
		return nil, s.restErr.NotFound("project not found")
	}

	requestLog, err := s.requestLogRepo.Find(ctx, &models.ProxyRequestLog{UID: logUID})
	if err != nil {
		return nil, s.restErr.NotFound("request log not found")
	}

	if requestLog.ProjectID != project.ID {
		return nil, s.restErr.NotFound("request log not found")
	}

	service, err := s.serviceRepo.Find(ctx, &models.Service{ID: requestLog.ServiceID})
	if err != nil || service == nil {
		return nil, s.restErr.NotFound("service not found")
	}

	if service.Scheme == models.SchemeHTTPS {
		s.proxyService.SetTlsPort(cfg.GetConf().TLSPort)
	}

	proxyURL := service.ProxyURL
	fullPath := requestLog.Path
	if fullPath == "" {
		fullPath = "/"
	}

	scheme := models.SchemeHTTP
	var vtlsConfig *tls.ConnectionState

	if service.Scheme == models.SchemeHTTPS {
		scheme = models.SchemeHTTPS
		vtlsConfig = &tls.ConnectionState{}
	}

	fullURL := scheme + "://" + proxyURL + fullPath
	if requestLog.QueryString != "" {
		fullURL += "?" + requestLog.QueryString
	}

	parsedURL, err := url.Parse(fullURL)
	if err != nil {
		s.logger.Errorf("failed to parse replay URL: %v", err)
		return nil, s.restErr.ServerError("failed to parse replay URL")
	}

	var bodyReader io.Reader
	if requestLog.RequestBody != "" {
		bodyReader = bytes.NewReader([]byte(requestLog.RequestBody))
	}

	httpReq, err := http.NewRequestWithContext(c.Request.Context(), requestLog.Method, parsedURL.String(), bodyReader)
	if err != nil {
		s.logger.Errorf("failed to create replay request: %v", err)
		return nil, s.restErr.ServerError("failed to create replay request")
	}

	var requestHeaders map[string]interface{}
	if len(requestLog.RequestHeaders) > 0 {
		_ = json.Unmarshal(requestLog.RequestHeaders, &requestHeaders)
		for key, value := range requestHeaders {
			if strValue, ok := value.(string); ok {
				httpReq.Header.Set(key, strValue)
			}
		}
	}

	httpReq.Host = proxyURL
	httpReq.Header.Set("Host", proxyURL)
	httpReq.TLS = vtlsConfig

	clientIP := c.ClientIP()
	userAgent := c.GetHeader("User-Agent")

	response := s.proxyService.ProxyRequest(httpReq, clientIP, userAgent)
	if response.Error != nil {
		s.logger.Errorf("failed to replay request: %v", response.Error)

		return nil, s.restErr.BadRequest(response.ErrorMessage)
	}

	bodyStr := string(response.Body)
	maxBodySize := 10 * 1024 // 10KB
	if len(bodyStr) > maxBodySize {
		bodyStr = bodyStr[:maxBodySize] + "..."
	}

	return gin.H{
		"status_code":      response.StatusCode,
		"response_time":    0,
		"response_body":    bodyStr,
		"response_size":    len(response.Body),
		"replayed_at":      time.Now(),
		"original_log_uid": requestLog.UID,
	}, nil
}
