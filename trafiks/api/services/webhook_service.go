package services

import (
	"encoding/json"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/trafiks/trafiks/api/dto"
	"github.com/trafiks/trafiks/api/repository"
	"github.com/trafiks/trafiks/models"
	"github.com/trafiks/trafiks/pkg"
)

type WebhookServiceClient interface {
	CreateWebhook(c *gin.Context, req dto.CreateWebhookRequest) (interface{}, *pkg.RestErr)
	ListWebhooks(c *gin.Context) (interface{}, *pkg.RestErr)
	GetWebhook(c *gin.Context) (interface{}, *pkg.RestErr)
	UpdateWebhook(c *gin.Context, req dto.UpdateWebhookRequest) (interface{}, *pkg.RestErr)
	DeleteWebhook(c *gin.Context) (interface{}, *pkg.RestErr)
}

type WebhookServiceStruct struct {
	logger      pkg.LoggerClient
	webhookRepo repository.WebhookRepoClient
	restErr     pkg.RestErrClient
}

func NewWebhookServiceStruct(
	logger pkg.LoggerClient,
	webhookRepo repository.WebhookRepoClient,
	restErr pkg.RestErrClient,
) WebhookServiceClient {
	return &WebhookServiceStruct{
		logger:      logger,
		webhookRepo: webhookRepo,
		restErr:     restErr,
	}
}

func (s *WebhookServiceStruct) CreateWebhook(c *gin.Context, req dto.CreateWebhookRequest) (interface{}, *pkg.RestErr) {
	user := GetUserFromContext(c)
	if user == nil {
		return nil, s.restErr.Unauthorized("unauthorized")
	}

	if err := req.Validate(); err != nil {
		return nil, s.restErr.BadRequest(err.Error())
	}

	ctx := c.Request.Context()

	// Generate secret if not provided
	secret := strings.TrimSpace(req.Secret)
	if secret == "" {
		randomBytes, err := pkg.GenerateToken(32)
		if err != nil {
			s.logger.Errorf("failed to generate webhook secret: %v", err)
			return nil, s.restErr.ServerError("failed to generate webhook secret")
		}
		secret = randomBytes
	}

	webhook := req.ToWebhook(user.ID)
	webhook.Secret = secret // Use generated or provided secret

	if err := s.webhookRepo.Create(ctx, webhook); err != nil {
		s.logger.Errorf("failed to create webhook: %v", err)
		return nil, s.restErr.ServerError("failed to create webhook")
	}

	var enabledEvents map[string]bool
	_ = json.Unmarshal(webhook.EnabledEvents, &enabledEvents)

	return gin.H{
		"uid":            webhook.UID,
		"url":            webhook.URL,
		"secret":         secret, // Return the secret so user can copy it (only shown once)
		"enabled_events": enabledEvents,
		"is_active":      webhook.IsActive,
		"created_at":     webhook.CreatedAt,
		"updated_at":     webhook.UpdatedAt,
	}, nil
}

func (s *WebhookServiceStruct) ListWebhooks(c *gin.Context) (interface{}, *pkg.RestErr) {
	user := GetUserFromContext(c)
	if user == nil {
		return nil, s.restErr.Unauthorized("unauthorized")
	}

	ctx := c.Request.Context()

	webhooks, err := s.webhookRepo.FindMany(ctx, &models.Webhook{UserID: user.ID})
	if err != nil {
		s.logger.Errorf("failed to fetch webhooks: %v", err)
		return nil, s.restErr.ServerError("failed to fetch webhooks")
	}

	result := make([]gin.H, 0, len(webhooks))
	for _, webhook := range webhooks {
		var enabledEvents map[string]bool
		_ = json.Unmarshal(webhook.EnabledEvents, &enabledEvents)

		result = append(result, gin.H{
			"uid":            webhook.UID,
			"url":            webhook.URL,
			"enabled_events": enabledEvents,
			"is_active":      webhook.IsActive,
			"created_at":     webhook.CreatedAt,
			"updated_at":     webhook.UpdatedAt,
		})
	}

	return gin.H{
		"webhooks": result,
	}, nil
}

func (s *WebhookServiceStruct) GetWebhook(c *gin.Context) (interface{}, *pkg.RestErr) {
	user := GetUserFromContext(c)
	if user == nil {
		return nil, s.restErr.Unauthorized("unauthorized")
	}

	webhookUID := c.Param("webhookId")
	if webhookUID == "" {
		return nil, s.restErr.BadRequest("webhook ID is required")
	}

	ctx := c.Request.Context()

	webhook, err := s.webhookRepo.Find(ctx, &models.Webhook{UID: webhookUID})
	if err != nil {
		return nil, s.restErr.NotFound("webhook not found")
	}

	// Verify webhook belongs to user
	if webhook.UserID != user.ID {
		return nil, s.restErr.NotFound("webhook not found")
	}

	var enabledEvents map[string]bool
	_ = json.Unmarshal(webhook.EnabledEvents, &enabledEvents)

	return gin.H{
		"uid":            webhook.UID,
		"url":            webhook.URL,
		"enabled_events": enabledEvents,
		"is_active":      webhook.IsActive,
		"created_at":     webhook.CreatedAt,
		"updated_at":     webhook.UpdatedAt,
	}, nil
}

func (s *WebhookServiceStruct) UpdateWebhook(c *gin.Context, req dto.UpdateWebhookRequest) (interface{}, *pkg.RestErr) {
	user := GetUserFromContext(c)
	if user == nil {
		return nil, s.restErr.Unauthorized("unauthorized")
	}

	ctx := c.Request.Context()

	webhookUID := c.Param("webhookId")
	if webhookUID == "" {
		return nil, s.restErr.BadRequest("webhook ID is required")
	}

	if err := req.Validate(); err != nil {
		return nil, s.restErr.BadRequest(err.Error())
	}

	webhook, err := s.webhookRepo.Find(ctx, &models.Webhook{UID: webhookUID})
	if err != nil {
		return nil, s.restErr.NotFound("webhook not found")
	}

	// Verify webhook belongs to user
	if webhook.UserID != user.ID {
		return nil, s.restErr.NotFound("webhook not found")
	}

	hasUpdates := false

	if req.URL != "" {
		webhook.URL = req.URL
		hasUpdates = true
	}

	if req.Secret != "" {
		webhook.Secret = req.Secret
		hasUpdates = true
	}

	if len(req.EnabledEvents) > 0 {
		eventsJSON, _ := json.Marshal(req.EnabledEvents)
		webhook.EnabledEvents = eventsJSON
		hasUpdates = true
	}

	if req.IsActive != nil {
		webhook.IsActive = req.IsActive
		hasUpdates = true
	}

	if !hasUpdates {
		return nil, s.restErr.BadRequest("no fields to update")
	}

	if err := s.webhookRepo.Update(ctx, webhook); err != nil {
		s.logger.Errorf("failed to update webhook: %v", err)
		return nil, s.restErr.ServerError("failed to update webhook")
	}

	var enabledEvents map[string]bool
	_ = json.Unmarshal(webhook.EnabledEvents, &enabledEvents)

	return gin.H{
		"uid":            webhook.UID,
		"url":            webhook.URL,
		"enabled_events": enabledEvents,
		"is_active":      webhook.IsActive,
		"created_at":     webhook.CreatedAt,
		"updated_at":     webhook.UpdatedAt,
	}, nil
}

func (s *WebhookServiceStruct) DeleteWebhook(c *gin.Context) (interface{}, *pkg.RestErr) {
	user := GetUserFromContext(c)
	if user == nil {
		return nil, s.restErr.Unauthorized("unauthorized")
	}

	webhookUID := c.Param("webhookId")
	if webhookUID == "" {
		return nil, s.restErr.BadRequest("webhook ID is required")
	}

	ctx := c.Request.Context()

	webhook, err := s.webhookRepo.Find(ctx, &models.Webhook{UID: webhookUID})
	if err != nil {
		return nil, s.restErr.NotFound("webhook not found")
	}

	// Verify webhook belongs to user
	if webhook.UserID != user.ID {
		return nil, s.restErr.NotFound("webhook not found")
	}

	if err := s.webhookRepo.Delete(ctx, webhook); err != nil {
		s.logger.Errorf("failed to delete webhook: %v", err)
		return nil, s.restErr.ServerError("failed to delete webhook")
	}

	return gin.H{
		"message": "Webhook deleted successfully",
	}, nil
}
