package controller

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/trafiks/trafiks/api/dto"
	"github.com/trafiks/trafiks/api/services"
	"github.com/trafiks/trafiks/pkg"
)

type ProjectClient interface {
	CreateProject(c *gin.Context)
	ListProjects(c *gin.Context)
	GetProject(c *gin.Context)
	UpdateProject(c *gin.Context)
	DeleteProject(c *gin.Context)
}

type Project struct {
	logger     pkg.LoggerClient
	projectSvc services.ProjectClient
}

func NewProject(
	logger pkg.LoggerClient,
	projectSvc services.ProjectClient,
) ProjectClient {
	return &Project{
		logger:     logger,
		projectSvc: projectSvc,
	}
}

func (p *Project) CreateProject(c *gin.Context) {
	var req dto.CreateProjectRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"message": err.Error(),
		})
		return
	}

	resp, err := p.projectSvc.CreateProject(c, req)
	if err != nil {
		c.JSON(err.StatusCode, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "ok",
		"data":    resp,
		"success": true,
	})
}

func (p *Project) ListProjects(c *gin.Context) {
	resp, err := p.projectSvc.ListProjects(c)
	if err != nil {
		c.JSON(err.StatusCode, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "ok",
		"data":    resp,
		"success": true,
	})
}

func (p *Project) GetProject(c *gin.Context) {
	resp, err := p.projectSvc.GetProject(c)
	if err != nil {
		c.JSON(err.StatusCode, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "ok",
		"data":    resp,
		"success": true,
	})
}

func (p *Project) UpdateProject(c *gin.Context) {
	var req dto.UpdateProjectRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"message": err.Error(),
		})
		return
	}

	resp, err := p.projectSvc.UpdateProject(c, req)
	if err != nil {
		c.JSON(err.StatusCode, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "ok",
		"data":    resp,
		"success": true,
	})
}

func (p *Project) DeleteProject(c *gin.Context) {
	resp, err := p.projectSvc.DeleteProject(c)
	if err != nil {
		c.JSON(err.StatusCode, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "ok",
		"data":    resp,
		"success": true,
	})
}
