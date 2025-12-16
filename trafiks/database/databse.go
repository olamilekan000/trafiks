package database

import (
	"github.com/trafiks/trafiks/database/postgres"
	"go.uber.org/fx"
)

var Module = fx.Options(
	fx.Provide(postgres.NewDatabase),
)
