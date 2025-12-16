package services

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/trafiks/trafiks/api/dto"
	"github.com/trafiks/trafiks/api/repository"
	"github.com/trafiks/trafiks/models"
	"github.com/trafiks/trafiks/pkg"
)

type MetricsClient interface {
	GetMetrics(c *gin.Context) (interface{}, *pkg.RestErr)
	StreamMetrics(c *gin.Context)
}

type Metrics struct {
	logger         pkg.LoggerClient
	requestLogRepo repository.ProxyRequestLogRepoClient
	projectRepo    repository.ProjectRepoClient
	restErr        pkg.RestErrClient
	streamHub      pkg.MetricsStreamHubClient
}

func NewMetrics(
	logger pkg.LoggerClient,
	requestLogRepo repository.ProxyRequestLogRepoClient,
	projectRepo repository.ProjectRepoClient,
	restErr pkg.RestErrClient,
	streamHub pkg.MetricsStreamHubClient,
) MetricsClient {
	return &Metrics{
		logger:         logger,
		requestLogRepo: requestLogRepo,
		projectRepo:    projectRepo,
		restErr:        restErr,
		streamHub:      streamHub,
	}
}

func (s *Metrics) GetMetrics(c *gin.Context) (interface{}, *pkg.RestErr) {
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

	var startTime, endTime *time.Time
	if startTimeStr := c.Query("start_time"); startTimeStr != "" {
		if parsed, err := time.Parse(time.RFC3339, startTimeStr); err == nil {
			startTime = &parsed
		}
	}
	if endTimeStr := c.Query("end_time"); endTimeStr != "" {
		if parsed, err := time.Parse(time.RFC3339, endTimeStr); err == nil {
			endTime = &parsed
		}
	}

	if startTime == nil && endTime == nil {
		now := time.Now()
		dayAgo := now.Add(-24 * time.Hour)
		startTime = &dayAgo
		endTime = &now
	}

	groupBy := c.Query("group_by")
	if groupBy == "" {
		groupBy = "1min"
	}
	validGroupBy := map[string]bool{
		"1min": true, "5min": true, "30min": true,
		"1h": true, "hour": true,
		"day": true, "week": true, "month": true,
	}
	if !validGroupBy[groupBy] {
		groupBy = "1min"
	}

	filters := &repository.RequestLogFilters{
		StartTime: startTime,
		EndTime:   endTime,
	}

	metrics, err := s.requestLogRepo.GetMetricsByProjectID(ctx, project.ID, filters)
	if err != nil {
		s.logger.Errorf("failed to get metrics: %v", err)
		return nil, s.restErr.ServerError("failed to get metrics")
	}

	timeSeries, err := s.requestLogRepo.GetTimeSeriesByProjectID(ctx, project.ID, filters, groupBy)
	if err != nil {
		s.logger.Errorf("failed to get time series: %v", err)
		return nil, s.restErr.ServerError("failed to get time series data")
	}

	topPathsLimit := 10
	if limitStr := c.Query("top_paths_limit"); limitStr != "" {
		if parsed, err := strconv.Atoi(limitStr); err == nil && parsed > 0 {
			if parsed > 50 {
				parsed = 50
			}
			topPathsLimit = parsed
		}
	}
	topPaths, err := s.requestLogRepo.GetTopPathsByProjectID(ctx, project.ID, filters, topPathsLimit)
	if err != nil {
		s.logger.Errorf("failed to get top paths: %v", err)
		return nil, s.restErr.ServerError("failed to get top paths")
	}

	statusCodes, err := s.requestLogRepo.GetStatusCodesByProjectID(ctx, project.ID, filters)
	if err != nil {
		s.logger.Errorf("failed to get status codes: %v", err)
		return nil, s.restErr.ServerError("failed to get status code distribution")
	}

	methods, err := s.requestLogRepo.GetMethodsByProjectID(ctx, project.ID, filters)
	if err != nil {
		s.logger.Errorf("failed to get methods: %v", err)
		return nil, s.restErr.ServerError("failed to get method distribution")
	}

	cacheHitRate := 0.0
	if metrics.TotalRequests > 0 {
		cacheHitRate = (float64(metrics.TotalCacheHits) / float64(metrics.TotalRequests)) * 100
	}

	errorRate := 0.0
	if metrics.TotalRequests > 0 {
		errorRate = (float64(metrics.TotalErrors) / float64(metrics.TotalRequests)) * 100
	}

	timeSeriesData := make([]dto.TimeSeriesDataPoint, len(timeSeries))
	for i, point := range timeSeries {
		timeSeriesData[i] = dto.TimeSeriesDataPoint{
			Time:                point.Time,
			Count:               point.Count,
			AverageResponseTime: point.AverageResponseTime,
			Errors:              point.Errors,
			CacheHits:           point.CacheHits,
		}
	}

	topPathsData := make([]dto.PathMetrics, len(topPaths))
	for i, path := range topPaths {
		topPathsData[i] = dto.PathMetrics{
			Path:                path.Path,
			Count:               path.Count,
			AverageResponseTime: path.AverageResponseTime,
			ErrorCount:          path.ErrorCount,
			CacheHitCount:       path.CacheHitCount,
		}
	}

	statusCodesArray := make([]dto.StatusCodeCount, 0, len(statusCodes))
	for statusCode, count := range statusCodes {
		statusCodesArray = append(statusCodesArray, dto.StatusCodeCount{
			StatusCode: statusCode,
			Count:      count,
		})
	}

	methodsArray := make([]dto.MethodCount, 0, len(methods))
	for method, count := range methods {
		methodsArray = append(methodsArray, dto.MethodCount{
			Method: method,
			Count:  count,
		})
	}

	response := dto.MetricsResponse{
		TotalRequests:  metrics.TotalRequests,
		TotalErrors:    metrics.TotalErrors,
		TotalCacheHits: metrics.TotalCacheHits,
		CacheHitRate:   cacheHitRate,
		// AverageResponseTime removed from top-level - only available in RequestsOverTime for graph
		MinResponseTime:   metrics.MinResponseTime,
		MaxResponseTime:   metrics.MaxResponseTime,
		TotalRequestSize:  metrics.TotalRequestSize,
		TotalResponseSize: metrics.TotalResponseSize,
		StatusCodes:       statusCodesArray,
		Methods:           methodsArray,
		RequestsOverTime:  timeSeriesData, // Contains average_response_time per time point
		TopPaths:          topPathsData,
		ErrorRate:         errorRate,
	}

	return response, nil
}

// StreamMetrics handles SSE streaming for real-time metrics updates
func (s *Metrics) StreamMetrics(c *gin.Context) {
	ctx := c.Request.Context()

	user := GetUserFromContext(c)
	if user == nil {
		c.JSON(401, gin.H{"error": "unauthorized"})
		return
	}

	projectUID := c.Param("projectId")
	if projectUID == "" {
		c.JSON(400, gin.H{"error": "project ID is required"})
		return
	}

	project, err := s.projectRepo.Find(ctx, &models.Project{
		UID:    projectUID,
		UserID: user.ID,
	})
	if err != nil {
		c.JSON(404, gin.H{"error": "project not found"})
		return
	}

	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")

	clientChan := s.streamHub.RegisterClient(project.ID)
	defer s.streamHub.UnregisterClient(project.ID, clientChan)

	// Send initial connection message
	c.SSEvent("connected", gin.H{
		"message":    "Connected to metrics stream",
		"project_id": project.ID,
	})
	c.Writer.Flush()

	go func() {
		now := time.Now()
		dayAgo := now.Add(-24 * time.Hour)
		filters := &repository.RequestLogFilters{
			StartTime: &dayAgo,
			EndTime:   &now,
		}

		metrics, err := s.requestLogRepo.GetMetricsByProjectID(ctx, project.ID, filters)
		if err == nil {
			event := pkg.MetricsEvent{
				Type:      "metrics_snapshot",
				ProjectID: project.ID,
				Data: map[string]interface{}{
					"total_requests":   metrics.TotalRequests,
					"total_errors":     metrics.TotalErrors,
					"total_cache_hits": metrics.TotalCacheHits,
				},
				Timestamp: time.Now(),
			}
			s.streamHub.Broadcast(project.ID, event)
		}
	}()

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case eventJSON := <-clientChan:
			c.Writer.WriteString("data: " + string(eventJSON) + "\n\n")
			c.Writer.Flush()
		case <-ticker.C:
			c.SSEvent("ping", gin.H{"timestamp": time.Now().Unix()})
			c.Writer.Flush()
		case <-c.Request.Context().Done():
			return
		}
	}
}
