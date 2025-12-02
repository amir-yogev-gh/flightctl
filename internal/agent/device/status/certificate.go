package status

import (
	"context"
	"fmt"
	"time"

	"github.com/flightctl/flightctl/api/v1beta1"
	"github.com/flightctl/flightctl/internal/agent/device/certmanager"
	"github.com/flightctl/flightctl/pkg/log"
)

// CertificateStatus contains certificate lifecycle information for device status.
// This is a local type that matches the API structure.
// TODO: After OpenAPI spec is updated and types are regenerated, use v1beta1.CertificateStatus
type CertificateStatus struct {
	// Expiration is when the certificate expires
	Expiration *time.Time `json:"expiration,omitempty"`

	// DaysUntilExpiration is the number of days until expiration (negative if expired)
	DaysUntilExpiration *int `json:"daysUntilExpiration,omitempty"`

	// State is the current certificate lifecycle state
	// Values: normal, expiring_soon, renewing, expired, recovering, renewal_failed
	State string `json:"state,omitempty"`

	// LastRenewed is when the certificate was last renewed
	LastRenewed *time.Time `json:"lastRenewed,omitempty"`

	// RenewalCount is the number of times the certificate has been renewed
	RenewalCount *int `json:"renewalCount,omitempty"`
}

// CertificateExporter exports certificate status information.
type CertificateExporter struct {
	certManager *certmanager.CertManager
	log         *log.PrefixLogger
}

// NewCertificateExporter creates a new certificate status exporter.
func NewCertificateExporter(certManager *certmanager.CertManager, log *log.PrefixLogger) *CertificateExporter {
	return &CertificateExporter{
		certManager: certManager,
		log:         log,
	}
}

// Status implements status.Exporter.
func (ce *CertificateExporter) Status(ctx context.Context, deviceStatus *v1beta1.DeviceStatus, opts ...CollectorOpt) error {
	if ce.certManager == nil {
		return nil // No certificate manager - skip
	}

	// Get certificate information from certificate manager
	certStatus, err := ce.getCertificateStatus(ctx)
	if err != nil {
		ce.log.Warnf("Failed to get certificate status: %v", err)
		// Continue with empty status
		return nil
	}

	// Add certificate status to device status
	// Note: After OpenAPI spec is updated and types are regenerated, this will use deviceStatus.Certificate
	// For now, use CustomInfo as a workaround until types are regenerated
	if certStatus != nil {
		// Try to set Certificate field directly (will work after types are regenerated)
		// For now, also add to CustomInfo as fallback
		if deviceStatus.SystemInfo.CustomInfo == nil {
			customInfo := make(v1beta1.CustomDeviceInfo)
			deviceStatus.SystemInfo.CustomInfo = &customInfo
		}
		if certStatus.State != "" {
			(*deviceStatus.SystemInfo.CustomInfo)["certificate.state"] = certStatus.State
		}
		if certStatus.DaysUntilExpiration != nil {
			(*deviceStatus.SystemInfo.CustomInfo)["certificate.daysUntilExpiration"] = fmt.Sprintf("%d", *certStatus.DaysUntilExpiration)
		}
		if certStatus.Expiration != nil {
			(*deviceStatus.SystemInfo.CustomInfo)["certificate.expiration"] = certStatus.Expiration.Format(time.RFC3339)
		}
		if certStatus.LastRenewed != nil {
			(*deviceStatus.SystemInfo.CustomInfo)["certificate.lastRenewed"] = certStatus.LastRenewed.Format(time.RFC3339)
		}
		if certStatus.RenewalCount != nil {
			(*deviceStatus.SystemInfo.CustomInfo)["certificate.renewalCount"] = fmt.Sprintf("%d", *certStatus.RenewalCount)
		}
	}

	return nil
}

// getCertificateStatus collects certificate status information.
func (ce *CertificateExporter) getCertificateStatus(ctx context.Context) (*CertificateStatus, error) {
	// Get certificate manager's lifecycle manager
	lifecycleManager := ce.certManager.GetLifecycleManager()
	if lifecycleManager == nil {
		return nil, nil // No lifecycle manager - return nil
	}

	// Get management certificate status
	// Assume certificate name is "management" and provider is "builtin"
	providerName := "builtin"
	certName := "management"

	// Get certificate state
	lifecycleState, days, expiration, err := lifecycleManager.GetCertificateStatus(ctx, providerName, certName)
	if err != nil {
		return nil, fmt.Errorf("failed to get certificate status: %w", err)
	}

	if lifecycleState == nil {
		return nil, nil // No state available
	}

	status := &CertificateStatus{
		State: string(lifecycleState.GetState()),
	}

	// Add expiration information
	if expiration != nil {
		status.Expiration = expiration
	}
	if days != 0 {
		daysVal := days
		status.DaysUntilExpiration = &daysVal
	}

	// Use expiration from lifecycle state if available (more accurate)
	if lifecycleState.ExpirationTime != nil {
		status.Expiration = lifecycleState.ExpirationTime
	}
	if lifecycleState.DaysUntilExpiration != 0 {
		daysVal := lifecycleState.DaysUntilExpiration
		status.DaysUntilExpiration = &daysVal
	}

	// TODO: Add renewal information (LastRenewed, RenewalCount)
	// This requires access to device store or certificate tracking
	// For now, leave as nil - will be added when renewal tracking is available

	return status, nil
}
