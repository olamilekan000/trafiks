package routes

import (
	"github.com/trafiks/trafiks/api/controller"
	"github.com/trafiks/trafiks/api/middleware"
	"github.com/trafiks/trafiks/pkg"
)

type Auth struct {
	logger pkg.LoggerClient

	handler        Router
	authController controller.AuthClient
	authValidator  middleware.AuthValidatorClient
}

func NewAuth(
	logger pkg.LoggerClient,
	handler Router,
	authController controller.AuthClient,
	authValidator middleware.AuthValidatorClient,
) Auth {
	return Auth{
		logger:         logger,
		handler:        handler,
		authController: authController,
		authValidator:  authValidator,
	}
}

func (a *Auth) Setup() {
	a.handler.POST("/auth/login", a.authController.Login)
	a.handler.POST("/auth/logout", a.authValidator.ValidateUser, a.authController.Logout)
	a.handler.POST("/auth/signup", a.authController.Signup)
	a.handler.GET("/auth/verify", a.authController.Verify)
	a.handler.POST("/auth/forgot-password", a.authController.ForgotPassword)
	a.handler.POST("/auth/reset-password", a.authController.ResetPassword)
}
