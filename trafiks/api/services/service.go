package services

import (
	"errors"

	"github.com/gin-gonic/gin"
	"gorm.io/datatypes"
	"gorm.io/gorm"

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
		if err := s.handleTLSCertificate(req, service); err != nil {
			return nil, err
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
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return gin.H{
				"project_id": project.UID,
			}, nil
		}

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

		if req.Scheme == models.SchemeHTTPS {
			if err := s.updateTLSCertificate(req, service, updates); err != nil {
				return nil, err
			}
		}

		if req.Scheme == models.SchemeHTTP {
			s.removeTLSCertificate(service, updates)
		}
	}

	if req.ProxyURL != "" && req.ProxyURL != service.ProxyURL {
		if err := s.handleProxyURLChange(req, service, updates); err != nil {
			return nil, err
		}
	}

	if req.TLSCertificate != "" && req.TLSKey != "" && service.Scheme == models.SchemeHTTPS {
		s.updateTLSCertificateOnly(req, service, updates)
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

func (s *Service) handleTLSCertificate(req dto.CreateServiceRequest, service *models.Service) *pkg.RestErr {
	if req.TLSCertificate != "" && req.TLSKey != "" {
		return s.setProvidedCertificate(req, service)
	}

	return s.generateSelfSignedCertificate(req.ProxyURL, service)
}

func (s *Service) setProvidedCertificate(req dto.CreateServiceRequest, service *models.Service) *pkg.RestErr {
	resolver := s.determineCertResolver(req.TLSCertResolver, "manual")
	s.setCertificateFields(req.TLSCertificate, req.TLSKey, resolver, service, nil)
	s.logger.Infof("Using provided certificate for domain: %s resolver: %s", req.ProxyURL, service.TLSCertResolver)
	s.loadCertificateToTLSManager(req.ProxyURL, []byte(req.TLSCertificate), []byte(req.TLSKey))
	return nil
}

func (s *Service) updateTLSCertificate(req dto.UpdateServiceRequest, service *models.Service, updates map[string]interface{}) *pkg.RestErr {
	if req.TLSCertificate != "" && req.TLSKey != "" {
		return s.setProvidedCertificateForUpdate(req, service, updates)
	}

	if service.TLSCertificate == "" {
		return s.generateSelfSignedCertificateForUpdate(service, updates)
	}

	return nil
}

func (s *Service) setProvidedCertificateForUpdate(req dto.UpdateServiceRequest, service *models.Service, updates map[string]interface{}) *pkg.RestErr {
	resolver := s.determineCertResolver(req.TLSCertResolver, "manual")
	s.setCertificateFields(req.TLSCertificate, req.TLSKey, resolver, service, updates)
	s.logger.Infof("Updated certificate for domain: %s (resolver: %s)", service.ProxyURL, service.TLSCertResolver)
	s.loadCertificateToTLSManager(service.ProxyURL, []byte(req.TLSCertificate), []byte(req.TLSKey))
	return nil
}

func (s *Service) generateSelfSignedCertificate(proxyURL string, service *models.Service) *pkg.RestErr {
	certPEM, keyPEM, err := tlsPkg.GenerateSelfSignedCert(proxyURL)
	if err != nil {
		s.logger.Errorf("failed to generate self-signed certificate: %v", err)
		return s.restErr.ServerError("failed to generate certificate")
	}

	s.setCertificateFields(string(certPEM), string(keyPEM), "selfsigned", service, nil)
	s.logger.Infof("Generated self-signed certificate for domain: %s", proxyURL)
	s.loadCertificateToTLSManager(proxyURL, certPEM, keyPEM)
	return nil
}

func (s *Service) generateSelfSignedCertificateForUpdate(service *models.Service, updates map[string]interface{}) *pkg.RestErr {
	certPEM, keyPEM, err := tlsPkg.GenerateSelfSignedCert(service.ProxyURL)
	if err != nil {
		s.logger.Errorf("failed to generate self-signed certificate: %v", err)
		return s.restErr.ServerError("failed to generate certificate")
	}

	s.setCertificateFields(string(certPEM), string(keyPEM), "selfsigned", service, updates)
	s.logger.Infof("Generated self-signed certificate for domain: %s", service.ProxyURL)
	s.loadCertificateToTLSManager(service.ProxyURL, certPEM, keyPEM)
	return nil
}

func (s *Service) removeTLSCertificate(service *models.Service, updates map[string]interface{}) {
	updates["tls_certificate"] = ""
	updates["tls_key"] = ""
	updates["tls_cert_resolver"] = ""

	if s.tlsManager != nil {
		s.tlsManager.RemoveCertificate(service.ProxyURL)
	}
}

func (s *Service) handleProxyURLChange(req dto.UpdateServiceRequest, service *models.Service, updates map[string]interface{}) *pkg.RestErr {
	if service.Scheme != models.SchemeHTTPS {
		return nil
	}

	if s.tlsManager != nil {
		s.tlsManager.RemoveCertificate(service.ProxyURL)
	}

	if req.TLSCertificate != "" && req.TLSKey != "" {
		return s.setCertificateForNewDomain(req, service, updates)
	}

	return s.generateCertificateForNewDomain(req.ProxyURL, service, updates)
}

func (s *Service) setCertificateForNewDomain(req dto.UpdateServiceRequest, service *models.Service, updates map[string]interface{}) *pkg.RestErr {
	resolver := s.determineCertResolver(req.TLSCertResolver, "")
	s.setCertificateFields(req.TLSCertificate, req.TLSKey, resolver, service, updates)
	s.logger.Infof("Updated certificate for new domain: %s", req.ProxyURL)
	s.loadCertificateToTLSManager(req.ProxyURL, []byte(req.TLSCertificate), []byte(req.TLSKey))
	return nil
}

func (s *Service) generateCertificateForNewDomain(proxyURL string, service *models.Service, updates map[string]interface{}) *pkg.RestErr {
	certPEM, keyPEM, err := tlsPkg.GenerateSelfSignedCert(proxyURL)
	if err != nil {
		s.logger.Errorf("failed to generate self-signed certificate: %v", err)
		return s.restErr.ServerError("failed to generate certificate")
	}

	s.setCertificateFields(string(certPEM), string(keyPEM), "selfsigned", service, updates)
	s.logger.Infof("Regenerated self-signed certificate for new domain: %s", proxyURL)
	s.loadCertificateToTLSManager(proxyURL, certPEM, keyPEM)
	return nil
}

func (s *Service) updateTLSCertificateOnly(req dto.UpdateServiceRequest, service *models.Service, updates map[string]interface{}) {
	resolver := s.determineCertResolver(req.TLSCertResolver, "")
	s.setCertificateFields(req.TLSCertificate, req.TLSKey, resolver, service, updates)
	s.loadCertificateToTLSManager(service.ProxyURL, []byte(req.TLSCertificate), []byte(req.TLSKey))
}

func (s *Service) setCertificateFields(certPEM, keyPEM, resolver string, service *models.Service, updates map[string]interface{}) {
	service.TLSCertificate = certPEM
	service.TLSKey = keyPEM

	if updates != nil {
		updates["tls_certificate"] = certPEM
		updates["tls_key"] = keyPEM
	}

	if resolver != "" {
		service.TLSCertResolver = resolver
		if updates != nil {
			updates["tls_cert_resolver"] = resolver
		}
	}
}

func (s *Service) determineCertResolver(resolver, defaultResolver string) string {
	if resolver != "" {
		return resolver
	}
	return defaultResolver
}

func (s *Service) loadCertificateToTLSManager(proxyURL string, certPEM, keyPEM []byte) {
	if s.tlsManager == nil {
		return
	}

	if err := s.tlsManager.LoadCertificate(proxyURL, certPEM, keyPEM); err != nil {
		s.logger.Warnf("Failed to load certificate into TLS manager cache: %v", err)
	}
}
