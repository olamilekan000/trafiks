package services

import (
	"go.uber.org/fx"
)

var Module = fx.Options(
	fx.Provide(NewAuth),
	fx.Provide(NewUser),
	fx.Provide(NewWebhookService),
	fx.Provide(NewWebhookServiceStruct),
	fx.Provide(NewWebhookDeliveryService),
	fx.Provide(NewAPIKey),
	fx.Provide(NewProject),
	fx.Provide(NewService),
	fx.Provide(NewProxyRequestLog),
	fx.Provide(NewMetrics),
	fx.Provide(NewTLSManager),
	fx.Provide(NewServiceSourceManager),
)
