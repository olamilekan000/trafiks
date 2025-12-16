package services

import (
	"github.com/gin-gonic/gin"

	"github.com/trafiks/trafiks/models"
)

const AppUserContext = "sanmouser"
const AuthMethodContext = "auth_method"

func GetUserFromContext(c *gin.Context) *models.User {
	u, ok := c.Get(AppUserContext)
	if !ok {
		return nil
	}

	currUser := u.(*models.User)

	return currUser
}

func IsAPIKeyAuth(c *gin.Context) bool {
	method, ok := c.Get(AuthMethodContext)
	if !ok {
		return false
	}
	return method == "api_key"
}
