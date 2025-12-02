# Developer Story: Enhanced Structured Logging for Certificate Operations

**Story ID:** EDM-323-EPIC-5-STORY-3  
**Epic:** EDM-323-EPIC-5 (Configuration and Observability)  
**Feature:** EDM-323 (Agent Certificate Rotation)  
**Story Points:** 3  
**Priority:** Medium

## Overview

Enhance logging for certificate operations with structured log entries that include operation context, success/failure status, error details, and duration. This enables effective troubleshooting of certificate issues.

## Implementation Tasks

### Task 1: Define Logging Structure

**File:** `internal/agent/device/certmanager/logging.go` (new)

**Objective:** Create logging utilities for certificate operations.

**Implementation Steps:**

1. **Create logging.go file:**
```go
package certmanager

import (
    "time"

    "github.com/flightctl/flightctl/internal/agent/device/certmanager/provider"
)

// CertificateLogContext contains context for certificate operation logging.
type CertificateLogContext struct {
    Operation      string    // renewal, recovery, swap, validation
    CertificateType string   // management, bootstrap
    CertificateName string
    DeviceName     string
    Reason         string    // proactive, expired
    ThresholdDays  int
    DaysUntilExpiration int
    Duration       time.Duration
    Error          error
    Success        bool
}

// LogCertificateOperation logs a certificate operation with structured context.
func LogCertificateOperation(log provider.Logger, ctx CertificateLogContext) {
    logFields := map[string]interface{}{
        "operation":           ctx.Operation,
        "certificate_type":    ctx.CertificateType,
        "certificate_name":    ctx.CertificateName,
        "device_name":         ctx.DeviceName,
    }

    if ctx.Reason != "" {
        logFields["reason"] = ctx.Reason
    }
    if ctx.ThresholdDays > 0 {
        logFields["threshold_days"] = ctx.ThresholdDays
    }
    if ctx.DaysUntilExpiration != 0 {
        logFields["days_until_expiration"] = ctx.DaysUntilExpiration
    }
    if ctx.Duration > 0 {
        logFields["duration_seconds"] = ctx.Duration.Seconds()
    }

    if ctx.Error != nil {
        logFields["error"] = ctx.Error.Error()
        logFields["success"] = false
        log.Errorf("Certificate operation failed: %+v", logFields)
    } else if ctx.Success {
        logFields["success"] = true
        log.Infof("Certificate operation completed: %+v", logFields)
    } else {
        log.Infof("Certificate operation: %+v", logFields)
    }
}
```

**Testing:**
- Test LogCertificateOperation logs correctly
- Test log fields are included

---

### Task 2: Add Logging to Renewal Operations

**File:** `internal/agent/device/certmanager/lifecycle.go` (modify)

**Objective:** Add structured logging to renewal operations.

**Implementation Steps:**

1. **Add logging to TriggerRenewal:**
```go
// In TriggerRenewal, add logging:
func (lm *LifecycleManager) TriggerRenewal(ctx context.Context, cert *certificate, cfg agent_config.CertificateRenewalConfig) error {
    startTime := time.Now()
    
    logCtx := CertificateLogContext{
        Operation:           "renewal",
        CertificateType:     "management",
        CertificateName:     cert.Name,
        DeviceName:          lm.deviceName,
        Reason:              "proactive",
        ThresholdDays:       cfg.ThresholdDays,
        DaysUntilExpiration: cert.Lifecycle.GetDaysUntilExpiration(),
    }
    
    LogCertificateOperation(lm.log, logCtx)
    
    // ... perform renewal ...
    
    logCtx.Duration = time.Since(startTime)
    if err != nil {
        logCtx.Error = err
        logCtx.Success = false
    } else {
        logCtx.Success = true
    }
    LogCertificateOperation(lm.log, logCtx)
    
    return err
}
```

**Testing:**
- Test renewal logging includes context
- Test renewal logging includes duration
- Test renewal logging includes errors

---

### Task 3: Add Logging to Recovery Operations

**File:** `internal/agent/device/certmanager/lifecycle.go` (modify)

**Objective:** Add structured logging to recovery operations.

**Implementation Steps:**

1. **Add logging to RecoverExpiredCertificate:**
```go
// In RecoverExpiredCertificate, add logging:
func (lm *LifecycleManager) RecoverExpiredCertificate(ctx context.Context, providerName string, certName string) error {
    startTime := time.Now()
    
    logCtx := CertificateLogContext{
        Operation:       "recovery",
        CertificateType: "management",
        CertificateName: certName,
        DeviceName:      lm.deviceName,
        Reason:          "expired",
    }
    
    LogCertificateOperation(lm.log, logCtx)
    
    // ... perform recovery ...
    
    logCtx.Duration = time.Since(startTime)
    if err != nil {
        logCtx.Error = err
        logCtx.Success = false
    } else {
        logCtx.Success = true
    }
    LogCertificateOperation(lm.log, logCtx)
    
    return err
}
```

**Testing:**
- Test recovery logging includes context
- Test recovery logging includes duration
- Test recovery logging includes errors

---

### Task 4: Add Logging to Atomic Swap

**File:** `internal/agent/device/certmanager/swap.go` (modify)

**Objective:** Add structured logging to atomic swap operations.

**Implementation Steps:**

1. **Add logging to AtomicSwap:**
```go
// In AtomicSwap, add logging:
func (cv *CertificateValidator) AtomicSwap(ctx context.Context, storage provider.StorageProvider) error {
    startTime := time.Now()
    
    logCtx := CertificateLogContext{
        Operation:       "swap",
        CertificateType: "management",
        CertificateName: cv.deviceName,
        DeviceName:      cv.deviceName,
    }
    
    LogCertificateOperation(cv.log, logCtx)
    
    // ... perform swap ...
    
    logCtx.Duration = time.Since(startTime)
    if err != nil {
        logCtx.Error = err
        logCtx.Success = false
    } else {
        logCtx.Success = true
    }
    LogCertificateOperation(cv.log, logCtx)
    
    return err
}
```

**Testing:**
- Test swap logging includes context
- Test swap logging includes duration
- Test swap logging includes errors

---

### Task 5: Add Logging to Certificate Validation

**File:** `internal/agent/device/certmanager/swap.go` (modify)

**Objective:** Add structured logging to certificate validation.

**Implementation Steps:**

1. **Add logging to ValidatePendingCertificate:**
```go
// In ValidatePendingCertificate, add logging:
func (cv *CertificateValidator) ValidatePendingCertificate(ctx context.Context, cert *x509.Certificate, keyPEM []byte, rw fileio.ReadWriter) error {
    startTime := time.Now()
    
    logCtx := CertificateLogContext{
        Operation:       "validation",
        CertificateType: "management",
        CertificateName: cv.deviceName,
        DeviceName:      cv.deviceName,
    }
    
    LogCertificateOperation(cv.log, logCtx)
    
    // ... perform validation ...
    
    logCtx.Duration = time.Since(startTime)
    if err != nil {
        logCtx.Error = err
        logCtx.Success = false
    } else {
        logCtx.Success = true
    }
    LogCertificateOperation(cv.log, logCtx)
    
    return err
}
```

**Testing:**
- Test validation logging includes context
- Test validation logging includes duration
- Test validation logging includes errors

---

### Task 6: Add Logging to CSR Operations

**File:** `internal/agent/device/certmanager/provider/provisioner/csr.go` (modify)

**Objective:** Add structured logging to CSR generation and submission.

**Implementation Steps:**

1. **Add logging to CSR provisioner:**
```go
// In Provision, add logging:
func (p *CSRProvisioner) Provision(ctx context.Context) (bool, *x509.Certificate, []byte, error) {
    logCtx := CertificateLogContext{
        Operation:       "csr_generation",
        CertificateType: "management",
        CertificateName: p.cfg.CommonName,
    }
    
    if p.renewalContext != nil {
        logCtx.Reason = p.renewalContext.Reason
        logCtx.DaysUntilExpiration = p.renewalContext.DaysUntilExpiration
    }
    
    LogCertificateOperation(p.log, logCtx)
    
    // ... generate CSR ...
    
    // Log CSR submission
    logCtx.Operation = "csr_submission"
    LogCertificateOperation(p.log, logCtx)
    
    // ... submit CSR ...
    
    return ready, cert, keyBytes, err
}
```

**Testing:**
- Test CSR logging includes context
- Test CSR logging includes renewal context

---

### Task 7: Enhance Error Logging

**File:** `internal/agent/device/certmanager/logging.go` (modify)

**Objective:** Add detailed error logging with context.

**Implementation Steps:**

1. **Add LogCertificateError method:**
```go
// LogCertificateError logs a certificate operation error with detailed context.
func LogCertificateError(log provider.Logger, operation string, certType string, certName string, err error, context map[string]interface{}) {
    logFields := map[string]interface{}{
        "operation":        operation,
        "certificate_type": certType,
        "certificate_name": certName,
        "error":           err.Error(),
    }
    
    // Add additional context
    for k, v := range context {
        logFields[k] = v
    }
    
    log.WithFields(logFields).Errorf("Certificate operation error: %v", err)
}
```

2. **Use LogCertificateError in error cases:**
```go
// In error handling, use LogCertificateError:
if err != nil {
    LogCertificateError(lm.log, "renewal", "management", cert.Name, err, map[string]interface{}{
        "reason":              "proactive",
        "days_until_expiration": cert.Lifecycle.GetDaysUntilExpiration(),
    })
    return err
}
```

**Testing:**
- Test error logging includes context
- Test error logging includes error details

---

## Unit Tests

### Test File: `internal/agent/device/certmanager/logging_test.go` (new)

**Test Cases:**

1. **TestLogCertificateOperation:**
   - Logs operation correctly
   - Includes all context fields
   - Handles success case
   - Handles error case

2. **TestLogCertificateError:**
   - Logs error correctly
   - Includes error details
   - Includes context

3. **TestLogFormat:**
   - Log format is consistent
   - Log fields are correct
   - Log levels are appropriate

---

## Integration Tests

### Test File: `test/integration/certificate_logging_test.go` (new)

**Test Cases:**

1. **TestRenewalLogging:**
   - Renewal operations are logged
   - Log entries include context
   - Log entries include duration

2. **TestRecoveryLogging:**
   - Recovery operations are logged
   - Log entries include context
   - Log entries include duration

3. **TestErrorLogging:**
   - Errors are logged with context
   - Error details are included
   - Log format is consistent

---

## Code Review Checklist

- [ ] Logging structure is well-defined
- [ ] Logging is added to all operations
- [ ] Log entries include context
- [ ] Log entries include duration
- [ ] Error logging is detailed
- [ ] Log format is consistent
- [ ] Log levels are appropriate
- [ ] Unit tests achieve >80% coverage
- [ ] Integration tests cover main flows
- [ ] Code follows existing patterns

---

## Definition of Done

- [ ] Logging structure defined
- [ ] Logging added to renewal operations
- [ ] Logging added to recovery operations
- [ ] Logging added to atomic swap
- [ ] Logging added to validation
- [ ] Logging added to CSR operations
- [ ] Error logging enhanced
- [ ] Unit tests written and passing (>80% coverage)
- [ ] Integration tests written and passing
- [ ] Code reviewed and approved
- [ ] Logging documented
- [ ] Documentation updated

---

## Related Files

- `internal/agent/device/certmanager/logging.go` - Logging utilities
- `internal/agent/device/certmanager/lifecycle.go` - Lifecycle manager
- `internal/agent/device/certmanager/swap.go` - Certificate swap
- `internal/agent/device/certmanager/provider/provisioner/csr.go` - CSR provisioner

---

## Dependencies

- **EDM-323-EPIC-2**: Proactive Renewal (must be completed)
  - Requires renewal operations
  
- **EDM-323-EPIC-3**: Atomic Swap (must be completed)
  - Requires atomic swap operations
  
- **EDM-323-EPIC-4**: Expired Recovery (must be completed)
  - Requires recovery operations

- **Existing Logging Infrastructure**: Uses existing logging infrastructure

---

## Notes

- **Structured Logging**: Log entries use structured format with consistent fields to enable log parsing and analysis.

- **Log Levels**: Log levels are chosen appropriately: INFO for normal operations, WARN for recoverable issues, ERROR for failures.

- **Context Fields**: Log entries include relevant context: operation type, certificate name, reason, duration, error details.

- **Duration Tracking**: Operation duration is tracked and included in log entries to enable performance analysis.

- **Error Details**: Error logging includes detailed error messages and context to enable troubleshooting.

- **Log Format**: Log format is consistent across all certificate operations to enable log aggregation and analysis.

---

**Document End**

