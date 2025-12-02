package certmanager

import (
	"fmt"
	"time"

	"github.com/flightctl/flightctl/internal/agent/device/certmanager/provider"
)

// CertificateLogContext contains context for certificate operation logging.
type CertificateLogContext struct {
	Operation           string // renewal, recovery, swap, validation, csr_generation, csr_submission
	CertificateType     string // management, bootstrap
	CertificateName     string
	DeviceName          string
	Reason              string // proactive, expired
	ThresholdDays       int
	DaysUntilExpiration int
	Duration            time.Duration
	Error               error
	Success             bool
}

// LogCertificateOperation logs a certificate operation with structured context.
// The log message includes all relevant context fields in a structured format.
func LogCertificateOperation(log provider.Logger, ctx CertificateLogContext) {
	// Build structured log message with key=value pairs
	var parts []string
	parts = append(parts, fmt.Sprintf("operation=%s", ctx.Operation))

	if ctx.CertificateType != "" {
		parts = append(parts, fmt.Sprintf("certificate_type=%s", ctx.CertificateType))
	}
	if ctx.CertificateName != "" {
		parts = append(parts, fmt.Sprintf("certificate_name=%s", ctx.CertificateName))
	}
	if ctx.DeviceName != "" {
		parts = append(parts, fmt.Sprintf("device_name=%s", ctx.DeviceName))
	}
	if ctx.Reason != "" {
		parts = append(parts, fmt.Sprintf("reason=%s", ctx.Reason))
	}
	if ctx.ThresholdDays > 0 {
		parts = append(parts, fmt.Sprintf("threshold_days=%d", ctx.ThresholdDays))
	}
	if ctx.DaysUntilExpiration != 0 {
		parts = append(parts, fmt.Sprintf("days_until_expiration=%d", ctx.DaysUntilExpiration))
	}
	if ctx.Duration > 0 {
		parts = append(parts, fmt.Sprintf("duration_seconds=%.3f", ctx.Duration.Seconds()))
	}
	if ctx.Error != nil {
		parts = append(parts, fmt.Sprintf("error=%v", ctx.Error))
		parts = append(parts, "success=false")
		log.Errorf("Certificate operation: %s", fmt.Sprint(parts))
	} else if ctx.Success {
		parts = append(parts, "success=true")
		log.Infof("Certificate operation: %s", fmt.Sprint(parts))
	} else {
		log.Infof("Certificate operation: %s", fmt.Sprint(parts))
	}
}

// LogCertificateError logs a certificate operation error with detailed context.
func LogCertificateError(log provider.Logger, operation string, certType string, certName string, err error, context map[string]interface{}) {
	var parts []string
	parts = append(parts, fmt.Sprintf("operation=%s", operation))
	parts = append(parts, fmt.Sprintf("certificate_type=%s", certType))
	parts = append(parts, fmt.Sprintf("certificate_name=%s", certName))
	parts = append(parts, fmt.Sprintf("error=%v", err))

	// Add additional context
	for k, v := range context {
		parts = append(parts, fmt.Sprintf("%s=%v", k, v))
	}

	log.Errorf("Certificate operation error: %s", fmt.Sprint(parts))
}
