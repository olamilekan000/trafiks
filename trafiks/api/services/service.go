package services

import (
	"github.com/gin-gonic/gin"
	"gorm.io/datatypes"

	"github.com/trafiks/trafiks/api/dto"
	"github.com/trafiks/trafiks/api/repository"
	"github.com/trafiks/trafiks/cfg"
	"github.com/trafiks/trafiks/models"
	"github.com/trafiks/trafiks/pkg"
	tlsPkg "github.com/trafiks/trafiks/pkg/tls"
)

type ServiceClient interface {
	CreateService(c *gin.Context, req dto.CreateServiceRequest) (interface{}, *pkg.RestErr)
	GetService(c *gin.Context) (interface{}, *pkg.RestErr)
	GetServiceByProject(c *gin.Context) (interface{}, *pkg.RestErr)
	UpdateService(c *gin.Context, req dto.UpdateServiceRequest) (interface{}, *pkg.RestErr)
}

type Service struct {
	logger      pkg.LoggerClient
	serviceRepo repository.ServiceRepoClient
	projectRepo repository.ProjectRepoClient
	restErr     pkg.RestErrClient
	config      *cfg.Config
	tlsManager  *TLSManager
}

func NewService(
	logger pkg.LoggerClient,
	serviceRepo repository.ServiceRepoClient,
	projectRepo repository.ProjectRepoClient,
	restErr pkg.RestErrClient,
	config *cfg.Config,
	tlsManager *TLSManager,
) ServiceClient {
	return &Service{
		logger:      logger,
		serviceRepo: serviceRepo,
		projectRepo: projectRepo,
		restErr:     restErr,
		config:      config,
		tlsManager:  tlsManager,
	}
}

func (s *Service) CreateService(c *gin.Context, req dto.CreateServiceRequest) (interface{}, *pkg.RestErr) {
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

	if req.Source == models.SourceKubernetes && !IsAPIKeyAuth(c) {
		return nil, s.restErr.RequestNotAllowed("kubernetes source services can only be created via the Kubernetes operator using API key authentication")
	}

	project, err := s.projectRepo.Find(ctx, &models.Project{
		UID:    projectUID,
		UserID: user.ID,
	})
	if err != nil {
		return nil, s.restErr.NotFound("project not found")
	}

	existingService, err := s.serviceRepo.Find(ctx, &models.Service{ProjectID: project.ID})
	if err == nil && existingService != nil {
		return nil, s.restErr.StatusConflict("service already exists for this project")
	}

	existingServiceByProxyURL, err := s.serviceRepo.Find(ctx, &models.Service{ProxyURL: req.ProxyURL})
	if err == nil && existingServiceByProxyURL != nil {
		return nil, s.restErr.StatusConflict("proxy URL is already in use")
	}

	scheme := req.Scheme
	if scheme == "" {
		scheme = models.SchemeHTTP
	}

	service := &models.Service{
		ProjectID:        project.ID,
		Source:           req.Source,
		Scheme:           scheme,
		TargetBackendURL: req.TargetBackendURL,
		ProxyURL:         req.ProxyURL,
		CacheEnabled:     req.CacheEnabled,
		CacheTTL:         req.CacheTTL,
	}

	if scheme == models.SchemeHTTPS {
		certPEM, keyPEM, err := tlsPkg.GenerateSelfSignedCert(req.ProxyURL)
		if err != nil {
			s.logger.Errorf("failed to generate self-signed certificate: %v", err)
			return nil, s.restErr.ServerError("failed to generate certificate")
		}
		service.TLSCertificate = string(certPEM)
		service.TLSKey = string(keyPEM)
		s.logger.Infof("Generated self-signed certificate for domain: %s", req.ProxyURL)

		if s.tlsManager != nil {
			if err := s.tlsManager.LoadCertificate(req.ProxyURL, certPEM, keyPEM); err != nil {
				s.logger.Warnf("Failed to load certificate into TLS manager cache: %v", err)
			}
		}
	}

	if req.Configuration != nil {
		configJSON, err := req.Configuration.ToJSON()
		if err != nil {
			s.logger.Errorf("failed to marshal configuration: %v", err)
			return nil, s.restErr.ServerError("failed to process configuration")
		}
		service.Configuration = datatypes.JSON(configJSON)
	}

	if err := s.serviceRepo.Create(ctx, service); err != nil {
		s.logger.Errorf("failed to create service: %v", err)
		return nil, s.restErr.ServerError("failed to create service")
	}

	config, _ := service.GetConfig()

	response := gin.H{
		"uid":                service.UID,
		"project_id":         project.UID,
		"source":             service.Source,
		"scheme":             service.Scheme,
		"target_backend_url": service.TargetBackendURL,
		"proxy_url":          service.ProxyURL,
		"cache_enabled":      service.CacheEnabled,
		"cache_ttl":          service.CacheTTL,
		"configuration":      config,
		"created_at":         service.CreatedAt,
		"updated_at":         service.UpdatedAt,
	}

	if service.Scheme == models.SchemeHTTPS && service.TLSCertificate != "" {
		certInfo, err := tlsPkg.ParseCertificateInfo(service.TLSCertificate, service.TLSCertResolver)
		if err == nil {
			response["certificate"] = certInfo
		}
	}

	return response, nil
}

func (s *Service) GetService(c *gin.Context) (interface{}, *pkg.RestErr) {
	ctx := c.Request.Context()

	user := GetUserFromContext(c)
	if user == nil {
		return nil, s.restErr.Unauthorized("unauthorized")
	}

	projectUID := c.Param("projectId")
	if projectUID == "" {
		return nil, s.restErr.BadRequest("project ID is required")
	}

	serviceId := c.Param("serviceId")
	if serviceId == "" {
		return nil, s.restErr.BadRequest("service ID is required")
	}

	project, err := s.projectRepo.Find(ctx, &models.Project{
		UID:    projectUID,
		UserID: user.ID,
	})
	if err != nil {
		return nil, s.restErr.NotFound("project not found")
	}

	service, err := s.serviceRepo.Find(ctx, &models.Service{
		UID:       serviceId,
		ProjectID: project.ID,
	})
	if err != nil {
		return nil, s.restErr.NotFound("service not found")
	}

	config, _ := service.GetConfig()

	response := gin.H{
		"uid":                service.UID,
		"project_id":         project.UID,
		"source":             service.Source,
		"scheme":             service.Scheme,
		"target_backend_url": service.TargetBackendURL,
		"proxy_url":          service.ProxyURL,
		"cache_enabled":      service.CacheEnabled,
		"cache_ttl":          service.CacheTTL,
		"configuration":      config,
		"created_at":         service.CreatedAt,
		"updated_at":         service.UpdatedAt,
	}

	if service.Scheme == models.SchemeHTTPS && service.TLSCertificate != "" {
		certInfo, err := tlsPkg.ParseCertificateInfo(service.TLSCertificate, service.TLSCertResolver)
		if err == nil {
			response["certificate"] = certInfo
		}
	}

	return response, nil
}

func (s *Service) GetServiceByProject(c *gin.Context) (interface{}, *pkg.RestErr) {
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
	if err != nil {
		return nil, s.restErr.NotFound("service not found for this project")
	}

	config, _ := service.GetConfig()

	response := gin.H{
		"uid":                service.UID,
		"project_id":         project.UID,
		"source":             service.Source,
		"scheme":             service.Scheme,
		"target_backend_url": service.TargetBackendURL,
		"proxy_url":          service.ProxyURL,
		"cache_enabled":      service.CacheEnabled,
		"cache_ttl":          service.CacheTTL,
		"configuration":      config,
		"created_at":         service.CreatedAt,
		"updated_at":         service.UpdatedAt,
	}

	if service.Scheme == models.SchemeHTTPS && service.TLSCertificate != "" {
		certInfo, err := tlsPkg.ParseCertificateInfo(service.TLSCertificate, service.TLSCertResolver)
		if err == nil {
			response["certificate"] = certInfo
		}
	}

	return response, nil
}

func (s *Service) UpdateService(c *gin.Context, req dto.UpdateServiceRequest) (interface{}, *pkg.RestErr) {
	ctx := c.Request.Context()

	user := GetUserFromContext(c)
	if user == nil {
		return nil, s.restErr.Unauthorized("unauthorized")
	}

	projectUID := c.Param("projectId")
	if projectUID == "" {
		return nil, s.restErr.BadRequest("project ID is required")
	}

	serviceId := c.Param("serviceId")
	if serviceId == "" {
		return nil, s.restErr.BadRequest("service ID is required")
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

	service, err := s.serviceRepo.Find(ctx, &models.Service{
		UID:       serviceId,
		ProjectID: project.ID,
	})
	if err != nil {
		return nil, s.restErr.NotFound("service not found")
	}

	if service.Source == models.SourceKubernetes && !IsAPIKeyAuth(c) {
		return nil, s.restErr.RequestNotAllowed("kubernetes source services can only be updated via the Kubernetes operator using API key authentication")
	}

	updates := make(map[string]interface{})

	if req.Source != "" {
		updates["source"] = req.Source
		service.Source = req.Source
	}

	if req.Scheme != "" {
		updates["scheme"] = req.Scheme
		service.Scheme = req.Scheme

		if req.Scheme == models.SchemeHTTPS && service.TLSCertificate == "" {
			certPEM, keyPEM, err := tlsPkg.GenerateSelfSignedCert(service.ProxyURL)
			if err != nil {
				s.logger.Errorf("failed to generate self-signed certificate: %v", err)
				return nil, s.restErr.ServerError("failed to generate certificate")
			}
			updates["tls_certificate"] = string(certPEM)
			updates["tls_key"] = string(keyPEM)
			updates["tls_cert_resolver"] = "selfsigned"
			service.TLSCertificate = string(certPEM)
			service.TLSKey = string(keyPEM)
			service.TLSCertResolver = "selfsigned"
			s.logger.Infof("Generated self-signed certificate for domain: %s", service.ProxyURL)

			if s.tlsManager != nil {
				if err := s.tlsManager.LoadCertificate(service.ProxyURL, certPEM, keyPEM); err != nil {
					s.logger.Warnf("Failed to load certificate into TLS manager cache: %v", err)
				}
			}
		}
		if req.Scheme == models.SchemeHTTP {
			updates["tls_certificate"] = ""
			updates["tls_key"] = ""
			updates["tls_cert_resolver"] = ""

			if s.tlsManager != nil {
				s.tlsManager.RemoveCertificate(service.ProxyURL)
			}
		}
	}

	if req.ProxyURL != "" {
		if service.Scheme == models.SchemeHTTPS {
			if s.tlsManager != nil {
				s.tlsManager.RemoveCertificate(service.ProxyURL)
			}

			certPEM, keyPEM, err := tlsPkg.GenerateSelfSignedCert(req.ProxyURL)
			if err != nil {
				s.logger.Errorf("failed to generate self-signed certificate: %v", err)
				return nil, s.restErr.ServerError("failed to generate certificate")
			}
			updates["tls_certificate"] = string(certPEM)
			updates["tls_key"] = string(keyPEM)
			service.TLSCertificate = string(certPEM)
			service.TLSKey = string(keyPEM)
			s.logger.Infof("Regenerated self-signed certificate for new domain: %s", req.ProxyURL)

			if s.tlsManager != nil {
				if err := s.tlsManager.LoadCertificate(req.ProxyURL, certPEM, keyPEM); err != nil {
					s.logger.Warnf("Failed to load certificate into TLS manager cache: %v", err)
				}
			}
		}
	}

	if req.TargetBackendURL != "" {
		updates["target_backend_url"] = req.TargetBackendURL
		service.TargetBackendURL = req.TargetBackendURL
	}

	if req.ProxyURL != "" {
		existingServiceByProxyURL, err := s.serviceRepo.Find(ctx, &models.Service{ProxyURL: req.ProxyURL})
		if err == nil && existingServiceByProxyURL != nil && existingServiceByProxyURL.ID != service.ID {
			return nil, s.restErr.StatusConflict("proxy URL is already in use")
		}
		updates["proxy_url"] = req.ProxyURL
		service.ProxyURL = req.ProxyURL
	}

	if req.CacheEnabled != nil {
		updates["cache_enabled"] = *req.CacheEnabled
		service.CacheEnabled = *req.CacheEnabled
	}

	if req.CacheTTL != nil {
		updates["cache_ttl"] = *req.CacheTTL
		service.CacheTTL = *req.CacheTTL
	}

	if req.Configuration != nil {
		configJSON, err := req.Configuration.ToJSON()
		if err != nil {
			s.logger.Errorf("failed to marshal configuration: %v", err)
			return nil, s.restErr.ServerError("failed to process configuration")
		}
		updates["configuration"] = datatypes.JSON(configJSON)
		service.Configuration = datatypes.JSON(configJSON)
	}

	if len(updates) == 0 {
		return nil, s.restErr.BadRequest("no fields to update")
	}

	if err := s.serviceRepo.Updates(ctx, &models.Service{
		ID: service.ID,
	}, updates); err != nil {
		s.logger.Errorf("failed to update service: %v", err)
		return nil, s.restErr.ServerError("failed to update service")
	}

	config, _ := service.GetConfig()

	response := gin.H{
		"uid":                service.UID,
		"project_id":         project.UID,
		"source":             service.Source,
		"scheme":             service.Scheme,
		"target_backend_url": service.TargetBackendURL,
		"proxy_url":          service.ProxyURL,
		"cache_enabled":      service.CacheEnabled,
		"cache_ttl":          service.CacheTTL,
		"configuration":      config,
		"created_at":         service.CreatedAt,
		"updated_at":         service.UpdatedAt,
	}

	if service.Scheme == models.SchemeHTTPS && service.TLSCertificate != "" {
		certInfo, err := tlsPkg.ParseCertificateInfo(service.TLSCertificate, service.TLSCertResolver)
		if err == nil {
			response["certificate"] = certInfo
		}
	}

	return response, nil
}
