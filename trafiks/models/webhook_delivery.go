package models

import (
	"time"

	"gorm.io/datatypes"
	"gorm.io/gorm"

	"github.com/trafiks/trafiks/pkg"
)

type WebhookDeliveryStatus string

const (
	DeliveryStatusPending    WebhookDeliveryStatus = "pending"
	DeliveryStatusProcessing WebhookDeliveryStatus = "processing"
	DeliveryStatusSuccess    WebhookDeliveryStatus = "success"
	DeliveryStatusFailed     WebhookDeliveryStatus = "failed"
)

type WebhookDelivery struct {
	gorm.Model
	ID             uint                  `json:"-" gorm:"primaryKey"`
	UID            string                `gorm:"not null;uniqueIndex"`
	WebhookID      uint                  `gorm:"not null;index"`
	UserID         uint                  `gorm:"not null;index"`
	EventType      string                `gorm:"not null"`
	Payload        datatypes.JSON        `gorm:"type:jsonb"`
	Status         WebhookDeliveryStatus `gorm:"not null;default:'pending'"`
	HTTPStatusCode *int                  `gorm:"type:integer"`
	ResponseBody   string                `gorm:"type:text"`
	ErrorMessage   string                `gorm:"type:text"`
	RetryCount     int                   `gorm:"default:0"`
	NextRetryAt    *time.Time
	IdempotencyKey string `gorm:"not null;index"`
	DeliveredAt    *time.Time
	CreatedAt      time.Time `json:"CreatedAt"`
	UpdatedAt      time.Time `json:"UpdatedAt"`

	Webhook Webhook `json:"-" gorm:"foreignKey:WebhookID"`
}

func (w *WebhookDelivery) BeforeCreate(tx *gorm.DB) (err error) {
	w.UID = pkg.GenerateUUIDV7()
	w.CreatedAt = time.Now().Local()
	w.UpdatedAt = time.Now().Local()
	return
}

func (w *WebhookDelivery) BeforeUpdate(tx *gorm.DB) (err error) {
	w.UpdatedAt = time.Now().Local()
	return
}

func (w *WebhookDelivery) IsRetryable() bool {
	return w.Status == DeliveryStatusFailed && w.RetryCount < 5
}

func (w *WebhookDelivery) ShouldRetry() bool {
	if !w.IsRetryable() {
		return false
	}
	if w.NextRetryAt == nil {
		return true
	}
	return time.Now().After(*w.NextRetryAt)
}
