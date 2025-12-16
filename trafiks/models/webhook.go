package models

import (
	"time"

	"gorm.io/datatypes"
	"gorm.io/gorm"

	"github.com/trafiks/trafiks/pkg"
)

type Webhook struct {
	gorm.Model
	ID            uint           `json:"-" gorm:"primaryKey"`
	UID           string         `gorm:"not null;uniqueIndex"`
	UserID        uint           `gorm:"not null;index"`
	URL           string         `gorm:"not null"`
	Secret        string         `gorm:"not null"`
	EnabledEvents datatypes.JSON `gorm:"type:jsonb"`
	IsActive      *bool          `gorm:"default:true;type:boolean"`
	CreatedAt     time.Time      `json:"CreatedAt"`
	UpdatedAt     time.Time      `json:"UpdatedAt"`

	User User `json:"-" gorm:"foreignKey:UserID"`
}

func (w *Webhook) BeforeCreate(tx *gorm.DB) (err error) {
	w.UID = pkg.GenerateUUIDV7()
	w.CreatedAt = time.Now().Local()
	w.UpdatedAt = time.Now().Local()
	return
}

func (w *Webhook) BeforeUpdate(tx *gorm.DB) (err error) {
	w.UpdatedAt = time.Now().Local()
	return
}

func (w *Webhook) Active() bool {
	return pkg.ToBoolValue(w.IsActive)
}
