package models

import (
	"fmt"
	"strings"
	"time"

	"github.com/trafiks/trafiks/pkg"

	"github.com/dgrijalva/jwt-go"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type GitProvider string

const Github GitProvider = "github"

// User role hierarchy:
// - super-admin: SyncTax internal users (system administrators)
// - admin: Higher role for customers (organization admins)
// - user: Regular customers
type User struct {
	gorm.Model               `json:"-"`
	ID                       uint   `json:"-" gorm:"primaryKey;unique"`
	UID                      string `gorm:"not null"`
	FirstName                string
	LastName                 string
	Password                 string `json:"-"`
	Email                    string `gorm:"uniqueIndex;not null"`
	AuthMethod               string
	Role                     string     `json:"-" gorm:"default:'user'"` // Either "user", "admin", or "super-admin"
	VerificationToken        string     `json:"-"`
	PasswordResetToken       string     `json:"-"`
	PasswordResetTokenExpiry *time.Time `json:"-"`
	CreatedAt                time.Time  `json:"JoinedAt"`
	UpdatedAt                time.Time  `json:"-"`
	VerifiedAt               *time.Time `json:"-"`
}

func (u *User) CheckPassword(password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(password))
	return err == nil
}

// IsAdmin checks if the user has admin role
func (u *User) IsAdmin() bool {
	return u.Role == "admin"
}

// IsSuperAdmin checks if the user has super-admin role
func (u *User) IsSuperAdmin() bool {
	return u.Role == "super-admin"
}

func (u *User) FullName() string {
	first := strings.TrimSpace(u.FirstName)
	last := strings.TrimSpace(u.LastName)

	if first != "" {
		first = strings.ToUpper(first[:1]) + strings.ToLower(first[1:])
	}
	if last != "" {
		last = strings.ToUpper(last[:1]) + strings.ToLower(last[1:])
	}

	switch {
	case first == "" && last == "":
		return ""
	case first == "":
		return last
	case last == "":
		return first
	default:
		return fmt.Sprintf("%s %s", first, last)
	}
}

func (u *User) BeforeCreate(tx *gorm.DB) (err error) {
	u.UID = pkg.GenerateUUIDV7()

	token, err := pkg.GenerateToken(32)
	if err != nil {
		return err
	}

	u.VerificationToken = token
	u.CreatedAt = time.Now().Local()
	u.UpdatedAt = time.Now().Local()

	return
}

func (u *User) BeforeUpdate(tx *gorm.DB) (err error) {
	u.UpdatedAt = time.Now().Local()
	return
}

func (u *User) GenerateJWT(jwtSecret string) (string, error) {
	expirationTime := time.Now().Add(5 * 24 * time.Hour)
	claims := jwt.MapClaims{
		"user_id":    u.UID,
		"first_name": u.FirstName,
		"exp":        expirationTime.Unix(),
		"role":       u.Role,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	return token.SignedString([]byte(jwtSecret))
}

func (u *User) GenerateVerificationURL(baseURL, token string) string {
	return fmt.Sprintf("%s/verify-email?token=%s", baseURL, token)
}

func (u *User) GeneratePasswordResetURL(baseURL, token string) string {
	return fmt.Sprintf("%s/reset-password?token=%s", baseURL, token)
}
