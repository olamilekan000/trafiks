package services

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	verifier "github.com/AfterShip/email-verifier"
	"github.com/gin-gonic/gin"
	"github.com/mssola/useragent"
	"gorm.io/gorm"

	"github.com/trafiks/trafiks/api/dto"
	"github.com/trafiks/trafiks/api/repository"
	"github.com/trafiks/trafiks/cfg"
	"github.com/trafiks/trafiks/models"
	"github.com/trafiks/trafiks/pkg"
)

type AuthClient interface {
	LoginHandler(c *gin.Context, payload dto.AuthPayload) (interface{}, *pkg.RestErr)

	Signup(ctx *gin.Context, user dto.SignupPayload) (interface{}, *pkg.RestErr)
	Verify(ctx *gin.Context) (interface{}, *pkg.RestErr)
	ForgotPassword(ctx *gin.Context, payload dto.ForgotPassword) (interface{}, *pkg.RestErr)
	ResetPassword(ctx *gin.Context, payload dto.ResetPassword) (interface{}, *pkg.RestErr)
	Logout(c *gin.Context)
}

type Auth struct {
	emailVerifier *verifier.Verifier

	logger pkg.LoggerClient

	userRepo repository.UserRepoClient
	restErr  pkg.RestErrClient
}

func NewAuth(
	logger pkg.LoggerClient,
	userRepo repository.UserRepoClient,
	restErr pkg.RestErrClient,
) AuthClient {
	emailVerifier := verifier.NewVerifier().
		EnableAutoUpdateDisposable()

	return &Auth{
		logger:        logger,
		userRepo:      userRepo,
		restErr:       restErr,
		emailVerifier: emailVerifier,
	}
}

func (a *Auth) LoginHandler(c *gin.Context, payload dto.AuthPayload) (interface{}, *pkg.RestErr) {
	ctx := c.Request.Context()
	user, err := a.userRepo.Find(ctx, &models.User{
		Email: payload.Email,
	})
	if err != nil {
		a.logger.Errorf("error fetching user: %q", err)

		return nil, a.restErr.BadRequest("invalid credentials")
	}

	if user.VerifiedAt == nil {
		return nil, a.restErr.BadRequestWithAction("kindly verify your account", "VERIFY_ACCOUNT")
	}
	if !user.CheckPassword(payload.Password) {
		return nil, a.restErr.BadRequest("invalid credentials")
	}

	secret := cfg.GetConf().JwtSecret

	token, err := user.GenerateJWT(secret)
	if err != nil {
		return nil, a.restErr.ServerError("an error occurred while creating user session")
	}

	AuthHandler(c, token)

	return map[string]string{
		"message": "Authentication successful",
	}, nil
}

func AuthHandler(c *gin.Context, token string) {
	isSecure := c.Request.TLS != nil

	cookie := &http.Cookie{
		Name:     "session",
		Value:    token,
		Path:     "/",
		MaxAge:   5 * 24 * 3600,
		HttpOnly: true,
		Secure:   isSecure,
	}

	// SameSiteNone requires Secure flag, use Lax for HTTP
	if isSecure {
		cookie.SameSite = http.SameSiteNoneMode
	} else {
		cookie.SameSite = http.SameSiteLaxMode
	}

	http.SetCookie(c.Writer, cookie)
}

func (a *Auth) Logout(c *gin.Context) {
	user := GetUserFromContext(c)
	if user == nil {
		a.logger.Errorf("error fetching user")
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid user",
		})
		return
	}

	isSecure := c.Request.TLS != nil

	cookie := &http.Cookie{
		Name:     "session",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   isSecure,
	}

	// SameSiteNone requires Secure flag, use Lax for HTTP
	if isSecure {
		cookie.SameSite = http.SameSiteNoneMode
	} else {
		cookie.SameSite = http.SameSiteLaxMode
	}

	http.SetCookie(c.Writer, cookie)

	c.JSON(http.StatusOK, gin.H{
		"message": "Logout successful",
	})
}

func (a *Auth) Signup(c *gin.Context, user dto.SignupPayload) (interface{}, *pkg.RestErr) {
	ctx := c.Request.Context()

	user.Sanitize()

	result, err := a.emailVerifier.Verify(user.Email)
	if err != nil {
		a.logger.Error("email verification failed", err)
		return nil, a.restErr.ServerError("could not verify email")
	}

	if result.Disposable {
		return nil, a.restErr.BadRequest("disposable email addresses are not allowed")
	}

	if err := user.Validate(); err != nil {
		return nil, a.restErr.BadRequest(err.Error())
	}

	existingUser, err := a.userRepo.Find(ctx, &models.User{
		Email: user.Email,
	})
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		a.logger.Error("failed to check user existence", err)
		return nil, a.restErr.ServerError("could not process signup")
	}

	if existingUser != nil {
		return nil, a.restErr.StatusConflict("invalid payload")
	}

	hashedPassword, err := pkg.HashPassword(user.Password)
	if err != nil {
		a.logger.Error("failed to hash password", err)
		return nil, a.restErr.ServerError("could not secure password")
	}

	newUser := &models.User{
		Email:      user.Email,
		Password:   hashedPassword,
		FirstName:  user.FirstName,
		LastName:   user.LastName,
		AuthMethod: "email",
		Role:       "admin",
	}

	// Audit log for successful signup
	actorName := newUser.FullName()
	if actorName == "" {
		actorName = newUser.Email
	}

	return map[string]string{
		"message": "a verification link has been sent to your mail.",
	}, nil
}

func (a *Auth) Verify(c *gin.Context) (interface{}, *pkg.RestErr) {
	ctx := c.Request.Context()

	token := c.Query("token")
	if token == "" {
		return nil, a.restErr.BadRequest("verification token is required")
	}

	user, err := a.userRepo.Find(ctx, &models.User{
		VerificationToken: token,
	})
	if err != nil {
		return nil, a.restErr.BadRequest("invalid or expired verification token")
	}

	now := time.Now()
	user.VerifiedAt = &now
	user.VerificationToken = ""

	if err := a.userRepo.Updates(ctx, &models.User{
		ID: user.ID,
	}, map[string]interface{}{
		"verification_token": user.VerificationToken,
		"verified_at":        user.VerifiedAt,
	}); err != nil {
		return nil, a.restErr.BadRequest("could not verify user")
	}

	return map[string]string{
		"message": "your verification was successful.",
	}, nil
}

func (a *Auth) ForgotPassword(c *gin.Context, payload dto.ForgotPassword) (interface{}, *pkg.RestErr) {
	ctx := c.Request.Context()

	user, err := a.userRepo.Find(ctx, &models.User{
		Email: payload.Email,
	})
	if err != nil {
		return map[string]string{
			"message": "if the email exists, a reset link has been sent.",
		}, nil
	}

	token, err := pkg.GenerateToken(32)
	if err != nil {
		return nil, a.restErr.BadRequest("could not generate reset token")
	}

	expiry := time.Now().Add(10 * time.Minute)

	if err := a.userRepo.Updates(ctx, &models.User{
		ID: user.ID,
	}, map[string]interface{}{
		"password_reset_token":        token,
		"password_reset_token_expiry": expiry,
	}); err != nil {
		return nil, a.restErr.ServerError("could not generate reset token")
	}

	return map[string]string{
		"message": "If the email exists, a reset link has been sent.",
	}, nil
}

func (a *Auth) ResetPassword(c *gin.Context, payload dto.ResetPassword) (interface{}, *pkg.RestErr) {
	ctx := c.Request.Context()

	userAgent := c.GetHeader("User-Agent")

	token := c.Query("token")
	if token == "" {
		return nil, a.restErr.BadRequest("verification token is required")
	}

	user, err := a.userRepo.Find(ctx, &models.User{
		PasswordResetToken: token,
	})
	if err != nil ||
		user.PasswordResetTokenExpiry == nil ||
		time.Now().After(*user.PasswordResetTokenExpiry) {
		return nil, a.restErr.BadRequest("invalid or expired reset token")
	}

	hashedPassword, err := pkg.HashPassword(payload.NewPassword)
	if err != nil {
		a.logger.Error("failed to hash password", err)
		return nil, a.restErr.BadRequest("could not secure password")
	}

	timestamp := time.Now().Format("Monday, Jan 2, 2006 at 3:04PM")
	fmt.Println("timestamp", timestamp)
	ua := useragent.New(userAgent)
	browser, _ := ua.Browser()
	fmt.Println("browser", browser)

	if err := a.userRepo.Updates(ctx, &models.User{
		ID: user.ID,
	}, map[string]interface{}{
		"password":                    hashedPassword,
		"password_reset_token":        "",
		"password_reset_token_expiry": nil,
	}); err != nil {
		return nil, a.restErr.ServerError("could not reset password")
	}

	return map[string]string{
		"message": "your password has been reset successfully.",
	}, nil
}
