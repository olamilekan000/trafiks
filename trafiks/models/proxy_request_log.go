package models

import (
	"time"

	"github.com/trafiks/trafiks/pkg"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type ProxyRequestLog struct {
	gorm.Model
	ID        uint   `json:"-" gorm:"primaryKey"`
	UID       string `gorm:"not null;uniqueIndex"`
	ServiceID uint   `gorm:"not null;index"`
	ProjectID uint   `gorm:"not null;index"`

	// Request details
	Method      string `gorm:"not null;index"`
	Path        string `gorm:"not null"`
	QueryString string `gorm:"type:text"`
	TargetURL   string `gorm:"not null"` // Final proxied URL

	// Request metadata
	RequestHeaders datatypes.JSON `gorm:"type:jsonb"` // Selected headers
	RequestSize    int64          `gorm:"default:0"`  // bytes
	RequestBody    string         `gorm:"type:text"`  // Optional: store body (can be large)

	// Response details
	StatusCode      int            `gorm:"not null;index"`
	ResponseHeaders datatypes.JSON `gorm:"type:jsonb"`
	ResponseSize    int64          `gorm:"default:0"` // bytes
	ResponseBody    string         `gorm:"type:text"` // Optional: store body

	// Performance
	ResponseTime int64 `gorm:"not null"` // milliseconds
	CacheHit     bool  `gorm:"default:false"`

	// Client info
	IPAddress string `gorm:"index"`
	UserAgent string `gorm:"type:text"`

	// Timestamps
	RequestedAt time.Time `gorm:"not null;index"`
	CreatedAt   time.Time `json:"CreatedAt"`

	// Relationships
	Service Service `json:"-" gorm:"foreignKey:ServiceID"`
	Project Project `json:"-" gorm:"foreignKey:ProjectID"`
}

func (p *ProxyRequestLog) BeforeCreate(tx *gorm.DB) (err error) {
	p.UID = pkg.GenerateUUIDV7()
	p.CreatedAt = time.Now().Local()
	return
}
