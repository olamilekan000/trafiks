package main

import (
	"go.uber.org/fx"

	"github.com/trafiks/trafiks/internal/agent"
)

func main() {
	fx.New(agent.Module).Run()
}
