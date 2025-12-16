package dto

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"gorm.io/datatypes"

	"github.com/trafiks/trafiks/models"
	"github.com/trafiks/trafiks/pkg"
)

var validWebhookEvents = map[string]bool{
	"upstream.unreachable":  true,
	"upstream.timeout":      true,
	"request.failed":        true,
	"cache.miss":            true,
	"project.activated":     true,
	"project.deactivated":   true,
	"secretkey.created":     true,
	"secretkey.deactivated": true,
	"error_rate.high":       true,
}

type CreateWebhookRequest struct {
	URL           string          `json:"url" binding:"required,url"`
	Secret        string          `json:"secret,omitempty"`
	EnabledEvents map[string]bool `json:"enabled_events" binding:"required"`
}

func (c *CreateWebhookRequest) Validate() error {
	if strings.TrimSpace(c.URL) == "" {
		return errors.New("url is required")
	}
	if c.Secret != "" && len(c.Secret) < 8 {
		return errors.New("secret must be at least 8 characters if provided")
	}
	if len(c.EnabledEvents) == 0 {
		return errors.New("at least one event must be enabled")
	}

	for eventType := range c.EnabledEvents {
		if !validWebhookEvents[eventType] {
			return fmt.Errorf("invalid event type: %s", eventType)
		}
	}

	return nil
}

func (c *CreateWebhookRequest) ToWebhook(userID uint) *models.Webhook {
	eventsJSON, _ := json.Marshal(c.EnabledEvents)
	return &models.Webhook{
		UserID:        userID,
		URL:           strings.TrimSpace(c.URL),
		Secret:        c.Secret,
		EnabledEvents: datatypes.JSON(eventsJSON),
		IsActive:      pkg.BoolPtr(true),
	}
}

type UpdateWebhookRequest struct {
	URL           string          `json:"url,omitempty"`
	Secret        string          `json:"secret,omitempty"`
	EnabledEvents map[string]bool `json:"enabled_events,omitempty"`
	IsActive      *bool           `json:"is_active,omitempty"`
}

func (u *UpdateWebhookRequest) Validate() error {
	if u.URL != "" && strings.TrimSpace(u.URL) == "" {
		return fmt.Errorf("URL cannot be empty")
	}
	if u.Secret != "" && len(u.Secret) < 8 {
		return fmt.Errorf("secret must be at least 8 characters")
	}

	if len(u.EnabledEvents) > 0 {
		for eventType := range u.EnabledEvents {
			if !validWebhookEvents[eventType] {
				return fmt.Errorf("invalid event type: %s", eventType)
			}
		}
	}

	return nil
}
