package routes

import (
	"crypto/tls"
	"net/http"
	"strings"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"

	"github.com/trafiks/trafiks/cfg"
	"github.com/trafiks/trafiks/pkg"
)

type Router struct {
	*gin.Engine
}

func NewRouter(logger pkg.LoggerClient) Router {
	const (
		xFrameOptions                = "X-Frame-Options"
		xFrameOptionsValue           = "DENY"
		xContentTypeOptions          = "X-Content-Type-Options"
		xContentTypeOptionsValue     = "nosniff"
		xssProtection                = "X-XSS-Protection"
		xssProtectionValue           = "1; mode=block"
		strictTransportSecurity      = "Strict-Transport-Security"
		strictTransportSecurityValue = "max-age=31536000; includeSubDomains; preload"
	)

	env := strings.ToLower(cfg.GetConf().Environment)

	mode := gin.ReleaseMode
	if env == "local" ||
		strings.HasPrefix(env, "stag") ||
		strings.HasPrefix(env, "dev") {
		mode = gin.DebugMode
	}

	gin.SetMode(mode)
	httpRouter := gin.New()

	// Add custom recovery middleware with logging
	httpRouter.Use(gin.CustomRecovery(func(c *gin.Context, recovered interface{}) {
		logger.Errorf("Panic recovered in HTTP handler: %v, Path: %s, Method: %s", recovered, c.Request.URL.Path, c.Request.Method)
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "Internal server error",
			"data":    nil,
		})
	}))

	// Add request logging middleware
	httpRouter.Use(gin.Logger())

	allowedOrigins := []string{
		"http://localhost:5173",
		"http://trafikscloud.cloud",
		"https://trafikscloud.cloud",
	}

	corsConf := cors.Config{
		AllowMethods: []string{"GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS"},
		AllowHeaders: []string{
			"Content-Type",
			"Authorization",
			"X-Requested-With",
			"Accept",
			"X-Idempotency",
			"content-security-policy",
			"Cookie",
		},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
		AllowOrigins:     allowedOrigins,
		ExposeHeaders:    []string{"Set-Cookie"},
	}

	httpRouter.Use(cors.New(corsConf))

	httpRouter.Use(func(c *gin.Context) {
		c.Header(xFrameOptions, xFrameOptionsValue)
		c.Header(xContentTypeOptions, xContentTypeOptionsValue)
		c.Header(xssProtection, xssProtectionValue)

		// Only set HSTS for HTTPS requests
		if c.Request.TLS != nil {
			c.Header(strictTransportSecurity, strictTransportSecurityValue)
		}

		c.Header("Last-Modified", time.Now().Format(http.TimeFormat))
		c.Next()
	})

	httpRouter.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"msg": "API Up and Running"})
	})

	httpRouter.NoRoute(func(c *gin.Context) {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"message": "not found",
			"data":    []any{},
		})
	})

	return Router{
		httpRouter,
	}
}

func (r *Router) CreateServer(addr string) *http.Server {
	conf := cfg.GetConf()
	serverConfig := conf.Proxy.Server

	return &http.Server{
		Addr:         addr,
		Handler:      r.Engine,
		ReadTimeout:  serverConfig.ReadTimeout,
		WriteTimeout: serverConfig.WriteTimeout,
		IdleTimeout:  serverConfig.IdleTimeout,
	}
}

func (r *Router) CreateTLSServer(addr string, tlsConfig *tls.Config) *http.Server {
	conf := cfg.GetConf()
	serverConfig := conf.Proxy.Server

	return &http.Server{
		Addr:         addr,
		Handler:      r.Engine,
		TLSConfig:    tlsConfig,
		ReadTimeout:  serverConfig.ReadTimeout,
		WriteTimeout: serverConfig.WriteTimeout,
		IdleTimeout:  serverConfig.IdleTimeout,
	}
}

func (r *Router) Run(addr string) error {
	server := r.CreateServer(addr)
	return server.ListenAndServe()
}

func (r *Router) RunTLS(addr string, tlsConfig *tls.Config) error {
	server := r.CreateTLSServer(addr, tlsConfig)
	return server.ListenAndServeTLS("", "") // Empty strings because we use TLSConfig.GetCertificate
}
