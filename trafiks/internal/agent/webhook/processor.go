package webhook

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"gorm.io/datatypes"

	"github.com/trafiks/trafiks/api/repository"
	"github.com/trafiks/trafiks/internal/queue"
	"github.com/trafiks/trafiks/models"
	"github.com/trafiks/trafiks/pkg"
)

type WebhookProcessorClient interface {
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
}

type WebhookProcessor struct {
	logger           pkg.LoggerClient
	webhookRepo      repository.WebhookRepoClient
	deliveryRepo     repository.WebhookDeliveryRepoClient
	queue            queue.Queue
	httpClient       *http.Client
	workerCount      int
	processingTicker *time.Ticker
}

func NewWebhookProcessor(
	logger pkg.LoggerClient,
	webhookRepo repository.WebhookRepoClient,
	deliveryRepo repository.WebhookDeliveryRepoClient,
	q queue.Queue,
) WebhookProcessorClient {
	return &WebhookProcessor{
		logger:       logger,
		webhookRepo:  webhookRepo,
		deliveryRepo: deliveryRepo,
		queue:        q,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		workerCount: 5,
	}
}

func (p *WebhookProcessor) Start(ctx context.Context) error {
	p.logger.Info("Starting webhook processor...")

	for i := 0; i < p.workerCount; i++ {
		go p.worker(ctx, i)
	}

	p.processingTicker = time.NewTicker(1 * time.Minute)
	go p.retryProcessor(ctx)

	<-ctx.Done()
	p.logger.Info("Webhook processor stopping...")

	if p.processingTicker != nil {
		p.processingTicker.Stop()
	}

	return nil
}

func (p *WebhookProcessor) Stop(ctx context.Context) error {
	p.logger.Info("Stopping webhook processor...")

	if p.processingTicker != nil {
		p.processingTicker.Stop()
	}

	return nil
}

func (p *WebhookProcessor) worker(ctx context.Context, workerID int) {
	p.logger.Infof("Webhook worker %d started", workerID)

	for {
		select {
		case <-ctx.Done():
			p.logger.Infof("Webhook worker %d stopping", workerID)
			return
		default:
			data, err := p.queue.PopWithTimeout(ctx, "webhook:deliveries", 5)
			if err != nil {
				p.logger.Errorf("Worker %d: Error popping from queue: %v", workerID, err)
				time.Sleep(1 * time.Second)
				continue
			}

			if data == nil {
				continue
			}

			if err := p.processDelivery(ctx, data); err != nil {
				p.logger.Errorf("Worker %d: Error processing delivery: %v", workerID, err)
			}
		}
	}
}

func (p *WebhookProcessor) processDelivery(ctx context.Context, data []byte) error {
	p.logger.Infof("Processing delivery: %s", string(data))

	var msg map[string]interface{}
	if err := json.Unmarshal(data, &msg); err != nil {
		return fmt.Errorf("failed to unmarshal delivery message: %w", err)
	}

	deliveryID, ok := msg["delivery_id"].(float64)
	if !ok {
		return fmt.Errorf("invalid delivery_id in message")
	}

	delivery, err := p.deliveryRepo.Find(ctx, &models.WebhookDelivery{ID: uint(deliveryID)})
	if err != nil {
		return fmt.Errorf("failed to find delivery %v: %w", deliveryID, err)
	}

	p.deliveryRepo.Updates(ctx, delivery, map[string]interface{}{
		"status": models.DeliveryStatusProcessing,
	})

	webhook, err := p.webhookRepo.Find(ctx, &models.Webhook{ID: delivery.WebhookID, IsActive: pkg.BoolPtr(true)})
	if err != nil {
		p.markDeliveryFailed(ctx, delivery, fmt.Sprintf("Webhook not found: %v", err))
		return err
	}

	if !webhook.Active() {
		p.markDeliveryFailed(ctx, delivery, "Webhook is inactive")
		return fmt.Errorf("webhook is inactive")
	}

	statusCode, responseBody, err := p.sendWebhook(
		webhook,
		delivery.Payload,
		delivery.EventType,
		delivery.IdempotencyKey,
	)

	if err != nil {
		p.markDeliveryFailed(ctx, delivery, err.Error())
		p.scheduleRetry(ctx, delivery)
		return err
	}

	if statusCode >= 200 && statusCode < 300 {
		now := time.Now()
		p.deliveryRepo.Updates(ctx, delivery, map[string]interface{}{
			"status":           models.DeliveryStatusSuccess,
			"http_status_code": statusCode,
			"response_body":    responseBody,
			"delivered_at":     &now,
		})
		p.logger.Infof("Successfully delivered webhook %s (delivery %s)", webhook.UID, delivery.UID)

		return nil
	}

	p.markDeliveryFailed(ctx, delivery, fmt.Sprintf("HTTP %d: %s", statusCode, responseBody))
	p.scheduleRetry(ctx, delivery)

	return fmt.Errorf("failed to deliver webhook: %d", statusCode)
}

func (p *WebhookProcessor) sendWebhook(webhook *models.Webhook, payload datatypes.JSON, eventType, idempotencyKey string) (int, string, error) {
	var payloadData map[string]interface{}
	if err := json.Unmarshal(payload, &payloadData); err != nil {
		return 0, "", fmt.Errorf("failed to unmarshal payload: %w", err)
	}

	payloadBytes, err := json.Marshal(payloadData)
	if err != nil {
		return 0, "", fmt.Errorf("failed to marshal payload: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, webhook.URL, bytes.NewBuffer(payloadBytes))
	if err != nil {
		return 0, "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Trafiks-Event", eventType)
	req.Header.Set("X-Trafiks-Signature", p.generateHMACSignature(payloadBytes, webhook.Secret))
	req.Header.Set("X-Idempotency-Key", idempotencyKey)
	req.Header.Set("User-Agent", "Trafiks-Webhook/1.0")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return 0, "", fmt.Errorf("failed to send webhook: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes := make([]byte, 10240)
	n, _ := resp.Body.Read(bodyBytes)
	responseBody := string(bodyBytes[:n])

	return resp.StatusCode, responseBody, nil
}

func (p *WebhookProcessor) generateHMACSignature(payload []byte, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	return "sha256=" + hex.EncodeToString(mac.Sum(nil))
}

func (p *WebhookProcessor) markDeliveryFailed(ctx context.Context, delivery *models.WebhookDelivery, errorMsg string) {
	p.deliveryRepo.Updates(ctx, delivery, map[string]interface{}{
		"status":        models.DeliveryStatusFailed,
		"error_message": errorMsg,
		"retry_count":   delivery.RetryCount + 1,
	})
}

func (p *WebhookProcessor) scheduleRetry(ctx context.Context, delivery *models.WebhookDelivery) {
	if !delivery.IsRetryable() {
		return
	}

	backoffDurations := []time.Duration{
		1 * time.Minute,
		5 * time.Minute,
		15 * time.Minute,
		1 * time.Hour,
		6 * time.Hour,
	}

	retryCount := delivery.RetryCount
	if retryCount >= len(backoffDurations) {
		retryCount = len(backoffDurations) - 1
	}

	nextRetry := time.Now().Add(backoffDurations[retryCount])
	p.deliveryRepo.Updates(ctx, delivery, map[string]interface{}{
		"next_retry_at": &nextRetry,
	})
}

func (p *WebhookProcessor) retryProcessor(ctx context.Context) {
	p.logger.Info("Starting retry processor...")

	for {
		select {
		case <-ctx.Done():
			return
		case <-p.processingTicker.C:
			deliveries, err := p.deliveryRepo.FindRetryable(ctx, 100)
			if err != nil {
				p.logger.Errorf("Error finding retryable deliveries: %v", err)
				continue
			}

			for _, delivery := range deliveries {
				if delivery.Status != models.DeliveryStatusFailed {
					continue
				}

				p.deliveryRepo.Updates(ctx, delivery, map[string]interface{}{
					"status": models.DeliveryStatusPending,
				})

				if err := p.requeueDelivery(ctx, delivery); err != nil {
					p.logger.Errorf("Error re-queuing delivery %s: %v", delivery.UID, err)
					p.deliveryRepo.Updates(ctx, delivery, map[string]interface{}{
						"status": models.DeliveryStatusFailed,
					})
				}
			}
		}
	}
}

func (p *WebhookProcessor) requeueDelivery(ctx context.Context, delivery *models.WebhookDelivery) error {
	payload := map[string]interface{}{
		"delivery_id": delivery.ID,
		"webhook_id":  delivery.WebhookID,
		"event_type":  delivery.EventType,
		"payload":     delivery.Payload,
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal delivery data: %w", err)
	}

	return p.queue.Push(ctx, "webhook:deliveries", jsonData)
}
