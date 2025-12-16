package controller

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/trafiks/trafiks/api/dto"
	"github.com/trafiks/trafiks/api/services"
	"github.com/trafiks/trafiks/pkg"
)

type WebhookClient interface {
	CreateWebhook(c *gin.Context)
	ListWebhooks(c *gin.Context)
	GetWebhook(c *gin.Context)
	UpdateWebhook(c *gin.Context)
	DeleteWebhook(c *gin.Context)
	ListDeliveries(c *gin.Context)
	GetDelivery(c *gin.Context)
}

type Webhook struct {
	logger          pkg.LoggerClient
	webhookService  services.WebhookServiceClient
	deliveryService services.WebhookDeliveryServiceClient
}

func NewWebhook(
	logger pkg.LoggerClient,
	webhookService services.WebhookServiceClient,
	deliveryService services.WebhookDeliveryServiceClient,
) WebhookClient {
	return &Webhook{
		logger:          logger,
		webhookService:  webhookService,
		deliveryService: deliveryService,
	}
}

func (w *Webhook) CreateWebhook(c *gin.Context) {
	var req dto.CreateWebhookRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"message": err.Error(),
		})
		return
	}

	resp, err := w.webhookService.CreateWebhook(c, req)
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

func (w *Webhook) ListWebhooks(c *gin.Context) {
	resp, err := w.webhookService.ListWebhooks(c)
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

func (w *Webhook) GetWebhook(c *gin.Context) {
	resp, err := w.webhookService.GetWebhook(c)
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

func (w *Webhook) UpdateWebhook(c *gin.Context) {
	var req dto.UpdateWebhookRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"message": err.Error(),
		})
		return
	}

	resp, err := w.webhookService.UpdateWebhook(c, req)
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

func (w *Webhook) DeleteWebhook(c *gin.Context) {
	resp, err := w.webhookService.DeleteWebhook(c)
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

func (w *Webhook) ListDeliveries(c *gin.Context) {
	resp, err := w.deliveryService.ListDeliveries(c)
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

func (w *Webhook) GetDelivery(c *gin.Context) {
	resp, err := w.deliveryService.GetDelivery(c)
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
