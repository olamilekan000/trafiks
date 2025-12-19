package routes

import (
	"embed"
	"io"
	"io/fs"
	"mime"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/trafiks/trafiks/cfg"
	"github.com/trafiks/trafiks/pkg"
)

//go:embed dist/*
var DashboardFiles embed.FS

type Dashboard struct {
	logger  pkg.LoggerClient
	handler Router
}

func NewDashboard(
	logger pkg.LoggerClient,
	handler Router,
) Dashboard {
	return Dashboard{
		logger:  logger,
		handler: handler,
	}
}

func (d *Dashboard) Setup() {
	config := cfg.GetConf()

	if config.Dashboard.Enabled != nil && !*config.Dashboard.Enabled {
		d.logger.Info("Dashboard is disabled in configuration")
		return
	}

	distFS, err := fs.Sub(DashboardFiles, "dist")
	if err != nil {
		d.logger.Warnf("failed to load embedded dashboard files: %v.", err)
		return
	}

	d.handler.GET("/dashboard", d.domainCheckMiddleware, func(c *gin.Context) {
		c.Redirect(http.StatusMovedPermanently, "/dashboard/")
	})

	d.handler.GET("/dashboard/*path", d.domainCheckMiddleware, func(c *gin.Context) {
		path := strings.TrimPrefix(c.Param("path"), "/")
		file, err := distFS.Open(path)
		if err != nil {
			file, err = distFS.Open("index.html")
			if err != nil {
				c.Status(http.StatusNotFound)
				return
			}
		}
		defer file.Close()

		data, err := io.ReadAll(file)
		if err != nil {
			c.Status(http.StatusInternalServerError)
			return
		}

		contentType := mime.TypeByExtension(filepath.Ext(path))
		if contentType == "" {
			contentType = "text/html"
		}

		c.Data(http.StatusOK, contentType, data)
	})
}

func (d *Dashboard) domainCheckMiddleware(c *gin.Context) {
	config := cfg.GetConf()

	allowedDomain := extractDomain(config.AppBaseURL)
	if allowedDomain == "" {
		c.Next()
		return
	}

	requestHost := c.Request.Host
	if strings.Contains(requestHost, ":") {
		requestHost = strings.Split(requestHost, ":")[0]
	}

	allowedDomain = normalizeDomain(allowedDomain)
	requestHost = normalizeDomain(requestHost)

	if requestHost != allowedDomain {
		d.logger.Warnf("Dashboard access denied: request host '%s' does not match app_base_url domain '%s'", c.Request.Host, allowedDomain)
		c.JSON(http.StatusForbidden, gin.H{
			"success": false,
			"message": "you do not have permission to access this route.",
			"data":    nil,
		})
		c.Abort()
		return
	}

	c.Next()
}

func extractDomain(urlString string) string {
	urlString = strings.TrimSpace(urlString)
	if urlString == "" {
		return ""
	}

	if !strings.HasPrefix(urlString, "http://") && !strings.HasPrefix(urlString, "https://") {
		return urlString
	}

	parsedURL, err := url.Parse(urlString)
	if err != nil {
		urlString = strings.TrimPrefix(strings.TrimPrefix(urlString, "http://"), "https://")
		if idx := strings.Index(urlString, "/"); idx != -1 {
			return urlString[:idx]
		}
		return urlString
	}

	return parsedURL.Hostname()
}

func normalizeDomain(domain string) string {
	domain = strings.ToLower(strings.TrimSpace(domain))
	domain = strings.TrimPrefix(domain, "www.")
	return domain
}
