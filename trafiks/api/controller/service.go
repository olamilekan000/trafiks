package controller

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/trafiks/trafiks/api/dto"
	"github.com/trafiks/trafiks/api/services"
	"github.com/trafiks/trafiks/pkg"
)

type ServiceClient interface {
	CreateService(c *gin.Context)
	GetService(c *gin.Context)
	GetServiceByProject(c *gin.Context)
	UpdateService(c *gin.Context)
}

type Service struct {
	logger     pkg.LoggerClient
	serviceSvc services.ServiceClient
}

func NewService(
	logger pkg.LoggerClient,
	serviceSvc services.ServiceClient,
) ServiceClient {
	return &Service{
		logger:     logger,
		serviceSvc: serviceSvc,
	}
}

func (s *Service) CreateService(c *gin.Context) {
	var req dto.CreateServiceRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"message": err.Error(),
		})
		return
	}

	resp, err := s.serviceSvc.CreateService(c, req)
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

func (s *Service) GetService(c *gin.Context) {
	resp, err := s.serviceSvc.GetService(c)
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

func (s *Service) GetServiceByProject(c *gin.Context) {
	resp, err := s.serviceSvc.GetServiceByProject(c)
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

func (s *Service) UpdateService(c *gin.Context) {
	var req dto.UpdateServiceRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"message": err.Error(),
		})
		return
	}

	resp, err := s.serviceSvc.UpdateService(c, req)
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
