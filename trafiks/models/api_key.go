package models

import (
	"time"

	"gorm.io/gorm"

	"github.com/trafiks/trafiks/pkg"
)

// APIKey represents an API key for programmatic access
type APIKey struct {
	gorm.Model
	ID         uint       `json:"-" gorm:"primaryKey;unique"`
	UID        string     `gorm:"not null;uniqueIndex"`
	UserID     uint       `gorm:"not null;index"`
	KeyHash    string     `gorm:"not null;uniqueIndex"` // Hashed version of the key
	KeyPrefix  string     `gorm:"not null"`             // First 8 chars for display (e.g., "tfk_xxxx")
	Name       string     `gorm:"not null"`             // User-friendly name for the key
	LastUsedAt *time.Time `json:"LastUsedAt,omitempty"`
	ExpiresAt  *time.Time `json:"ExpiresAt,omitempty"`
	RevokedAt  *time.Time `json:"RevokedAt,omitempty"`
	CreatedAt  time.Time  `json:"CreatedAt"`
	UpdatedAt  time.Time  `json:"UpdatedAt"`

	// Relationships
	User User `json:"-" gorm:"foreignKey:UserID"`
}

// IsActive checks if the API key is active (not revoked and not expired)
func (k *APIKey) IsActive() bool {
	if k.RevokedAt != nil {
		return false
	}
	if k.ExpiresAt != nil && time.Now().After(*k.ExpiresAt) {
		return false
	}
	return true
}

// BeforeCreate hook to generate UID and key prefix
func (k *APIKey) BeforeCreate(tx *gorm.DB) (err error) {
	k.UID = pkg.GenerateUUIDV7()
	k.CreatedAt = time.Now().Local()
	k.UpdatedAt = time.Now().Local()
	return
}

// BeforeUpdate hook
func (k *APIKey) BeforeUpdate(tx *gorm.DB) (err error) {
	k.UpdatedAt = time.Now().Local()
	return
}
