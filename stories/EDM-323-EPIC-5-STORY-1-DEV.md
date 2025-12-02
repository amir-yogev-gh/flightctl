# Developer Story: Comprehensive Certificate Metrics

**Story ID:** EDM-323-EPIC-5-STORY-1  
**Epic:** EDM-323-EPIC-5 (Configuration and Observability)  
**Feature:** EDM-323 (Agent Certificate Rotation)  
**Story Points:** 5  
**Priority:** Medium

## Overview

Implement comprehensive Prometheus metrics for certificate lifecycle operations. Metrics should track certificate expiration, renewal operations, recovery operations, and operation durations to enable fleet-wide monitoring.

## Implementation Tasks

### Task 1: Create Certificate Metrics Collector

**File:** `internal/agent/instrumentation/metrics/certificate_collector.go` (new)

**Objective:** Create new metrics collector for certificate operations.

**Implementation Steps:**

1. **Create certificate_collector.go file:**
```go
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
    certificateRenewalSuccess   *prometheus.CounterVec
    certificateRenewalFailures  *prometheus.CounterVec
    certificateRenewalDuration *prometheus.HistogramVec

    // Recovery operation metrics
    certificateRecoveryAttempts *prometheus.CounterVec
    certificateRecoverySuccess   *prometheus.CounterVec
    certificateRecoveryFailures  *prometheus.CounterVec
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
```

**Testing:**
- Test CertificateCollector is created correctly
- Test all metrics are registered

---

### Task 2: Implement Metrics Collection Methods

**File:** `internal/agent/instrumentation/metrics/certificate_collector.go` (modify)

**Objective:** Add methods to record certificate metrics.

**Implementation Steps:**

1. **Add Describe and Collect methods:**
```go
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
```

2. **Add metric recording methods:**
```go
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
```

**Testing:**
- Test metrics are recorded correctly
- Test metrics are collected correctly
- Test thread safety

---

### Task 3: Integrate Metrics into Certificate Manager

**File:** `internal/agent/device/certmanager/manager.go` (modify)

**Objective:** Integrate metrics collection into certificate operations.

**Implementation Steps:**

1. **Add metrics collector to CertManager:**
```go
// In CertManager struct, add:
type CertManager struct {
    // ... existing fields ...
    metricsCollector *metrics.CertificateCollector
}

// In NewManager, add metrics collector:
func NewManager(ctx context.Context, log provider.Logger, opts ...ManagerOption) (*CertManager, error) {
    // ... existing code ...
    
    // Create metrics collector if metrics are enabled
    var metricsCollector *metrics.CertificateCollector
    if config != nil && config.MetricsEnabled {
        metricsCollector = metrics.NewCertificateCollector(log)
    }
    
    cm := &CertManager{
        // ... existing fields ...
        metricsCollector: metricsCollector,
    }
    
    return cm, nil
}
```

2. **Record metrics during operations:**
```go
// In ensureCertificate_do, record renewal metrics:
if isRenewal {
    startTime := time.Now()
    reason := "proactive"
    if cert.Info.NotAfter != nil && time.Now().After(*cert.Info.NotAfter) {
        reason = "expired"
    }
    
    if cm.metricsCollector != nil {
        cm.metricsCollector.RecordRenewalAttempt("management", cert.Name, reason)
    }
    
    // ... perform renewal ...
    
    if err == nil {
        duration := time.Since(startTime)
        if cm.metricsCollector != nil {
            cm.metricsCollector.RecordRenewalSuccess("management", cert.Name, reason, duration)
        }
    } else {
        duration := time.Since(startTime)
        if cm.metricsCollector != nil {
            cm.metricsCollector.RecordRenewalFailure("management", cert.Name, reason, duration)
        }
    }
}
```

**Testing:**
- Test metrics are recorded during operations
- Test metrics are not recorded when disabled

---

### Task 4: Record Expiration Metrics

**File:** `internal/agent/device/certmanager/manager.go` (modify)

**Objective:** Record certificate expiration metrics.

**Implementation Steps:**

1. **Add expiration metrics recording:**
```go
// In addCertificateInfo, record expiration metrics:
func (cm *CertManager) addCertificateInfo(cert *certificate, parsedCert *x509.Certificate) {
    cert.Info.NotBefore = &parsedCert.NotBefore
    cert.Info.NotAfter = &parsedCert.NotAfter

    // Record expiration metrics
    if cm.metricsCollector != nil {
        now := time.Now()
        daysUntilExpiration := int(parsedCert.NotAfter.Sub(now).Hours() / 24)
        cm.metricsCollector.RecordCertificateExpiration("management", cert.Name, parsedCert.NotAfter, daysUntilExpiration)
    }
}
```

2. **Update expiration metrics periodically:**
```go
// In CheckExpiredCertificates, update expiration metrics:
func (lm *LifecycleManager) CheckExpiredCertificates(ctx context.Context) error {
    // ... existing code ...
    
    // Update expiration metrics
    if cm.metricsCollector != nil {
        daysUntilExpiration := days
        expirationTime := cert.Info.NotAfter
        if expirationTime != nil {
            cm.metricsCollector.RecordCertificateExpiration("management", certName, *expirationTime, daysUntilExpiration)
        }
    }
    
    // ... rest of method ...
}
```

**Testing:**
- Test expiration metrics are recorded
- Test expiration metrics are updated

---

### Task 5: Record Recovery Metrics

**File:** `internal/agent/device/certmanager/lifecycle.go` (modify)

**Objective:** Record recovery operation metrics.

**Implementation Steps:**

1. **Add recovery metrics recording:**
```go
// In RecoverExpiredCertificate, record recovery metrics:
func (lm *LifecycleManager) RecoverExpiredCertificate(ctx context.Context, providerName string, certName string) error {
    startTime := time.Now()
    
    // Record recovery attempt
    if lm.metricsCollector != nil {
        lm.metricsCollector.RecordRecoveryAttempt("management", certName)
    }
    
    // ... perform recovery ...
    
    if err == nil {
        duration := time.Since(startTime)
        if lm.metricsCollector != nil {
            lm.metricsCollector.RecordRecoverySuccess("management", certName, duration)
        }
    } else {
        duration := time.Since(startTime)
        if lm.metricsCollector != nil {
            lm.metricsCollector.RecordRecoveryFailure("management", certName, duration)
        }
    }
    
    return err
}
```

**Testing:**
- Test recovery metrics are recorded
- Test recovery metrics include duration

---

### Task 6: Register Metrics Collector

**File:** `internal/agent/instrumentation/instrumentation.go` (modify)

**Objective:** Register certificate metrics collector with metrics server.

**Implementation Steps:**

1. **Register certificate collector:**
```go
// In NewMetricsServer, register certificate collector:
func NewMetricsServer(l *log.PrefixLogger, cfg *agent_config.Config) *metricsServer {
    ms := &metricsServer{log: l}
    if cfg == nil || !cfg.MetricsEnabled {
        return ms
    }

    rpcCollector := agentmetrics.NewRPCCollector(l)
    
    // Provide the Observe hook to producers
    cfg.SetEnrollmentMetricsCallback(rpcCollector.Observe)
    cfg.SetManagementMetricsCallback(rpcCollector.Observe)

    // Create certificate collector
    certCollector := agentmetrics.NewCertificateCollector(l)
    
    // Register collectors with metrics server
    ms.srv = instmetrics.NewMetricsServer(l, rpcCollector, certCollector)
    return ms
}
```

**Note:** This assumes the metrics server can accept multiple collectors. If not, we may need to create a combined collector.

**Testing:**
- Test metrics collector is registered
- Test metrics are exposed via Prometheus

---

### Task 7: Add Service-Side Metrics (Optional)

**File:** `internal/service/certificatesigningrequest.go` (modify)

**Objective:** Add service-side metrics for certificate operations.

**Implementation Steps:**

1. **Add service metrics (if metrics infrastructure exists):**
```go
// In CreateCertificateSigningRequest, record metrics:
if h.metricsCollector != nil {
    if h.isRenewalRequest(result) {
        reason := "proactive"
        if labels := result.Metadata.Labels; labels != nil {
            if r, ok := (*labels)["flightctl.io/renewal-reason"]; ok {
                reason = r
            }
        }
        h.metricsCollector.RecordRenewalRequest("renewal", reason)
    }
}

// In signApprovedCertificateSigningRequest, record metrics:
if h.metricsCollector != nil {
    startTime := time.Now()
    // ... sign certificate ...
    duration := time.Since(startTime)
    h.metricsCollector.RecordRenewalIssued("renewal", reason, duration)
}
```

**Note:** Service-side metrics depend on service metrics infrastructure. This may need to be implemented separately.

**Testing:**
- Test service metrics are recorded (if implemented)
- Test metrics are exposed correctly

---

## Unit Tests

### Test File: `internal/agent/instrumentation/metrics/certificate_collector_test.go` (new)

**Test Cases:**

1. **TestCertificateCollector:**
   - Collector is created correctly
   - All metrics are registered
   - Metrics are collected correctly

2. **TestRecordCertificateExpiration:**
   - Expiration metrics are recorded
   - Days until expiration is correct
   - Timestamp is correct

3. **TestRecordRenewalMetrics:**
   - Renewal attempts are recorded
   - Renewal success is recorded
   - Renewal failures are recorded
   - Duration is recorded

4. **TestRecordRecoveryMetrics:**
   - Recovery attempts are recorded
   - Recovery success is recorded
   - Recovery failures are recorded
   - Duration is recorded

---

## Integration Tests

### Test File: `test/integration/certificate_metrics_test.go` (new)

**Test Cases:**

1. **TestMetricsExposure:**
   - Metrics are exposed via Prometheus
   - Metrics follow naming conventions
   - Metrics have correct labels

2. **TestMetricsUpdates:**
   - Metrics are updated during operations
   - Metrics reflect current state
   - Metrics persist across restarts

---

## Code Review Checklist

- [ ] All metrics are defined correctly
- [ ] Metrics follow Prometheus conventions
- [ ] Metrics are recorded during operations
- [ ] Metrics are exposed via Prometheus
- [ ] Thread safety is ensured
- [ ] Unit tests achieve >80% coverage
- [ ] Integration tests cover main flows
- [ ] Code follows existing patterns

---

## Definition of Done

- [ ] CertificateCollector created
- [ ] All metrics defined
- [ ] Metrics recording methods implemented
- [ ] Integration with certificate manager added
- [ ] Metrics collector registered
- [ ] Service-side metrics added (if applicable)
- [ ] Unit tests written and passing (>80% coverage)
- [ ] Integration tests written and passing
- [ ] Code reviewed and approved
- [ ] Metrics documented
- [ ] Documentation updated

---

## Related Files

- `internal/agent/instrumentation/metrics/certificate_collector.go` - Certificate metrics
- `internal/agent/instrumentation/instrumentation.go` - Metrics server
- `internal/agent/device/certmanager/manager.go` - Certificate manager
- `internal/agent/device/certmanager/lifecycle.go` - Lifecycle manager

---

## Dependencies

- **EDM-323-EPIC-2**: Proactive Renewal (must be completed)
  - Requires renewal operations to be implemented
  
- **EDM-323-EPIC-4**: Expired Recovery (must be completed)
  - Requires recovery operations to be implemented

- **Existing Metrics Infrastructure**: Uses existing Prometheus metrics infrastructure

---

## Notes

- **Prometheus Conventions**: All metrics follow Prometheus naming conventions: lowercase with underscores, descriptive help text, appropriate metric types.

- **Metric Labels**: Metrics use labels (certificate_type, certificate_name, reason) to enable filtering and aggregation.

- **Thread Safety**: All metric operations are thread-safe using mutexes.

- **Performance**: Metric recording is lightweight and doesn't impact certificate operations.

- **Optional Metrics**: Metrics are only recorded if metrics are enabled in configuration. This prevents overhead when metrics are disabled.

- **Service Metrics**: Service-side metrics may require separate implementation depending on service metrics infrastructure.

---

**Document End**

