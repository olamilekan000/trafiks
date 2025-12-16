package controller

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/trafiks/trafiks/api/dto"
	"github.com/trafiks/trafiks/api/services"
	"github.com/trafiks/trafiks/pkg"
)

type APIKeyClient interface {
	GenerateAPIKey(c *gin.Context)
	ListAPIKeys(c *gin.Context)
	RevokeAPIKey(c *gin.Context)
}

type APIKey struct {
	logger    pkg.LoggerClient
	apiKeySvc services.APIKeyClient
}

func NewAPIKey(
	logger pkg.LoggerClient,
	apiKeySvc services.APIKeyClient,
) APIKeyClient {
	return &APIKey{
		logger:    logger,
		apiKeySvc: apiKeySvc,
	}
}

func (ak *APIKey) GenerateAPIKey(c *gin.Context) {
	var req dto.GenerateAPIKeyRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"message": err.Error(),
		})
		return
	}

	resp, err := ak.apiKeySvc.GenerateAPIKey(c, req)
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

func (ak *APIKey) ListAPIKeys(c *gin.Context) {
	resp, err := ak.apiKeySvc.ListAPIKeys(c)
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

func (ak *APIKey) RevokeAPIKey(c *gin.Context) {
	keyID := c.Param("keyId")
	if keyID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"message": "key ID is required",
		})
		return
	}

	resp, err := ak.apiKeySvc.RevokeAPIKey(c, keyID)
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
