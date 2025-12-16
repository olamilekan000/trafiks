package routes

import (
	"embed"
	"io"
	"io/fs"
	"mime"
	"net/http"
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
		d.logger.Warnf("Failed to load embedded dashboard files: %v. Dashboard will not be available. Please run 'make build-ui' and rebuild the Go binary.", err)
		return
	}

	d.logger.Info("Serving embedded dashboard from binary")

	d.handler.GET("/dashboard", func(c *gin.Context) {
		c.Redirect(http.StatusMovedPermanently, "/dashboard/")
	})

	d.handler.GET("/dashboard/*path", func(c *gin.Context) {
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
