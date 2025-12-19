//go:generate mockgen -source=proxy_request_log.go -destination=../../tests/mocks/proxy_request_log.go -package=mocks

package repository

import (
	"context"
	"time"

	"github.com/trafiks/trafiks/api/dto"
	"github.com/trafiks/trafiks/database/postgres"
	"github.com/trafiks/trafiks/models"
	"gorm.io/gorm"
)

type RequestLogFilters struct {
	Method     string
	StatusCode int
	StartTime  *time.Time
	EndTime    *time.Time
	CacheHit   *bool
	Path       string // partial match
}

type ProxyRequestLogRepoClient interface {
	Create(ctx context.Context, log *models.ProxyRequestLog) error
	Find(ctx context.Context, filter *models.ProxyRequestLog) (*models.ProxyRequestLog, error)
	FindMany(ctx context.Context, filter *models.ProxyRequestLog, pagination *dto.Pagination, filters *RequestLogFilters) ([]*models.ProxyRequestLog, error)
	Count(ctx context.Context, filter *models.ProxyRequestLog, filters *RequestLogFilters) (int64, error)
	// Metrics aggregation methods
	GetMetricsByProjectID(ctx context.Context, projectID uint, filters *RequestLogFilters) (*MetricsAggregation, error)
	GetTimeSeriesByProjectID(ctx context.Context, projectID uint, filters *RequestLogFilters, groupBy string) ([]TimeSeriesPoint, error)
	GetTopPathsByProjectID(ctx context.Context, projectID uint, filters *RequestLogFilters, limit int) ([]PathAggregation, error)
	GetStatusCodesByProjectID(ctx context.Context, projectID uint, filters *RequestLogFilters) (map[int]int64, error)
	GetMethodsByProjectID(ctx context.Context, projectID uint, filters *RequestLogFilters) (map[string]int64, error)
}

// MetricsAggregation represents aggregated metrics data
type MetricsAggregation struct {
	TotalRequests       int64
	TotalErrors         int64
	TotalCacheHits      int64
	AverageResponseTime float64
	MinResponseTime     int64
	MaxResponseTime     int64
	TotalRequestSize    int64
	TotalResponseSize   int64
}

// TimeSeriesPoint represents a time series data point
type TimeSeriesPoint struct {
	Time                time.Time
	Count               int64
	AverageResponseTime float64
	Errors              int64
	CacheHits           int64
}

// PathAggregation represents aggregated data for a path
type PathAggregation struct {
	Path                string
	Count               int64
	AverageResponseTime float64
	ErrorCount          int64
	CacheHitCount       int64
}

type ProxyRequestLogRepo struct {
	db *postgres.PostgresDB
}

func NewProxyRequestLogRepo(db postgres.PostgresDB) ProxyRequestLogRepoClient {
	return &ProxyRequestLogRepo{
		db: &db,
	}
}

func (r *ProxyRequestLogRepo) Create(ctx context.Context, log *models.ProxyRequestLog) error {
	return r.db.WithContext(ctx).Create(log).Error
}

func (r *ProxyRequestLogRepo) Find(ctx context.Context, filter *models.ProxyRequestLog) (*models.ProxyRequestLog, error) {
	var log models.ProxyRequestLog
	if err := r.db.WithContext(ctx).Where(filter).First(&log).Error; err != nil {
		return nil, err
	}
	return &log, nil
}

func (r *ProxyRequestLogRepo) FindMany(ctx context.Context, filter *models.ProxyRequestLog, pagination *dto.Pagination, filters *RequestLogFilters) ([]*models.ProxyRequestLog, error) {
	var logs []*models.ProxyRequestLog

	if pagination.Page < 1 {
		pagination.Page = 1
	}
	if pagination.Limit < 1 {
		pagination.Limit = 20
	}
	if pagination.Limit > 100 {
		pagination.Limit = 100
	}

	query := r.db.WithContext(ctx).Model(&models.ProxyRequestLog{}).Where(filter)

	// Apply additional filters
	query = applyRequestLogFilters(query, filters)

	// Count total
	if err := query.Count(&pagination.Total).Error; err != nil {
		return nil, err
	}

	// Fetch paginated results
	offset := pagination.Offset()
	if err := query.Order("requested_at DESC").
		Limit(pagination.Limit).
		Offset(offset).
		Find(&logs).Error; err != nil {
		return nil, err
	}

	return logs, nil
}

func (r *ProxyRequestLogRepo) Count(ctx context.Context, filter *models.ProxyRequestLog, filters *RequestLogFilters) (int64, error) {
	var count int64
	query := r.db.WithContext(ctx).Model(&models.ProxyRequestLog{}).Where(filter)
	query = applyRequestLogFilters(query, filters)
	if err := query.Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

// applyRequestLogFilters applies filters to the query
func applyRequestLogFilters(query *gorm.DB, filters *RequestLogFilters) *gorm.DB {
	if filters == nil {
		return query
	}

	if filters.Method != "" {
		query = query.Where("method = ?", filters.Method)
	}

	if filters.StatusCode > 0 {
		query = query.Where("status_code = ?", filters.StatusCode)
	}

	if filters.StartTime != nil {
		query = query.Where("requested_at >= ?", *filters.StartTime)
	}

	if filters.EndTime != nil {
		query = query.Where("requested_at <= ?", *filters.EndTime)
	}

	if filters.CacheHit != nil {
		query = query.Where("cache_hit = ?", *filters.CacheHit)
	}

	if filters.Path != "" {
		query = query.Where("path LIKE ?", "%"+filters.Path+"%")
	}

	return query
}

// GetMetricsByProjectID gets aggregated metrics for a project
func (r *ProxyRequestLogRepo) GetMetricsByProjectID(ctx context.Context, projectID uint, filters *RequestLogFilters) (*MetricsAggregation, error) {
	var result MetricsAggregation

	query := r.db.WithContext(ctx).Model(&models.ProxyRequestLog{}).Where("project_id = ?", projectID)
	query = applyRequestLogFilters(query, filters)

	// Get counts
	if err := query.Count(&result.TotalRequests).Error; err != nil {
		return nil, err
	}

	// Count errors (4xx and 5xx)
	errorQuery := query.Where("status_code >= ? AND status_code < ?", 400, 600)
	if err := errorQuery.Count(&result.TotalErrors).Error; err != nil {
		return nil, err
	}

	// Count cache hits
	cacheQuery := query.Where("cache_hit = ?", true)
	if err := cacheQuery.Count(&result.TotalCacheHits).Error; err != nil {
		return nil, err
	}

	// Get response time stats
	type ResponseTimeStats struct {
		Avg float64
		Min int64
		Max int64
	}
	var stats ResponseTimeStats
	if err := query.Select("AVG(response_time) as avg, MIN(response_time) as min, MAX(response_time) as max").
		Scan(&stats).Error; err != nil {
		return nil, err
	}
	result.AverageResponseTime = stats.Avg
	result.MinResponseTime = stats.Min
	result.MaxResponseTime = stats.Max

	// Get size stats
	type SizeStats struct {
		TotalRequestSize  int64
		TotalResponseSize int64
	}
	var sizeStats SizeStats
	if err := query.Select("COALESCE(SUM(request_size), 0) as total_request_size, COALESCE(SUM(response_size), 0) as total_response_size").
		Scan(&sizeStats).Error; err != nil {
		return nil, err
	}
	result.TotalRequestSize = sizeStats.TotalRequestSize
	result.TotalResponseSize = sizeStats.TotalResponseSize

	return &result, nil
}

// GetTimeSeriesByProjectID gets time series data for a project
func (r *ProxyRequestLogRepo) GetTimeSeriesByProjectID(ctx context.Context, projectID uint, filters *RequestLogFilters, groupBy string) ([]TimeSeriesPoint, error) {
	query := r.db.WithContext(ctx).Model(&models.ProxyRequestLog{}).Where("project_id = ?", projectID)
	query = applyRequestLogFilters(query, filters)

	// Determine time truncation based on groupBy
	var timeFormat string
	switch groupBy {
	case "1min":
		timeFormat = "DATE_TRUNC('minute', requested_at)"
	case "5min":
		// Group by 5-minute intervals: divide epoch seconds by 300 (5*60) and multiply back
		timeFormat = "to_timestamp((EXTRACT(EPOCH FROM requested_at)::bigint / 300) * 300)"
	case "30min":
		// Group by 30-minute intervals: divide epoch seconds by 1800 (30*60) and multiply back
		timeFormat = "to_timestamp((EXTRACT(EPOCH FROM requested_at)::bigint / 1800) * 1800)"
	case "1h", "hour":
		timeFormat = "DATE_TRUNC('hour', requested_at)"
	case "day":
		timeFormat = "DATE_TRUNC('day', requested_at)"
	case "week":
		timeFormat = "DATE_TRUNC('week', requested_at)"
	case "month":
		timeFormat = "DATE_TRUNC('month', requested_at)"
	default:
		timeFormat = "DATE_TRUNC('hour', requested_at)" // default to hour
	}

	type TimeSeriesResult struct {
		Time                time.Time `gorm:"column:time"`
		Count               int64     `gorm:"column:count"`
		AverageResponseTime float64   `gorm:"column:avg_response_time"`
		Errors              int64     `gorm:"column:errors"`
		CacheHits           int64     `gorm:"column:cache_hits"`
	}

	var results []TimeSeriesResult
	if err := query.Select(
		timeFormat+" as time",
		"COUNT(*) as count",
		"COALESCE(AVG(response_time), 0) as avg_response_time",
		"COUNT(CASE WHEN status_code >= 400 AND status_code < 600 THEN 1 END) as errors",
		"COUNT(CASE WHEN cache_hit = true THEN 1 END) as cache_hits",
	).
		Group("time").
		Order("time ASC").
		Scan(&results).Error; err != nil {
		return nil, err
	}

	points := make([]TimeSeriesPoint, len(results))
	for i, r := range results {
		points[i] = TimeSeriesPoint{
			Time:                r.Time,
			Count:               r.Count,
			AverageResponseTime: r.AverageResponseTime,
			Errors:              r.Errors,
			CacheHits:           r.CacheHits,
		}
	}

	return points, nil
}

// GetTopPathsByProjectID gets top paths by request count
func (r *ProxyRequestLogRepo) GetTopPathsByProjectID(ctx context.Context, projectID uint, filters *RequestLogFilters, limit int) ([]PathAggregation, error) {
	if limit <= 0 {
		limit = 10
	}
	if limit > 50 {
		limit = 50
	}

	query := r.db.WithContext(ctx).Model(&models.ProxyRequestLog{}).Where("project_id = ?", projectID)
	query = applyRequestLogFilters(query, filters)

	type PathResult struct {
		Path                string  `gorm:"column:path"`
		Count               int64   `gorm:"column:count"`
		AverageResponseTime float64 `gorm:"column:avg_response_time"`
		ErrorCount          int64   `gorm:"column:error_count"`
		CacheHitCount       int64   `gorm:"column:cache_hit_count"`
	}

	var results []PathResult
	if err := query.Select(
		"path",
		"COUNT(*) as count",
		"COALESCE(AVG(response_time), 0) as avg_response_time",
		"COUNT(CASE WHEN status_code >= 400 AND status_code < 600 THEN 1 END) as error_count",
		"COUNT(CASE WHEN cache_hit = true THEN 1 END) as cache_hit_count",
	).
		Group("path").
		Order("count DESC").
		Limit(limit).
		Scan(&results).Error; err != nil {
		return nil, err
	}

	paths := make([]PathAggregation, len(results))
	for i, r := range results {
		paths[i] = PathAggregation{
			Path:                r.Path,
			Count:               r.Count,
			AverageResponseTime: r.AverageResponseTime,
			ErrorCount:          r.ErrorCount,
			CacheHitCount:       r.CacheHitCount,
		}
	}

	return paths, nil
}

// GetStatusCodesByProjectID gets status code distribution
func (r *ProxyRequestLogRepo) GetStatusCodesByProjectID(ctx context.Context, projectID uint, filters *RequestLogFilters) (map[int]int64, error) {
	query := r.db.WithContext(ctx).Model(&models.ProxyRequestLog{}).Where("project_id = ?", projectID)
	query = applyRequestLogFilters(query, filters)

	type StatusCodeResult struct {
		StatusCode int   `gorm:"column:status_code"`
		Count      int64 `gorm:"column:count"`
	}

	var results []StatusCodeResult
	if err := query.Select("status_code, COUNT(*) as count").
		Group("status_code").
		Order("status_code ASC").
		Scan(&results).Error; err != nil {
		return nil, err
	}

	statusCodes := make(map[int]int64)
	for _, r := range results {
		statusCodes[r.StatusCode] = r.Count
	}

	return statusCodes, nil
}

// GetMethodsByProjectID gets HTTP method distribution
func (r *ProxyRequestLogRepo) GetMethodsByProjectID(ctx context.Context, projectID uint, filters *RequestLogFilters) (map[string]int64, error) {
	query := r.db.WithContext(ctx).Model(&models.ProxyRequestLog{}).Where("project_id = ?", projectID)
	query = applyRequestLogFilters(query, filters)

	type MethodResult struct {
		Method string `gorm:"column:method"`
		Count  int64  `gorm:"column:count"`
	}

	var results []MethodResult
	if err := query.Select("method, COUNT(*) as count").
		Group("method").
		Order("count DESC").
		Scan(&results).Error; err != nil {
		return nil, err
	}

	methods := make(map[string]int64)
	for _, r := range results {
		methods[r.Method] = r.Count
	}

	return methods, nil
}
