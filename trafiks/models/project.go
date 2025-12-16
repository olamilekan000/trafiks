package models

import (
	"fmt"
	"time"

	"gorm.io/gorm"

	"github.com/trafiks/trafiks/pkg"
)

type Project struct {
	gorm.Model
	ID          uint      `json:"-" gorm:"primaryKey;unique"`
	UID         string    `gorm:"not null;uniqueIndex"`
	UserID      uint      `gorm:"not null;index"`
	Name        string    `gorm:"not null"`
	Slug        string    `gorm:"not null;uniqueIndex"`
	Description string    `gorm:"type:text"`
	IsActive    *bool     `gorm:"default:true"`
	CreatedAt   time.Time `json:"CreatedAt"`
	UpdatedAt   time.Time `json:"UpdatedAt"`

	User     User      `json:"-" gorm:"foreignKey:UserID"`
	Services []Service `json:"-" gorm:"foreignKey:ProjectID"` // One-to-many relationship
}

func (p *Project) BeforeCreate(tx *gorm.DB) (err error) {
	p.UID = pkg.GenerateUUIDV7()

	nameSlug := pkg.ToSlug(p.Name)

	randomToken, err := pkg.GenerateToken(4)
	if err != nil {
		return err
	}

	p.Slug = fmt.Sprintf("%s-%s", nameSlug, randomToken)

	p.CreatedAt = time.Now().Local()
	p.UpdatedAt = time.Now().Local()

	return
}

func (p *Project) BeforeUpdate(tx *gorm.DB) (err error) {
	p.UpdatedAt = time.Now().Local()
	return
}
