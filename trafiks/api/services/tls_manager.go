package services

import (
	"context"
	"crypto/tls"
	"fmt"
	"sync"

	"github.com/trafiks/trafiks/api/repository"
	"github.com/trafiks/trafiks/models"
	"github.com/trafiks/trafiks/pkg"
)

// Manager manages TLS certificates for multiple domains with SNI support
type TLSManager struct {
	certificates map[string]*tls.Certificate
	mu           sync.RWMutex
	logger       pkg.LoggerClient
}

func NewTLSManager(logger pkg.LoggerClient) *TLSManager {
	return &TLSManager{
		certificates: make(map[string]*tls.Certificate),
		logger:       logger,
	}
}

func (m *TLSManager) LoadCertificate(domain string, certPEM, keyPEM []byte) error {
	if len(certPEM) == 0 || len(keyPEM) == 0 {
		return fmt.Errorf("certificate or key is empty for domain: %s", domain)
	}

	cert, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		return fmt.Errorf("failed to parse certificate for domain %s: %w", domain, err)
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	m.certificates[domain] = &cert

	return nil
}

func (m *TLSManager) LoadCertificateFromService(service *models.Service) error {
	if service.Scheme != models.SchemeHTTPS {
		return nil
	}

	if service.TLSCertificate == "" || service.TLSKey == "" {
		return fmt.Errorf("service %s has HTTPS scheme but no certificate", service.ProxyURL)
	}

	return m.LoadCertificate(service.ProxyURL, []byte(service.TLSCertificate), []byte(service.TLSKey))
}

func (m *TLSManager) RemoveCertificate(domain string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.certificates, domain)
}

func (m *TLSManager) GetCertificate(clientHello *tls.ClientHelloInfo) (*tls.Certificate, error) {
	domain := clientHello.ServerName
	if domain == "" {
		m.mu.RLock()
		defer m.mu.RUnlock()
		if len(m.certificates) == 0 {
			return nil, fmt.Errorf("no certificates available")
		}
		// Return first certificate.. not ideal, but better than failing
		for d, cert := range m.certificates {
			m.logger.Infof("TLS: No SNI provided, using first available certificate for domain: %s", d)
			return cert, nil
		}
	}

	m.mu.RLock()
	availableDomains := make([]string, 0, len(m.certificates))
	for d := range m.certificates {
		availableDomains = append(availableDomains, d)
	}
	m.mu.RUnlock()

	m.mu.RLock()
	cert, ok := m.certificates[domain]
	m.mu.RUnlock()

	if !ok {
		m.logger.Errorf("TLS handshake error: no certificate found for domain '%s'. Available domains: %v", domain, availableDomains)
		return nil, fmt.Errorf("no certificate found for domain: %s (available: %v)", domain, availableDomains)
	}

	m.logger.Infof("TLS: Found certificate for domain: %s", domain)
	return cert, nil
}

func (m *TLSManager) GetTLSConfig() *tls.Config {
	return &tls.Config{
		GetCertificate: m.GetCertificate,
		MinVersion:     tls.VersionTLS12,
	}
}

func (m *TLSManager) HasCertificates() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.certificates) > 0
}

func (m *TLSManager) GetDomainCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.certificates)
}

func (m *TLSManager) LoadAllFromDatabase(ctx context.Context, serviceRepo repository.ServiceRepoClient) error {
	services, err := serviceRepo.FindMany(ctx, &models.Service{Scheme: models.SchemeHTTPS})
	if err != nil {
		return fmt.Errorf("failed to load HTTPS services: %w", err)
	}

	loadedCount := 0
	for _, service := range services {
		if service.TLSCertificate != "" && service.TLSKey != "" {
			if err := m.LoadCertificateFromService(service); err != nil {
				m.logger.Errorf("Failed to load certificate for service %s (domain: %s): %v", service.UID, service.ProxyURL, err)
				continue
			}
			m.logger.Infof("Loaded certificate for domain: %s (service: %s)", service.ProxyURL, service.UID)
			loadedCount++

			continue
		}

		m.logger.Warnf("Service %s domain: %s has HTTPS scheme but no certificate/key", service.UID, service.ProxyURL)
	}

	if loadedCount == 0 && len(services) > 0 {
		m.logger.Warnf("No certificates were loaded from %d HTTPS service(s)", len(services))
	}

	return nil
}
