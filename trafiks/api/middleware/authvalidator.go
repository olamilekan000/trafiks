package middleware

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"

	"github.com/trafiks/trafiks/api/repository"
	"github.com/trafiks/trafiks/api/services"
	"github.com/trafiks/trafiks/cfg"
	"github.com/trafiks/trafiks/models"
	"github.com/trafiks/trafiks/pkg"
)

type AuthValidatorClient interface {
	ValidateUser(c *gin.Context)
	IsAdmin(c *gin.Context)
}

type AuthValidator struct {
	userRepo   repository.UserRepoClient
	apiKeyRepo repository.APIKeyRepoClient
	logger     pkg.LoggerClient
}

func NewAuthValidator(
	userRepo repository.UserRepoClient,
	apiKeyRepo repository.APIKeyRepoClient,
	logger pkg.LoggerClient,
) AuthValidatorClient {
	return &AuthValidator{
		userRepo:   userRepo,
		apiKeyRepo: apiKeyRepo,
		logger:     logger,
	}
}

func (av *AuthValidator) ValidateUser(c *gin.Context) {
	ctx := c.Request.Context()

	apiKey := av.extractAPIKey(c)
	if apiKey != "" {
		if user := av.validateAPIKey(ctx, c, apiKey); user != nil {
			c.Set(services.AppUserContext, user)
			c.Set("auth_method", "api_key")
			c.Next()
			return
		}
	}

	tokenString, err := c.Cookie("session")
	if err != nil || tokenString == "" {
		av.logger.Errorf("missing or invalid session cookie: %v", err)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized, session cookie missing or invalid"})
		c.Abort()
		return
	}

	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}

		return []byte(cfg.GetConf().JwtSecret), nil
	})
	if err != nil {
		av.logger.Errorf("error parsing token: %v", err)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid session token"})
		c.Abort()
		return
	}

	if !token.Valid {
		av.logger.Errorf("invalid token: %v", err)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized, invalid token"})
		c.Abort()
		return
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		av.logger.Errorf("invalid claims in token")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized, invalid claims"})
		c.Abort()
		return
	}

	expirationTime, ok := claims["exp"].(float64)
	if !ok {
		av.logger.Errorf("token does not contain 'exp' claim")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized, expiration time not found"})
		c.Abort()
		return
	}

	expiration := time.Unix(int64(expirationTime), 0)
	if time.Now().After(expiration) {
		av.logger.Errorf("token expired: %v", err)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized, token expired"})
		c.Abort()
		return
	}

	userID, ok := claims["user_id"].(string)
	if !ok {
		av.logger.Errorf("user_id not found in token")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized, invalid user"})
		c.Abort()
		return
	}

	user, err := av.userRepo.Find(ctx, &models.User{
		UID: userID,
	})
	if err != nil {
		av.logger.Errorf("error fetching user details: %v", err)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized, invalid user"})
		c.Abort()
		return
	}

	c.Set(services.AppUserContext, user)
	c.Set("auth_method", "session")

	c.Next()
}

// extractAPIKey extracts API key from request headers
func (av *AuthValidator) extractAPIKey(c *gin.Context) string {
	apiKey := c.GetHeader("X-API-Key")
	if apiKey != "" {
		return apiKey
	}

	authHeader := c.GetHeader("Authorization")
	if authHeader != "" {
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) == 2 && (strings.ToLower(parts[0]) == "bearer" || strings.ToLower(parts[0]) == "apikey") {
			return parts[1]
		}
		if !strings.Contains(authHeader, " ") {
			return authHeader
		}
	}

	return ""
}

func (av *AuthValidator) validateAPIKey(ctx context.Context, c *gin.Context, apiKeyValue string) *models.User {
	if !strings.HasPrefix(apiKeyValue, "tfk_") {
		return nil
	}

	keyPrefix := apiKeyValue[:11]

	apiKey, err := av.apiKeyRepo.Find(ctx, &models.APIKey{
		KeyPrefix: keyPrefix,
		RevokedAt: nil,
		ExpiresAt: nil,
	})
	if err != nil {
		av.logger.Errorf("failed to fetch API key: %v", err)
		return nil
	}

	if !apiKey.IsActive() {
		return nil
	}

	err = bcrypt.CompareHashAndPassword([]byte(apiKey.KeyHash), []byte(apiKeyValue))
	if err != nil {
		return nil
	}

	now := time.Now()
	av.apiKeyRepo.Updates(ctx, apiKey, map[string]interface{}{
		"last_used_at": now,
	})

	user, err := av.userRepo.Find(ctx, &models.User{ID: apiKey.UserID})
	if err != nil {
		av.logger.Errorf("failed to fetch user for API key: %v", err)
		return nil
	}

	return user
}

func (av *AuthValidator) IsAdmin(c *gin.Context) {
	user := services.GetUserFromContext(c)
	if user == nil || (!user.IsAdmin() && !user.IsSuperAdmin()) {
		av.logger.Errorf("error fetching user")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	c.Next()
}
