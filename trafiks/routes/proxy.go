package routes

import (
	"github.com/trafiks/trafiks/api/repository"
	"github.com/trafiks/trafiks/api/services"
	"github.com/trafiks/trafiks/cfg"
	"github.com/trafiks/trafiks/pkg"
	"github.com/trafiks/trafiks/pkg/cache"
	"github.com/trafiks/trafiks/pkg/source"
)

type Proxy struct {
	logger         pkg.LoggerClient
	handler        Router
	config         *cfg.Config
	serviceRepo    repository.ServiceRepoClient
	projectRepo    repository.ProjectRepoClient
	requestLogRepo repository.ProxyRequestLogRepoClient
	webhookRepo    repository.WebhookRepoClient
	cacheAdapter   cache.Cache
	streamHub      pkg.MetricsStreamHubClient
	webhookService services.WebhookEventSender
	sourceManager  *source.ServiceSourceManager
}

func NewProxy(
	logger pkg.LoggerClient,
	handler Router,
	config *cfg.Config,
	serviceRepo repository.ServiceRepoClient,
	projectRepo repository.ProjectRepoClient,
	requestLogRepo repository.ProxyRequestLogRepoClient,
	redisClient cache.RedisClient,
	streamHub pkg.MetricsStreamHubClient,
	webhookService services.WebhookEventSender,
	sourceManager *source.ServiceSourceManager,
	webhookRepo repository.WebhookRepoClient,
) Proxy {
	cacheAdapter := cache.AsCache(redisClient)
	return Proxy{
		logger:         logger,
		handler:        handler,
		config:         config,
		serviceRepo:    serviceRepo,
		projectRepo:    projectRepo,
		requestLogRepo: requestLogRepo,
		cacheAdapter:   cacheAdapter,
		streamHub:      streamHub,
		webhookService: webhookService,
		sourceManager:  sourceManager,
		webhookRepo:    webhookRepo,
	}
}

func (p *Proxy) Setup() {
	serviceProxyHandler := services.ServiceProxyHandler(
		p.logger,
		p.serviceRepo,
		p.projectRepo,
		p.requestLogRepo,
		p.cacheAdapter,
		p.config.AppBaseURL,
		p.config.TLSPort,
		p.streamHub,
		p.webhookService,
		p.sourceManager,
		p.webhookRepo,
	)
	p.handler.NoRoute(serviceProxyHandler)
}
