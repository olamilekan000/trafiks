package services

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/trafiks/trafiks/api/dto"
	"github.com/trafiks/trafiks/api/repository"
	"github.com/trafiks/trafiks/cfg"
	"github.com/trafiks/trafiks/models"
	"github.com/trafiks/trafiks/pkg"
)

type ProjectClient interface {
	CreateProject(c *gin.Context, req dto.CreateProjectRequest) (interface{}, *pkg.RestErr)
	ListProjects(c *gin.Context) (interface{}, *pkg.RestErr)
	GetProject(c *gin.Context) (interface{}, *pkg.RestErr)
	UpdateProject(c *gin.Context, req dto.UpdateProjectRequest) (interface{}, *pkg.RestErr)
	DeleteProject(c *gin.Context) (interface{}, *pkg.RestErr)
}

type Project struct {
	logger         pkg.LoggerClient
	projectRepo    repository.ProjectRepoClient
	serviceRepo    repository.ServiceRepoClient
	webhookRepo    repository.WebhookRepoClient
	restErr        pkg.RestErrClient
	config         *cfg.Config
	webhookService WebhookEventSender
}

func NewProject(
	logger pkg.LoggerClient,
	projectRepo repository.ProjectRepoClient,
	serviceRepo repository.ServiceRepoClient,
	webhookRepo repository.WebhookRepoClient,
	restErr pkg.RestErrClient,
	config *cfg.Config,
	webhookService WebhookEventSender,
) ProjectClient {
	return &Project{
		logger:         logger,
		projectRepo:    projectRepo,
		serviceRepo:    serviceRepo,
		webhookRepo:    webhookRepo,
		restErr:        restErr,
		config:         config,
		webhookService: webhookService,
	}
}

func (s *Project) CreateProject(c *gin.Context, req dto.CreateProjectRequest) (interface{}, *pkg.RestErr) {
	ctx := c.Request.Context()

	user := GetUserFromContext(c)
	if user == nil {
		return nil, s.restErr.Unauthorized("unauthorized")
	}

	if err := req.Validate(); err != nil {
		return nil, s.restErr.BadRequest(err.Error())
	}

	project := &models.Project{
		UserID:      user.ID,
		Name:        req.Name,
		Description: req.Description,
	}

	if err := s.projectRepo.Create(ctx, project); err != nil {
		s.logger.Errorf("failed to create project: %v", err)
		return nil, s.restErr.ServerError("failed to create project")
	}

	return gin.H{
		"uid":         project.UID,
		"name":        project.Name,
		"slug":        project.Slug,
		"description": project.Description,
		"is_active":   project.IsActive,
		"created_at":  project.CreatedAt,
		"updated_at":  project.UpdatedAt,
	}, nil
}

func (s *Project) ListProjects(c *gin.Context) (interface{}, *pkg.RestErr) {
	ctx := c.Request.Context()

	user := GetUserFromContext(c)
	if user == nil {
		return nil, s.restErr.Unauthorized("unauthorized")
	}

	pagination := &dto.Pagination{
		Page:  1,
		Limit: 20,
	}

	if pageStr := c.Query("page"); pageStr != "" {
		if parsedPage, err := strconv.Atoi(pageStr); err == nil && parsedPage > 0 {
			pagination.Page = parsedPage
		}
	}

	if limitStr := c.Query("limit"); limitStr != "" {
		if parsedLimit, err := strconv.Atoi(limitStr); err == nil && parsedLimit > 0 {
			if parsedLimit > 100 {
				parsedLimit = 100
			}
			pagination.Limit = parsedLimit
		}
	}

	projects, err := s.projectRepo.FindMany(ctx, &models.Project{UserID: user.ID}, pagination)
	if err != nil {
		s.logger.Errorf("failed to fetch projects: %v", err)
		return nil, s.restErr.ServerError("failed to fetch projects")
	}

	result := make([]gin.H, 0, len(projects))
	for _, project := range projects {
		projectData := gin.H{
			"uid":         project.UID,
			"name":        project.Name,
			"slug":        project.Slug,
			"description": project.Description,
			"is_active":   project.IsActive,
			"created_at":  project.CreatedAt,
			"updated_at":  project.UpdatedAt,
		}

		if len(project.Services) > 0 && project.Services[0].ProxyURL != "" {
			projectData["proxy_url"] = project.Services[0].ProxyURL
			projectData["scheme"] = project.Services[0].Scheme
		}

		result = append(result, projectData)
	}

	return gin.H{
		"projects": result,
		"pagination": gin.H{
			"page":        pagination.Page,
			"limit":       pagination.Limit,
			"total":       pagination.Total,
			"total_pages": (int(pagination.Total) + pagination.Limit - 1) / pagination.Limit,
		},
	}, nil
}

func (s *Project) GetProject(c *gin.Context) (interface{}, *pkg.RestErr) {
	ctx := c.Request.Context()

	user := GetUserFromContext(c)
	if user == nil {
		return nil, s.restErr.Unauthorized("unauthorized")
	}

	projectUID := c.Param("projectId")
	if projectUID == "" {
		return nil, s.restErr.BadRequest("project ID is required")
	}

	project, err := s.projectRepo.Find(ctx, &models.Project{
		UID:    projectUID,
		UserID: user.ID,
	})
	if err != nil {
		return nil, s.restErr.NotFound("project not found")
	}

	return gin.H{
		"uid":         project.UID,
		"name":        project.Name,
		"slug":        project.Slug,
		"description": project.Description,
		"is_active":   project.IsActive,
		"created_at":  project.CreatedAt,
		"updated_at":  project.UpdatedAt,
	}, nil
}

func (s *Project) UpdateProject(c *gin.Context, req dto.UpdateProjectRequest) (interface{}, *pkg.RestErr) {
	ctx := c.Request.Context()

	user := GetUserFromContext(c)
	if user == nil {
		return nil, s.restErr.Unauthorized("unauthorized")
	}

	projectUID := c.Param("projectId")
	if projectUID == "" {
		return nil, s.restErr.BadRequest("project ID is required")
	}

	if err := req.Validate(); err != nil {
		return nil, s.restErr.BadRequest(err.Error())
	}

	project, err := s.projectRepo.Find(ctx, &models.Project{
		UID:    projectUID,
		UserID: user.ID,
	})
	if err != nil {
		return nil, s.restErr.NotFound("project not found")
	}

	updates := make(map[string]interface{})

	if req.Description != "" {
		updates["description"] = req.Description
		project.Description = req.Description
	}

	if req.IsActive != nil {
		webhook, err := s.webhookRepo.Find(
			ctx, &models.Webhook{UserID: user.ID, IsActive: pkg.BoolPtr(true)})
		if err != nil {
			s.logger.Warnf("failed to fetch webhook %v", err)
		}

		if webhook == nil {
			return nil, nil
		}

		wasActive := project.IsActive != nil && *project.IsActive
		updates["is_active"] = *req.IsActive
		project.IsActive = req.IsActive

		if s.webhookService != nil {
			if *req.IsActive && !wasActive {
				go s.webhookService.SendEvent(
					webhook.ID,
					EventProjectActivated,
					map[string]interface{}{
						"project_id":   project.ID,
						"project_uid":  project.UID,
						"project_name": project.Name,
						"project_slug": project.Slug,
					},
				)
			} else if !*req.IsActive && wasActive {
				go s.webhookService.SendEvent(
					webhook.ID,
					EventProjectDeactivated,
					map[string]interface{}{
						"project_uid":  project.UID,
						"project_name": project.Name,
						"project_slug": project.Slug,
					},
				)
			}
		}
	}

	if len(updates) == 0 {
		return nil, s.restErr.BadRequest("no fields to update")
	}

	if err := s.projectRepo.Updates(ctx, &models.Project{
		ID: project.ID,
	}, updates); err != nil {
		s.logger.Errorf("failed to update project: %v", err)
		return nil, s.restErr.ServerError("failed to update project")
	}

	return gin.H{
		"uid":         project.UID,
		"name":        project.Name,
		"slug":        project.Slug,
		"description": project.Description,
		"is_active":   project.IsActive,
		"created_at":  project.CreatedAt,
		"updated_at":  project.UpdatedAt,
	}, nil
}

func (s *Project) DeleteProject(c *gin.Context) (interface{}, *pkg.RestErr) {
	ctx := c.Request.Context()

	user := GetUserFromContext(c)
	if user == nil {
		return nil, s.restErr.Unauthorized("unauthorized")
	}

	projectUID := c.Param("projectId")
	if projectUID == "" {
		return nil, s.restErr.BadRequest("project ID is required")
	}

	project, err := s.projectRepo.Find(ctx, &models.Project{
		UID:    projectUID,
		UserID: user.ID,
	})
	if err != nil {
		return nil, s.restErr.NotFound("project not found")
	}

	service, err := s.serviceRepo.Find(ctx, &models.Service{ProjectID: project.ID})
	if err == nil && service != nil {
		if err := s.serviceRepo.Delete(ctx, service); err != nil {
			s.logger.Errorf("failed to delete associated service: %v", err)
		}
	}

	if err := s.projectRepo.Delete(ctx, &models.Project{
		UID:    projectUID,
		UserID: user.ID,
	}); err != nil {
		s.logger.Errorf("failed to delete project: %v", err)
		return nil, s.restErr.ServerError("failed to delete project")
	}

	return gin.H{
		"message": "project deleted successfully",
	}, nil
}
