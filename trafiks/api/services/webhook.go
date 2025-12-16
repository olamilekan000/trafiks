package services

import (
	"context"
	"encoding/json"
	"time"

	"gorm.io/datatypes"

	"github.com/trafiks/trafiks/api/repository"
	"github.com/trafiks/trafiks/internal/queue"
	"github.com/trafiks/trafiks/models"
	"github.com/trafiks/trafiks/pkg"
)

type WebhookEvent struct {
	Event     string                 `json:"event"`
	Timestamp time.Time              `json:"timestamp"`
	UserID    uint                   `json:"user_id"`
	WebhookID uint                   `json:"webhook_id"`
	Data      map[string]interface{} `json:"data"`
}

type WebhookEventSender interface {
	SendEvent(webhookID uint, eventType string, data map[string]interface{})
}

type WebhookService struct {
	logger       pkg.LoggerClient
	webhookRepo  repository.WebhookRepoClient
	deliveryRepo repository.WebhookDeliveryRepoClient
	projectRepo  repository.ProjectRepoClient
	queue        queue.Queue
}

func NewWebhookService(
	logger pkg.LoggerClient,
	webhookRepo repository.WebhookRepoClient,
	deliveryRepo repository.WebhookDeliveryRepoClient,
	projectRepo repository.ProjectRepoClient,
	q queue.Queue,
) WebhookEventSender {
	return &WebhookService{
		logger:       logger,
		webhookRepo:  webhookRepo,
		deliveryRepo: deliveryRepo,
		projectRepo:  projectRepo,
		queue:        q,
	}
}

// SendEvent sends a webhook event to all active webhooks for a user that have the event enabled
// It creates webhook deliveries and pushes them to the queue for processing by the agent
// projectUID is optional and only included for project-specific events
func (s *WebhookService) SendEvent(webhookID uint, eventType string, data map[string]interface{}) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	webhook, err := s.webhookRepo.Find(
		ctx, &models.Webhook{ID: webhookID, IsActive: pkg.BoolPtr(true)})
	if err != nil {
		s.logger.Warnf("failed to fetch webhook %v", err)
		return
	}

	eventData := make(map[string]interface{})
	for k, v := range data {
		eventData[k] = v
	}

	if !s.isEventEnabled(webhook, eventType) {
		s.logger.Warnf("event %s is not enabled for webhook %s", eventType, webhook.UID)
		return
	}

	event := WebhookEvent{
		Event:     eventType,
		Timestamp: time.Now(),
		UserID:    webhook.UserID,
		WebhookID: webhook.ID,
		Data:      eventData,
	}

	go s.createAndQueueDelivery(webhook, eventType, event)
}

func (s *WebhookService) createAndQueueDelivery(webhook *models.Webhook, eventType string, event WebhookEvent) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(10*time.Second))
	defer cancel()

	payloadJSON, err := json.Marshal(event)
	if err != nil {
		s.logger.Errorf("failed to marshal webhook event: %v", err)
		return
	}

	idempotencyKey := pkg.GenerateUUIDV7()
	delivery := &models.WebhookDelivery{
		WebhookID:      webhook.ID,
		UserID:         webhook.UserID,
		EventType:      eventType,
		Payload:        datatypes.JSON(payloadJSON),
		Status:         models.DeliveryStatusPending,
		IdempotencyKey: idempotencyKey,
	}

	if err := s.deliveryRepo.Create(ctx, delivery); err != nil {
		s.logger.Errorf("failed to create webhook delivery: %v", err)
		return
	}

	queueData := map[string]interface{}{
		"delivery_id": delivery.ID,
		"webhook_id":  webhook.ID,
		"event_type":  eventType,
		"payload":     delivery.Payload,
	}

	jsonData, err := json.Marshal(queueData)
	if err != nil {
		s.logger.Errorf("failed to marshal queue data: %v", err)
		return
	}

	if err := s.queue.Push(ctx, "webhook:deliveries", jsonData); err != nil {
		s.logger.Errorf("failed to push webhook delivery to queue: %v", err)
		s.deliveryRepo.Updates(ctx, delivery, map[string]interface{}{
			"status":        models.DeliveryStatusFailed,
			"error_message": "failed to queue delivery",
		})
		return
	}

	s.logger.Infof("queued webhook delivery %s for webhook %s", delivery.UID, webhook.UID)
}

func (s *WebhookService) isEventEnabled(webhook *models.Webhook, eventType string) bool {
	var enabledEvents map[string]bool
	if err := json.Unmarshal(webhook.EnabledEvents, &enabledEvents); err != nil {
		s.logger.Warnf("failed to unmarshal enabled events for webhook %s: %v", webhook.UID, err)
		return false
	}

	enabled, exists := enabledEvents[eventType]
	return exists && enabled
}

// isProjectSpecificEvent determines if an event type is project-specific
// Project-specific events must always have a project link
func (s *WebhookService) isProjectSpecificEvent(eventType string) bool {
	projectEvents := map[string]bool{
		EventProjectActivated:    true,
		EventProjectDeactivated:  true,
		EventCacheMiss:           true,
		EventCacheHit:            true,
		EventUpstreamUnreachable: true,
		EventUpstreamTimeout:     true,
		EventRequestFailed:       true,
		EventErrorRateHigh:       true,
	}
	return projectEvents[eventType]
}
