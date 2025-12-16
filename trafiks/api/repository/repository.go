package repository

import "go.uber.org/fx"

var Module = fx.Options(
	fx.Provide(NewUserRepo),
	fx.Provide(NewAPIKeyRepo),
	fx.Provide(NewProjectRepo),
	fx.Provide(NewServiceRepo),
	fx.Provide(NewProxyRequestLogRepo),
	fx.Provide(NewWebhookRepo),
	fx.Provide(NewWebhookDeliveryRepo),
)
