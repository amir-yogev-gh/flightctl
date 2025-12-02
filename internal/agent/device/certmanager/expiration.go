package certmanager

import (
	"crypto/x509"
	"fmt"
	"time"

	"github.com/flightctl/flightctl/internal/agent/device/certmanager/provider"
)

// ExpirationMonitor handles certificate expiration monitoring and calculations.
type ExpirationMonitor struct {
	log provider.Logger
}

// NewExpirationMonitor creates a new expiration monitor.
func NewExpirationMonitor(log provider.Logger) *ExpirationMonitor {
	return &ExpirationMonitor{
		log: log,
	}
}

// ParseCertificateExpiration extracts the expiration date from an X.509 certificate.
// Returns the expiration time (NotAfter) and any error encountered.
func (em *ExpirationMonitor) ParseCertificateExpiration(cert *x509.Certificate) (time.Time, error) {
	if cert == nil {
		return time.Time{}, fmt.Errorf("certificate is nil")
	}

	if cert.NotAfter.IsZero() {
		return time.Time{}, fmt.Errorf("certificate has no expiration date")
	}

	return cert.NotAfter, nil
}

// CalculateDaysUntilExpiration calculates the number of days until certificate expiration.
// Uses UTC timezone for consistent calculations across timezones.
// Returns negative days if certificate is already expired.
func (em *ExpirationMonitor) CalculateDaysUntilExpiration(cert *x509.Certificate) (int, error) {
	expiration, err := em.ParseCertificateExpiration(cert)
	if err != nil {
		return 0, err
	}

	now := time.Now().UTC()
	expirationUTC := expiration.UTC()

	// Calculate duration until expiration
	duration := expirationUTC.Sub(now)
	days := int(duration.Hours() / 24)

	return days, nil
}

// IsExpired checks if a certificate has expired.
// Returns true if the current time is after the certificate's NotAfter time.
func (em *ExpirationMonitor) IsExpired(cert *x509.Certificate) (bool, error) {
	if cert == nil {
		return false, fmt.Errorf("certificate is nil")
	}

	if cert.NotAfter.IsZero() {
		return false, fmt.Errorf("certificate has no expiration date")
	}

	now := time.Now().UTC()
	expirationUTC := cert.NotAfter.UTC()

	return now.After(expirationUTC), nil
}

// IsExpiringSoon checks if a certificate is expiring within the specified threshold.
// thresholdDays is the number of days before expiration to consider "soon".
// Returns true if certificate expires within thresholdDays.
func (em *ExpirationMonitor) IsExpiringSoon(cert *x509.Certificate, thresholdDays int) (bool, error) {
	if thresholdDays < 0 {
		return false, fmt.Errorf("threshold days must be non-negative")
	}

	daysUntilExpiration, err := em.CalculateDaysUntilExpiration(cert)
	if err != nil {
		return false, err
	}

	// Certificate is expiring soon if days until expiration <= threshold
	return daysUntilExpiration <= thresholdDays, nil
}
