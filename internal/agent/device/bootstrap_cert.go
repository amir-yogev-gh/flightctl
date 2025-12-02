package device

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"path/filepath"
	"time"

	agent_config "github.com/flightctl/flightctl/internal/agent/config"
	"github.com/flightctl/flightctl/internal/agent/device/fileio"
	fccrypto "github.com/flightctl/flightctl/pkg/crypto"
	"github.com/flightctl/flightctl/pkg/log"
)

// BootstrapCertificateHandler handles bootstrap certificate operations for recovery.
type BootstrapCertificateHandler struct {
	certPath string
	keyPath  string
	rw       fileio.ReadWriter
	log      *log.PrefixLogger
}

// NewBootstrapCertificateHandler creates a new bootstrap certificate handler.
func NewBootstrapCertificateHandler(dataDir string, rw fileio.ReadWriter, log *log.PrefixLogger) *BootstrapCertificateHandler {
	certPath := filepath.Join(dataDir, agent_config.DefaultCertsDirName, agent_config.EnrollmentCertFile)
	keyPath := filepath.Join(dataDir, agent_config.DefaultCertsDirName, agent_config.EnrollmentKeyFile)

	return &BootstrapCertificateHandler{
		certPath: certPath,
		keyPath:  keyPath,
		rw:       rw,
		log:      log,
	}
}

// GetBootstrapCertificate loads and returns the bootstrap certificate and key.
// Returns error if certificate is missing or invalid.
func (b *BootstrapCertificateHandler) GetBootstrapCertificate(ctx context.Context) (*x509.Certificate, []byte, error) {
	b.log.Debugf("Loading bootstrap certificate from %s", b.certPath)

	// Check if certificate file exists
	exists, err := b.rw.PathExists(b.certPath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to check bootstrap certificate existence: %w", err)
	}
	if !exists {
		return nil, nil, fmt.Errorf("bootstrap certificate not found at %s", b.certPath)
	}

	// Load certificate
	certPEM, err := b.rw.ReadFile(b.certPath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read bootstrap certificate: %w", err)
	}

	cert, err := fccrypto.ParsePEMCertificate(certPEM)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to parse bootstrap certificate: %w", err)
	}

	// Load key
	keyPEM, err := b.rw.ReadFile(b.keyPath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read bootstrap key: %w", err)
	}

	b.log.Debugf("Successfully loaded bootstrap certificate from %s", b.certPath)
	return cert, keyPEM, nil
}

// ValidateBootstrapCertificate validates that the bootstrap certificate is not expired.
func (b *BootstrapCertificateHandler) ValidateBootstrapCertificate(cert *x509.Certificate) error {
	now := time.Now()

	// Check if certificate is expired
	if cert.NotAfter.Before(now) {
		return fmt.Errorf("bootstrap certificate is expired (expired at %v)", cert.NotAfter)
	}

	// Check if certificate is not yet valid
	if cert.NotBefore.After(now) {
		return fmt.Errorf("bootstrap certificate is not yet valid (valid from %v)", cert.NotBefore)
	}

	return nil
}

// HasValidBootstrapCertificate checks if a valid bootstrap certificate exists.
func (b *BootstrapCertificateHandler) HasValidBootstrapCertificate(ctx context.Context) (bool, error) {
	cert, _, err := b.GetBootstrapCertificate(ctx)
	if err != nil {
		return false, nil // Certificate doesn't exist or can't be loaded
	}

	if err := b.ValidateBootstrapCertificate(cert); err != nil {
		b.log.Warnf("Bootstrap certificate exists but is invalid: %v", err)
		return false, nil
	}

	return true, nil
}

// GetCertificateForAuth returns the appropriate certificate for authentication.
// If management certificate is expired, falls back to bootstrap certificate.
func (b *BootstrapCertificateHandler) GetCertificateForAuth(ctx context.Context, managementCertPath, managementKeyPath string) (*tls.Certificate, error) {
	// First, try to load management certificate
	mgmtCertExists, err := b.rw.PathExists(managementCertPath)
	if err == nil && mgmtCertExists {
		mgmtCertPEM, err := b.rw.ReadFile(managementCertPath)
		if err == nil {
			mgmtCert, err := fccrypto.ParsePEMCertificate(mgmtCertPEM)
			if err == nil {
				// Check if management certificate is expired
				if time.Now().Before(mgmtCert.NotAfter) {
					// Management certificate is valid - use it
					mgmtKeyPEM, err := b.rw.ReadFile(managementKeyPath)
					if err == nil {
						tlsCert, err := tls.X509KeyPair(mgmtCertPEM, mgmtKeyPEM)
						if err == nil {
							b.log.Debug("Using management certificate for authentication")
							return &tlsCert, nil
						}
					}
				} else {
					b.log.Warnf("Management certificate is expired (expired at %v), falling back to bootstrap", mgmtCert.NotAfter)
				}
			}
		}
	}

	// Management certificate is expired or missing - fall back to bootstrap
	b.log.Infof("Falling back to bootstrap certificate for authentication")
	cert, keyPEM, err := b.GetBootstrapCertificate(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get bootstrap certificate: %w", err)
	}

	if err := b.ValidateBootstrapCertificate(cert); err != nil {
		return nil, fmt.Errorf("bootstrap certificate validation failed: %w", err)
	}

	// Create TLS certificate from bootstrap cert and key
	certPEM, err := fccrypto.EncodeCertificatePEM(cert)
	if err != nil {
		return nil, fmt.Errorf("failed to encode bootstrap certificate: %w", err)
	}

	tlsCert, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		return nil, fmt.Errorf("failed to create TLS certificate from bootstrap cert: %w", err)
	}

	b.log.Infof("Using bootstrap certificate for authentication")
	return &tlsCert, nil
}
