package routes

import (
	"github.com/trafiks/trafiks/api/controller"
	"github.com/trafiks/trafiks/api/middleware"
	"github.com/trafiks/trafiks/pkg"
)

type Project struct {
	logger            pkg.LoggerClient
	handler           Router
	projectController controller.ProjectClient
	authValidator     middleware.AuthValidatorClient
}

func NewProject(
	logger pkg.LoggerClient,
	handler Router,
	projectController controller.ProjectClient,
	authValidator middleware.AuthValidatorClient,
) Project {
	return Project{
		logger:            logger,
		handler:           handler,
		projectController: projectController,
		authValidator:     authValidator,
	}
}

func (p *Project) Setup() {
	projects := p.handler.Group("/projects")

	projects.Use(p.authValidator.ValidateUser)
	{
		projects.POST("", p.projectController.CreateProject)
		projects.GET("", p.projectController.ListProjects)
		projects.GET("/:projectId", p.projectController.GetProject)
		projects.PUT("/:projectId", p.projectController.UpdateProject)
		projects.DELETE("/:projectId", p.projectController.DeleteProject)
	}
}
