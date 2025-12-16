package dto

import (
	"fmt"
	"strings"
)

type CreateProjectRequest struct {
	Name        string `json:"Name" binding:"required"`
	Description string `json:"Description"`
}

func (c *CreateProjectRequest) Validate() error {
	c.Name = strings.TrimSpace(c.Name)
	c.Description = strings.TrimSpace(c.Description)

	if c.Name == "" {
		return fmt.Errorf("name is required")
	}

	if len(c.Name) > 100 {
		return fmt.Errorf("name must not exceed 100 characters")
	}

	if len(c.Description) > 500 {
		return fmt.Errorf("description must not exceed 500 characters")
	}

	return nil
}

type UpdateProjectRequest struct {
	Description string `json:"Description"`
	IsActive    *bool  `json:"IsActive,omitempty"`
}

func (u *UpdateProjectRequest) Validate() error {
	u.Description = strings.TrimSpace(u.Description)

	if len(u.Description) > 500 {
		return fmt.Errorf("description must not exceed 500 characters")
	}

	return nil
}
