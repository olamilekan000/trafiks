package main

import (
	"go.uber.org/fx"

	"github.com/trafiks/trafiks/app"
)

func main() {
	fx.New(app.Module).Run()
}
