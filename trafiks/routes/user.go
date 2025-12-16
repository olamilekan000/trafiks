package routes

import (
	"github.com/trafiks/trafiks/api/controller"
	"github.com/trafiks/trafiks/api/middleware"
	"github.com/trafiks/trafiks/pkg"
)

type User struct {
	logger pkg.LoggerClient

	handler        Router
	userController controller.UserClient
	authValidator  middleware.AuthValidatorClient
}

func NewUser(
	logger pkg.LoggerClient,
	handler Router,
	userController controller.UserClient,
	authValidator middleware.AuthValidatorClient,
) User {
	return User{
		logger:         logger,
		handler:        handler,
		userController: userController,
		authValidator:  authValidator,
	}
}

func (u *User) Setup() {
	user := u.handler.Group("/users")

	user.Use(u.authValidator.ValidateUser)
	{
		user.GET("/data", u.userController.Profile)
		user.PUT("/data", u.userController.UpdateProfile)
		user.POST("/change-password", u.userController.ChangePassword)
	}
}
