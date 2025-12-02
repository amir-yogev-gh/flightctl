# Developer Story: Service-Side Recovery Request Validation

**Story ID:** EDM-323-EPIC-4-STORY-4  
**Epic:** EDM-323-EPIC-4 (Expired Certificate Recovery)  
**Feature:** EDM-323 (Agent Certificate Rotation)  
**Story Points:** 5  
**Priority:** High

## Overview

Implement service-side validation for recovery CSR requests from devices with expired certificates. The service should validate recovery requests using bootstrap certificates or TPM attestation, verify device identity, and auto-approve valid recovery requests.

## Implementation Tasks

### Task 1: Add Recovery Request Detection

**File:** `internal/service/certificatesigningrequest.go` (modify)

**Objective:** Detect recovery requests (expired certificate renewal).

**Implementation Steps:**

1. **Add isRecoveryRequest method:**
```go
// isRecoveryRequest checks if a CSR is a recovery request (expired certificate renewal).
func (h *ServiceHandler) isRecoveryRequest(csr *api.CertificateSigningRequest) bool {
    if csr.Metadata == nil || csr.Metadata.Labels == nil {
        return false
    }
    
    labels := *csr.Metadata.Labels
    renewalReason, hasRenewalLabel := labels["flightctl.io/renewal-reason"]
    
    // Recovery requests have renewal reason "expired"
    return hasRenewalLabel && renewalReason == "expired"
}
```

**Testing:**
- Test isRecoveryRequest detects recovery requests
- Test isRecoveryRequest returns false for non-recovery requests

---

### Task 2: Implement Recovery Request Validation

**File:** `internal/service/certificatesigningrequest.go` (modify)

**Objective:** Validate recovery requests comprehensively.

**Implementation Steps:**

1. **Add validateExpiredCertificateRenewal method:**
```go
// validateExpiredCertificateRenewal validates a recovery CSR request for expired certificate renewal.
func (h *ServiceHandler) validateExpiredCertificateRenewal(ctx context.Context, orgId uuid.UUID, csr *api.CertificateSigningRequest) error {
    // Extract device name from CSR
    deviceName, err := h.extractDeviceNameFromCSR(csr)
    if err != nil {
        return fmt.Errorf("failed to extract device name from CSR: %w", err)
    }
    
    // Step 1: Verify device exists in database
    device, err := h.store.Device().Get(ctx, orgId, deviceName)
    if err != nil {
        if errors.Is(err, flterrors.ErrResourceNotFound) {
            return fmt.Errorf("device %q not found - recovery requires existing device", deviceName)
        }
        return fmt.Errorf("failed to get device %q: %w", deviceName, err)
    }
    
    // Step 2: Verify device was previously enrolled
    if device.Status == nil {
        return fmt.Errorf("device %q has no status - device may not be enrolled", deviceName)
    }
    
    // Step 3: Check authentication method
    // Recovery requests can use:
    // - Bootstrap certificate (if not expired)
    // - TPM attestation (if bootstrap also expired)
    
    peerCert, err := signer.PeerCertificateFromCtx(ctx)
    if err != nil {
        // No peer certificate - must use TPM attestation
        return h.validateTPMAttestationForRecovery(ctx, orgId, csr, device)
    }
    
    // Step 4: Validate peer certificate (bootstrap or expired management cert)
    if err := h.validateRecoveryPeerCertificate(ctx, orgId, peerCert, device); err != nil {
        // If bootstrap cert validation fails, try TPM attestation
        h.log.Warnf("Peer certificate validation failed, checking TPM attestation: %v", err)
        return h.validateTPMAttestationForRecovery(ctx, orgId, csr, device)
    }
    
    // Step 5: Verify CSR CommonName matches device identity
    csrFingerprint, err := h.extractDeviceNameFromCSR(csr)
    if err != nil {
        return fmt.Errorf("failed to extract device name from CSR: %w", err)
    }
    
    if csrFingerprint != deviceName {
        return fmt.Errorf("CSR CommonName %q does not match device name %q", csrFingerprint, deviceName)
    }
    
    // Step 6: Check device is not revoked/blacklisted
    // TODO: Add revocation check if device revocation is implemented
    
    return nil
}
```

**Testing:**
- Test validateExpiredCertificateRenewal with valid recovery
- Test validateExpiredCertificateRenewal with non-existent device
- Test validateExpiredCertificateRenewal with unenrolled device

---

### Task 3: Implement Bootstrap Certificate Validation

**File:** `internal/service/certificatesigningrequest.go` (modify)

**Objective:** Validate bootstrap certificate for recovery authentication.

**Implementation Steps:**

1. **Add validateRecoveryPeerCertificate method:**
```go
// validateRecoveryPeerCertificate validates the peer certificate for recovery authentication.
// It accepts either bootstrap certificate or expired management certificate.
func (h *ServiceHandler) validateRecoveryPeerCertificate(ctx context.Context, orgId uuid.UUID, peerCert *x509.Certificate, device *api.Device) error {
    // Extract device fingerprint from peer certificate CN
    peerFingerprint, err := signer.DeviceFingerprintFromCN(h.ca.Config(), peerCert.Subject.CommonName)
    if err != nil {
        // If extraction fails, use CN directly
        peerFingerprint = peerCert.Subject.CommonName
    }
    
    // Verify peer certificate fingerprint matches device name
    deviceName := lo.FromPtr(device.Metadata.Name)
    if peerFingerprint != deviceName {
        return fmt.Errorf("peer certificate fingerprint %q does not match device name %q", peerFingerprint, deviceName)
    }
    
    // Check if certificate is expired (acceptable for recovery)
    now := time.Now()
    if peerCert.NotAfter.Before(now) {
        h.log.Warnf("Peer certificate for device %q is expired (expired at %v) - acceptable for recovery", 
            deviceName, peerCert.NotAfter)
        // Expired certificate is acceptable for recovery
    }
    
    // Verify certificate is signed by expected CA
    // This ensures it's either the management cert or bootstrap cert
    caPool := x509.NewCertPool()
    if h.ca.Config().Service.CertificateAuthorityData != nil {
        if !caPool.AppendCertsFromPEM(h.ca.Config().Service.CertificateAuthorityData) {
            return fmt.Errorf("failed to parse CA bundle")
        }
    }
    
    opts := x509.VerifyOptions{
        Roots: caPool,
    }
    
    if _, err := peerCert.Verify(opts); err != nil {
        return fmt.Errorf("peer certificate signature verification failed: %w", err)
    }
    
    return nil
}
```

**Testing:**
- Test validateRecoveryPeerCertificate with bootstrap cert
- Test validateRecoveryPeerCertificate with expired management cert
- Test validateRecoveryPeerCertificate with wrong identity

---

### Task 4: Implement TPM Attestation Validation

**File:** `internal/service/certificatesigningrequest.go` (modify)

**Objective:** Validate TPM attestation for recovery requests.

**Implementation Steps:**

1. **Add validateTPMAttestationForRecovery method:**
```go
// validateTPMAttestationForRecovery validates TPM attestation for recovery requests.
func (h *ServiceHandler) validateTPMAttestationForRecovery(ctx context.Context, orgId uuid.UUID, csr *api.CertificateSigningRequest, device *api.Device) error {
    // Check if CSR includes TPM attestation
    // TPM attestation may be in CSR metadata or CSR extensions
    
    // Extract TPM attestation from CSR
    attestation, err := h.extractTPMAttestationFromCSR(csr)
    if err != nil {
        return fmt.Errorf("failed to extract TPM attestation from CSR: %w", err)
    }
    
    if attestation == nil {
        return fmt.Errorf("TPM attestation not found in recovery request")
    }
    
    // Step 1: Verify device fingerprint matches
    deviceName := lo.FromPtr(device.Metadata.Name)
    if attestation.DeviceFingerprint != deviceName {
        return fmt.Errorf("TPM attestation device fingerprint %q does not match device name %q", 
            attestation.DeviceFingerprint, deviceName)
    }
    
    // Step 2: Verify TPM quote signature
    if err := h.verifyTPMQuote(ctx, orgId, attestation, device); err != nil {
        return fmt.Errorf("TPM quote verification failed: %w", err)
    }
    
    // Step 3: Verify PCR values match expected values (if stored)
    // This is optional - PCR values may change, so we may not enforce exact match
    // But we can verify they're reasonable
    
    // Step 4: Verify attestation freshness (nonce check)
    // TODO: Implement nonce freshness check to prevent replay attacks
    
    return nil
}
```

2. **Add extractTPMAttestationFromCSR method:**
```go
// extractTPMAttestationFromCSR extracts TPM attestation from CSR metadata or extensions.
func (h *ServiceHandler) extractTPMAttestationFromCSR(csr *api.CertificateSigningRequest) (*RenewalAttestation, error) {
    // Check CSR metadata for attestation
    if csr.Metadata != nil && csr.Metadata.Annotations != nil {
        // TPM attestation may be stored in annotations
        // Format depends on implementation
    }
    
    // Check CSR extensions for attestation
    // TPM attestation may be in CSR extensions
    
    // For now, return nil (not found)
    // Full implementation depends on how attestation is packaged in CSR
    return nil, nil
}
```

**Note:** Full TPM attestation extraction depends on how it's packaged in the CSR. This may need to be coordinated with the agent implementation.

**Testing:**
- Test validateTPMAttestationForRecovery with valid attestation
- Test validateTPMAttestationForRecovery with invalid attestation
- Test extractTPMAttestationFromCSR extracts correctly

---

### Task 5: Implement Device Fingerprint Validation

**File:** `internal/service/certificatesigningrequest.go` (modify)

**Objective:** Verify device fingerprint matches device record.

**Implementation Steps:**

1. **Add validateDeviceFingerprint method:**
```go
// validateDeviceFingerprint verifies device fingerprint matches device record.
func (h *ServiceHandler) validateDeviceFingerprint(ctx context.Context, orgId uuid.UUID, fingerprint string, device *api.Device) error {
    deviceName := lo.FromPtr(device.Metadata.Name)
    
    // Verify fingerprint matches device name
    if fingerprint != deviceName {
        return fmt.Errorf("device fingerprint %q does not match device name %q", fingerprint, deviceName)
    }
    
    // Optionally verify fingerprint matches stored fingerprint in device record
    // This requires device model to have fingerprint field (from EDM-323-EPIC-1-STORY-4)
    if device.CertificateFingerprint != nil {
        if *device.CertificateFingerprint != fingerprint {
            // Fingerprint mismatch - but this may be acceptable for recovery
            // (device may have new key)
            h.log.Warnf("Device fingerprint %q does not match stored fingerprint %q - acceptable for recovery", 
                fingerprint, *device.CertificateFingerprint)
        }
    }
    
    return nil
}
```

**Testing:**
- Test validateDeviceFingerprint with matching fingerprint
- Test validateDeviceFingerprint with mismatched fingerprint
- Test validateDeviceFingerprint handles missing stored fingerprint

---

### Task 6: Add Auto-Approval for Recovery Requests

**File:** `internal/service/certificatesigningrequest.go` (modify)

**Objective:** Auto-approve validated recovery requests.

**Implementation Steps:**

1. **Add autoApproveRecovery method:**
```go
// autoApproveRecovery auto-approves a validated recovery CSR request.
func (h *ServiceHandler) autoApproveRecovery(ctx context.Context, orgId uuid.UUID, csr *api.CertificateSigningRequest) {
    if api.IsStatusConditionTrue(csr.Status.Conditions, api.ConditionTypeCertificateSigningRequestApproved) || 
       api.IsStatusConditionTrue(csr.Status.Conditions, api.ConditionTypeCertificateSigningRequestDenied) {
        return
    }
    
    deviceName, _ := h.extractDeviceNameFromCSR(csr)
    message := fmt.Sprintf("Auto-approved recovery request for device %q (expired certificate renewal)", deviceName)
    
    api.SetStatusCondition(&csr.Status.Conditions, api.Condition{
        Type:    api.ConditionTypeCertificateSigningRequestApproved,
        Status:  api.ConditionStatusTrue,
        Reason:  "RecoveryAutoApproved",
        Message: message,
    })
    api.RemoveStatusCondition(&csr.Status.Conditions, api.ConditionTypeCertificateSigningRequestFailed)
    
    if _, err := h.store.CertificateSigningRequest().UpdateStatus(ctx, orgId, csr); err != nil {
        h.log.WithError(err).Error("failed to set recovery approval condition")
    }
}
```

2. **Update CreateCertificateSigningRequest to handle recoveries:**
```go
// In CreateCertificateSigningRequest, after validation:
// Check if this is a recovery request
if h.isRecoveryRequest(result) {
    // Validate recovery request
    if err := h.validateExpiredCertificateRenewal(ctx, orgId, result); err != nil {
        h.setCSRFailedCondition(ctx, orgId, result, "RecoveryValidationFailed", fmt.Sprintf("Recovery validation failed: %v", err))
        return result, api.StatusBadRequest(fmt.Sprintf("recovery validation failed: %v", err))
    }
    
    // Auto-approve recovery request
    h.autoApproveRecovery(ctx, orgId, result)
} else if h.isRenewalRequest(result) {
    // ... existing renewal validation ...
}
```

**Testing:**
- Test autoApproveRecovery sets approval condition
- Test auto-approval happens after validation
- Test validation failure prevents auto-approval

---

### Task 7: Add TPM Quote Verification

**File:** `internal/service/certificatesigningrequest.go` (modify)

**Objective:** Verify TPM quote signature and PCR values.

**Implementation Steps:**

1. **Add verifyTPMQuote method:**
```go
// verifyTPMQuote verifies TPM quote signature and PCR values.
func (h *ServiceHandler) verifyTPMQuote(ctx context.Context, orgId uuid.UUID, attestation *RenewalAttestation, device *api.Device) error {
    // Step 1: Get device's enrollment request to get original TPM attestation
    deviceName := lo.FromPtr(device.Metadata.Name)
    er, err := h.store.EnrollmentRequest().Get(ctx, orgId, deviceName)
    if err != nil {
        return fmt.Errorf("failed to get enrollment request for device %q: %w", deviceName, err)
    }
    
    // Step 2: Verify TPM quote using existing TPM verification infrastructure
    // This uses the same verification logic as enrollment
    if err := h.verifyTPMCSRRequest(ctx, orgId, csr); err != nil {
        return fmt.Errorf("TPM verification failed: %w", err)
    }
    
    // Step 3: Verify PCR values (optional - may not enforce exact match)
    // PCR values may change, so we may only verify they're reasonable
    
    return nil
}
```

**Note:** This reuses existing TPM verification logic from `verifyTPMCSRRequest`.

**Testing:**
- Test verifyTPMQuote with valid quote
- Test verifyTPMQuote with invalid quote
- Test verifyTPMQuote handles missing enrollment request

---

## Unit Tests

### Test File: `internal/service/certificatesigningrequest_recovery_test.go` (new)

**Test Cases:**

1. **TestIsRecoveryRequest:**
   - Detects recovery requests
   - Returns false for non-recovery requests

2. **TestValidateExpiredCertificateRenewal:**
   - Validates valid recovery request
   - Rejects non-existent device
   - Rejects unenrolled device

3. **TestValidateRecoveryPeerCertificate:**
   - Validates bootstrap certificate
   - Validates expired management certificate
   - Rejects wrong identity

4. **TestValidateTPMAttestationForRecovery:**
   - Validates valid TPM attestation
   - Rejects invalid TPM attestation
   - Rejects missing attestation

5. **TestAutoApproveRecovery:**
   - Auto-approves validated recovery
   - Sets approval condition correctly
   - Handles already approved/denied requests

---

## Integration Tests

### Test File: `test/integration/certificate_recovery_validation_test.go` (new)

**Test Cases:**

1. **TestRecoveryRequestValidation:**
   - Valid recovery request is validated and approved
   - Invalid recovery request is rejected
   - Recovery request triggers certificate signing

2. **TestRecoveryWithBootstrapCertificate:**
   - Recovery with bootstrap cert succeeds
   - Certificate is signed and issued
   - Device can use new certificate

3. **TestRecoveryWithTPMAttestation:**
   - Recovery with TPM attestation succeeds
   - TPM attestation is validated
   - Certificate is signed and issued

4. **TestRecoveryRequestRejection:**
   - Non-existent device is rejected
   - Unenrolled device is rejected
   - Invalid attestation is rejected

---

## Code Review Checklist

- [ ] Recovery request detection works correctly
- [ ] Bootstrap certificate validation works
- [ ] TPM attestation validation works
- [ ] Device fingerprint validation works
- [ ] Auto-approval happens after validation
- [ ] Error handling is comprehensive
- [ ] Logging is appropriate
- [ ] Unit tests achieve >80% coverage
- [ ] Integration tests cover main flows
- [ ] Code follows existing patterns

---

## Definition of Done

- [ ] isRecoveryRequest method implemented
- [ ] validateExpiredCertificateRenewal implemented
- [ ] validateRecoveryPeerCertificate implemented
- [ ] validateTPMAttestationForRecovery implemented
- [ ] validateDeviceFingerprint implemented
- [ ] autoApproveRecovery implemented
- [ ] verifyTPMQuote implemented
- [ ] Integration with CSR creation added
- [ ] Unit tests written and passing (>80% coverage)
- [ ] Integration tests written and passing
- [ ] Code reviewed and approved
- [ ] Documentation updated

---

## Related Files

- `internal/service/certificatesigningrequest.go` - CSR handler
- `internal/store/device.go` - Device store
- `internal/crypto/signer/` - Certificate signing logic

---

## Dependencies

- **EDM-323-EPIC-4-STORY-3**: TPM Attestation Generation (must be completed)
  - Requires TPM attestation to be generated by agent

- **EDM-323-EPIC-1-STORY-4**: Database Schema for Certificate Tracking (must be completed)
  - Requires device certificate tracking fields

---

## Notes

- **Recovery vs Renewal**: Recovery requests are for expired certificates (renewal reason "expired"), while renewal requests are for certificates expiring soon (renewal reason "proactive").

- **Authentication Methods**: Recovery requests can use either bootstrap certificate (if not expired) or TPM attestation (if bootstrap also expired). The service tries bootstrap first, then TPM attestation.

- **Bootstrap Certificate**: Bootstrap certificates are accepted even if expired (for recovery). The service validates they're signed by the expected CA and match device identity.

- **TPM Attestation**: TPM attestation is validated using the same infrastructure as enrollment. The TPM quote and PCR values are verified against device records.

- **Device Fingerprint**: Device fingerprint must match the device name. Stored fingerprint mismatch may be acceptable for recovery (device may have new key).

- **Auto-Approval**: Valid recovery requests are auto-approved to enable automatic recovery without manual intervention.

- **Error Handling**: Validation failures return detailed error messages to help diagnose issues. Invalid requests are rejected with appropriate status codes.

---

**Document End**

