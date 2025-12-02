# Developer Story: Agent-Side Certificate Renewal Trigger

**Story ID:** EDM-323-EPIC-2-STORY-1  
**Epic:** EDM-323-EPIC-2 (Proactive Certificate Renewal)  
**Feature:** EDM-323 (Agent Certificate Rotation)  
**Story Points:** 5  
**Priority:** High

## Overview

Implement automatic certificate renewal triggering when certificates are approaching expiration. The trigger integrates with the expiration monitor and lifecycle manager to determine when renewal is needed and initiates the renewal process.

## Implementation Tasks

### Task 1: Add Renewal Check to Certificate Manager

**File:** `internal/agent/device/certmanager/manager.go` (modify)

**Objective:** Add method to check if a certificate needs renewal and trigger renewal if needed.

**Implementation Steps:**

1. **Add shouldRenewCertificate method:**
```go
// shouldRenewCertificate determines if a certificate needs renewal based on expiration threshold.
// Returns true if renewal is needed, days until expiration, and any error.
func (cm *CertManager) shouldRenewCertificate(ctx context.Context, providerName string, cert *certificate, thresholdDays int) (bool, int, error) {
    if cm.lifecycleManager == nil {
        return false, 0, fmt.Errorf("lifecycle manager not initialized")
    }
    
    if thresholdDays < 0 {
        return false, 0, fmt.Errorf("threshold days must be non-negative, got %d", thresholdDays)
    }
    
    // Use lifecycle manager to check if renewal is needed
    needsRenewal, days, err := cm.lifecycleManager.CheckRenewal(ctx, providerName, cert.Name, thresholdDays)
    if err != nil {
        return false, 0, fmt.Errorf("failed to check renewal: %w", err)
    }
    
    return needsRenewal, days, nil
}
```

2. **Add triggerRenewal method:**
```go
// triggerRenewal initiates the certificate renewal process.
// It sets the certificate state to "renewing" and queues the certificate for renewal.
func (cm *CertManager) triggerRenewal(ctx context.Context, providerName string, cert *certificate, cfg provider.CertificateConfig) error {
    if cm.lifecycleManager == nil {
        return fmt.Errorf("lifecycle manager not initialized")
    }
    
    // Set certificate state to "renewing"
    if err := cm.lifecycleManager.SetCertificateState(ctx, providerName, cert.Name, CertificateStateRenewing); err != nil {
        cm.log.Warnf("Failed to set certificate state to renewing for %q/%q: %v", providerName, cert.Name, err)
        // Continue anyway - state update failure shouldn't block renewal
    }
    
    // Queue certificate for renewal
    // Note: We use the same provisioning queue, but with renewal context
    // The CSR provisioner will handle renewal-specific logic in the next story
    if err := cm.provisionCertificate(ctx, providerName, cert, cfg); err != nil {
        // If queuing fails, reset state
        _ = cm.lifecycleManager.SetCertificateState(ctx, providerName, cert.Name, CertificateStateExpiringSoon)
        return fmt.Errorf("failed to queue certificate for renewal: %w", err)
    }
    
    cm.log.Infof("Triggered renewal for certificate %q/%q", providerName, cert.Name)
    return nil
}
```

**Testing:**
- Test shouldRenewCertificate with various expiration states
- Test triggerRenewal sets state correctly
- Test triggerRenewal queues certificate
- Test error handling

---

### Task 2: Integrate Renewal Check into Sync Flow

**File:** `internal/agent/device/certmanager/manager.go` (modify)

**Objective:** Add renewal check to the certificate sync flow.

**Implementation Steps:**

1. **Modify syncCertificate to check for renewal:**
```go
// syncCertificate synchronizes a single certificate.
func (cm *CertManager) syncCertificate(ctx context.Context, provider provider.ConfigProvider, cfg provider.CertificateConfig) error {
    var err error
    providerName := provider.Name()
    certName := cfg.Name

    cert, err := cm.certificates.ReadCertificate(providerName, certName)
    if err != nil {
        cert = cm.createCertificate(ctx, provider, cfg)
    }

    cert.mu.Lock()
    defer cert.mu.Unlock()

    if cm.processingQueue.IsProcessing(providerName, cert.Name) {
        _, usedCfg := cm.processingQueue.Get(providerName, cert.Name)

        if !usedCfg.Equal(cfg) {
            // Remove old queued item
            cm.processingQueue.Remove(providerName, cert.Name)

            // Re-queue with new config
            if err := cm.provisionCertificate(ctx, providerName, cert, cfg); err != nil {
                return fmt.Errorf("failed to provision certificate %q from provider %q: %w", cert.Name, providerName, err)
            }
            cm.log.Debugf("Config changed during processing — re-queued provision for certificate %q of provider %q", certName, providerName)
        }
        return nil
    }

    // Check if certificate needs provisioning (initial or config change)
    if !cm.shouldprovisionCertificate(providerName, cert, cfg) {
        cert.Config = cfg
        
        // NEW: Check if certificate needs renewal
        if cm.lifecycleManager != nil && cm.config != nil {
            thresholdDays := cm.config.Certificate.Renewal.ThresholdDays
            if thresholdDays == 0 {
                thresholdDays = 30 // Default threshold
            }
            
            // Only check renewal if certificate has expiration info
            if cert.Info.NotAfter != nil {
                needsRenewal, days, err := cm.shouldRenewCertificate(ctx, providerName, cert, thresholdDays)
                if err != nil {
                    cm.log.Warnf("Failed to check renewal for certificate %q/%q: %v", providerName, certName, err)
                } else if needsRenewal {
                    cm.log.Infof("Certificate %q/%q needs renewal (expires in %d days)", providerName, certName, days)
                    
                    // Check current state - don't trigger if already renewing
                    currentState, err := cm.lifecycleManager.GetCertificateState(ctx, providerName, certName)
                    if err == nil && currentState.GetState() != CertificateStateRenewing {
                        if err := cm.triggerRenewal(ctx, providerName, cert, cfg); err != nil {
                            cm.log.Errorf("Failed to trigger renewal for certificate %q/%q: %v", providerName, certName, err)
                            // Continue - renewal will be retried on next sync
                        }
                    }
                }
            }
        }
        
        cm.log.Debugf("Certificate %q for provider %q: no provision required", certName, providerName)
        return nil
    }

    if err := cm.provisionCertificate(ctx, providerName, cert, cfg); err != nil {
        return fmt.Errorf("failed to provision certificate %q from provider %q: %w", cert.Name, providerName, err)
    }

    cm.log.Debugf("Provision triggered for certificate %q of provider %q", certName, providerName)
    return nil
}
```

**Testing:**
- Test renewal check during sync
- Test renewal is not triggered if already renewing
- Test renewal is triggered when needed
- Test renewal is not triggered when not needed
- Test error handling during renewal check

---

### Task 3: Add Configuration Access to CertManager

**File:** `internal/agent/device/certmanager/manager.go` (modify)

**Objective:** Store configuration reference in CertManager to access renewal settings.

**Implementation Steps:**

1. **Add config field to CertManager:**
```go
type CertManager struct {
    log provider.Logger
    // ... existing fields ...
    expirationMonitor *ExpirationMonitor
    lifecycleManager  *LifecycleManager
    config            *config.Config  // NEW: Store config for renewal settings
    // ... rest of fields ...
}
```

2. **Update NewManager to accept config (or add option):**
```go
// WithConfig sets the agent configuration for the certificate manager.
func WithConfig(cfg *config.Config) ManagerOption {
    return func(cm *CertManager) error {
        if cfg == nil {
            return fmt.Errorf("config cannot be nil")
        }
        cm.config = cfg
        return nil
    }
}
```

3. **Update Sync method to store config:**
```go
// Sync performs a full synchronization of all certificate providers.
func (cm *CertManager) Sync(ctx context.Context, cfg *config.Config) error {
    // Store config for use in renewal checks
    cm.config = cfg
    
    cm.log.Debug("Starting certificate sync")
    if err := cm.sync(ctx); err != nil {
        cm.log.Errorf("certificate management sync failed: %v", err)
        return err
    }
    return nil
}
```

**Testing:**
- Test config is stored correctly
- Test config is accessible for renewal checks
- Test nil config handling

---

### Task 4: Add Renewal State Management

**File:** `internal/agent/device/certmanager/manager.go` (modify)

**Objective:** Ensure certificate state is properly managed during renewal.

**Implementation Steps:**

1. **Update state when renewal starts:**
The `triggerRenewal` method already sets state to "renewing". Ensure this is done correctly.

2. **Update state when renewal completes:**
In `ensureCertificate_do`, after successful certificate write, update state:

```go
// In ensureCertificate_do, after successful certificate write:
if err := cert.Storage.Write(crt, keyBytes); err != nil {
    return nil, err
}

cm.addCertificateInfo(cert, crt)

// Update lifecycle state based on renewal context
if cm.lifecycleManager != nil {
    // Check if this was a renewal (state was "renewing")
    currentState, err := cm.lifecycleManager.GetCertificateState(ctx, providerName, cert.Name)
    if err == nil && currentState.GetState() == CertificateStateRenewing {
        // Renewal completed successfully
        days, expiration, err := cm.lifecycleManager.CheckRenewal(ctx, providerName, cert.Name, 365) // Use large threshold to get days
        if err == nil {
            _ = cm.lifecycleManager.UpdateCertificateState(ctx, providerName, cert.Name, 
                CertificateStateNormal, days, expiration)
            cm.log.Infof("Certificate %q/%q renewal completed successfully", providerName, cert.Name)
        }
    } else {
        // Initial provisioning or other operation
        _ = cm.lifecycleManager.SetCertificateState(ctx, providerName, cert.Name, CertificateStateNormal)
    }
}

cert.Config = config
// ... rest of method ...
```

3. **Update state when renewal fails:**
In `ensureCertificate`, on error:

```go
// In ensureCertificate, on error:
if err != nil {
    cm.log.Errorf("failed to ensure certificate %q from provider %q: %v", cert.Name, providerName, err)
    
    // Update lifecycle state if this was a renewal
    if cm.lifecycleManager != nil {
        currentState, stateErr := cm.lifecycleManager.GetCertificateState(ctx, providerName, cert.Name)
        if stateErr == nil && currentState.GetState() == CertificateStateRenewing {
            // Renewal failed
            _ = cm.lifecycleManager.RecordError(ctx, providerName, cert.Name, err)
            // State will be set to renewal_failed by RecordError
        }
    }
    
    // ... rest of error handling ...
}
```

**Testing:**
- Test state is set to "renewing" when renewal starts
- Test state is updated to "normal" when renewal succeeds
- Test state is updated to "renewal_failed" when renewal fails
- Test state transitions are logged

---

### Task 5: Prevent Duplicate Renewal Triggers

**File:** `internal/agent/device/certmanager/manager.go` (modify)

**Objective:** Ensure renewal is not triggered multiple times for the same certificate.

**Implementation Steps:**

1. **Check state before triggering renewal:**
The renewal trigger already checks if certificate is already renewing. Enhance this check:

```go
// In syncCertificate, before triggering renewal:
// Check current state - don't trigger if already renewing
currentState, err := cm.lifecycleManager.GetCertificateState(ctx, providerName, certName)
if err == nil {
    state := currentState.GetState()
    if state == CertificateStateRenewing {
        cm.log.Debugf("Certificate %q/%q is already renewing, skipping trigger", providerName, certName)
        return nil
    }
    if state == CertificateStateRenewalFailed {
        // Allow retry after some time (handled by retry logic)
        cm.log.Debugf("Certificate %q/%q renewal previously failed, will retry", providerName, certName)
    }
}

// Check if certificate is already in processing queue
if cm.processingQueue.IsProcessing(providerName, certName) {
    cm.log.Debugf("Certificate %q/%q is already being processed, skipping renewal trigger", providerName, certName)
    return nil
}
```

2. **Add renewal tracking to prevent rapid re-triggers:**
```go
// Add field to track last renewal attempt
type CertManager struct {
    // ... existing fields ...
    lastRenewalAttempt map[string]time.Time  // Key: "providerName/certName"
    renewalAttemptMu  sync.RWMutex
}

// In NewManager:
cm.lastRenewalAttempt = make(map[string]time.Time)

// In triggerRenewal, check last attempt:
func (cm *CertManager) triggerRenewal(ctx context.Context, providerName string, cert *certificate, cfg provider.CertificateConfig) error {
    key := fmt.Sprintf("%s/%s", providerName, cert.Name)
    
    cm.renewalAttemptMu.RLock()
    lastAttempt, recentlyAttempted := cm.lastRenewalAttempt[key]
    cm.renewalAttemptMu.RUnlock()
    
    // Prevent rapid re-triggers (minimum 1 hour between attempts)
    if recentlyAttempted && time.Since(lastAttempt) < time.Hour {
        cm.log.Debugf("Certificate %q/%q renewal was recently attempted, skipping", providerName, cert.Name)
        return nil
    }
    
    // Record renewal attempt
    cm.renewalAttemptMu.Lock()
    cm.lastRenewalAttempt[key] = time.Now()
    cm.renewalAttemptMu.Unlock()
    
    // ... rest of triggerRenewal ...
}
```

**Testing:**
- Test renewal is not triggered if already renewing
- Test renewal is not triggered if recently attempted
- Test renewal can be retried after delay
- Test duplicate prevention works correctly

---

### Task 6: Add Renewal Metrics

**File:** `internal/agent/device/certmanager/manager.go` (modify)

**Objective:** Add metrics for renewal triggers (if metrics infrastructure exists).

**Implementation Steps:**

1. **Add renewal trigger metric (if metrics available):**
```go
// In triggerRenewal, after successful trigger:
if cm.metricsCallback != nil {
    cm.metricsCallback("certificate_renewal_triggered_total", 1.0, nil)
}

// In triggerRenewal, on failure:
if cm.metricsCallback != nil {
    cm.metricsCallback("certificate_renewal_trigger_failures_total", 1.0, err)
}
```

**Note:** This assumes a metrics callback exists. If not, this can be added in a later story.

**Testing:**
- Test metrics are recorded (if metrics infrastructure exists)
- Test metrics include correct labels

---

### Task 7: Add Logging for Renewal Triggers

**File:** `internal/agent/device/certmanager/manager.go` (modify)

**Objective:** Add appropriate logging for renewal trigger operations.

**Implementation Steps:**

1. **Add logging in renewal check:**
```go
// In syncCertificate, when checking renewal:
needsRenewal, days, err := cm.shouldRenewCertificate(ctx, providerName, cert, thresholdDays)
if err != nil {
    cm.log.Warnf("Failed to check renewal for certificate %q/%q: %v", providerName, certName, err)
} else if needsRenewal {
    cm.log.Infof("Certificate %q/%q needs renewal (expires in %d days, threshold: %d days)", 
        providerName, certName, days, thresholdDays)
    // ... trigger renewal ...
}
```

2. **Add logging in triggerRenewal:**
```go
// In triggerRenewal:
cm.log.Infof("Triggering renewal for certificate %q/%q (expires in %d days)", 
    providerName, cert.Name, days)

// On success:
cm.log.Infof("Successfully triggered renewal for certificate %q/%q", providerName, cert.Name)

// On failure:
cm.log.Errorf("Failed to trigger renewal for certificate %q/%q: %v", providerName, cert.Name, err)
```

**Testing:**
- Test logging includes relevant information
- Test log levels are appropriate (info for normal, warn/error for issues)

---

## Unit Tests

### Test File: `internal/agent/device/certmanager/manager_renewal_test.go` (new)

**Test Cases:**

1. **TestShouldRenewCertificate:**
   - Certificate expiring soon (needs renewal)
   - Certificate expired (needs renewal)
   - Certificate normal (no renewal)
   - Certificate not found (error)
   - Invalid threshold (error)
   - Lifecycle manager not initialized (error)

2. **TestTriggerRenewal:**
   - Successfully triggers renewal
   - Sets state to "renewing"
   - Queues certificate for processing
   - Handles lifecycle manager errors gracefully
   - Handles queue errors

3. **TestSyncCertificateWithRenewalCheck:**
   - Renewal check during sync
   - Renewal triggered when needed
   - Renewal not triggered when not needed
   - Renewal not triggered if already renewing
   - Renewal not triggered if recently attempted
   - Error handling during renewal check

4. **TestRenewalStateManagement:**
   - State set to "renewing" when triggered
   - State updated to "normal" on success
   - State updated to "renewal_failed" on failure
   - State preserved during other operations

5. **TestDuplicateRenewalPrevention:**
   - Renewal not triggered if already renewing
   - Renewal not triggered if recently attempted
   - Renewal can be retried after delay
   - Multiple certificates can renew simultaneously

6. **TestRenewalWithConfiguration:**
   - Uses configured threshold
   - Uses default threshold if not configured
   - Handles disabled renewal
   - Handles missing config gracefully

---

## Integration Tests

### Test File: `test/integration/certificate_renewal_trigger_test.go` (new)

**Test Cases:**

1. **TestRenewalTriggerFlow:**
   - Certificate approaching expiration triggers renewal
   - Certificate state transitions correctly
   - Renewal is queued for processing
   - Multiple syncs don't trigger duplicate renewals

2. **TestRenewalWithExpirationMonitor:**
   - Expiration monitor detects expiring certificate
   - Lifecycle manager determines renewal needed
   - Renewal is triggered
   - State is updated correctly

3. **TestRenewalThresholdConfiguration:**
   - Different thresholds trigger at different times
   - Default threshold works correctly
   - Configuration changes are applied

---

## Code Review Checklist

- [ ] Renewal check integrates with lifecycle manager
- [ ] Renewal trigger sets state correctly
- [ ] Renewal is queued for processing
- [ ] Duplicate renewal prevention works
- [ ] State transitions are handled correctly
- [ ] Error handling is comprehensive
- [ ] Logging is appropriate
- [ ] Configuration is accessed correctly
- [ ] Unit tests achieve >80% coverage
- [ ] Integration tests cover main flows
- [ ] Code follows existing patterns

---

## Definition of Done

- [ ] shouldRenewCertificate method implemented
- [ ] triggerRenewal method implemented
- [ ] Renewal check integrated into sync flow
- [ ] Configuration access added to CertManager
- [ ] Renewal state management implemented
- [ ] Duplicate renewal prevention implemented
- [ ] Logging added for renewal operations
- [ ] Unit tests written and passing (>80% coverage)
- [ ] Integration tests written and passing
- [ ] Code reviewed and approved
- [ ] Documentation updated

---

## Related Files

- `internal/agent/device/certmanager/manager.go` - Certificate manager
- `internal/agent/device/certmanager/lifecycle.go` - Lifecycle manager (from STORY-2)
- `internal/agent/device/certmanager/expiration.go` - Expiration monitor (from STORY-1)
- `internal/agent/config/config.go` - Agent configuration
- `internal/agent/device/certmanager/cert_processing_queue.go` - Processing queue

---

## Dependencies

- **EDM-323-EPIC-1-STORY-1**: Certificate Expiration Monitoring (must be completed)
  - Requires `ExpirationMonitor` to be available
  
- **EDM-323-EPIC-1-STORY-2**: Certificate Lifecycle Manager (must be completed)
  - Requires `LifecycleManager` to be available
  
- **EDM-323-EPIC-1-STORY-3**: Configuration Schema (must be completed)
  - Requires renewal configuration to be available

---

## Notes

- **Renewal vs Provisioning**: Renewal uses the same provisioning queue but with different context. The CSR provisioner will handle renewal-specific logic in the next story.

- **State Management**: Certificate state must be carefully managed to prevent duplicate renewals and track renewal progress.

- **Retry Logic**: Failed renewals will be retried by the processing queue. The lifecycle manager tracks failures separately.

- **Configuration**: Renewal threshold comes from configuration. If not set, defaults to 30 days.

- **Performance**: Renewal checks should be fast. They only check expiration and compare with threshold. Actual renewal work is done asynchronously.

- **Idempotency**: Renewal trigger should be idempotent - triggering multiple times should not cause issues. The processing queue and state management prevent duplicates.

- **Error Handling**: Errors during renewal check should not block certificate sync. Log warnings and continue.

---

**Document End**

