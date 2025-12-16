package services

import (
	"github.com/gin-gonic/gin"

	"github.com/trafiks/trafiks/models"
)

const AppUserContext = "sanmouser"

func GetUserFromContext(c *gin.Context) *models.User {
	u, ok := c.Get(AppUserContext)
	if !ok {
		return nil
	}

	currUser := u.(*models.User)

	return currUser
}
