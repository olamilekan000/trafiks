package pkg

import (
	"go.uber.org/fx"

	"github.com/trafiks/trafiks/pkg/cache"
)

var Module = fx.Options(
	fx.Provide(NewLogger),
	fx.Provide(NewRestErr),
	fx.Provide(cache.NewRedisClient),
	fx.Provide(NewMetricsStreamHub),
)
