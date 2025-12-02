package metrics

import (
	"testing"
	"time"

	"github.com/flightctl/flightctl/pkg/log"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewCertificateCollector(t *testing.T) {
	logger := log.NewPrefixLogger("test")
	collector := NewCertificateCollector(logger)

	assert.NotNil(t, collector)
	assert.NotNil(t, collector.certificateExpirationTimestamp)
	assert.NotNil(t, collector.certificateDaysUntilExpiration)
	assert.NotNil(t, collector.certificateRenewalAttempts)
	assert.NotNil(t, collector.certificateRenewalSuccess)
	assert.NotNil(t, collector.certificateRenewalFailures)
	assert.NotNil(t, collector.certificateRenewalDuration)
	assert.NotNil(t, collector.certificateRecoveryAttempts)
	assert.NotNil(t, collector.certificateRecoverySuccess)
	assert.NotNil(t, collector.certificateRecoveryFailures)
	assert.NotNil(t, collector.certificateRecoveryDuration)
}

func TestCertificateCollector_Describe(t *testing.T) {
	logger := log.NewPrefixLogger("test")
	collector := NewCertificateCollector(logger)

	ch := make(chan *prometheus.Desc, 100)
	go func() {
		collector.Describe(ch)
		close(ch)
	}()

	var descs []*prometheus.Desc
	for desc := range ch {
		descs = append(descs, desc)
	}

	// Should have at least 10 metric descriptions (one for each metric)
	assert.GreaterOrEqual(t, len(descs), 10)
}

func TestCertificateCollector_Collect(t *testing.T) {
	logger := log.NewPrefixLogger("test")
	collector := NewCertificateCollector(logger)

	// Record some metrics first
	now := time.Now()
	collector.RecordCertificateExpiration("management", "test-cert", now, 30)
	collector.RecordRenewalAttempt("management", "test-cert", "proactive")
	collector.RecordRecoveryAttempt("management", "test-cert")

	ch := make(chan prometheus.Metric, 100)
	go func() {
		collector.Collect(ch)
		close(ch)
	}()

	var metrics []prometheus.Metric
	for metric := range ch {
		metrics = append(metrics, metric)
	}

	// Should have metrics collected
	assert.Greater(t, len(metrics), 0)
}

func TestRecordCertificateExpiration(t *testing.T) {
	logger := log.NewPrefixLogger("test")
	collector := NewCertificateCollector(logger)

	now := time.Now()
	expirationTime := now.Add(30 * 24 * time.Hour) // 30 days from now
	daysUntilExpiration := 30

	collector.RecordCertificateExpiration("management", "test-cert", expirationTime, daysUntilExpiration)

	// Verify metrics were recorded by collecting them
	registry := prometheus.NewRegistry()
	err := registry.Register(collector)
	require.NoError(t, err)

	metrics, err := registry.Gather()
	require.NoError(t, err)

	// Find expiration timestamp metric
	var foundTimestamp bool
	var foundDays bool
	for _, mf := range metrics {
		switch *mf.Name {
		case "flightctl_agent_certificate_expiration_timestamp":
			foundTimestamp = true
			assert.Len(t, mf.Metric, 1)
			assert.Equal(t, float64(expirationTime.Unix()), *mf.Metric[0].Gauge.Value)
		case "flightctl_agent_certificate_days_until_expiration":
			foundDays = true
			assert.Len(t, mf.Metric, 1)
			assert.Equal(t, float64(daysUntilExpiration), *mf.Metric[0].Gauge.Value)
		}
	}

	assert.True(t, foundTimestamp, "expiration timestamp metric not found")
	assert.True(t, foundDays, "days until expiration metric not found")
}

func TestRecordRenewalMetrics(t *testing.T) {
	logger := log.NewPrefixLogger("test")
	collector := NewCertificateCollector(logger)

	certType := "management"
	certName := "test-cert"
	reason := "proactive"
	duration := 5 * time.Second

	// Record renewal attempt
	collector.RecordRenewalAttempt(certType, certName, reason)

	// Record renewal success
	collector.RecordRenewalSuccess(certType, certName, reason, duration)

	// Record renewal failure
	collector.RecordRenewalFailure(certType, certName, "expired", duration)

	// Verify metrics were recorded
	registry := prometheus.NewRegistry()
	err := registry.Register(collector)
	require.NoError(t, err)

	metrics, err := registry.Gather()
	require.NoError(t, err)

	// Find renewal metrics
	var foundAttempts bool
	var foundSuccess bool
	var foundFailures bool
	var foundDuration bool

	for _, mf := range metrics {
		switch *mf.Name {
		case "flightctl_agent_certificate_renewal_attempts_total":
			foundAttempts = true
			// Should have 2 attempts (one proactive, one expired)
			assert.GreaterOrEqual(t, len(mf.Metric), 1)
		case "flightctl_agent_certificate_renewal_success_total":
			foundSuccess = true
			assert.Len(t, mf.Metric, 1)
			assert.Equal(t, 1.0, *mf.Metric[0].Counter.Value)
		case "flightctl_agent_certificate_renewal_failures_total":
			foundFailures = true
			assert.Len(t, mf.Metric, 1)
			assert.Equal(t, 1.0, *mf.Metric[0].Counter.Value)
		case "flightctl_agent_certificate_renewal_duration_seconds":
			foundDuration = true
			// Should have duration recorded for both success and failure
			assert.GreaterOrEqual(t, len(mf.Metric), 1)
		}
	}

	assert.True(t, foundAttempts, "renewal attempts metric not found")
	assert.True(t, foundSuccess, "renewal success metric not found")
	assert.True(t, foundFailures, "renewal failures metric not found")
	assert.True(t, foundDuration, "renewal duration metric not found")
}

func TestRecordRecoveryMetrics(t *testing.T) {
	logger := log.NewPrefixLogger("test")
	collector := NewCertificateCollector(logger)

	certType := "management"
	certName := "test-cert"
	duration := 10 * time.Second

	// Record recovery attempt
	collector.RecordRecoveryAttempt(certType, certName)

	// Record recovery success
	collector.RecordRecoverySuccess(certType, certName, duration)

	// Record recovery failure
	collector.RecordRecoveryFailure(certType, certName, duration)

	// Verify metrics were recorded
	registry := prometheus.NewRegistry()
	err := registry.Register(collector)
	require.NoError(t, err)

	metrics, err := registry.Gather()
	require.NoError(t, err)

	// Find recovery metrics
	var foundAttempts bool
	var foundSuccess bool
	var foundFailures bool
	var foundDuration bool

	for _, mf := range metrics {
		switch *mf.Name {
		case "flightctl_agent_certificate_recovery_attempts_total":
			foundAttempts = true
			assert.Len(t, mf.Metric, 1)
			assert.Equal(t, 1.0, *mf.Metric[0].Counter.Value)
		case "flightctl_agent_certificate_recovery_success_total":
			foundSuccess = true
			assert.Len(t, mf.Metric, 1)
			assert.Equal(t, 1.0, *mf.Metric[0].Counter.Value)
		case "flightctl_agent_certificate_recovery_failures_total":
			foundFailures = true
			assert.Len(t, mf.Metric, 1)
			assert.Equal(t, 1.0, *mf.Metric[0].Counter.Value)
		case "flightctl_agent_certificate_recovery_duration_seconds":
			foundDuration = true
			// Should have duration recorded for both success and failure
			assert.GreaterOrEqual(t, len(mf.Metric), 1)
		}
	}

	assert.True(t, foundAttempts, "recovery attempts metric not found")
	assert.True(t, foundSuccess, "recovery success metric not found")
	assert.True(t, foundFailures, "recovery failures metric not found")
	assert.True(t, foundDuration, "recovery duration metric not found")
}

func TestCertificateCollector_ThreadSafety(t *testing.T) {
	logger := log.NewPrefixLogger("test")
	collector := NewCertificateCollector(logger)

	// Test concurrent access
	done := make(chan bool)
	numGoroutines := 10
	operationsPerGoroutine := 10

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			for j := 0; j < operationsPerGoroutine; j++ {
				now := time.Now()
				collector.RecordCertificateExpiration("management", "test-cert", now, 30)
				collector.RecordRenewalAttempt("management", "test-cert", "proactive")
				collector.RecordRenewalSuccess("management", "test-cert", "proactive", time.Second)
				collector.RecordRecoveryAttempt("management", "test-cert")
				collector.RecordRecoverySuccess("management", "test-cert", time.Second)
			}
			done <- true
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < numGoroutines; i++ {
		<-done
	}

	// Verify no panics occurred and metrics were recorded
	registry := prometheus.NewRegistry()
	err := registry.Register(collector)
	require.NoError(t, err)

	metrics, err := registry.Gather()
	require.NoError(t, err)
	assert.NotEmpty(t, metrics)
}

func TestCertificateCollector_AllMetricsRegistered(t *testing.T) {
	logger := log.NewPrefixLogger("test")
	collector := NewCertificateCollector(logger)

	// Record at least one metric for each type to ensure they're exposed
	now := time.Now()
	collector.RecordCertificateExpiration("management", "test-cert", now, 30)
	collector.RecordRenewalAttempt("management", "test-cert", "proactive")
	collector.RecordRenewalSuccess("management", "test-cert", "proactive", time.Second)
	collector.RecordRenewalFailure("management", "test-cert", "expired", time.Second)
	collector.RecordRecoveryAttempt("management", "test-cert")
	collector.RecordRecoverySuccess("management", "test-cert", time.Second)
	collector.RecordRecoveryFailure("management", "test-cert", time.Second)

	registry := prometheus.NewRegistry()
	err := registry.Register(collector)
	require.NoError(t, err)

	metrics, err := registry.Gather()
	require.NoError(t, err)

	metricNames := make(map[string]bool)
	for _, mf := range metrics {
		metricNames[*mf.Name] = true
	}

	expectedMetrics := []string{
		"flightctl_agent_certificate_expiration_timestamp",
		"flightctl_agent_certificate_days_until_expiration",
		"flightctl_agent_certificate_renewal_attempts_total",
		"flightctl_agent_certificate_renewal_success_total",
		"flightctl_agent_certificate_renewal_failures_total",
		"flightctl_agent_certificate_renewal_duration_seconds",
		"flightctl_agent_certificate_recovery_attempts_total",
		"flightctl_agent_certificate_recovery_success_total",
		"flightctl_agent_certificate_recovery_failures_total",
		"flightctl_agent_certificate_recovery_duration_seconds",
	}

	for _, expectedMetric := range expectedMetrics {
		assert.True(t, metricNames[expectedMetric], "Expected metric %s not found", expectedMetric)
	}
}
