package services

import (
	"fmt"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"

	"github.com/trafiks/trafiks/api/dto"
	"github.com/trafiks/trafiks/api/repository"
	"github.com/trafiks/trafiks/models"
	"github.com/trafiks/trafiks/pkg"
)

type APIKeyClient interface {
	GenerateAPIKey(c *gin.Context, req dto.GenerateAPIKeyRequest) (interface{}, *pkg.RestErr)
	ListAPIKeys(c *gin.Context) (interface{}, *pkg.RestErr)
	RevokeAPIKey(c *gin.Context, keyID string) (interface{}, *pkg.RestErr)
}

type APIKey struct {
	logger         pkg.LoggerClient
	apiKeyRepo     repository.APIKeyRepoClient
	userRepo       repository.UserRepoClient
	webhookRepo    repository.WebhookRepoClient
	restErr        pkg.RestErrClient
	webhookService WebhookEventSender
}

func NewAPIKey(
	logger pkg.LoggerClient,
	apiKeyRepo repository.APIKeyRepoClient,
	userRepo repository.UserRepoClient,
	webhookRepo repository.WebhookRepoClient,
	restErr pkg.RestErrClient,
	webhookService WebhookEventSender,
) APIKeyClient {
	return &APIKey{
		logger:         logger,
		apiKeyRepo:     apiKeyRepo,
		userRepo:       userRepo,
		webhookRepo:    webhookRepo,
		restErr:        restErr,
		webhookService: webhookService,
	}
}

func (s *APIKey) GenerateAPIKey(c *gin.Context, req dto.GenerateAPIKeyRequest) (interface{}, *pkg.RestErr) {
	user := GetUserFromContext(c)
	if user == nil {
		return nil, s.restErr.Unauthorized("unauthorized")
	}

	if strings.TrimSpace(req.Name) == "" {
		return nil, s.restErr.BadRequest("name is required")
	}

	apiKeys, err := s.apiKeyRepo.FindMany(
		c.Request.Context(),
		&models.APIKey{UserID: user.ID, RevokedAt: nil, ExpiresAt: nil},
	)
	if err != nil {
		s.logger.Errorf("failed to find API keys: %v", err)
		return nil, s.restErr.ServerError("failed to find API keys")
	}

	for _, key := range apiKeys {
		if key.IsActive() {
			return nil, s.restErr.BadRequest("you already have an active API key. Please revoke it before creating a new one")
		}
	}

	randomBytes, err := pkg.GenerateToken(32)
	if err != nil {
		s.logger.Errorf("failed to generate API key token: %v", err)
		return nil, s.restErr.ServerError("failed to generate API key")
	}

	apiKeyValue := fmt.Sprintf("tfk_%s", randomBytes)
	keyPrefix := apiKeyValue[:11]

	keyHash, err := bcrypt.GenerateFromPassword([]byte(apiKeyValue), bcrypt.DefaultCost)
	if err != nil {
		s.logger.Errorf("failed to hash API key: %v", err)
		return nil, s.restErr.ServerError("failed to secure API key")
	}

	apiKey := &models.APIKey{
		UserID:    user.ID,
		KeyHash:   string(keyHash),
		KeyPrefix: keyPrefix,
		Name:      strings.TrimSpace(req.Name),
	}

	if req.ExpiresInDays > 0 {
		expiresAt := time.Now().Add(time.Duration(req.ExpiresInDays) * 24 * time.Hour)
		apiKey.ExpiresAt = &expiresAt
	}

	if err := s.apiKeyRepo.Create(c.Request.Context(), apiKey); err != nil {
		s.logger.Errorf("failed to create API key: %v", err)
		return nil, s.restErr.ServerError("failed to create API key")
	}

	if s.webhookService != nil {
		webhook, err := s.webhookRepo.Find(
			c.Request.Context(), &models.Webhook{UserID: user.ID, IsActive: pkg.BoolPtr(true)})
		if err != nil {
			s.logger.Warnf("failed to fetch webhook: %v", err)
		}

		if webhook != nil {
			go s.webhookService.SendEvent(
				webhook.ID,
				EventSecretKeyCreated,
				map[string]interface{}{
					"key_id":     apiKey.UID,
					"key_prefix": apiKey.KeyPrefix,
					"name":       apiKey.Name,
					"user_id":    user.ID,
					"webhook_id": webhook.ID,
				},
			)
		}
	}

	return gin.H{
		"api_key":    apiKeyValue,
		"uid":        apiKey.UID,
		"key_prefix": keyPrefix,
		"name":       apiKey.Name,
		"created_at": apiKey.CreatedAt,
		"expires_at": apiKey.ExpiresAt,
		"message":    "API key generated successfully. Store it securely as it won't be shown again.",
	}, nil
}

func (s *APIKey) ListAPIKeys(c *gin.Context) (interface{}, *pkg.RestErr) {
	user := GetUserFromContext(c)
	if user == nil {
		return nil, s.restErr.Unauthorized("unauthorized")
	}

	apiKeys, err := s.apiKeyRepo.FindMany(c.Request.Context(), &models.APIKey{UserID: user.ID})
	if err != nil {
		s.logger.Errorf("failed to fetch API keys: %v", err)
		return nil, s.restErr.ServerError("failed to fetch API keys")
	}

	result := make([]gin.H, 0, len(apiKeys))
	for _, key := range apiKeys {
		result = append(result, gin.H{
			"key_id":       key.UID,
			"key_prefix":   key.KeyPrefix,
			"name":         key.Name,
			"created_at":   key.CreatedAt,
			"last_used_at": key.LastUsedAt,
			"expires_at":   key.ExpiresAt,
			"revoked_at":   key.RevokedAt,
			"is_active":    key.IsActive(),
		})
	}

	return gin.H{
		"api_keys": result,
	}, nil
}

func (s *APIKey) RevokeAPIKey(c *gin.Context, keyID string) (interface{}, *pkg.RestErr) {
	user := GetUserFromContext(c)
	if user == nil {
		return nil, s.restErr.Unauthorized("unauthorized")
	}

	apiKey, err := s.apiKeyRepo.Find(c.Request.Context(), &models.APIKey{
		UID:    keyID,
		UserID: user.ID,
	})
	if err != nil {
		return nil, s.restErr.NotFound("API key not found")
	}

	if apiKey.RevokedAt != nil {
		return nil, s.restErr.BadRequest("API key is already revoked")
	}

	now := time.Now()
	if err := s.apiKeyRepo.Updates(c.Request.Context(), apiKey, map[string]interface{}{
		"revoked_at": &now,
	}); err != nil {
		s.logger.Errorf("failed to revoke API key: %v", err)
		return nil, s.restErr.ServerError("failed to revoke API key")
	}

	if s.webhookService != nil {
		webhook, err := s.webhookRepo.Find(
			c.Request.Context(), &models.Webhook{UserID: user.ID, IsActive: pkg.BoolPtr(true)})
		if err != nil {
			s.logger.Warnf("failed to fetch webhook: %v", err)
		}

		if webhook != nil {
			go s.webhookService.SendEvent(
				webhook.ID,
				EventSecretKeyDeactivated,
				map[string]interface{}{
					"key_id":     apiKey.UID,
					"key_prefix": apiKey.KeyPrefix,
					"name":       apiKey.Name,
					"user_id":    user.ID,
				},
			)
		}
	}

	return gin.H{
		"message": "API key revoked successfully",
	}, nil
}
