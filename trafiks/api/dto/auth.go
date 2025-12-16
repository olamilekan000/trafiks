package dto

import (
	"fmt"
	"strings"
)

type AuthPayload struct {
	Email    string `json:"Email" binding:"required,email"`
	Password string `json:"Password" binding:"required"`
}

type SignupPayload struct {
	Email     string `json:"Email" binding:"required,email"`
	Password  string `json:"Password" binding:"required"`
	FirstName string `json:"FirstName" binding:"required"`
	LastName  string `json:"LastName" binding:"required"`
}

func (s *SignupPayload) Sanitize() {
	s.Email = strings.TrimSpace(strings.ToLower(s.Email))
	s.Password = strings.TrimSpace(s.Password)
	s.FirstName = strings.TrimSpace(s.FirstName)
	s.LastName = strings.TrimSpace(s.LastName)
}

func (s *SignupPayload) Validate() error {
	if len(s.FirstName) > 50 {
		return fmt.Errorf("first name must not exceed 50 characters")
	}

	if len(s.LastName) > 50 {
		return fmt.Errorf("last name must not exceed 50 characters")
	}

	if len(s.Email) > 50 {
		return fmt.Errorf("email must not exceed 50 characters")
	}

	if len(s.Password) < 8 || len(s.Password) > 64 {
		return fmt.Errorf("password must be between 8 and 64 characters")
	}

	return nil
}

type ForgotPassword struct {
	Email string `json:"Email" binding:"required,email"`
}

type ResetPassword struct {
	NewPassword string `json:"NewPassword" binding:"required,min=8"`
}

type ChangePwd struct {
	OldPassword string `json:"OldPassword" binding:"required"`
	NewPassword string `json:"NewPassword" binding:"required,min=8"`
}

type UpdateProfile struct {
	FirstName string `json:"FirstName"`
	LastName  string `json:"LastName"`
}

type GenerateAPIKeyRequest struct {
	Name          string `json:"Name" binding:"required"`
	ExpiresInDays int    `json:"ExpiresInDays"` // Optional: 0 means no expiration
}

type Pagination struct {
	Page  int
	Limit int
	Total int64
}

func (p *Pagination) Offset() int {
	if p.Page < 1 {
		p.Page = 1
	}

	return (p.Page - 1) * p.Limit
}
