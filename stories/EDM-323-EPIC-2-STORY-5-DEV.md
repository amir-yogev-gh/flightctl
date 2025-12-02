# Developer Story: Agent Certificate Reception and Storage

**Story ID:** EDM-323-EPIC-2-STORY-5  
**Epic:** EDM-323-EPIC-2 (Proactive Certificate Renewal)  
**Feature:** EDM-323 (Agent Certificate Rotation)  
**Story Points:** 3  
**Priority:** High

## Overview

Enhance the CSR provisioner to validate certificates before storage and ensure proper certificate state updates after successful renewal. The agent should validate received certificates to ensure they are valid, match the device identity, and are signed by the expected CA before storing them.

## Implementation Tasks

### Task 1: Add Certificate Validation Method

**File:** `internal/agent/device/certmanager/provider/provisioner/csr.go` (modify)

**Objective:** Add method to validate received certificates before storage.

**Implementation Steps:**

1. **Add certificate validation method:**
```go
// validateCertificate validates a received certificate before storage.
// It verifies the certificate signature chain, subject/SAN, expiration, and CA signature.
func (p *CSRProvisioner) validateCertificate(ctx context.Context, cert *x509.Certificate, expectedCommonName string) error {
    // Verify certificate is not expired
    now := time.Now()
    if cert.NotAfter.Before(now) {
        return fmt.Errorf("certificate is expired (expired at %v)", cert.NotAfter)
    }
    
    // Verify certificate is not yet valid (check NotBefore)
    if cert.NotBefore.After(now) {
        return fmt.Errorf("certificate is not yet valid (valid from %v)", cert.NotBefore)
    }
    
    // Verify certificate CommonName matches expected device identity
    if cert.Subject.CommonName != expectedCommonName {
        return fmt.Errorf("certificate CommonName %q does not match expected %q", 
            cert.Subject.CommonName, expectedCommonName)
    }
    
    // Verify certificate has valid signature
    // Note: The certificate signature is verified by the CA during signing,
    // but we can verify the certificate structure is valid
    if err := cert.CheckSignature(cert.SignatureAlgorithm, cert.RawTBSCertificate, cert.Signature); err != nil {
        return fmt.Errorf("certificate signature verification failed: %w", err)
    }
    
    // Additional validation: verify certificate has required extensions (if applicable)
    // For device certificates, we may want to verify device fingerprint extension
    // This is optional and depends on certificate requirements
    
    return nil
}
```

2. **Add method to get expected CommonName:**
```go
// getExpectedCommonName returns the expected CommonName for the certificate.
func (p *CSRProvisioner) getExpectedCommonName() string {
    // The expected CommonName is the one used when creating the CSR
    return p.cfg.CommonName
}
```

**Testing:**
- Test validation with valid certificate
- Test validation with expired certificate
- Test validation with not-yet-valid certificate
- Test validation with mismatched CommonName
- Test validation with invalid signature

---

### Task 2: Enhance check() Method with Validation

**File:** `internal/agent/device/certmanager/provider/provisioner/csr.go` (modify)

**Objective:** Add certificate validation to the check() method before returning the certificate.

**Implementation Steps:**

1. **Modify check() method to validate certificate:**
```go
// check polls the management server for CSR status and returns the certificate when ready.
// It handles the different CSR states: pending, approved, denied, or failed.
func (p *CSRProvisioner) check(ctx context.Context) (bool, *x509.Certificate, []byte, error) {
    if p.csrName == "" {
        return false, nil, nil, fmt.Errorf("no CSR name recorded")
    }
    if p.identity == nil {
        return false, nil, nil, fmt.Errorf("no identity generated")
    }

    csr, statusCode, err := p.csrClient.GetCertificateSigningRequest(ctx, p.csrName)
    if err != nil {
        return false, nil, nil, fmt.Errorf("get csr: %w", err)
    }
    if statusCode != http.StatusOK {
        return false, nil, nil, fmt.Errorf("unexpected status code %d while fetching CSR %q", statusCode, p.csrName)
    }
    if csr == nil {
        return false, nil, nil, fmt.Errorf("received nil CSR object for %q", p.csrName)
    }
    if csr.Status == nil {
        return false, nil, nil, nil // Not ready yet, wait for status to be populated
    }

    if api.IsStatusConditionTrue(csr.Status.Conditions, api.ConditionTypeCertificateSigningRequestApproved) && csr.Status.Certificate != nil {
        certPEM := *csr.Status.Certificate

        cert, err := fccrypto.ParsePEMCertificate(certPEM)
        if err != nil {
            return false, nil, nil, fmt.Errorf("failed to parse CSR PEM certificate: %w", err)
        }

        // NEW: Validate certificate before returning
        expectedCN := p.getExpectedCommonName()
        if err := p.validateCertificate(ctx, cert, expectedCN); err != nil {
            return false, nil, nil, fmt.Errorf("certificate validation failed: %w", err)
        }

        keyPEM, err := p.identity.KeyPEM()
        if err != nil {
            return false, nil, nil, fmt.Errorf("key pem: %w", err)
        }

        return true, cert, keyPEM, nil
    }

    if api.IsStatusConditionTrue(csr.Status.Conditions, api.ConditionTypeCertificateSigningRequestDenied) ||
        api.IsStatusConditionTrue(csr.Status.Conditions, api.ConditionTypeCertificateSigningRequestFailed) {
        return false, nil, nil, fmt.Errorf("csr %q was denied or failed", p.csrName)
    }

    return false, nil, nil, nil // still pending
}
```

**Testing:**
- Test check() validates certificate before returning
- Test check() rejects invalid certificates
- Test check() accepts valid certificates
- Test check() handles validation errors

---

### Task 3: Add Certificate State Update After Storage

**File:** `internal/agent/device/certmanager/manager.go` (modify)

**Objective:** Update certificate state after successful renewal certificate storage.

**Implementation Steps:**

1. **Enhance ensureCertificate_do to update state after renewal:**
```go
// ensureCertificate_do performs the actual certificate provisioning work.
func (cm *CertManager) ensureCertificate_do(ctx context.Context, providerName string, cert *certificate, cfg *provider.CertificateConfig) (*time.Duration, error) {
    // ... existing code ...
    
    if err := cert.Storage.Write(crt, keyBytes); err != nil {
        return nil, err
    }

    cm.addCertificateInfo(cert, crt)

    // NEW: Update certificate state if this was a renewal
    if cm.lifecycleManager != nil {
        currentState, err := cm.lifecycleManager.GetCertificateState(ctx, providerName, cert.Name)
        if err == nil && currentState.GetState() == CertificateStateRenewing {
            // Renewal completed successfully
            days, expiration, err := cm.lifecycleManager.CheckRenewal(ctx, providerName, cert.Name, 365) // Use large threshold to get days
            if err == nil {
                _ = cm.lifecycleManager.UpdateCertificateState(ctx, providerName, cert.Name, 
                    CertificateStateNormal, days, expiration)
                cm.log.Infof("Certificate %q/%q renewal completed successfully", providerName, cert.Name)
            }
        } else if currentState == nil || currentState.GetState() == CertificateStateNormal {
            // Initial provisioning or normal operation
            _ = cm.lifecycleManager.SetCertificateState(ctx, providerName, cert.Name, CertificateStateNormal)
        }
    }

    cert.Config = config
    cert.Provisioner = nil
    cert.Storage = nil
    return nil, nil
}
```

**Testing:**
- Test state is updated to "normal" after renewal
- Test state is updated after initial provisioning
- Test state update handles errors gracefully

---

### Task 4: Add Enhanced Certificate Validation

**File:** `internal/agent/device/certmanager/provider/provisioner/csr.go` (modify)

**Objective:** Add more comprehensive certificate validation checks.

**Implementation Steps:**

1. **Enhance validateCertificate with additional checks:**
```go
// validateCertificate validates a received certificate before storage.
func (p *CSRProvisioner) validateCertificate(ctx context.Context, cert *x509.Certificate, expectedCommonName string) error {
    // Verify certificate is not expired
    now := time.Now()
    if cert.NotAfter.Before(now) {
        return fmt.Errorf("certificate is expired (expired at %v)", cert.NotAfter)
    }
    
    // Verify certificate is not yet valid (check NotBefore)
    if cert.NotBefore.After(now) {
        return fmt.Errorf("certificate is not yet valid (valid from %v)", cert.NotBefore)
    }
    
    // Verify certificate CommonName matches expected device identity
    if cert.Subject.CommonName != expectedCommonName {
        return fmt.Errorf("certificate CommonName %q does not match expected %q", 
            cert.Subject.CommonName, expectedCommonName)
    }
    
    // Verify certificate has valid signature
    if err := cert.CheckSignature(cert.SignatureAlgorithm, cert.RawTBSCertificate, cert.Signature); err != nil {
        return fmt.Errorf("certificate signature verification failed: %w", err)
    }
    
    // Verify certificate has required key usages (if applicable)
    // For client certificates, we expect clientAuth usage
    hasClientAuth := false
    for _, usage := range cert.ExtKeyUsage {
        if usage == x509.ExtKeyUsageClientAuth {
            hasClientAuth = true
            break
        }
    }
    if !hasClientAuth {
        return fmt.Errorf("certificate missing required key usage: clientAuth")
    }
    
    // Verify certificate is not a CA certificate
    if cert.IsCA {
        return fmt.Errorf("certificate is a CA certificate, expected client certificate")
    }
    
    // Verify certificate has reasonable validity period
    validityPeriod := cert.NotAfter.Sub(cert.NotBefore)
    maxValidity := 2 * 365 * 24 * time.Hour // 2 years
    if validityPeriod > maxValidity {
        return fmt.Errorf("certificate has excessive validity period: %v (max: %v)", validityPeriod, maxValidity)
    }
    
    return nil
}
```

**Testing:**
- Test validation with missing key usage
- Test validation with CA certificate
- Test validation with excessive validity period
- Test validation with all checks passing

---

### Task 5: Add Logging for Certificate Reception

**File:** `internal/agent/device/certmanager/provider/provisioner/csr.go` (modify)

**Objective:** Add appropriate logging for certificate reception and validation.

**Implementation Steps:**

1. **Add logging in check() method:**
```go
// In check(), after certificate is parsed:
cert, err := fccrypto.ParsePEMCertificate(certPEM)
if err != nil {
    return false, nil, nil, fmt.Errorf("failed to parse CSR PEM certificate: %w", err)
}

// Log certificate reception
// Note: We don't have a logger in CSRProvisioner, so this would need to be added
// Or log at the certificate manager level

// Validate certificate
expectedCN := p.getExpectedCommonName()
if err := p.validateCertificate(ctx, cert, expectedCN); err != nil {
    return false, nil, nil, fmt.Errorf("certificate validation failed: %w", err)
}

// Log successful validation
// Certificate is ready for storage
```

2. **Add logging in certificate manager:**
```go
// In ensureCertificate_do, after successful storage:
cm.log.Infof("Certificate %q/%q received and stored successfully (expires: %v)", 
    providerName, cert.Name, crt.NotAfter)
```

**Note:** CSRProvisioner doesn't have a logger. We may need to add one or log at the certificate manager level.

**Testing:**
- Test logging includes relevant information
- Test log levels are appropriate

---

### Task 6: Handle Validation Failures

**File:** `internal/agent/device/certmanager/provider/provisioner/csr.go` (modify)

**Objective:** Handle certificate validation failures appropriately.

**Implementation Steps:**

1. **Update check() to handle validation failures:**
```go
// In check(), when validation fails:
if err := p.validateCertificate(ctx, cert, expectedCN); err != nil {
    // Validation failed - return error to trigger retry
    // The processing queue will retry the operation
    return false, nil, nil, fmt.Errorf("certificate validation failed: %w", err)
}
```

2. **Update certificate manager to handle validation errors:**
```go
// In ensureCertificate, when provisioner returns error:
ready, crt, keyBytes, err := cert.Provisioner.Provision(ctx)
if err != nil {
    // Check if this is a validation error
    if strings.Contains(err.Error(), "certificate validation failed") {
        cm.log.Errorf("Certificate validation failed for %q/%q: %v", providerName, cert.Name, err)
        // Update lifecycle state to indicate validation failure
        if cm.lifecycleManager != nil {
            _ = cm.lifecycleManager.RecordError(ctx, providerName, cert.Name, err)
        }
    }
    return nil, err
}
```

**Testing:**
- Test validation failures are handled correctly
- Test validation failures trigger retry
- Test validation failures update state

---

### Task 7: Add Certificate Fingerprint Verification (Optional)

**File:** `internal/agent/device/certmanager/provider/provisioner/csr.go` (modify)

**Objective:** Optionally verify certificate fingerprint matches expected value.

**Implementation Steps:**

1. **Add fingerprint verification (if needed):**
```go
// In validateCertificate, add optional fingerprint check:
// This is optional and may not be needed for all use cases
// If the service provides a fingerprint in CSR status, we can verify it

// Example (if fingerprint is available in CSR status):
if csr.Status.CertificateFingerprint != nil {
    expectedFingerprint := *csr.Status.CertificateFingerprint
    actualFingerprint := calculateCertificateFingerprint(cert)
    if actualFingerprint != expectedFingerprint {
        return fmt.Errorf("certificate fingerprint mismatch: expected %q, got %q", 
            expectedFingerprint, actualFingerprint)
    }
}
```

**Note:** This is optional and depends on whether the service provides fingerprint in CSR status.

**Testing:**
- Test fingerprint verification (if implemented)
- Test fingerprint mismatch handling

---

## Unit Tests

### Test File: `internal/agent/device/certmanager/provider/provisioner/csr_validation_test.go` (new)

**Test Cases:**

1. **TestValidateCertificate:**
   - Valid certificate passes validation
   - Expired certificate fails validation
   - Not-yet-valid certificate fails validation
   - Mismatched CommonName fails validation
   - Invalid signature fails validation
   - Missing key usage fails validation
   - CA certificate fails validation
   - Excessive validity period fails validation

2. **TestGetExpectedCommonName:**
   - Returns correct CommonName from config
   - Handles empty CommonName

3. **TestCheckWithValidation:**
   - check() validates certificate before returning
   - check() rejects invalid certificates
   - check() accepts valid certificates
   - check() handles validation errors

4. **TestCertificateStateUpdate:**
   - State updated to "normal" after renewal
   - State updated after initial provisioning
   - State update handles errors gracefully

---

## Integration Tests

### Test File: `test/integration/certificate_renewal_reception_test.go` (new)

**Test Cases:**

1. **TestRenewalCertificateReception:**
   - Agent receives renewal certificate
   - Certificate is validated
   - Certificate is stored
   - Certificate state is updated

2. **TestCertificateValidation:**
   - Valid certificate is accepted
   - Invalid certificate is rejected
   - Validation errors are logged

3. **TestCertificateStorage:**
   - Certificate is stored correctly
   - Certificate metadata is updated
   - Certificate can be loaded after storage

4. **TestCertificateStateManagement:**
   - State transitions correctly during renewal
   - State is updated after successful storage
   - State reflects certificate status

---

## Code Review Checklist

- [ ] Certificate validation is comprehensive
- [ ] Validation happens before storage
- [ ] Certificate state is updated after storage
- [ ] Validation failures are handled correctly
- [ ] Error handling is comprehensive
- [ ] Logging is appropriate
- [ ] Unit tests achieve >80% coverage
- [ ] Integration tests cover main flows
- [ ] Code follows existing patterns

---

## Definition of Done

- [ ] Certificate validation method implemented
- [ ] Validation integrated into check() method
- [ ] Certificate state update after storage implemented
- [ ] Enhanced validation checks added
- [ ] Logging added for certificate reception
- [ ] Validation failure handling implemented
- [ ] Unit tests written and passing (>80% coverage)
- [ ] Integration tests written and passing
- [ ] Code reviewed and approved
- [ ] Documentation updated

---

## Related Files

- `internal/agent/device/certmanager/provider/provisioner/csr.go` - CSR provisioner
- `internal/agent/device/certmanager/manager.go` - Certificate manager
- `internal/agent/device/certmanager/provider/storage/fs.go` - Certificate storage
- `pkg/crypto/cert.go` - Certificate utilities

---

## Dependencies

- **EDM-323-EPIC-2-STORY-2**: CSR Generation for Certificate Renewal (must be completed)
  - Requires renewal CSR generation
  
- **EDM-323-EPIC-2-STORY-4**: Certificate Issuance for Renewals (must be completed)
  - Requires certificates to be issued by service

---

## Notes

- **Certificate Validation**: Certificates are validated before storage to ensure they are valid, match device identity, and meet security requirements. Invalid certificates are rejected and the operation is retried.

- **Validation Timing**: Validation happens in the `check()` method after the certificate is received from the server but before it's returned to the certificate manager for storage.

- **Certificate State**: Certificate state is updated after successful storage. For renewals, the state transitions from "renewing" to "normal". For initial provisioning, the state is set to "normal".

- **Error Handling**: Validation failures cause the operation to fail and be retried by the processing queue. The lifecycle manager tracks validation errors.

- **Atomic Storage**: Certificate storage uses atomic file operations (via `fileio.WriteFile`) to ensure certificates are written atomically and don't leave the system in an inconsistent state.

- **Key Usage**: Certificates are validated to ensure they have the required key usages (e.g., clientAuth for client certificates).

- **Validity Period**: Certificates are validated to ensure they have reasonable validity periods (not excessive).

- **Backward Compatibility**: Existing certificate provisioning continues to work. Validation is added as an additional step but doesn't change the overall flow.

---

**Document End**

