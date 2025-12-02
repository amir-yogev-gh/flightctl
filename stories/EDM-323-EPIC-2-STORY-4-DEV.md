# Developer Story: Certificate Issuance for Renewals

**Story ID:** EDM-323-EPIC-2-STORY-4  
**Epic:** EDM-323-EPIC-2 (Proactive Certificate Renewal)  
**Feature:** EDM-323 (Agent Certificate Rotation)  
**Story Points:** 3  
**Priority:** High

## Overview

Enhance certificate signing to handle renewal requests and update device certificate tracking fields in the database after certificate issuance. Ensure renewal certificates preserve device identity and use standard validity periods.

## Implementation Tasks

### Task 1: Enhance Certificate Signing for Renewals

**File:** `internal/service/certificatesigningrequest.go` (modify)

**Objective:** Enhance `signApprovedCertificateSigningRequest` to handle renewal requests and extract certificate information.

**Implementation Steps:**

1. **Add method to extract device name from CSR:**
```go
// extractDeviceNameFromCSR extracts the device name from CSR CommonName.
// This is a helper method that can be reused from renewal validation.
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

2. **Enhance signApprovedCertificateSigningRequest:**
```go
func (h *ServiceHandler) signApprovedCertificateSigningRequest(ctx context.Context, orgId uuid.UUID, csr *api.CertificateSigningRequest) {
    if csr.Status.Certificate != nil && len(*csr.Status.Certificate) > 0 {
        return
    }

    request, _, err := newSignRequestFromCertificateSigningRequest(csr)
    if err != nil {
        h.setCSRFailedCondition(ctx, orgId, csr, "SigningFailed", fmt.Sprintf("Failed to sign certificate: %v", err))
        return
    }

    // For renewal requests, ensure standard validity (365 days)
    isRenewal := h.isRenewalRequest(csr)
    if isRenewal {
        // Set expiration to 365 days for renewals (standard validity)
        expirySeconds := int32(365 * 24 * 60 * 60) // 365 days
        if request.ExpirationSeconds() == nil || *request.ExpirationSeconds() != expirySeconds {
            // Create new request with standard expiration
            request = signer.NewSignRequestFromBytes(
                request.SignerName(),
                request.CSRBytes(),
                signer.WithExpirationSeconds(expirySeconds),
            )
        }
    }

    certPEM, err := signer.SignAsPEM(ctx, h.ca, request)
    if err != nil {
        h.setCSRFailedCondition(ctx, orgId, csr, "SigningFailed", fmt.Sprintf("Failed to sign certificate: %v", err))
        return
    }

    csr.Status.Certificate = &certPEM
    if _, err := h.store.CertificateSigningRequest().UpdateStatus(ctx, orgId, csr); err != nil {
        h.log.WithError(err).Error("failed to set signed certificate")
        return
    }

    // Update device certificate tracking if this is a renewal
    if isRenewal {
        if err := h.updateDeviceCertificateTracking(ctx, orgId, csr, certPEM); err != nil {
            h.log.WithError(err).Warn("failed to update device certificate tracking after renewal")
            // Don't fail the CSR - certificate was issued successfully
        }
    }
}
```

**Testing:**
- Test certificate signing for renewals
- Test standard validity period (365 days) is used
- Test certificate identity is preserved
- Test device tracking is updated

---

### Task 2: Parse Certificate from PEM

**File:** `internal/service/certificatesigningrequest.go` (modify)

**Objective:** Add method to parse issued certificate from PEM for tracking updates.

**Implementation Steps:**

1. **Add method to parse certificate:**
```go
// parseCertificateFromPEM parses a PEM-encoded certificate.
func (h *ServiceHandler) parseCertificateFromPEM(certPEM string) (*x509.Certificate, error) {
    cert, err := fccrypto.ParsePEMCertificate([]byte(certPEM))
    if err != nil {
        return nil, fmt.Errorf("failed to parse certificate PEM: %w", err)
    }
    return cert, nil
}
```

**Note:** Import `fccrypto` and `crypto/x509` if not already imported.

**Testing:**
- Test parsing valid PEM certificate
- Test parsing invalid PEM certificate
- Test parsing empty PEM

---

### Task 3: Update Device Certificate Tracking

**File:** `internal/service/certificatesigningrequest.go` (modify)

**Objective:** Update device certificate tracking fields after certificate issuance.

**Implementation Steps:**

1. **Add updateDeviceCertificateTracking method:**
```go
// updateDeviceCertificateTracking updates device certificate tracking fields after renewal.
func (h *ServiceHandler) updateDeviceCertificateTracking(ctx context.Context, orgId uuid.UUID, csr *api.CertificateSigningRequest, certPEM string) error {
    // Extract device name from CSR
    deviceName, err := h.extractDeviceNameFromCSR(csr)
    if err != nil {
        return fmt.Errorf("failed to extract device name from CSR: %w", err)
    }
    
    // Parse certificate to get expiration
    cert, err := h.parseCertificateFromPEM(certPEM)
    if err != nil {
        return fmt.Errorf("failed to parse issued certificate: %w", err)
    }
    
    // Calculate certificate fingerprint (SHA256 hash)
    fingerprint, err := fccrypto.CalculateCertificateFingerprint(cert)
    if err != nil {
        h.log.WithError(err).Warn("failed to calculate certificate fingerprint")
        fingerprint = nil // Continue without fingerprint
    }
    
    // Get current renewal count
    device, err := h.store.Device().Get(ctx, orgId, deviceName)
    if err != nil {
        return fmt.Errorf("failed to get device %q: %w", deviceName, err)
    }
    
    // Calculate renewal count (increment if device has certificate tracking)
    renewalCount := 0
    if device != nil {
        // If device model has certificate_renewal_count field, get it
        // Otherwise, start at 0 for first renewal
        // Note: This assumes the device model has been updated with certificate tracking fields
        // from EDM-323-EPIC-1-STORY-4
    }
    
    // Update device certificate tracking
    expirationTime := cert.NotAfter.UTC()
    lastRenewedTime := time.Now().UTC()
    renewalCount++ // Increment renewal count
    
    // Use device store method to update certificate tracking
    // Note: This method should be available from EDM-323-EPIC-1-STORY-4
    if err := h.store.Device().UpdateCertificateExpiration(ctx, orgId, deviceName, &expirationTime); err != nil {
        return fmt.Errorf("failed to update certificate expiration: %w", err)
    }
    
    if err := h.store.Device().UpdateCertificateRenewalInfo(ctx, orgId, deviceName, &lastRenewedTime, renewalCount, fingerprint); err != nil {
        return fmt.Errorf("failed to update certificate renewal info: %w", err)
    }
    
    h.log.Infof("Updated certificate tracking for device %q: expiration=%v, renewal_count=%d", 
        deviceName, expirationTime, renewalCount)
    
    return nil
}
```

**Note:** This assumes the device store methods `UpdateCertificateExpiration` and `UpdateCertificateRenewalInfo` are available from EDM-323-EPIC-1-STORY-4. If not yet implemented, we'll need to add them.

**Testing:**
- Test certificate tracking update for renewals
- Test expiration date is set correctly
- Test renewal count is incremented
- Test fingerprint is calculated correctly
- Test error handling

---

### Task 4: Handle Certificate Fingerprint Calculation

**File:** `internal/service/certificatesigningrequest.go` (modify)

**Objective:** Calculate certificate fingerprint for tracking.

**Implementation Steps:**

1. **Check if fingerprint calculation utility exists:**
```go
// In updateDeviceCertificateTracking, calculate fingerprint:
// Check if fccrypto has CalculateCertificateFingerprint
// If not, implement simple SHA256 hash:
import (
    "crypto/sha256"
    "encoding/hex"
)

// calculateCertificateFingerprint calculates SHA256 fingerprint of certificate.
func (h *ServiceHandler) calculateCertificateFingerprint(cert *x509.Certificate) (string, error) {
    hash := sha256.Sum256(cert.Raw)
    return hex.EncodeToString(hash[:]), nil
}
```

2. **Use fingerprint in tracking update:**
```go
// In updateDeviceCertificateTracking:
fingerprint, err := h.calculateCertificateFingerprint(cert)
if err != nil {
    h.log.WithError(err).Warn("failed to calculate certificate fingerprint")
    fingerprint = nil // Continue without fingerprint
} else {
    fingerprintStr := fingerprint
    fingerprint = &fingerprintStr
}
```

**Testing:**
- Test fingerprint calculation
- Test fingerprint is unique per certificate
- Test fingerprint format is correct

---

### Task 5: Get Current Renewal Count

**File:** `internal/service/certificatesigningrequest.go` (modify)

**Objective:** Retrieve and increment current renewal count from device.

**Implementation Steps:**

1. **Add method to get renewal count:**
```go
// getDeviceRenewalCount retrieves the current renewal count for a device.
func (h *ServiceHandler) getDeviceRenewalCount(ctx context.Context, orgId uuid.UUID, deviceName string) (int, error) {
    device, err := h.store.Device().Get(ctx, orgId, deviceName)
    if err != nil {
        return 0, fmt.Errorf("failed to get device %q: %w", deviceName, err)
    }
    
    // Access certificate_renewal_count from device model
    // Note: This assumes the device model has been updated with certificate tracking fields
    // from EDM-323-EPIC-1-STORY-4
    // For now, return 0 if field doesn't exist (first renewal)
    
    // If device model has CertificateRenewalCount field:
    // return device.CertificateRenewalCount, nil
    
    // Otherwise, return 0 (first renewal)
    return 0, nil
}
```

2. **Use renewal count in tracking update:**
```go
// In updateDeviceCertificateTracking:
renewalCount, err := h.getDeviceRenewalCount(ctx, orgId, deviceName)
if err != nil {
    h.log.WithError(err).Warn("failed to get current renewal count, starting at 0")
    renewalCount = 0
}
renewalCount++ // Increment for this renewal
```

**Testing:**
- Test getting renewal count from device
- Test incrementing renewal count
- Test handling missing renewal count (first renewal)

---

### Task 6: Add Logging for Certificate Issuance

**File:** `internal/service/certificatesigningrequest.go` (modify)

**Objective:** Add appropriate logging for certificate issuance and tracking updates.

**Implementation Steps:**

1. **Add logging in signApprovedCertificateSigningRequest:**
```go
// In signApprovedCertificateSigningRequest, after successful signing:
if isRenewal {
    deviceName, _ := h.extractDeviceNameFromCSR(csr)
    h.log.Infof("Issued renewal certificate for device %q (expires: %v)", 
        deviceName, cert.NotAfter)
} else {
    h.log.Debugf("Issued certificate for CSR %q", lo.FromPtr(csr.Metadata.Name))
}
```

2. **Add logging in updateDeviceCertificateTracking:**
```go
// Already added in Task 3, but ensure it's comprehensive:
h.log.Infof("Updated certificate tracking for device %q: expiration=%v, last_renewed=%v, renewal_count=%d, fingerprint=%s", 
    deviceName, expirationTime, lastRenewedTime, renewalCount, lo.FromPtrOr(fingerprint, "none"))
```

**Testing:**
- Test logging includes relevant information
- Test log levels are appropriate

---

### Task 7: Handle Errors Gracefully

**File:** `internal/service/certificatesigningrequest.go` (modify)

**Objective:** Ensure certificate issuance succeeds even if tracking update fails.

**Implementation Steps:**

1. **Ensure tracking update doesn't block issuance:**
```go
// In signApprovedCertificateSigningRequest, after certificate issuance:
// Update device certificate tracking if this is a renewal
if isRenewal {
    if err := h.updateDeviceCertificateTracking(ctx, orgId, csr, certPEM); err != nil {
        h.log.WithError(err).Warn("failed to update device certificate tracking after renewal")
        // Don't fail the CSR - certificate was issued successfully
        // Tracking update is best-effort
    }
}
```

2. **Add retry logic for tracking updates (optional):**
```go
// In updateDeviceCertificateTracking, add retry for critical updates:
// Retry update up to 3 times with exponential backoff
maxRetries := 3
for i := 0; i < maxRetries; i++ {
    err := h.store.Device().UpdateCertificateExpiration(ctx, orgId, deviceName, &expirationTime)
    if err == nil {
        break
    }
    if i < maxRetries-1 {
        time.Sleep(time.Duration(i+1) * 100 * time.Millisecond)
        continue
    }
    return fmt.Errorf("failed to update certificate expiration after %d retries: %w", maxRetries, err)
}
```

**Testing:**
- Test certificate issuance succeeds even if tracking fails
- Test retry logic works
- Test error messages are clear

---

## Unit Tests

### Test File: `internal/service/certificatesigningrequest_renewal_issuance_test.go` (new)

**Test Cases:**

1. **TestExtractDeviceNameFromCSR:**
   - Extracts device name from bootstrap signer CSR
   - Extracts device name from other signer CSR
   - Handles empty CommonName
   - Handles parsing errors

2. **TestParseCertificateFromPEM:**
   - Parses valid PEM certificate
   - Parses invalid PEM certificate
   - Handles empty PEM

3. **TestCalculateCertificateFingerprint:**
   - Calculates fingerprint correctly
   - Fingerprint is unique per certificate
   - Fingerprint format is correct

4. **TestGetDeviceRenewalCount:**
   - Gets renewal count from device
   - Returns 0 for first renewal
   - Handles missing device

5. **TestUpdateDeviceCertificateTracking:**
   - Updates expiration date correctly
   - Updates renewal count correctly
   - Updates fingerprint correctly
   - Handles errors gracefully

6. **TestSignApprovedCertificateSigningRequestForRenewal:**
   - Issues certificate with 365-day validity
   - Preserves device identity
   - Updates device tracking
   - Handles tracking update failures gracefully

---

## Integration Tests

### Test File: `test/integration/certificate_renewal_issuance_test.go` (new)

**Test Cases:**

1. **TestRenewalCertificateIssuance:**
   - Renewal CSR is signed successfully
   - Certificate has 365-day validity
   - Certificate preserves device identity
   - Device tracking is updated

2. **TestRenewalCertificateTracking:**
   - Expiration date is updated correctly
   - Renewal count is incremented
   - Last renewed timestamp is set
   - Fingerprint is stored

3. **TestRenewalCertificateIdentity:**
   - Certificate CommonName matches device
   - Certificate SANs match original
   - Certificate is signed by same CA

4. **TestRenewalCertificateFailureHandling:**
   - Certificate issuance succeeds even if tracking fails
   - Tracking update is retried on failure
   - Errors are logged appropriately

---

## Code Review Checklist

- [ ] Certificate signing handles renewals correctly
- [ ] Standard validity (365 days) is used for renewals
- [ ] Device identity is preserved
- [ ] Certificate tracking is updated after issuance
- [ ] Error handling is comprehensive
- [ ] Logging is appropriate
- [ ] Tracking update doesn't block issuance
- [ ] Unit tests achieve >80% coverage
- [ ] Integration tests cover main flows
- [ ] Code follows existing patterns

---

## Definition of Done

- [ ] Certificate signing enhanced for renewals
- [ ] Standard validity period enforced
- [ ] Device identity preserved
- [ ] Certificate tracking update implemented
- [ ] Fingerprint calculation implemented
- [ ] Renewal count tracking implemented
- [ ] Error handling and logging added
- [ ] Unit tests written and passing (>80% coverage)
- [ ] Integration tests written and passing
- [ ] Code reviewed and approved
- [ ] Documentation updated

---

## Related Files

- `internal/service/certificatesigningrequest.go` - CSR handler
- `internal/store/device.go` - Device store (certificate tracking methods)
- `internal/crypto/signer/` - Certificate signing logic
- `internal/crypto/cert.go` - Certificate utilities

---

## Dependencies

- **EDM-323-EPIC-1-STORY-4**: Database Schema for Certificate Tracking (must be completed)
  - Requires device certificate tracking fields in database
  - Requires `UpdateCertificateExpiration` and `UpdateCertificateRenewalInfo` methods
  
- **EDM-323-EPIC-2-STORY-3**: Service-Side Renewal Request Validation (must be completed)
  - Requires renewal request detection
  - Requires device name extraction

---

## Notes

- **Certificate Validity**: Renewal certificates use standard 365-day validity. This can be overridden by CSR expiration request, but defaults to 365 days for renewals.

- **Device Identity**: Certificate identity (CommonName, SANs) is preserved from the original certificate. The CSR CommonName must match the device identity.

- **Certificate Tracking**: Device certificate tracking fields are updated after successful certificate issuance. If tracking update fails, certificate issuance still succeeds (best-effort tracking).

- **Renewal Count**: Renewal count starts at 0 for the first renewal and increments with each subsequent renewal. The count is stored in the device record.

- **Certificate Fingerprint**: SHA256 fingerprint of the certificate is calculated and stored for tracking. This helps identify the specific certificate instance.

- **Error Handling**: Certificate issuance must succeed even if tracking update fails. Tracking updates are best-effort and should not block certificate issuance.

- **Database Methods**: This story assumes the device store methods from EDM-323-EPIC-1-STORY-4 are available. If not, they need to be implemented first.

- **Backward Compatibility**: Non-renewal certificate issuance continues to work as before. Only renewal requests trigger tracking updates.

---

**Document End**

