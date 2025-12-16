package routes

import (
	"github.com/trafiks/trafiks/api/controller"
	"github.com/trafiks/trafiks/api/middleware"
	"github.com/trafiks/trafiks/pkg"
)

type APIKey struct {
	logger           pkg.LoggerClient
	handler          Router
	apiKeyController controller.APIKeyClient
	authValidator    middleware.AuthValidatorClient
}

func NewAPIKey(
	logger pkg.LoggerClient,
	handler Router,
	apiKeyController controller.APIKeyClient,
	authValidator middleware.AuthValidatorClient,
) APIKey {
	return APIKey{
		logger:           logger,
		handler:          handler,
		apiKeyController: apiKeyController,
		authValidator:    authValidator,
	}
}

func (ak *APIKey) Setup() {
	apiKeys := ak.handler.Group("/api-keys")

	apiKeys.Use(ak.authValidator.ValidateUser)
	{
		apiKeys.POST("", ak.apiKeyController.GenerateAPIKey)
		apiKeys.GET("", ak.apiKeyController.ListAPIKeys)
		apiKeys.DELETE("/:keyId", ak.apiKeyController.RevokeAPIKey)
	}
}
