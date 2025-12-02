# Developer Story: Expired Certificate Detection

**Story ID:** EDM-323-EPIC-4-STORY-1  
**Epic:** EDM-323-EPIC-4 (Expired Certificate Recovery)  
**Feature:** EDM-323 (Agent Certificate Rotation)  
**Story Points:** 3  
**Priority:** High

## Overview

Implement expired certificate detection that runs on agent startup and periodically during operation. The detection should differentiate between "expiring soon" and "already expired" states and trigger recovery when certificates are expired.

## Implementation Tasks

### Task 1: Add Expired Certificate Detection Method

**File:** `internal/agent/device/certmanager/lifecycle.go` (modify)

**Objective:** Add method to detect expired certificates.

**Implementation Steps:**

1. **Add DetectExpiredCertificate method:**
```go
// DetectExpiredCertificate detects if a certificate is expired or expiring soon.
// Returns the expiration state and days until expiration (negative if expired).
func (lm *LifecycleManager) DetectExpiredCertificate(ctx context.Context, providerName string, certName string) (CertificateState, int, error) {
    // Get certificate expiration info
    cert, err := lm.certManager.certificates.ReadCertificate(providerName, certName)
    if err != nil {
        return CertificateStateNormal, 0, fmt.Errorf("failed to read certificate: %w", err)
    }

    cert.mu.RLock()
    defer cert.mu.RUnlock()

    // Check if certificate has expiration info
    if cert.Info.NotAfter == nil {
        // No expiration info - assume normal (may be initial provisioning)
        return CertificateStateNormal, 0, nil
    }

    // Calculate days until expiration
    now := time.Now()
    expirationTime := cert.Info.NotAfter
    daysUntilExpiration := int(expirationTime.Sub(now).Hours() / 24)

    // Determine state based on expiration
    if expirationTime.Before(now) {
        // Certificate is expired
        return CertificateStateExpired, daysUntilExpiration, nil
    }

    // Check if expiring soon (within threshold)
    thresholdDays := 30 // Default threshold
    if lm.config != nil {
        thresholdDays = lm.config.Certificate.Renewal.ThresholdDays
        if thresholdDays == 0 {
            thresholdDays = 30
        }
    }

    if daysUntilExpiration <= thresholdDays {
        // Certificate is expiring soon
        return CertificateStateExpiringSoon, daysUntilExpiration, nil
    }

    // Certificate is normal
    return CertificateStateNormal, daysUntilExpiration, nil
}
```

**Testing:**
- Test DetectExpiredCertificate with expired certificate
- Test DetectExpiredCertificate with expiring soon certificate
- Test DetectExpiredCertificate with normal certificate
- Test DetectExpiredCertificate with missing expiration info

---

### Task 2: Add Periodic Expiration Checking

**File:** `internal/agent/device/certmanager/lifecycle.go` (modify)

**Objective:** Add periodic checking for expired certificates.

**Implementation Steps:**

1. **Add CheckExpiredCertificates method:**
```go
// CheckExpiredCertificates checks all managed certificates for expiration.
// This should be called periodically to detect expired certificates.
func (lm *LifecycleManager) CheckExpiredCertificates(ctx context.Context) error {
    lm.log.Debug("Checking for expired certificates")

    // Iterate through all certificates
    for providerName, provider := range lm.certManager.certificates.providers {
        for certName := range provider.Certificates {
            state, days, err := lm.DetectExpiredCertificate(ctx, providerName, certName)
            if err != nil {
                lm.log.Warnf("Failed to check expiration for certificate %q/%q: %v", providerName, certName, err)
                continue
            }

            // Update certificate state
            currentState, err := lm.GetCertificateState(ctx, providerName, certName)
            if err == nil {
                if currentState.GetState() != state {
                    // State changed - update it
                    expirationTime := time.Now().Add(time.Duration(days) * 24 * time.Hour)
                    if err := lm.UpdateCertificateState(ctx, providerName, certName, state, days, &expirationTime); err != nil {
                        lm.log.Warnf("Failed to update certificate state: %v", err)
                    }

                    // If certificate is expired, trigger recovery
                    if state == CertificateStateExpired {
                        lm.log.Warnf("Certificate %q/%q is expired (expired %d days ago), triggering recovery", 
                            providerName, certName, -days)
                        if err := lm.TriggerRecovery(ctx, providerName, certName); err != nil {
                            lm.log.Errorf("Failed to trigger recovery for expired certificate %q/%q: %v", 
                                providerName, certName, err)
                        }
                    }
                }
            }
        }
    }

    return nil
}
```

**Testing:**
- Test CheckExpiredCertificates detects expired certificates
- Test CheckExpiredCertificates updates state
- Test CheckExpiredCertificates triggers recovery

---

### Task 3: Add Startup Expiration Check

**File:** `internal/agent/device/certmanager/manager.go` (modify)

**Objective:** Check for expired certificates on agent startup.

**Implementation Steps:**

1. **Add CheckExpiredCertificatesOnStartup method:**
```go
// CheckExpiredCertificatesOnStartup checks for expired certificates on agent startup.
// This should be called during agent initialization.
func (cm *CertManager) CheckExpiredCertificatesOnStartup(ctx context.Context) error {
    if cm.lifecycleManager == nil {
        return nil // No lifecycle manager - skip check
    }

    cm.log.Debug("Checking for expired certificates on startup")

    // Check all certificates
    if err := cm.lifecycleManager.CheckExpiredCertificates(ctx); err != nil {
        cm.log.Warnf("Failed to check expired certificates on startup: %v", err)
        return err
    }

    return nil
}
```

2. **Call from agent startup:**
```go
// In agent initialization, after CertManager is created:
if err := certManager.CheckExpiredCertificatesOnStartup(ctx); err != nil {
    log.Warnf("Failed to check expired certificates on startup: %v", err)
}
```

**Testing:**
- Test startup check detects expired certificates
- Test startup check triggers recovery
- Test startup check handles errors

---

### Task 4: Add Periodic Expiration Check Loop

**File:** `internal/agent/device/certmanager/manager.go` (modify)

**Objective:** Add periodic expiration checking during agent operation.

**Implementation Steps:**

1. **Add StartExpirationMonitoring method:**
```go
// StartExpirationMonitoring starts a goroutine that periodically checks for expired certificates.
func (cm *CertManager) StartExpirationMonitoring(ctx context.Context) {
    if cm.lifecycleManager == nil {
        return // No lifecycle manager - skip monitoring
    }

    checkInterval := 24 * time.Hour // Default: check daily
    if cm.config != nil {
        checkInterval = time.Duration(cm.config.Certificate.Renewal.CheckInterval)
        if checkInterval == 0 {
            checkInterval = 24 * time.Hour
        }
    }

    go func() {
        ticker := time.NewTicker(checkInterval)
        defer ticker.Stop()

        // Check immediately on startup
        if err := cm.lifecycleManager.CheckExpiredCertificates(ctx); err != nil {
            cm.log.Warnf("Failed to check expired certificates: %v", err)
        }

        for {
            select {
            case <-ctx.Done():
                cm.log.Debug("Expiration monitoring stopped")
                return
            case <-ticker.C:
                if err := cm.lifecycleManager.CheckExpiredCertificates(ctx); err != nil {
                    cm.log.Warnf("Failed to check expired certificates: %v", err)
                }
            }
        }
    }()
}
```

2. **Call from agent startup:**
```go
// In agent initialization, after CertManager is created:
certManager.StartExpirationMonitoring(ctx)
```

**Testing:**
- Test periodic checking runs at correct interval
- Test periodic checking stops on context cancellation
- Test periodic checking detects expired certificates

---

### Task 5: Add TriggerRecovery Method (Placeholder)

**File:** `internal/agent/device/certmanager/lifecycle.go` (modify)

**Objective:** Add placeholder for recovery triggering (will be implemented in later story).

**Implementation Steps:**

1. **Add TriggerRecovery method:**
```go
// TriggerRecovery triggers recovery for an expired certificate.
// This is a placeholder that will be fully implemented in EDM-323-EPIC-4-STORY-5.
func (lm *LifecycleManager) TriggerRecovery(ctx context.Context, providerName string, certName string) error {
    lm.log.Infof("Triggering recovery for expired certificate %q/%q", providerName, certName)

    // Update state to recovering
    if err := lm.SetCertificateState(ctx, providerName, certName, CertificateStateRecovering); err != nil {
        return fmt.Errorf("failed to set recovery state: %w", err)
    }

    // TODO: Implement full recovery flow in EDM-323-EPIC-4-STORY-5
    // For now, just log and update state
    lm.log.Warnf("Recovery triggered for certificate %q/%q (full implementation in next story)", providerName, certName)

    return nil
}
```

**Testing:**
- Test TriggerRecovery updates state
- Test TriggerRecovery logs appropriately

---

### Task 6: Integrate with Certificate Manager Sync

**File:** `internal/agent/device/certmanager/manager.go` (modify)

**Objective:** Check for expired certificates during certificate sync.

**Implementation Steps:**

1. **Add expiration check to sync:**
```go
// In sync method, after syncing certificates:
func (cm *CertManager) sync(ctx context.Context) error {
    // ... existing sync code ...

    // NEW: Check for expired certificates after sync
    if cm.lifecycleManager != nil {
        if err := cm.lifecycleManager.CheckExpiredCertificates(ctx); err != nil {
            cm.log.Warnf("Failed to check expired certificates during sync: %v", err)
            // Don't fail sync if expiration check fails
        }
    }

    return nil
}
```

**Testing:**
- Test expiration check during sync
- Test sync continues if check fails

---

### Task 7: Add Logging and Error Handling

**File:** `internal/agent/device/certmanager/lifecycle.go` (modify)

**Objective:** Add comprehensive logging for expiration detection.

**Implementation Steps:**

1. **Add logging in DetectExpiredCertificate:**
```go
// In DetectExpiredCertificate, add logging:
if state == CertificateStateExpired {
    lm.log.Warnf("Certificate %q/%q is expired (expired %d days ago)", providerName, certName, -days)
} else if state == CertificateStateExpiringSoon {
    lm.log.Infof("Certificate %q/%q is expiring soon (%d days remaining)", providerName, certName, days)
}
```

2. **Add logging in CheckExpiredCertificates:**
```go
// In CheckExpiredCertificates, add logging:
lm.log.Debugf("Checking expiration for certificate %q/%q", providerName, certName)
lm.log.Warnf("Certificate %q/%q is expired, triggering recovery", providerName, certName)
```

**Testing:**
- Test logging includes relevant information
- Test log levels are appropriate

---

## Unit Tests

### Test File: `internal/agent/device/certmanager/lifecycle_expiration_test.go` (new)

**Test Cases:**

1. **TestDetectExpiredCertificate:**
   - Detects expired certificate
   - Detects expiring soon certificate
   - Detects normal certificate
   - Handles missing expiration info

2. **TestCheckExpiredCertificates:**
   - Checks all certificates
   - Updates state correctly
   - Triggers recovery for expired certificates
   - Handles errors gracefully

3. **TestStartupExpirationCheck:**
   - Startup check detects expired certificates
   - Startup check triggers recovery
   - Startup check handles errors

4. **TestPeriodicExpirationCheck:**
   - Periodic check runs at correct interval
   - Periodic check stops on cancellation
   - Periodic check detects expired certificates

---

## Integration Tests

### Test File: `test/integration/certificate_expiration_detection_test.go` (new)

**Test Cases:**

1. **TestExpirationDetectionOnStartup:**
   - Agent detects expired certificate on startup
   - Agent triggers recovery
   - Agent continues operating

2. **TestPeriodicExpirationDetection:**
   - Periodic check detects expired certificates
   - Periodic check updates state
   - Periodic check triggers recovery

3. **TestExpirationStateTransitions:**
   - State transitions from normal to expiring soon
   - State transitions from expiring soon to expired
   - State transitions trigger appropriate actions

---

## Code Review Checklist

- [ ] Expiration detection is accurate
- [ ] State differentiation works correctly
- [ ] Periodic checking is implemented
- [ ] Startup checking is implemented
- [ ] Recovery triggering is integrated
- [ ] Error handling is comprehensive
- [ ] Logging is appropriate
- [ ] Unit tests achieve >80% coverage
- [ ] Integration tests cover main flows
- [ ] Code follows existing patterns

---

## Definition of Done

- [ ] DetectExpiredCertificate method implemented
- [ ] CheckExpiredCertificates method implemented
- [ ] Startup expiration check implemented
- [ ] Periodic expiration check implemented
- [ ] TriggerRecovery placeholder added
- [ ] Integration with certificate manager added
- [ ] Logging and error handling added
- [ ] Unit tests written and passing (>80% coverage)
- [ ] Integration tests written and passing
- [ ] Code reviewed and approved
- [ ] Documentation updated

---

## Related Files

- `internal/agent/device/certmanager/lifecycle.go` - Lifecycle manager
- `internal/agent/device/certmanager/manager.go` - Certificate manager
- `internal/agent/agent.go` - Agent initialization

---

## Dependencies

- **EDM-323-EPIC-1-STORY-1**: Certificate Expiration Monitoring (must be completed)
  - Requires expiration monitor to be available
  
- **EDM-323-EPIC-1-STORY-2**: Certificate Lifecycle Manager (must be completed)
  - Requires lifecycle manager to be available

---

## Notes

- **State Differentiation**: The detection differentiates between "expiring soon" (within threshold) and "already expired" (past expiration date). This allows different handling for each case.

- **Periodic Checking**: Expiration checking runs periodically (default: daily) to detect certificates that expire between agent restarts.

- **Startup Checking**: Expiration checking also runs on agent startup to detect expired certificates immediately.

- **Recovery Triggering**: When an expired certificate is detected, recovery is triggered. The full recovery flow will be implemented in later stories.

- **Error Handling**: Expiration detection errors don't block certificate operations. Errors are logged and operations continue.

- **Performance**: Expiration checking is lightweight (just time comparisons). It doesn't impact normal certificate operations.

---

**Document End**

