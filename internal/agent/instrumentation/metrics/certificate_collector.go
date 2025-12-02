package metrics

import (
	"sync"
	"time"

	"github.com/flightctl/flightctl/pkg/log"
	"github.com/prometheus/client_golang/prometheus"
)

// CertificateCollector collects metrics for certificate lifecycle operations.
type CertificateCollector struct {
	log *log.PrefixLogger
	mu  sync.RWMutex

	// Certificate expiration metrics
	certificateExpirationTimestamp *prometheus.GaugeVec
	certificateDaysUntilExpiration *prometheus.GaugeVec

	// Renewal operation metrics
	certificateRenewalAttempts *prometheus.CounterVec
	certificateRenewalSuccess  *prometheus.CounterVec
	certificateRenewalFailures *prometheus.CounterVec
	certificateRenewalDuration *prometheus.HistogramVec

	// Recovery operation metrics
	certificateRecoveryAttempts *prometheus.CounterVec
	certificateRecoverySuccess  *prometheus.CounterVec
	certificateRecoveryFailures *prometheus.CounterVec
	certificateRecoveryDuration *prometheus.HistogramVec
}

// NewCertificateCollector creates a new certificate metrics collector.
func NewCertificateCollector(l *log.PrefixLogger) *CertificateCollector {
	return &CertificateCollector{
		log: l,
		certificateExpirationTimestamp: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "flightctl_agent_certificate_expiration_timestamp",
				Help: "Certificate expiration timestamp (Unix seconds)",
			},
			[]string{"certificate_type", "certificate_name"},
		),
		certificateDaysUntilExpiration: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "flightctl_agent_certificate_days_until_expiration",
				Help: "Days until certificate expiration (negative if expired)",
			},
			[]string{"certificate_type", "certificate_name"},
		),
		certificateRenewalAttempts: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "flightctl_agent_certificate_renewal_attempts_total",
				Help: "Total number of certificate renewal attempts",
			},
			[]string{"certificate_type", "certificate_name", "reason"},
		),
		certificateRenewalSuccess: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "flightctl_agent_certificate_renewal_success_total",
				Help: "Total number of successful certificate renewals",
			},
			[]string{"certificate_type", "certificate_name", "reason"},
		),
		certificateRenewalFailures: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "flightctl_agent_certificate_renewal_failures_total",
				Help: "Total number of failed certificate renewals",
			},
			[]string{"certificate_type", "certificate_name", "reason"},
		),
		certificateRenewalDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "flightctl_agent_certificate_renewal_duration_seconds",
				Help:    "Certificate renewal operation duration in seconds",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"certificate_type", "certificate_name", "reason"},
		),
		certificateRecoveryAttempts: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "flightctl_agent_certificate_recovery_attempts_total",
				Help: "Total number of certificate recovery attempts",
			},
			[]string{"certificate_type", "certificate_name"},
		),
		certificateRecoverySuccess: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "flightctl_agent_certificate_recovery_success_total",
				Help: "Total number of successful certificate recoveries",
			},
			[]string{"certificate_type", "certificate_name"},
		),
		certificateRecoveryFailures: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "flightctl_agent_certificate_recovery_failures_total",
				Help: "Total number of failed certificate recoveries",
			},
			[]string{"certificate_type", "certificate_name"},
		),
		certificateRecoveryDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "flightctl_agent_certificate_recovery_duration_seconds",
				Help:    "Certificate recovery operation duration in seconds",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"certificate_type", "certificate_name"},
		),
	}
}

// Describe implements prometheus.Collector.
func (c *CertificateCollector) Describe(ch chan<- *prometheus.Desc) {
	c.certificateExpirationTimestamp.Describe(ch)
	c.certificateDaysUntilExpiration.Describe(ch)
	c.certificateRenewalAttempts.Describe(ch)
	c.certificateRenewalSuccess.Describe(ch)
	c.certificateRenewalFailures.Describe(ch)
	c.certificateRenewalDuration.Describe(ch)
	c.certificateRecoveryAttempts.Describe(ch)
	c.certificateRecoverySuccess.Describe(ch)
	c.certificateRecoveryFailures.Describe(ch)
	c.certificateRecoveryDuration.Describe(ch)
}

// Collect implements prometheus.Collector.
func (c *CertificateCollector) Collect(ch chan<- prometheus.Metric) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	c.certificateExpirationTimestamp.Collect(ch)
	c.certificateDaysUntilExpiration.Collect(ch)
	c.certificateRenewalAttempts.Collect(ch)
	c.certificateRenewalSuccess.Collect(ch)
	c.certificateRenewalFailures.Collect(ch)
	c.certificateRenewalDuration.Collect(ch)
	c.certificateRecoveryAttempts.Collect(ch)
	c.certificateRecoverySuccess.Collect(ch)
	c.certificateRecoveryFailures.Collect(ch)
	c.certificateRecoveryDuration.Collect(ch)
}

// RecordCertificateExpiration records certificate expiration information.
func (c *CertificateCollector) RecordCertificateExpiration(certType, certName string, expirationTime time.Time, daysUntilExpiration int) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.certificateExpirationTimestamp.WithLabelValues(certType, certName).Set(float64(expirationTime.Unix()))
	c.certificateDaysUntilExpiration.WithLabelValues(certType, certName).Set(float64(daysUntilExpiration))
}

// RecordRenewalAttempt records a certificate renewal attempt.
func (c *CertificateCollector) RecordRenewalAttempt(certType, certName, reason string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.certificateRenewalAttempts.WithLabelValues(certType, certName, reason).Inc()
}

// RecordRenewalSuccess records a successful certificate renewal.
func (c *CertificateCollector) RecordRenewalSuccess(certType, certName, reason string, duration time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.certificateRenewalSuccess.WithLabelValues(certType, certName, reason).Inc()
	c.certificateRenewalDuration.WithLabelValues(certType, certName, reason).Observe(duration.Seconds())
}

// RecordRenewalFailure records a failed certificate renewal.
func (c *CertificateCollector) RecordRenewalFailure(certType, certName, reason string, duration time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.certificateRenewalFailures.WithLabelValues(certType, certName, reason).Inc()
	c.certificateRenewalDuration.WithLabelValues(certType, certName, reason).Observe(duration.Seconds())
}

// RecordRecoveryAttempt records a certificate recovery attempt.
func (c *CertificateCollector) RecordRecoveryAttempt(certType, certName string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.certificateRecoveryAttempts.WithLabelValues(certType, certName).Inc()
}

// RecordRecoverySuccess records a successful certificate recovery.
func (c *CertificateCollector) RecordRecoverySuccess(certType, certName string, duration time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.certificateRecoverySuccess.WithLabelValues(certType, certName).Inc()
	c.certificateRecoveryDuration.WithLabelValues(certType, certName).Observe(duration.Seconds())
}

// RecordRecoveryFailure records a failed certificate recovery.
func (c *CertificateCollector) RecordRecoveryFailure(certType, certName string, duration time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.certificateRecoveryFailures.WithLabelValues(certType, certName).Inc()
	c.certificateRecoveryDuration.WithLabelValues(certType, certName).Observe(duration.Seconds())
}
