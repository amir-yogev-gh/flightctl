# Developer Story: Service-Side Renewal Request Validation

**Story ID:** EDM-323-EPIC-2-STORY-3  
**Epic:** EDM-323-EPIC-2 (Proactive Certificate Renewal)  
**Feature:** EDM-323 (Agent Certificate Rotation)  
**Story Points:** 5  
**Priority:** High

## Overview

Implement validation and auto-approval logic for certificate renewal CSR requests. The service should validate that renewal requests come from valid, previously enrolled devices with valid certificates, and automatically approve them without manual intervention.

## Implementation Tasks

### Task 1: Add Renewal Request Detection

**File:** `internal/service/certificatesigningrequest.go` (modify)

**Objective:** Add method to detect if a CSR is a renewal request based on labels.

**Implementation Steps:**

1. **Add isRenewalRequest method:**
```go
// isRenewalRequest checks if a CSR is a renewal request based on metadata labels.
func (h *ServiceHandler) isRenewalRequest(csr *api.CertificateSigningRequest) bool {
    if csr.Metadata == nil || csr.Metadata.Labels == nil {
        return false
    }
    
    labels := *csr.Metadata.Labels
    renewalReason, hasRenewalLabel := labels["flightctl.io/renewal-reason"]
    
    return hasRenewalLabel && (renewalReason == "proactive" || renewalReason == "expired")
}
```

2. **Add getRenewalContext method:**
```go
// getRenewalContext extracts renewal context from CSR labels.
func (h *ServiceHandler) getRenewalContext(csr *api.CertificateSigningRequest) (reason string, thresholdDays int, daysUntilExpiration int) {
    if !h.isRenewalRequest(csr) {
        return "", 0, 0
    }
    
    labels := *csr.Metadata.Labels
    reason = labels["flightctl.io/renewal-reason"]
    
    if thresholdStr, ok := labels["flightctl.io/renewal-threshold-days"]; ok {
        if threshold, err := strconv.Atoi(thresholdStr); err == nil {
            thresholdDays = threshold
        }
    }
    
    if daysStr, ok := labels["flightctl.io/renewal-days-until-expiration"]; ok {
        if days, err := strconv.Atoi(daysStr); err == nil {
            daysUntilExpiration = days
        }
    }
    
    return reason, thresholdDays, daysUntilExpiration
}
```

**Testing:**
- Test isRenewalRequest detects renewal requests
- Test isRenewalRequest returns false for non-renewal requests
- Test getRenewalContext extracts context correctly
- Test getRenewalContext handles missing labels

---

### Task 2: Extract Device Name from CSR

**File:** `internal/service/certificatesigningrequest.go` (modify)

**Objective:** Extract device name/identity from CSR CommonName.

**Implementation Steps:**

1. **Add extractDeviceNameFromCSR method:**
```go
// extractDeviceNameFromCSR extracts the device name from CSR CommonName.
func (h *ServiceHandler) extractDeviceNameFromCSR(csr *api.CertificateSigningRequest) (string, error) {
    request, _, err := newSignRequestFromCertificateSigningRequest(csr)
    if err != nil {
        return "", fmt.Errorf("failed to parse CSR: %w", err)
    }
    
    x509CSR := request.X509()
    commonName := x509CSR.Subject.CommonName
    
    if commonName == "" {
        return "", fmt.Errorf("CSR CommonName is empty")
    }
    
    // For bootstrap signer, CommonName is the device name
    if csr.Spec.SignerName == h.ca.Cfg.ClientBootstrapSignerName {
        // Extract device fingerprint from CN
        fingerprint, err := signer.DeviceFingerprintFromCN(h.ca.Config(), commonName)
        if err != nil {
            // If extraction fails, try using CN directly
            return commonName, nil
        }
        return fingerprint, nil
    }
    
    // For other signers, use CommonName directly
    return commonName, nil
}
```

**Testing:**
- Test extractDeviceNameFromCSR extracts name correctly
- Test extractDeviceNameFromCSR handles bootstrap signer
- Test extractDeviceNameFromCSR handles other signers
- Test extractDeviceNameFromCSR handles errors

---

### Task 3: Validate Renewal Request

**File:** `internal/service/certificatesigningrequest.go` (modify)

**Objective:** Implement comprehensive validation for renewal requests.

**Implementation Steps:**

1. **Add validateRenewalRequest method:**
```go
// validateRenewalRequest validates a renewal CSR request.
// It verifies the device exists, was previously enrolled, and has a valid certificate.
func (h *ServiceHandler) validateRenewalRequest(ctx context.Context, orgId uuid.UUID, csr *api.CertificateSigningRequest) error {
    // Extract device name from CSR
    deviceName, err := h.extractDeviceNameFromCSR(csr)
    if err != nil {
        return fmt.Errorf("failed to extract device name from CSR: %w", err)
    }
    
    // Verify device exists in database
    device, err := h.store.Device().Get(ctx, orgId, deviceName)
    if err != nil {
        if errors.Is(err, flterrors.ErrResourceNotFound) {
            return fmt.Errorf("device %q not found - renewal requires existing device", deviceName)
        }
        return fmt.Errorf("failed to get device %q: %w", deviceName, err)
    }
    
    // Verify device was previously enrolled (has status indicating enrollment)
    if device.Status == nil {
        return fmt.Errorf("device %q has no status - device may not be enrolled", deviceName)
    }
    
    // Verify current certificate is valid (from mTLS peer certificate)
    peerCert, err := signer.PeerCertificateFromCtx(ctx)
    if err != nil {
        return fmt.Errorf("failed to get peer certificate from context: %w", err)
    }
    
    // Verify certificate is not expired
    now := time.Now()
    if peerCert.NotAfter.Before(now) {
        return fmt.Errorf("peer certificate is expired (expired at %v)", peerCert.NotAfter)
    }
    
    // Verify certificate matches device identity
    // Extract device fingerprint from peer certificate CN
    peerFingerprint, err := signer.DeviceFingerprintFromCN(h.ca.Config(), peerCert.Subject.CommonName)
    if err != nil {
        // If extraction fails, use CN directly
        peerFingerprint = peerCert.Subject.CommonName
    }
    
    // Verify peer certificate fingerprint matches device name
    if peerFingerprint != deviceName {
        return fmt.Errorf("peer certificate fingerprint %q does not match device name %q", peerFingerprint, deviceName)
    }
    
    // Verify CSR CommonName matches device identity
    csrFingerprint, err := h.extractDeviceNameFromCSR(csr)
    if err != nil {
        return fmt.Errorf("failed to extract device name from CSR: %w", err)
    }
    
    if csrFingerprint != deviceName {
        return fmt.Errorf("CSR CommonName %q does not match device name %q", csrFingerprint, deviceName)
    }
    
    return nil
}
```

**Testing:**
- Test validateRenewalRequest with valid renewal
- Test validateRenewalRequest with non-existent device
- Test validateRenewalRequest with expired certificate
- Test validateRenewalRequest with mismatched identity
- Test validateRenewalRequest with unenrolled device

---

### Task 4: Add Auto-Approval for Renewal Requests

**File:** `internal/service/certificatesigningrequest.go` (modify)

**Objective:** Auto-approve validated renewal requests.

**Implementation Steps:**

1. **Add autoApproveRenewal method:**
```go
// autoApproveRenewal auto-approves a validated renewal CSR request.
func (h *ServiceHandler) autoApproveRenewal(ctx context.Context, orgId uuid.UUID, csr *api.CertificateSigningRequest) {
    if api.IsStatusConditionTrue(csr.Status.Conditions, api.ConditionTypeCertificateSigningRequestApproved) || 
       api.IsStatusConditionTrue(csr.Status.Conditions, api.ConditionTypeCertificateSigningRequestDenied) {
        return
    }
    
    reason, thresholdDays, daysUntilExpiration := h.getRenewalContext(csr)
    message := fmt.Sprintf("Auto-approved renewal request (reason: %s", reason)
    if thresholdDays > 0 {
        message += fmt.Sprintf(", threshold: %d days", thresholdDays)
    }
    if daysUntilExpiration != 0 {
        message += fmt.Sprintf(", days until expiration: %d", daysUntilExpiration)
    }
    message += ")"
    
    api.SetStatusCondition(&csr.Status.Conditions, api.Condition{
        Type:    api.ConditionTypeCertificateSigningRequestApproved,
        Status:  api.ConditionStatusTrue,
        Reason:  "RenewalAutoApproved",
        Message: message,
    })
    api.RemoveStatusCondition(&csr.Status.Conditions, api.ConditionTypeCertificateSigningRequestFailed)
    
    if _, err := h.store.CertificateSigningRequest().UpdateStatus(ctx, orgId, csr); err != nil {
        h.log.WithError(err).Error("failed to set renewal approval condition")
    }
}
```

2. **Update CreateCertificateSigningRequest to handle renewals:**
```go
func (h *ServiceHandler) CreateCertificateSigningRequest(ctx context.Context, orgId uuid.UUID, csr api.CertificateSigningRequest) (*api.CertificateSigningRequest, api.Status) {
    // ... existing validation code ...
    
    result, err := h.store.CertificateSigningRequest().Create(ctx, orgId, &csr, h.callbackCertificateSigningRequestUpdated)
    if err != nil {
        return nil, StoreErrorToApiStatus(err, true, api.CertificateSigningRequestKind, csr.Metadata.Name)
    }

    // Check if this is a renewal request
    if h.isRenewalRequest(result) {
        // Validate renewal request
        if err := h.validateRenewalRequest(ctx, orgId, result); err != nil {
            h.setCSRFailedCondition(ctx, orgId, result, "RenewalValidationFailed", fmt.Sprintf("Renewal validation failed: %v", err))
            return result, api.StatusBadRequest(fmt.Sprintf("renewal validation failed: %v", err))
        }
        
        // Auto-approve renewal request
        h.autoApproveRenewal(ctx, orgId, result)
    } else if result.Spec.SignerName == h.ca.Cfg.ClientBootstrapSignerName {
        // Existing auto-approval for bootstrap signer (non-renewal)
        h.autoApprove(ctx, orgId, result)
    }

    if api.IsStatusConditionTrue(result.Status.Conditions, api.ConditionTypeCertificateSigningRequestApproved) {
        h.signApprovedCertificateSigningRequest(ctx, orgId, result)
    }

    return result, api.StatusCreated()
}
```

3. **Update ReplaceCertificateSigningRequest similarly:**
```go
func (h *ServiceHandler) ReplaceCertificateSigningRequest(ctx context.Context, orgId uuid.UUID, name string, csr api.CertificateSigningRequest) (*api.CertificateSigningRequest, api.Status) {
    // ... existing validation code ...
    
    result, created, err := h.store.CertificateSigningRequest().CreateOrUpdate(ctx, orgId, &csr, h.callbackCertificateSigningRequestUpdated)
    if err != nil {
        return nil, StoreErrorToApiStatus(err, created, api.CertificateSigningRequestKind, &name)
    }

    // Check if this is a renewal request
    if h.isRenewalRequest(result) {
        // Validate renewal request
        if err := h.validateRenewalRequest(ctx, orgId, result); err != nil {
            h.setCSRFailedCondition(ctx, orgId, result, "RenewalValidationFailed", fmt.Sprintf("Renewal validation failed: %v", err))
            return result, api.StatusBadRequest(fmt.Sprintf("renewal validation failed: %v", err))
        }
        
        // Auto-approve renewal request
        h.autoApproveRenewal(ctx, orgId, result)
    } else if result.Spec.SignerName == h.ca.Cfg.ClientBootstrapSignerName {
        // Existing auto-approval for bootstrap signer (non-renewal)
        h.autoApprove(ctx, orgId, result)
    }
    
    if api.IsStatusConditionTrue(result.Status.Conditions, api.ConditionTypeCertificateSigningRequestApproved) {
        h.signApprovedCertificateSigningRequest(ctx, orgId, result)
    }

    return result, StoreErrorToApiStatus(nil, created, api.CertificateSigningRequestKind, &name)
}
```

**Testing:**
- Test autoApproveRenewal sets approval condition
- Test autoApproveRenewal includes renewal context in message
- Test auto-approval happens after validation
- Test validation failure prevents auto-approval

---

### Task 5: Add Error Handling and Logging

**File:** `internal/service/certificatesigningrequest.go` (modify)

**Objective:** Add appropriate error handling and logging for renewal validation.

**Implementation Steps:**

1. **Add logging in validation:**
```go
// In validateRenewalRequest, add logging:
func (h *ServiceHandler) validateRenewalRequest(ctx context.Context, orgId uuid.UUID, csr *api.CertificateSigningRequest) error {
    deviceName, err := h.extractDeviceNameFromCSR(csr)
    if err != nil {
        h.log.WithError(err).Warn("Failed to extract device name from renewal CSR")
        return fmt.Errorf("failed to extract device name from CSR: %w", err)
    }
    
    h.log.Debugf("Validating renewal request for device %q", deviceName)
    
    // ... rest of validation ...
    
    h.log.Debugf("Renewal request validation successful for device %q", deviceName)
    return nil
}
```

2. **Add logging in auto-approval:**
```go
// In autoApproveRenewal, add logging:
func (h *ServiceHandler) autoApproveRenewal(ctx context.Context, orgId uuid.UUID, csr *api.CertificateSigningRequest) {
    // ... existing code ...
    
    deviceName, _ := h.extractDeviceNameFromCSR(csr)
    reason, thresholdDays, daysUntilExpiration := h.getRenewalContext(csr)
    
    h.log.Infof("Auto-approving renewal request for device %q (reason: %s, threshold: %d days, days until expiration: %d)", 
        deviceName, reason, thresholdDays, daysUntilExpiration)
    
    // ... rest of method ...
}
```

**Testing:**
- Test logging includes relevant information
- Test log levels are appropriate
- Test errors are logged correctly

---

### Task 6: Handle Edge Cases

**File:** `internal/service/certificatesigningrequest.go` (modify)

**Objective:** Handle edge cases in renewal validation.

**Implementation Steps:**

1. **Handle missing peer certificate gracefully:**
```go
// In validateRenewalRequest, handle missing peer certificate:
peerCert, err := signer.PeerCertificateFromCtx(ctx)
if err != nil {
    // For renewal, peer certificate should always be present (mTLS required)
    // But handle gracefully for testing/debugging
    h.log.Warnf("Peer certificate not found in context for renewal request - this may indicate missing mTLS")
    return fmt.Errorf("peer certificate required for renewal validation: %w", err)
}
```

2. **Handle expired certificates:**
```go
// In validateRenewalRequest, check certificate expiration:
now := time.Now()
if peerCert.NotAfter.Before(now) {
    // For expired certificates, renewal is still allowed (recovery scenario)
    h.log.Warnf("Peer certificate for device %q is expired (expired at %v) - allowing renewal for recovery", 
        deviceName, peerCert.NotAfter)
    // Don't return error - expired certificates can be renewed
}
```

3. **Handle devices without status:**
```go
// In validateRenewalRequest, handle devices without status:
if device.Status == nil {
    // Device may be newly created but not yet enrolled
    // For renewal, we require enrollment, so this is an error
    return fmt.Errorf("device %q has no status - device may not be enrolled", deviceName)
}
```

**Testing:**
- Test missing peer certificate handling
- Test expired certificate handling
- Test device without status handling
- Test other edge cases

---

### Task 7: Add Metrics (Optional)

**File:** `internal/service/certificatesigningrequest.go` (modify)

**Objective:** Add metrics for renewal validation (if metrics infrastructure exists).

**Implementation Steps:**

1. **Add renewal validation metrics:**
```go
// In validateRenewalRequest, add metrics:
if h.metricsCallback != nil {
    h.metricsCallback("certificate_renewal_validation_total", 1.0, nil)
}

// On validation failure:
if h.metricsCallback != nil {
    h.metricsCallback("certificate_renewal_validation_failures_total", 1.0, err)
}

// In autoApproveRenewal, add metrics:
if h.metricsCallback != nil {
    h.metricsCallback("certificate_renewal_auto_approved_total", 1.0, nil)
}
```

**Note:** This assumes a metrics callback exists. If not, this can be added in a later story.

**Testing:**
- Test metrics are recorded (if metrics infrastructure exists)
- Test metrics include correct labels

---

## Unit Tests

### Test File: `internal/service/certificatesigningrequest_renewal_test.go` (new)

**Test Cases:**

1. **TestIsRenewalRequest:**
   - Detects renewal request with "proactive" reason
   - Detects renewal request with "expired" reason
   - Returns false for non-renewal requests
   - Returns false for requests without labels

2. **TestGetRenewalContext:**
   - Extracts renewal context correctly
   - Handles missing threshold days
   - Handles missing days until expiration
   - Returns empty context for non-renewal requests

3. **TestExtractDeviceNameFromCSR:**
   - Extracts device name from bootstrap signer CSR
   - Extracts device name from other signer CSR
   - Handles empty CommonName
   - Handles parsing errors

4. **TestValidateRenewalRequest:**
   - Validates valid renewal request
   - Rejects non-existent device
   - Rejects unenrolled device
   - Rejects mismatched identity
   - Handles expired certificate (allows for recovery)
   - Handles missing peer certificate

5. **TestAutoApproveRenewal:**
   - Auto-approves validated renewal
   - Sets approval condition correctly
   - Includes renewal context in message
   - Doesn't approve already approved/denied requests

6. **TestRenewalRequestFlow:**
   - Complete flow: create -> validate -> approve -> sign
   - Validation failure prevents approval
   - Approval triggers signing

---

## Integration Tests

### Test File: `test/integration/certificate_renewal_validation_test.go` (new)

**Test Cases:**

1. **TestRenewalRequestValidation:**
   - Valid renewal request is validated and approved
   - Invalid renewal request is rejected
   - Renewal request triggers certificate signing

2. **TestRenewalWithValidCertificate:**
   - Renewal with valid certificate succeeds
   - Certificate is signed and issued
   - Device can use new certificate

3. **TestRenewalWithExpiredCertificate:**
   - Renewal with expired certificate succeeds (recovery)
   - Certificate is signed and issued
   - Device can recover with new certificate

4. **TestRenewalRequestRejection:**
   - Non-existent device is rejected
   - Unenrolled device is rejected
   - Mismatched identity is rejected

---

## Code Review Checklist

- [ ] Renewal request detection works correctly
- [ ] Device name extraction is accurate
- [ ] Validation covers all required checks
- [ ] Auto-approval happens after validation
- [ ] Error handling is comprehensive
- [ ] Logging is appropriate
- [ ] Edge cases are handled
- [ ] Unit tests achieve >80% coverage
- [ ] Integration tests cover main flows
- [ ] Code follows existing patterns

---

## Definition of Done

- [ ] isRenewalRequest method implemented
- [ ] getRenewalContext method implemented
- [ ] extractDeviceNameFromCSR method implemented
- [ ] validateRenewalRequest method implemented
- [ ] autoApproveRenewal method implemented
- [ ] Renewal validation integrated into CSR creation
- [ ] Error handling and logging added
- [ ] Edge cases handled
- [ ] Unit tests written and passing (>80% coverage)
- [ ] Integration tests written and passing
- [ ] Code reviewed and approved
- [ ] Documentation updated

---

## Related Files

- `internal/service/certificatesigningrequest.go` - CSR handler
- `internal/crypto/signer/` - Certificate signing logic
- `internal/store/device.go` - Device store
- `internal/crypto/signer/common.go` - Common signer utilities

---

## Dependencies

- **EDM-323-EPIC-1-STORY-4**: Database Schema for Certificate Tracking (must be completed)
  - Requires device certificate tracking fields
  
- **EDM-323-EPIC-2-STORY-2**: CSR Generation for Certificate Renewal (must be completed)
  - Requires renewal labels in CSR requests

---

## Notes

- **Peer Certificate**: Renewal requests must be authenticated using mTLS with the current valid certificate. The peer certificate is extracted from the request context.

- **Device Identity**: Device identity is determined by the CommonName in the certificate. The CSR CommonName must match the device name.

- **Expired Certificates**: Expired certificates can still be renewed (recovery scenario). The validation allows renewal even if the peer certificate is expired.

- **Auto-Approval**: Renewal requests are auto-approved after validation. This is different from initial enrollment, which may require manual approval.

- **Validation Order**: Validation happens before auto-approval. If validation fails, the request is rejected with an appropriate error.

- **Backward Compatibility**: Non-renewal requests continue to work as before. Only requests with renewal labels are subject to renewal validation.

- **TPM Support**: TPM-based renewals are validated using the existing TPM verification logic. Renewal validation is in addition to TPM verification.

---

**Document End**

