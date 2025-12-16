package dto

import "time"

type MetricsRequest struct {
	StartTime *time.Time `form:"start_time"`
	EndTime   *time.Time `form:"end_time"`
	GroupBy   string     `form:"group_by"` // hour, day, week, month
}

// MetricsResponse represents aggregated metrics
type MetricsResponse struct {
	TotalRequests       int64                 `json:"total_requests"`
	TotalErrors         int64                 `json:"total_errors"`
	TotalCacheHits      int64                 `json:"total_cache_hits"`
	CacheHitRate        float64               `json:"cache_hit_rate"`        // percentage
	AverageResponseTime float64               `json:"average_response_time"` // milliseconds
	MinResponseTime     int64                 `json:"min_response_time"`     // milliseconds
	MaxResponseTime     int64                 `json:"max_response_time"`     // milliseconds
	TotalRequestSize    int64                 `json:"total_request_size"`    // bytes
	TotalResponseSize   int64                 `json:"total_response_size"`   // bytes
	StatusCodes         []StatusCodeCount     `json:"status_codes"`          // status code -> count (array for easier charting)
	Methods             []MethodCount         `json:"methods"`               // method -> count (array for easier charting)
	RequestsOverTime    []TimeSeriesDataPoint `json:"requests_over_time"`    // Perfect for time series bar charts
	TopPaths            []PathMetrics         `json:"top_paths"`             // Perfect for bar charts
	ErrorRate           float64               `json:"error_rate"`            // percentage
}

type StatusCodeCount struct {
	StatusCode int   `json:"status_code"`
	Count      int64 `json:"count"`
}

type MethodCount struct {
	Method string `json:"method"`
	Count  int64  `json:"count"`
}

type TimeSeriesDataPoint struct {
	Time                time.Time `json:"time"`
	Count               int64     `json:"count"`
	AverageResponseTime float64   `json:"average_response_time"`
	Errors              int64     `json:"errors"`
	CacheHits           int64     `json:"cache_hits"`
}

type PathMetrics struct {
	Path                string  `json:"path"`
	Count               int64   `json:"count"`
	AverageResponseTime float64 `json:"average_response_time"`
	ErrorCount          int64   `json:"error_count"`
	CacheHitCount       int64   `json:"cache_hit_count"`
}
