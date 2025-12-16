package routes

import (
	"github.com/trafiks/trafiks/api/controller"
	"github.com/trafiks/trafiks/api/middleware"
	"github.com/trafiks/trafiks/pkg"
)

type Metrics struct {
	logger            pkg.LoggerClient
	handler           Router
	metricsController controller.MetricsClient
	authValidator     middleware.AuthValidatorClient
}

func NewMetrics(
	logger pkg.LoggerClient,
	handler Router,
	metricsController controller.MetricsClient,
	authValidator middleware.AuthValidatorClient,
) Metrics {
	return Metrics{
		logger:            logger,
		handler:           handler,
		metricsController: metricsController,
		authValidator:     authValidator,
	}
}

func (m *Metrics) Setup() {
	metrics := m.handler.Group("/projects/:projectId/metrics")

	metrics.Use(m.authValidator.ValidateUser)
	{
		metrics.GET("", m.metricsController.GetMetrics)
		metrics.GET("/stream", m.metricsController.StreamMetrics)
	}
}
