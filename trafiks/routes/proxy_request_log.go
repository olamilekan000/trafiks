package routes

import (
	"github.com/trafiks/trafiks/api/controller"
	"github.com/trafiks/trafiks/api/middleware"
	"github.com/trafiks/trafiks/pkg"
)

type ProxyRequestLog struct {
	logger               pkg.LoggerClient
	handler              Router
	requestLogController controller.ProxyRequestLogClient
	authValidator        middleware.AuthValidatorClient
}

func NewProxyRequestLog(
	logger pkg.LoggerClient,
	handler Router,
	requestLogController controller.ProxyRequestLogClient,
	authValidator middleware.AuthValidatorClient,
) ProxyRequestLog {
	return ProxyRequestLog{
		logger:               logger,
		handler:              handler,
		requestLogController: requestLogController,
		authValidator:        authValidator,
	}
}

func (p *ProxyRequestLog) Setup() {
	requestLogs := p.handler.Group("/projects/:projectId/request-logs")

	requestLogs.Use(p.authValidator.ValidateUser)
	{
		requestLogs.GET("", p.requestLogController.ListRequestLogs)
		requestLogs.GET("/:logId", p.requestLogController.GetRequestLog)
		requestLogs.POST("/:logId/replay", p.requestLogController.ReplayRequest)
	}
}
