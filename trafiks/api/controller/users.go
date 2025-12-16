package controller

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/trafiks/trafiks/api/dto"
	"github.com/trafiks/trafiks/api/services"
	"github.com/trafiks/trafiks/pkg"
)

type UserClient interface {
	Profile(c *gin.Context)
	ChangePassword(c *gin.Context)
	UpdateProfile(c *gin.Context)
}

type User struct {
	logger  pkg.LoggerClient
	authSvc services.UserClient
}

func NewUser(
	logger pkg.LoggerClient,
	userSvc services.UserClient,
) UserClient {
	return &User{
		logger:  logger,
		authSvc: userSvc,
	}
}

func (u *User) Profile(c *gin.Context) {
	resp, err := u.authSvc.Profile(c)
	if err != nil {
		c.JSON(err.StatusCode, err)

		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "ok",
		"data":    resp,
		"success": true,
	})
}

func (u *User) ChangePassword(c *gin.Context) {
	var pwds dto.ChangePwd

	if err := c.ShouldBindJSON(&pwds); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"message": translateValidationErrors(err),
		})

		return
	}

	resp, err := u.authSvc.ChangePassword(c, pwds)
	if err != nil {
		c.JSON(err.StatusCode, err)

		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "ok",
		"data":    resp,
		"success": true,
	})
}

func (u *User) UpdateProfile(c *gin.Context) {
	var profile dto.UpdateProfile

	if err := c.ShouldBindJSON(&profile); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"message": err.Error(),
		})

		return
	}

	resp, err := u.authSvc.UpdateProfile(c, profile)
	if err != nil {
		c.JSON(err.StatusCode, err)

		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "ok",
		"data":    resp,
		"success": true,
	})
}
