package tls

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"net"
	"time"
)

// GenerateSelfSignedCert generates a self-signed TLS certificate for the given domain
// Returns PEM-encoded certificate and private key
func GenerateSelfSignedCert(domain string) (certPEM []byte, keyPEM []byte, err error) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to generate private key: %w", err)
	}

	notBefore := time.Now()
	notAfter := notBefore.Add(365 * 24 * time.Hour)

	serialNumber, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to generate serial number: %w", err)
	}

	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization:  []string{"Trafiks"},
			Country:       []string{"US"},
			Province:      []string{""},
			Locality:      []string{""},
			StreetAddress: []string{""},
			PostalCode:    []string{""},
			CommonName:    domain,
		},
		NotBefore:             notBefore,
		NotAfter:              notAfter,
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		IsCA:                  false,
	}

	template.DNSNames = []string{domain}

	if ip := net.ParseIP(domain); ip != nil {
		template.IPAddresses = []net.IP{ip}
	}

	derBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, &privateKey.PublicKey, privateKey)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create certificate: %w", err)
	}

	certPEM = pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: derBytes,
	})

	keyPEM = pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
	})

	return certPEM, keyPEM, nil
}

// CertificateInfo represents parsed certificate information
type CertificateInfo struct {
	CommonName      string    `json:"common_name"`
	Issuer          string    `json:"issuer"`
	Subject         string    `json:"subject"`
	ValidFrom       time.Time `json:"valid_from"`
	ValidTo         time.Time `json:"valid_to"`
	SerialNumber    string    `json:"serial_number"`
	DNSNames        []string  `json:"dns_names"`
	IsExpired       bool      `json:"is_expired"`
	DaysUntilExpiry int       `json:"days_until_expiry"`
	Type            string    `json:"type"` // "selfsigned", "letsencrypt", "manual"
}

// ParseCertificateInfo parses certificate details from PEM-encoded certificate
func ParseCertificateInfo(certPEM string, certResolver string) (*CertificateInfo, error) {
	if certPEM == "" {
		return nil, fmt.Errorf("certificate PEM is empty")
	}

	block, _ := pem.Decode([]byte(certPEM))
	if block == nil {
		return nil, fmt.Errorf("failed to decode PEM block")
	}

	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse certificate: %w", err)
	}

	now := time.Now()
	isExpired := now.After(cert.NotAfter)
	daysUntilExpiry := int(cert.NotAfter.Sub(now).Hours() / 24)
	if isExpired {
		daysUntilExpiry = 0
	}

	info := &CertificateInfo{
		CommonName:      cert.Subject.CommonName,
		Issuer:          cert.Issuer.String(),
		Subject:         cert.Subject.String(),
		ValidFrom:       cert.NotBefore,
		ValidTo:         cert.NotAfter,
		SerialNumber:    cert.SerialNumber.String(),
		DNSNames:        cert.DNSNames,
		IsExpired:       isExpired,
		DaysUntilExpiry: daysUntilExpiry,
		Type:            certResolver,
	}

	return info, nil
}
