package routes

import (
	"github.com/trafiks/trafiks/api/controller"
	"github.com/trafiks/trafiks/api/middleware"
	"github.com/trafiks/trafiks/pkg"
)

type Service struct {
	logger            pkg.LoggerClient
	handler           Router
	serviceController controller.ServiceClient
	authValidator     middleware.AuthValidatorClient
}

func NewService(
	logger pkg.LoggerClient,
	handler Router,
	serviceController controller.ServiceClient,
	authValidator middleware.AuthValidatorClient,
) Service {
	return Service{
		logger:            logger,
		handler:           handler,
		serviceController: serviceController,
		authValidator:     authValidator,
	}
}

func (s *Service) Setup() {
	// Services are nested under projects
	projects := s.handler.Group("/projects")
	projects.Use(s.authValidator.ValidateUser)
	{
		projects.POST("/:projectId/services", s.serviceController.CreateService)
		projects.GET("/:projectId/service", s.serviceController.GetServiceByProject)
		projects.GET("/:projectId/services/:serviceId", s.serviceController.GetService)
		projects.PUT("/:projectId/services/:serviceId", s.serviceController.UpdateService)
	}
}
