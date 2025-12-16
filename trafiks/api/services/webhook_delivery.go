package services

import (
	"encoding/json"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/trafiks/trafiks/api/dto"
	"github.com/trafiks/trafiks/api/repository"
	"github.com/trafiks/trafiks/models"
	"github.com/trafiks/trafiks/pkg"
)

type WebhookDeliveryServiceClient interface {
	ListDeliveries(c *gin.Context) (interface{}, *pkg.RestErr)
	GetDelivery(c *gin.Context) (interface{}, *pkg.RestErr)
}

type WebhookDeliveryService struct {
	logger       pkg.LoggerClient
	deliveryRepo repository.WebhookDeliveryRepoClient
	webhookRepo  repository.WebhookRepoClient
	projectRepo  repository.ProjectRepoClient
	restErr      pkg.RestErrClient
}

func NewWebhookDeliveryService(
	logger pkg.LoggerClient,
	deliveryRepo repository.WebhookDeliveryRepoClient,
	webhookRepo repository.WebhookRepoClient,
	projectRepo repository.ProjectRepoClient,
	restErr pkg.RestErrClient,
) WebhookDeliveryServiceClient {
	return &WebhookDeliveryService{
		logger:       logger,
		deliveryRepo: deliveryRepo,
		webhookRepo:  webhookRepo,
		projectRepo:  projectRepo,
		restErr:      restErr,
	}
}

func (s *WebhookDeliveryService) ListDeliveries(c *gin.Context) (interface{}, *pkg.RestErr) {
	user := GetUserFromContext(c)
	if user == nil {
		return nil, s.restErr.Unauthorized("unauthorized")
	}

	ctx := c.Request.Context()

	// Parse pagination
	pagination := &dto.Pagination{
		Page:  1,
		Limit: 20,
	}

	if pageStr := c.Query("page"); pageStr != "" {
		if parsedPage, err := strconv.Atoi(pageStr); err == nil && parsedPage > 0 {
			pagination.Page = parsedPage
		}
	}

	if limitStr := c.Query("limit"); limitStr != "" {
		if parsedLimit, err := strconv.Atoi(limitStr); err == nil && parsedLimit > 0 {
			if parsedLimit > 100 {
				parsedLimit = 100
			}
			pagination.Limit = parsedLimit
		}
	}

	filters := &repository.WebhookDeliveryFilters{}

	if status := c.Query("status"); status != "" {
		filters.Status = status
	}

	if eventType := c.Query("event_type"); eventType != "" {
		filters.EventType = eventType
	}

	if startTimeStr := c.Query("start_time"); startTimeStr != "" {
		if startTime, err := time.Parse(time.RFC3339, startTimeStr); err == nil {
			filters.StartTime = &startTime
		}
	}

	if endTimeStr := c.Query("end_time"); endTimeStr != "" {
		if endTime, err := time.Parse(time.RFC3339, endTimeStr); err == nil {
			filters.EndTime = &endTime
		}
	}

	webhookUID := c.Param("webhookId")
	if webhookUID == "" {
		webhookUID = c.Query("webhook_id")
	}

	var deliveries []*models.WebhookDelivery
	var total int64

	if webhookUID != "" {
		// Verify webhook exists and belongs to user
		webhook, err := s.webhookRepo.Find(ctx, &models.Webhook{UID: webhookUID})
		if err != nil {
			return nil, s.restErr.NotFound("webhook not found")
		}
		if webhook.UserID != user.ID {
			return nil, s.restErr.NotFound("webhook not found")
		}

		deliveries, err = s.deliveryRepo.FindMany(ctx, &models.WebhookDelivery{WebhookID: webhook.ID}, pagination, filters)
		if err != nil {
			s.logger.Errorf("failed to fetch webhook deliveries: %v", err)
			return nil, s.restErr.ServerError("failed to fetch webhook deliveries")
		}

		total, err = s.deliveryRepo.Count(ctx, &models.WebhookDelivery{WebhookID: webhook.ID}, filters)
		if err != nil {
			s.logger.Errorf("failed to count webhook deliveries: %v", err)
			return nil, s.restErr.ServerError("failed to count webhook deliveries")
		}
	} else {
		// Fetch all deliveries for user's webhooks
		// Get all user's webhooks first
		webhooks, err := s.webhookRepo.FindMany(ctx, &models.Webhook{UserID: user.ID})
		if err != nil {
			s.logger.Errorf("failed to fetch user webhooks: %v", err)
			return nil, s.restErr.ServerError("failed to fetch webhook deliveries")
		}

		if len(webhooks) == 0 {
			return gin.H{
				"deliveries": []gin.H{},
				"pagination": gin.H{
					"page":        pagination.Page,
					"limit":       pagination.Limit,
					"total":       0,
					"total_pages": 0,
				},
			}, nil
		}

		// Get webhook IDs
		webhookIDs := make([]uint, len(webhooks))
		for i, w := range webhooks {
			webhookIDs[i] = w.ID
		}

		// Note: This requires updating the repository to support filtering by multiple webhook IDs
		// For now, we'll fetch deliveries for the first webhook as a workaround
		// TODO: Update repository to support IN queries
		deliveries, err = s.deliveryRepo.FindMany(ctx, &models.WebhookDelivery{WebhookID: webhookIDs[0]}, pagination, filters)
		if err != nil {
			s.logger.Errorf("failed to fetch webhook deliveries: %v", err)
			return nil, s.restErr.ServerError("failed to fetch webhook deliveries")
		}

		total, err = s.deliveryRepo.Count(ctx, &models.WebhookDelivery{WebhookID: webhookIDs[0]}, filters)
		if err != nil {
			s.logger.Errorf("failed to count webhook deliveries: %v", err)
			return nil, s.restErr.ServerError("failed to count webhook deliveries")
		}
	}

	pagination.Total = total

	// Format deliveries for response
	result := make([]gin.H, 0, len(deliveries))
	for _, delivery := range deliveries {
		deliveryData := gin.H{
			"uid":             delivery.UID,
			"webhook_id":      delivery.WebhookID,
			"event_type":      delivery.EventType,
			"status":          string(delivery.Status),
			"retry_count":     delivery.RetryCount,
			"idempotency_key": delivery.IdempotencyKey,
			"created_at":      delivery.CreatedAt,
			"updated_at":      delivery.UpdatedAt,
		}

		// Add payload
		if len(delivery.Payload) > 0 {
			var payload map[string]interface{}
			if err := json.Unmarshal(delivery.Payload, &payload); err == nil {
				deliveryData["payload"] = payload
			}
		}

		// Add HTTP status code if available
		if delivery.HTTPStatusCode != nil {
			deliveryData["http_status_code"] = *delivery.HTTPStatusCode
		}

		// Add response body if available
		if delivery.ResponseBody != "" {
			deliveryData["response_body"] = delivery.ResponseBody
		}

		// Add error message if available
		if delivery.ErrorMessage != "" {
			deliveryData["error_message"] = delivery.ErrorMessage
		}

		// Add delivered at if available
		if delivery.DeliveredAt != nil {
			deliveryData["delivered_at"] = delivery.DeliveredAt
		}

		// Add next retry at if available
		if delivery.NextRetryAt != nil {
			deliveryData["next_retry_at"] = delivery.NextRetryAt
		}

		result = append(result, deliveryData)
	}

	return gin.H{
		"deliveries": result,
		"pagination": gin.H{
			"page":        pagination.Page,
			"limit":       pagination.Limit,
			"total":       pagination.Total,
			"total_pages": (int(pagination.Total) + pagination.Limit - 1) / pagination.Limit,
		},
	}, nil
}

func (s *WebhookDeliveryService) GetDelivery(c *gin.Context) (interface{}, *pkg.RestErr) {
	user := GetUserFromContext(c)
	if user == nil {
		return nil, s.restErr.Unauthorized("unauthorized")
	}

	deliveryUID := c.Param("deliveryId")
	if deliveryUID == "" {
		return nil, s.restErr.BadRequest("delivery ID is required")
	}

	ctx := c.Request.Context()

	delivery, err := s.deliveryRepo.Find(ctx, &models.WebhookDelivery{UID: deliveryUID})
	if err != nil {
		return nil, s.restErr.NotFound("delivery not found")
	}

	// Verify delivery belongs to user's webhook
	webhook, err := s.webhookRepo.Find(ctx, &models.Webhook{ID: delivery.WebhookID})
	if err != nil {
		return nil, s.restErr.NotFound("delivery not found")
	}

	if webhook.UserID != user.ID {
		return nil, s.restErr.NotFound("delivery not found")
	}

	deliveryData := gin.H{
		"uid":             delivery.UID,
		"webhook_id":      delivery.WebhookID,
		"event_type":      delivery.EventType,
		"status":          string(delivery.Status),
		"retry_count":     delivery.RetryCount,
		"idempotency_key": delivery.IdempotencyKey,
		"created_at":      delivery.CreatedAt,
		"updated_at":      delivery.UpdatedAt,
	}

	// Add payload
	if len(delivery.Payload) > 0 {
		var payload map[string]interface{}
		if err := json.Unmarshal(delivery.Payload, &payload); err == nil {
			deliveryData["payload"] = payload
		}
	}

	// Add HTTP status code if available
	if delivery.HTTPStatusCode != nil {
		deliveryData["http_status_code"] = *delivery.HTTPStatusCode
	}

	// Add response body if available
	if delivery.ResponseBody != "" {
		deliveryData["response_body"] = delivery.ResponseBody
	}

	// Add error message if available
	if delivery.ErrorMessage != "" {
		deliveryData["error_message"] = delivery.ErrorMessage
	}

	// Add delivered at if available
	if delivery.DeliveredAt != nil {
		deliveryData["delivered_at"] = delivery.DeliveredAt
	}

	// Add next retry at if available
	if delivery.NextRetryAt != nil {
		deliveryData["next_retry_at"] = delivery.NextRetryAt
	}

	return deliveryData, nil
}
