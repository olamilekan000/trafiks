package routes

import (
	"go.uber.org/fx"

	"github.com/trafiks/trafiks/pkg"
)

var Module = fx.Options(
	fx.Provide(NewRouter),
	fx.Provide(NewRoute),
	fx.Provide(NewRoutes),
	fx.Provide(NewAuth),
	fx.Provide(NewUser),
	fx.Provide(NewProxy),
	fx.Provide(NewAPIKey),
	fx.Provide(NewProject),
	fx.Provide(NewService),
	fx.Provide(NewProxyRequestLog),
	fx.Provide(NewMetrics),
	fx.Provide(NewDashboard),
	fx.Provide(NewWebhook),
)

type Routes []IRoute

type IRoute interface {
	Setup()
}

type Route struct {
	handler Router
	logger  pkg.LoggerClient
}

func NewRoute(logger pkg.LoggerClient, handler Router) Route {
	return Route{
		logger:  logger,
		handler: handler,
	}
}

func NewRoutes(
	auth Auth,
	user User,
	proxy Proxy,
	apiKey APIKey,
	project Project,
	service Service,
	proxyRequestLog ProxyRequestLog,
	metrics Metrics,
	dashboard Dashboard,
	webhook Webhook,
) Routes {
	return Routes{
		&dashboard, // Register dashboard before proxy (which uses NoRoute)
		&auth,
		&user,
		&proxy,
		&apiKey,
		&project,
		&service,
		&proxyRequestLog,
		&metrics,
		&webhook,
	}
}

func (r Routes) Setup() {
	for _, route := range r {
		route.Setup()
	}
}
