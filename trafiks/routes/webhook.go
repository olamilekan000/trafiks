package routes

import (
	"github.com/trafiks/trafiks/api/controller"
	"github.com/trafiks/trafiks/api/middleware"
	"github.com/trafiks/trafiks/pkg"
)

type Webhook struct {
	logger            pkg.LoggerClient
	handler           Router
	webhookController controller.WebhookClient
	authValidator     middleware.AuthValidatorClient
}

func NewWebhook(
	logger pkg.LoggerClient,
	handler Router,
	webhookController controller.WebhookClient,
	authValidator middleware.AuthValidatorClient,
) Webhook {
	return Webhook{
		logger:            logger,
		handler:           handler,
		webhookController: webhookController,
		authValidator:     authValidator,
	}
}

func (w *Webhook) Setup() {
	webhooks := w.handler.Group("/webhooks")
	webhooks.Use(w.authValidator.ValidateUser)
	{
		webhooks.POST("", w.webhookController.CreateWebhook)
		webhooks.GET("", w.webhookController.ListWebhooks)
		webhooks.GET("/:webhookId", w.webhookController.GetWebhook)
		webhooks.PUT("/:webhookId", w.webhookController.UpdateWebhook)
		webhooks.DELETE("/:webhookId", w.webhookController.DeleteWebhook)

		// Webhook deliveries
		webhooks.GET("/:webhookId/deliveries", w.webhookController.ListDeliveries)
		webhooks.GET("/deliveries", w.webhookController.ListDeliveries)
		webhooks.GET("/deliveries/:deliveryId", w.webhookController.GetDelivery)
	}
}
