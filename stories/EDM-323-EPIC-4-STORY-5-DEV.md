# Developer Story: Complete Recovery Flow Implementation

**Story ID:** EDM-323-EPIC-4-STORY-5  
**Epic:** EDM-323-EPIC-4 (Expired Certificate Recovery)  
**Feature:** EDM-323 (Agent Certificate Rotation)  
**Story Points:** 5  
**Priority:** High

## Overview

Implement the complete recovery flow from expired certificate detection to new certificate installation. The flow should handle bootstrap certificate fallback, TPM attestation, CSR generation, certificate reception, and atomic swap.

## Implementation Tasks

### Task 1: Implement Complete Recovery Flow Method

**File:** `internal/agent/device/certmanager/lifecycle.go` (modify)

**Objective:** Implement the complete recovery flow orchestration.

**Implementation Steps:**

1. **Implement RecoverExpiredCertificate method:**
```go
// RecoverExpiredCertificate implements the complete recovery flow for expired certificates.
func (lm *LifecycleManager) RecoverExpiredCertificate(ctx context.Context, providerName string, certName string) error {
    lm.log.Infof("Starting recovery flow for expired certificate %q/%q", providerName, certName)

    // Step 1: Update state to recovering
    if err := lm.SetCertificateState(ctx, providerName, certName, CertificateStateRecovering); err != nil {
        lm.log.Warnf("Failed to set recovery state: %v", err)
        // Continue anyway
    }

    // Step 2: Determine authentication method
    authMethod, err := lm.determineRecoveryAuthMethod(ctx)
    if err != nil {
        return fmt.Errorf("failed to determine recovery authentication method: %w", err)
    }

    lm.log.Infof("Using authentication method: %s for recovery", authMethod)

    // Step 3: Generate renewal CSR with appropriate authentication
    csr, attestation, err := lm.generateRecoveryCSR(ctx, providerName, certName, authMethod)
    if err != nil {
        lm.RecordError(ctx, providerName, certName, err)
        return fmt.Errorf("failed to generate recovery CSR: %w", err)
    }

    // Step 4: Submit CSR to service
    csrName, err := lm.submitRecoveryCSR(ctx, csr, attestation)
    if err != nil {
        lm.RecordError(ctx, providerName, certName, err)
        return fmt.Errorf("failed to submit recovery CSR: %w", err)
    }

    lm.log.Infof("Recovery CSR submitted: %s", csrName)

    // Step 5: Poll for certificate approval and reception
    newCert, newKey, err := lm.pollForRecoveryCertificate(ctx, csrName)
    if err != nil {
        lm.RecordError(ctx, providerName, certName, err)
        return fmt.Errorf("failed to receive recovery certificate: %w", err)
    }

    // Step 6: Validate and atomically swap certificate
    if err := lm.installRecoveryCertificate(ctx, providerName, certName, newCert, newKey); err != nil {
        lm.RecordError(ctx, providerName, certName, err)
        return fmt.Errorf("failed to install recovery certificate: %w", err)
    }

    // Step 7: Update state to normal
    days, expiration, err := lm.CheckRenewal(ctx, providerName, certName, 365)
    if err == nil {
        _ = lm.UpdateCertificateState(ctx, providerName, certName, CertificateStateNormal, days, expiration)
    }

    lm.log.Infof("Recovery completed successfully for certificate %q/%q", providerName, certName)
    return nil
}
```

**Testing:**
- Test RecoverExpiredCertificate completes full flow
- Test RecoverExpiredCertificate handles errors
- Test RecoverExpiredCertificate updates state correctly

---

### Task 2: Implement Authentication Method Determination

**File:** `internal/agent/device/certmanager/lifecycle.go` (modify)

**Objective:** Determine which authentication method to use for recovery.

**Implementation Steps:**

1. **Add determineRecoveryAuthMethod method:**
```go
// RecoveryAuthMethod represents the authentication method for recovery.
type RecoveryAuthMethod string

const (
    RecoveryAuthMethodBootstrap RecoveryAuthMethod = "bootstrap"
    RecoveryAuthMethodTPM       RecoveryAuthMethod = "tpm"
)

// determineRecoveryAuthMethod determines which authentication method to use for recovery.
func (lm *LifecycleManager) determineRecoveryAuthMethod(ctx context.Context) (RecoveryAuthMethod, error) {
    // Step 1: Try bootstrap certificate first
    if lm.bootstrapHandler != nil {
        hasBootstrap, err := lm.bootstrapHandler.HasValidBootstrapCertificate(ctx)
        if err == nil && hasBootstrap {
            lm.log.Debug("Bootstrap certificate available, using for recovery")
            return RecoveryAuthMethodBootstrap, nil
        }
        if err != nil {
            lm.log.Warnf("Failed to check bootstrap certificate: %v", err)
        }
    }

    // Step 2: Fall back to TPM attestation
    if lm.tpmClient != nil {
        lm.log.Debug("Bootstrap certificate not available, using TPM attestation for recovery")
        return RecoveryAuthMethodTPM, nil
    }

    return "", fmt.Errorf("no authentication method available for recovery (bootstrap expired and no TPM)")
}
```

**Testing:**
- Test determineRecoveryAuthMethod prefers bootstrap
- Test determineRecoveryAuthMethod falls back to TPM
- Test determineRecoveryAuthMethod handles errors

---

### Task 3: Implement Recovery CSR Generation

**File:** `internal/agent/device/certmanager/lifecycle.go` (modify)

**Objective:** Generate CSR with appropriate authentication for recovery.

**Implementation Steps:**

1. **Add generateRecoveryCSR method:**
```go
// generateRecoveryCSR generates a renewal CSR for recovery with appropriate authentication.
func (lm *LifecycleManager) generateRecoveryCSR(ctx context.Context, providerName string, certName string, authMethod RecoveryAuthMethod) (*api.CertificateSigningRequest, *RenewalAttestation, error) {
    // Get certificate config
    cert, err := lm.certManager.certificates.ReadCertificate(providerName, certName)
    if err != nil {
        return nil, nil, fmt.Errorf("failed to read certificate: %w", err)
    }

    cert.mu.RLock()
    cfg := cert.Config
    cert.mu.RUnlock()

    // Generate CSR using provisioner
    // The provisioner will handle CSR generation
    // We need to add renewal context with "expired" reason
    
    var attestation *RenewalAttestation
    
    // If using TPM attestation, generate it
    if authMethod == RecoveryAuthMethodTPM {
        if lm.tpmRenewalProvider != nil {
            att, err := lm.tpmRenewalProvider.GenerateRenewalAttestation(ctx)
            if err != nil {
                return nil, nil, fmt.Errorf("failed to generate TPM attestation: %w", err)
            }
            attestation = att
        }
    }

    // Create renewal context
    renewalContext := &provisioner.RenewalContext{
        Reason:              "expired",
        DaysUntilExpiration: -1, // Expired
    }

    // Generate CSR (this will be done by the provisioner)
    // For now, return placeholder
    // Full implementation will use the CSR provisioner
    
    return nil, attestation, fmt.Errorf("CSR generation not yet fully implemented")
}
```

**Note:** Full CSR generation will use the existing CSR provisioner with renewal context.

**Testing:**
- Test generateRecoveryCSR with bootstrap auth
- Test generateRecoveryCSR with TPM auth
- Test generateRecoveryCSR includes attestation

---

### Task 4: Implement CSR Submission with Authentication

**File:** `internal/agent/device/certmanager/lifecycle.go` (modify)

**Objective:** Submit recovery CSR using appropriate authentication.

**Implementation Steps:**

1. **Add submitRecoveryCSR method:**
```go
// submitRecoveryCSR submits a recovery CSR to the service.
// It uses the appropriate authentication method (bootstrap cert or TPM).
func (lm *LifecycleManager) submitRecoveryCSR(ctx context.Context, csr *api.CertificateSigningRequest, attestation *RenewalAttestation) (string, error) {
    // Create management client with appropriate authentication
    var managementClient client.Management
    var err error

    // Determine which client to use based on authentication method
    if attestation == nil {
        // Using bootstrap certificate - create client with bootstrap cert
        if lm.bootstrapHandler != nil {
            tlsCert, err := lm.bootstrapHandler.GetCertificateForAuth(ctx, 
                lm.managementCertPath, lm.managementKeyPath)
            if err != nil {
                return "", fmt.Errorf("failed to get bootstrap certificate: %w", err)
            }
            
            // Create client with bootstrap certificate
            // This requires modifying client creation to accept TLS certificate
            // For now, placeholder
            managementClient = lm.managementClient // Use existing client if it supports cert switching
        }
    } else {
        // Using TPM attestation - may need different client setup
        // TPM attestation is included in CSR, not used for mTLS
        managementClient = lm.managementClient
    }

    // Submit CSR
    if managementClient != nil {
        submittedCSR, statusCode, err := managementClient.CreateCertificateSigningRequest(ctx, *csr)
        if err != nil {
            return "", fmt.Errorf("failed to submit recovery CSR: %w", err)
        }
        if statusCode != 200 && statusCode != 201 {
            return "", fmt.Errorf("unexpected status code %d when submitting recovery CSR", statusCode)
        }
        
        csrName := lo.FromPtr(submittedCSR.Metadata.Name)
        return csrName, nil
    }

    return "", fmt.Errorf("management client not available")
}
```

**Testing:**
- Test submitRecoveryCSR with bootstrap auth
- Test submitRecoveryCSR with TPM auth
- Test submitRecoveryCSR handles errors

---

### Task 5: Implement Certificate Polling

**File:** `internal/agent/device/certmanager/lifecycle.go` (modify)

**Objective:** Poll for recovery certificate approval and reception.

**Implementation Steps:**

1. **Add pollForRecoveryCertificate method:**
```go
// pollForRecoveryCertificate polls for recovery certificate approval and reception.
func (lm *LifecycleManager) pollForRecoveryCertificate(ctx context.Context, csrName string) (*x509.Certificate, []byte, error) {
    lm.log.Debugf("Polling for recovery certificate: %s", csrName)

    // Poll with exponential backoff
    maxAttempts := 30
    baseDelay := 10 * time.Second
    maxDelay := 5 * time.Minute
    
    for attempt := 0; attempt < maxAttempts; attempt++ {
        // Check CSR status
        csr, statusCode, err := lm.managementClient.GetCertificateSigningRequest(ctx, csrName)
        if err != nil {
            lm.log.Warnf("Failed to get CSR status (attempt %d/%d): %v", attempt+1, maxAttempts, err)
            // Continue polling
        } else if statusCode == 200 && csr != nil {
            // Check if approved and certificate is available
            if api.IsStatusConditionTrue(csr.Status.Conditions, api.ConditionTypeCertificateSigningRequestApproved) && 
               csr.Status.Certificate != nil {
                // Certificate is ready
                certPEM := *csr.Status.Certificate
                cert, err := fccrypto.ParsePEMCertificate(certPEM)
                if err != nil {
                    return nil, nil, fmt.Errorf("failed to parse certificate: %w", err)
                }

                // Get key from identity provider
                // For recovery, we need to get the key that matches the CSR
                // This depends on how the CSR was generated
                keyPEM, err := lm.getRecoveryKey(ctx)
                if err != nil {
                    return nil, nil, fmt.Errorf("failed to get recovery key: %w", err)
                }

                return cert, keyPEM, nil
            }

            // Check if denied or failed
            if api.IsStatusConditionTrue(csr.Status.Conditions, api.ConditionTypeCertificateSigningRequestDenied) ||
               api.IsStatusConditionTrue(csr.Status.Conditions, api.ConditionTypeCertificateSigningRequestFailed) {
                return nil, nil, fmt.Errorf("recovery CSR was denied or failed")
            }
        }

        // Wait before next poll
        delay := baseDelay * time.Duration(1<<uint(attempt))
        if delay > maxDelay {
            delay = maxDelay
        }
        
        select {
        case <-ctx.Done():
            return nil, nil, ctx.Err()
        case <-time.After(delay):
            // Continue polling
        }
    }

    return nil, nil, fmt.Errorf("timeout waiting for recovery certificate after %d attempts", maxAttempts)
}
```

**Testing:**
- Test pollForRecoveryCertificate polls correctly
- Test pollForRecoveryCertificate handles approval
- Test pollForRecoveryCertificate handles denial

---

### Task 6: Implement Certificate Installation

**File:** `internal/agent/device/certmanager/lifecycle.go` (modify)

**Objective:** Install recovery certificate using atomic swap.

**Implementation Steps:**

1. **Add installRecoveryCertificate method:**
```go
// installRecoveryCertificate installs the recovery certificate using atomic swap.
func (lm *LifecycleManager) installRecoveryCertificate(ctx context.Context, providerName string, certName string, cert *x509.Certificate, keyPEM []byte) error {
    lm.log.Infof("Installing recovery certificate for %q/%q", providerName, certName)

    // Get certificate storage
    certObj, err := lm.certManager.certificates.ReadCertificate(providerName, certName)
    if err != nil {
        return fmt.Errorf("failed to read certificate: %w", err)
    }

    certObj.mu.Lock()
    defer certObj.mu.Unlock()

    // Step 1: Write to pending location
    if err := certObj.Storage.WritePending(cert, keyPEM); err != nil {
        return fmt.Errorf("failed to write pending certificate: %w", err)
    }

    // Step 2: Validate pending certificate
    caBundlePath := lm.certManager.getCABundlePath(certObj.Config)
    validator := NewCertificateValidator(caBundlePath, certName, lm.log)
    
    pendingCert, err := certObj.Storage.LoadPendingCertificate(ctx)
    if err != nil {
        _ = certObj.Storage.CleanupPending(ctx)
        return fmt.Errorf("failed to load pending certificate: %w", err)
    }
    
    pendingKey, err := certObj.Storage.LoadPendingKey(ctx)
    if err != nil {
        _ = certObj.Storage.CleanupPending(ctx)
        return fmt.Errorf("failed to load pending key: %w", err)
    }
    
    if err := validator.ValidatePendingCertificate(ctx, pendingCert, pendingKey, lm.certManager.readWriter); err != nil {
        _ = certObj.Storage.CleanupPending(ctx)
        return fmt.Errorf("pending certificate validation failed: %w", err)
    }

    // Step 3: Atomically swap certificate
    if err := certObj.Storage.AtomicSwap(ctx); err != nil {
        _ = certObj.Storage.CleanupPending(ctx)
        return fmt.Errorf("failed to atomically swap certificate: %w", err)
    }

    // Step 4: Update certificate info
    lm.certManager.addCertificateInfo(certObj, cert)

    lm.log.Infof("Recovery certificate installed successfully for %q/%q", providerName, certName)
    return nil
}
```

**Testing:**
- Test installRecoveryCertificate writes to pending
- Test installRecoveryCertificate validates certificate
- Test installRecoveryCertificate performs atomic swap

---

### Task 7: Integrate Recovery into TriggerRecovery

**File:** `internal/agent/device/certmanager/lifecycle.go` (modify)

**Objective:** Complete the TriggerRecovery implementation.

**Implementation Steps:**

1. **Complete TriggerRecovery method:**
```go
// TriggerRecovery triggers recovery for an expired certificate.
func (lm *LifecycleManager) TriggerRecovery(ctx context.Context, providerName string, certName string) error {
    lm.log.Infof("Triggering recovery for expired certificate %q/%q", providerName, certName)

    // Update state to recovering
    if err := lm.SetCertificateState(ctx, providerName, certName, CertificateStateRecovering); err != nil {
        lm.log.Warnf("Failed to set recovery state: %v", err)
        // Continue anyway
    }

    // Execute recovery flow with retry logic
    maxRetries := 3
    baseDelay := 1 * time.Minute
    
    for attempt := 0; attempt < maxRetries; attempt++ {
        if attempt > 0 {
            lm.log.Infof("Retrying recovery (attempt %d/%d)", attempt+1, maxRetries)
            delay := baseDelay * time.Duration(1<<uint(attempt-1))
            select {
            case <-ctx.Done():
                return ctx.Err()
            case <-time.After(delay):
                // Continue retry
            }
        }

        err := lm.RecoverExpiredCertificate(ctx, providerName, certName)
        if err == nil {
            // Recovery successful
            return nil
        }

        lm.log.Warnf("Recovery attempt %d/%d failed: %v", attempt+1, maxRetries, err)
        
        // If this is the last attempt, return error
        if attempt == maxRetries-1 {
            lm.RecordError(ctx, providerName, certName, err)
            return fmt.Errorf("recovery failed after %d attempts: %w", maxRetries, err)
        }
    }

    return nil
}
```

**Testing:**
- Test TriggerRecovery executes recovery flow
- Test TriggerRecovery retries on failure
- Test TriggerRecovery updates state correctly

---

### Task 8: Add Error Handling and Logging

**File:** `internal/agent/device/certmanager/lifecycle.go` (modify)

**Objective:** Add comprehensive error handling and logging.

**Implementation Steps:**

1. **Add logging throughout recovery flow:**
```go
// In RecoverExpiredCertificate, add detailed logging:
lm.log.Infof("Starting recovery flow for expired certificate %q/%q", providerName, certName)
lm.log.Debugf("Using authentication method: %s", authMethod)
lm.log.Infof("Recovery CSR submitted: %s", csrName)
lm.log.Debugf("Polling for recovery certificate: %s", csrName)
lm.log.Infof("Recovery certificate received")
lm.log.Infof("Installing recovery certificate")
lm.log.Infof("Recovery completed successfully")
```

2. **Add error handling:**
```go
// In each step, add error handling:
if err != nil {
    lm.log.Errorf("Recovery step failed: %v", err)
    lm.RecordError(ctx, providerName, certName, err)
    return err
}
```

**Testing:**
- Test logging includes relevant information
- Test error handling is comprehensive

---

## Unit Tests

### Test File: `internal/agent/device/certmanager/lifecycle_recovery_test.go` (new)

**Test Cases:**

1. **TestRecoverExpiredCertificate:**
   - Completes full recovery flow
   - Handles errors at each step
   - Updates state correctly

2. **TestDetermineRecoveryAuthMethod:**
   - Prefers bootstrap certificate
   - Falls back to TPM
   - Handles missing methods

3. **TestGenerateRecoveryCSR:**
   - Generates CSR with bootstrap auth
   - Generates CSR with TPM auth
   - Includes attestation

4. **TestSubmitRecoveryCSR:**
   - Submits CSR successfully
   - Handles submission errors
   - Uses correct authentication

5. **TestPollForRecoveryCertificate:**
   - Polls correctly
   - Handles approval
   - Handles timeout

6. **TestInstallRecoveryCertificate:**
   - Writes to pending
   - Validates certificate
   - Performs atomic swap

---

## Integration Tests

### Test File: `test/integration/certificate_recovery_flow_test.go` (new)

**Test Cases:**

1. **TestCompleteRecoveryFlow:**
   - Complete flow from expired to new certificate
   - Uses bootstrap certificate
   - Installs new certificate

2. **TestRecoveryFlowWithTPM:**
   - Complete flow with TPM attestation
   - TPM attestation is validated
   - New certificate is installed

3. **TestRecoveryFlowRetry:**
   - Recovery retries on failure
   - Eventually succeeds
   - State is updated correctly

4. **TestRecoveryFlowFailure:**
   - Recovery handles failures gracefully
   - State reflects failure
   - Can be retried later

---

## Code Review Checklist

- [ ] Complete recovery flow is implemented
- [ ] Authentication method determination works
- [ ] CSR generation works
- [ ] CSR submission works
- [ ] Certificate polling works
- [ ] Certificate installation works
- [ ] Error handling is comprehensive
- [ ] Retry logic is implemented
- [ ] Logging is appropriate
- [ ] Unit tests achieve >80% coverage
- [ ] Integration tests cover main flows
- [ ] E2E tests cover complete scenarios
- [ ] Code follows existing patterns

---

## Definition of Done

- [ ] RecoverExpiredCertificate method implemented
- [ ] determineRecoveryAuthMethod implemented
- [ ] generateRecoveryCSR implemented
- [ ] submitRecoveryCSR implemented
- [ ] pollForRecoveryCertificate implemented
- [ ] installRecoveryCertificate implemented
- [ ] TriggerRecovery completed
- [ ] Error handling and logging added
- [ ] Unit tests written and passing (>80% coverage)
- [ ] Integration tests written and passing
- [ ] E2E tests written and passing
- [ ] Code reviewed and approved
- [ ] Documentation updated

---

## Related Files

- `internal/agent/device/certmanager/lifecycle.go` - Lifecycle manager
- `internal/agent/device/certmanager/manager.go` - Certificate manager
- `internal/agent/device/bootstrap_cert.go` - Bootstrap certificate handler
- `internal/agent/identity/tpm_renewal.go` - TPM renewal attestation
- `internal/agent/device/certmanager/swap.go` - Certificate swap

---

## Dependencies

- **EDM-323-EPIC-4-STORY-1**: Expired Certificate Detection (must be completed)
- **EDM-323-EPIC-4-STORY-2**: Bootstrap Certificate Fallback (must be completed)
- **EDM-323-EPIC-4-STORY-3**: TPM Attestation Generation (must be completed)
- **EDM-323-EPIC-4-STORY-4**: Service-Side Recovery Validation (must be completed)
- **EDM-323-EPIC-3**: Atomic Certificate Swap (must be completed)

---

## Notes

- **Recovery Flow**: The complete recovery flow orchestrates all components: detection, authentication, CSR generation, submission, polling, and installation.

- **Authentication Methods**: Recovery supports two authentication methods: bootstrap certificate (preferred) and TPM attestation (fallback). The method is determined automatically.

- **Retry Logic**: Recovery includes retry logic with exponential backoff to handle transient failures. The number of retries and delays are configurable.

- **State Management**: Certificate state is updated throughout the recovery flow: recovering → normal (on success) or renewal_failed (on failure).

- **Atomic Installation**: Recovery certificates are installed using atomic swap to ensure zero-downtime and data integrity.

- **Error Handling**: Each step of the recovery flow has comprehensive error handling. Errors are logged and state is updated appropriately.

- **Integration**: The recovery flow integrates with all previous stories: expiration detection, bootstrap fallback, TPM attestation, service validation, and atomic swap.

---

**Document End**

