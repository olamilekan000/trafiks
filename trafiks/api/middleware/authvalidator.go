package middleware

import (
	"errors"
	"net/http"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/gin-gonic/gin"

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
	userRepo repository.UserRepoClient
	logger   pkg.LoggerClient
}

func NewAuthValidator(userRepo repository.UserRepoClient, logger pkg.LoggerClient) AuthValidatorClient {
	return &AuthValidator{
		userRepo: userRepo,
		logger:   logger,
	}
}

func (av *AuthValidator) ValidateUser(c *gin.Context) {
	ctx := c.Request.Context()

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

	c.Next()
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
