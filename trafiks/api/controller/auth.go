package controller

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/trafiks/trafiks/api/dto"
	"github.com/trafiks/trafiks/api/services"
	"github.com/trafiks/trafiks/pkg"
)

type AuthClient interface {
	Login(c *gin.Context)
	Signup(c *gin.Context)
	Verify(c *gin.Context)
	ForgotPassword(c *gin.Context)
	ResetPassword(c *gin.Context)
	Logout(c *gin.Context)
}

type Auth struct {
	logger  pkg.LoggerClient
	authSvc services.AuthClient
}

func NewAuth(
	logger pkg.LoggerClient,
	authSvc services.AuthClient,
) AuthClient {
	return &Auth{
		logger:  logger,
		authSvc: authSvc,
	}
}

func (au *Auth) Login(c *gin.Context) {
	var req dto.AuthPayload

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"message": err.Error(),
		})

		return
	}

	resp, err := au.authSvc.LoginHandler(c, req)
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

func (au *Auth) Signup(c *gin.Context) {
	var req dto.SignupPayload

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"message": err.Error(),
		})

		return
	}

	resp, err := au.authSvc.Signup(c, req)
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

func (au *Auth) Verify(c *gin.Context) {
	resp, err := au.authSvc.Verify(c)
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

func (au *Auth) Logout(c *gin.Context) {
	au.authSvc.Logout(c)
}

func (au *Auth) ForgotPassword(c *gin.Context) {
	var req dto.ForgotPassword

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"message": err.Error(),
		})

		return
	}

	resp, err := au.authSvc.ForgotPassword(c, req)
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

func (au *Auth) ResetPassword(c *gin.Context) {
	var req dto.ResetPassword

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"message": translateValidationErrors(err),
		})

		return
	}

	resp, err := au.authSvc.ResetPassword(c, req)
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
