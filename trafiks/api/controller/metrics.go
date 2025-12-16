package controller

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/trafiks/trafiks/api/services"
	"github.com/trafiks/trafiks/pkg"
)

type MetricsClient interface {
	GetMetrics(c *gin.Context)
	StreamMetrics(c *gin.Context)
}

type Metrics struct {
	logger         pkg.LoggerClient
	metricsService services.MetricsClient
}

func NewMetrics(
	logger pkg.LoggerClient,
	metricsService services.MetricsClient,
) MetricsClient {
	return &Metrics{
		logger:         logger,
		metricsService: metricsService,
	}
}

func (m *Metrics) GetMetrics(c *gin.Context) {
	resp, err := m.metricsService.GetMetrics(c)
	if err != nil {
		c.JSON(err.StatusCode, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "ok",
		"data":    resp,
		"success": true,
	})
}

func (m *Metrics) StreamMetrics(c *gin.Context) {
	m.metricsService.StreamMetrics(c)
}
