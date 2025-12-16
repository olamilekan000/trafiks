package controller

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/trafiks/trafiks/api/services"
	"github.com/trafiks/trafiks/pkg"
)

type ProxyRequestLogClient interface {
	ListRequestLogs(c *gin.Context)
	GetRequestLog(c *gin.Context)
	ReplayRequest(c *gin.Context)
}

type ProxyRequestLog struct {
	logger            pkg.LoggerClient
	requestLogService services.ProxyRequestLogClient
}

func NewProxyRequestLog(
	logger pkg.LoggerClient,
	requestLogService services.ProxyRequestLogClient,
) ProxyRequestLogClient {
	return &ProxyRequestLog{
		logger:            logger,
		requestLogService: requestLogService,
	}
}

func (p *ProxyRequestLog) ListRequestLogs(c *gin.Context) {
	resp, err := p.requestLogService.ListRequestLogs(c)
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

func (p *ProxyRequestLog) GetRequestLog(c *gin.Context) {
	resp, err := p.requestLogService.GetRequestLog(c)
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

func (p *ProxyRequestLog) ReplayRequest(c *gin.Context) {
	resp, err := p.requestLogService.ReplayRequest(c)
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
