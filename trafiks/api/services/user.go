package services

import (
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/trafiks/trafiks/api/dto"
	"github.com/trafiks/trafiks/api/repository"
	"github.com/trafiks/trafiks/models"
	"github.com/trafiks/trafiks/pkg"
)

type UserClient interface {
	Profile(ctx *gin.Context) (interface{}, *pkg.RestErr)
	ChangePassword(c *gin.Context, req dto.ChangePwd) (interface{}, *pkg.RestErr)
	UpdateProfile(c *gin.Context, req dto.UpdateProfile) (interface{}, *pkg.RestErr)
}

type User struct {
	logger   pkg.LoggerClient
	userRepo repository.UserRepoClient
	restErr  pkg.RestErrClient
}

func NewUser(
	logger pkg.LoggerClient,
	userRepo repository.UserRepoClient,
	restErr pkg.RestErrClient,
) UserClient {
	return &User{
		logger:   logger,
		userRepo: userRepo,
		restErr:  restErr,
	}
}

func (u *User) Profile(c *gin.Context) (interface{}, *pkg.RestErr) {
	user := GetUserFromContext(c)
	if user == nil {
		u.logger.Errorf("error fetching user")
		return nil, u.restErr.BadRequest("invalid user")
	}

	return user, nil
}

func (u *User) ChangePassword(c *gin.Context, req dto.ChangePwd) (interface{}, *pkg.RestErr) {
	ctx := c.Request.Context()

	user := GetUserFromContext(c)
	if user == nil {
		u.logger.Errorf("unauthorized access - user not found in context")
		return nil, u.restErr.Unauthorized("unauthorized")
	}

	if req.OldPassword == req.NewPassword {
		return nil, u.restErr.BadRequest("new password must be different from old password")
	}

	if !user.CheckPassword(req.OldPassword) {
		return nil, u.restErr.BadRequest("old password is incorrect")
	}

	hashed, err := pkg.HashPassword(req.NewPassword)
	if err != nil {
		u.logger.Errorf("failed to hash new password: %v", err)
		return nil, u.restErr.ServerError("could not update password")
	}

	if err := u.userRepo.Updates(ctx, &models.User{
		ID: user.ID,
	}, map[string]interface{}{
		"password": hashed,
	}); err != nil {
		u.logger.Errorf("failed to update password: %v", err)
		return nil, u.restErr.ServerError("could not update password")
	}

	return gin.H{
		"message": "password updated successfully",
	}, nil
}

func (u *User) UpdateProfile(c *gin.Context, req dto.UpdateProfile) (interface{}, *pkg.RestErr) {
	ctx := c.Request.Context()

	user := GetUserFromContext(c)
	if user == nil {
		u.logger.Errorf("unauthorized access - user not found in context")
		return nil, u.restErr.Unauthorized("unauthorized")
	}

	updateFields := make(map[string]interface{})

	if strings.TrimSpace(req.FirstName) != "" {
		updateFields["first_name"] = req.FirstName
	}

	if strings.TrimSpace(req.LastName) != "" {
		updateFields["last_name"] = req.LastName
	}

	if len(updateFields) == 0 {
		return nil, u.restErr.BadRequest("no valid fields to update")
	}

	if err := u.userRepo.Updates(ctx, &models.User{ID: user.ID}, updateFields); err != nil {
		u.logger.Errorf("failed to update user profile: %v", err)
		return nil, u.restErr.ServerError("could not update profile")
	}

	if strings.TrimSpace(req.FirstName) != "" {
		user.FirstName = req.FirstName
	}

	if strings.TrimSpace(req.LastName) != "" {
		user.LastName = req.LastName
	}

	return user, nil
}
