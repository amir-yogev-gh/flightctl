# Developer Story: Certificate Validation Before Activation

**Story ID:** EDM-323-EPIC-3-STORY-2  
**Epic:** EDM-323-EPIC-3 (Atomic Certificate Swap)  
**Feature:** EDM-323 (Agent Certificate Rotation)  
**Story Points:** 5  
**Priority:** High

## Overview

Implement comprehensive certificate validation before activation. Validate pending certificates to ensure they are valid, match device identity, are signed by the expected CA, and can be used with the provided key before making them active.

## Implementation Tasks

### Task 1: Create Certificate Swap Package

**File:** `internal/agent/device/certmanager/swap.go` (new)

**Objective:** Create new file for certificate swap and validation operations.

**Implementation Steps:**

1. **Create swap.go file with package structure:**
```go
package certmanager

import (
    "context"
    "crypto/tls"
    "crypto/x509"
    "fmt"
    "time"

    "github.com/flightctl/flightctl/internal/agent/device/certmanager/provider"
    fccrypto "github.com/flightctl/flightctl/pkg/crypto"
)

// CertificateValidator handles validation of pending certificates before activation.
type CertificateValidator struct {
    caBundlePath string
    deviceName   string
    log          provider.Logger
}
```

**Testing:**
- Test file structure is correct
- Test imports are valid

---

### Task 2: Implement CA Bundle Loading

**File:** `internal/agent/device/certmanager/swap.go` (modify)

**Objective:** Add method to load CA bundle for certificate verification.

**Implementation Steps:**

1. **Add loadCABundle method:**
```go
// loadCABundle loads the CA certificate bundle from the filesystem.
func (cv *CertificateValidator) loadCABundle(ctx context.Context, rw fileio.ReadWriter) (*x509.CertPool, error) {
    caPEM, err := rw.ReadFile(cv.caBundlePath)
    if err != nil {
        return nil, fmt.Errorf("failed to read CA bundle from %s: %w", cv.caBundlePath, err)
    }

    caPool := x509.NewCertPool()
    if !caPool.AppendCertsFromPEM(caPEM) {
        return nil, fmt.Errorf("failed to parse CA bundle from %s", cv.caBundlePath)
    }

    return caPool, nil
}
```

**Testing:**
- Test loadCABundle loads CA bundle correctly
- Test loadCABundle handles missing file
- Test loadCABundle handles invalid PEM

---

### Task 3: Implement Certificate Signature Verification

**File:** `internal/agent/device/certmanager/swap.go` (modify)

**Objective:** Verify certificate signature chain against CA bundle.

**Implementation Steps:**

1. **Add verifyCertificateSignature method:**
```go
// verifyCertificateSignature verifies the certificate signature chain against the CA bundle.
func (cv *CertificateValidator) verifyCertificateSignature(ctx context.Context, cert *x509.Certificate, caPool *x509.CertPool) error {
    // Verify certificate against CA bundle
    opts := x509.VerifyOptions{
        Roots: caPool,
    }

    _, err := cert.Verify(opts)
    if err != nil {
        return fmt.Errorf("certificate signature verification failed: %w", err)
    }

    return nil
}
```

**Testing:**
- Test verifyCertificateSignature with valid certificate
- Test verifyCertificateSignature with invalid signature
- Test verifyCertificateSignature with wrong CA

---

### Task 4: Implement Certificate Identity Verification

**File:** `internal/agent/device/certmanager/swap.go` (modify)

**Objective:** Verify certificate subject/SAN matches device identity.

**Implementation Steps:**

1. **Add verifyCertificateIdentity method:**
```go
// verifyCertificateIdentity verifies the certificate subject/SAN matches device identity.
func (cv *CertificateValidator) verifyCertificateIdentity(cert *x509.Certificate) error {
    // Verify CommonName matches device name
    if cert.Subject.CommonName != cv.deviceName {
        return fmt.Errorf("certificate CommonName %q does not match device name %q", 
            cert.Subject.CommonName, cv.deviceName)
    }

    // Verify SANs (if present) also match device identity
    // For now, we check CommonName. SANs can be added later if needed.
    
    return nil
}
```

**Testing:**
- Test verifyCertificateIdentity with matching identity
- Test verifyCertificateIdentity with mismatched identity
- Test verifyCertificateIdentity handles empty CommonName

---

### Task 5: Implement Certificate Expiration Check

**File:** `internal/agent/device/certmanager/swap.go` (modify)

**Objective:** Verify certificate is not expired and is valid.

**Implementation Steps:**

1. **Add verifyCertificateExpiration method:**
```go
// verifyCertificateExpiration verifies the certificate expiration date.
func (cv *CertificateValidator) verifyCertificateExpiration(cert *x509.Certificate) error {
    now := time.Now()
    
    // Check if certificate is expired
    if cert.NotAfter.Before(now) {
        return fmt.Errorf("certificate is expired (expired at %v, current time %v)", 
            cert.NotAfter, now)
    }
    
    // Check if certificate is not yet valid
    if cert.NotBefore.After(now) {
        return fmt.Errorf("certificate is not yet valid (valid from %v, current time %v)", 
            cert.NotBefore, now)
    }
    
    // Check if certificate has reasonable validity period remaining
    // Warn if certificate expires soon (within 7 days)
    timeUntilExpiration := cert.NotAfter.Sub(now)
    if timeUntilExpiration < 7*24*time.Hour {
        cv.log.Warnf("Certificate expires soon (in %v)", timeUntilExpiration)
    }
    
    return nil
}
```

**Testing:**
- Test verifyCertificateExpiration with valid certificate
- Test verifyCertificateExpiration with expired certificate
- Test verifyCertificateExpiration with not-yet-valid certificate
- Test verifyCertificateExpiration warns on soon-to-expire certificate

---

### Task 6: Implement Key Pair Verification

**File:** `internal/agent/device/certmanager/swap.go` (modify)

**Objective:** Verify certificate and key match (can be used together).

**Implementation Steps:**

1. **Add verifyKeyPair method:**
```go
// verifyKeyPair verifies that the certificate and private key match.
func (cv *CertificateValidator) verifyKeyPair(cert *x509.Certificate, keyPEM []byte) error {
    // Parse private key
    key, err := fccrypto.ParsePEMKey(keyPEM)
    if err != nil {
        return fmt.Errorf("failed to parse private key: %w", err)
    }

    // Get public key from certificate
    certPublicKey := cert.PublicKey

    // Get public key from private key
    keyPublicKey, err := fccrypto.GetPublicKeyFromPrivateKey(key)
    if err != nil {
        return fmt.Errorf("failed to extract public key from private key: %w", err)
    }

    // Compare public keys
    if !fccrypto.PublicKeysEqual(certPublicKey, keyPublicKey) {
        return fmt.Errorf("certificate and private key do not match")
    }

    // Additional verification: try to create a TLS certificate
    // This ensures the key pair can actually be used together
    _, err = tls.X509KeyPair(
        []byte(fccrypto.EncodeCertificatePEM(cert)),
        keyPEM,
    )
    if err != nil {
        return fmt.Errorf("failed to create TLS certificate from key pair: %w", err)
    }

    return nil
}
```

**Note:** This assumes `fccrypto` has methods like `ParsePEMKey`, `GetPublicKeyFromPrivateKey`, and `PublicKeysEqual`. If not, we'll need to implement them or use standard crypto libraries.

**Testing:**
- Test verifyKeyPair with matching key pair
- Test verifyKeyPair with mismatched key pair
- Test verifyKeyPair with invalid key
- Test verifyKeyPair creates valid TLS certificate

---

### Task 7: Implement Complete Validation Method

**File:** `internal/agent/device/certmanager/swap.go` (modify)

**Objective:** Combine all validation steps into a single method.

**Implementation Steps:**

1. **Add ValidatePendingCertificate method:**
```go
// ValidatePendingCertificate validates a pending certificate before activation.
// It performs all validation checks: signature, identity, expiration, and key pair.
func (cv *CertificateValidator) ValidatePendingCertificate(ctx context.Context, cert *x509.Certificate, keyPEM []byte, rw fileio.ReadWriter) error {
    cv.log.Debugf("Validating pending certificate for device %q", cv.deviceName)

    // Step 1: Load CA bundle
    caPool, err := cv.loadCABundle(ctx, rw)
    if err != nil {
        return fmt.Errorf("failed to load CA bundle: %w", err)
    }

    // Step 2: Verify certificate signature chain
    if err := cv.verifyCertificateSignature(ctx, cert, caPool); err != nil {
        return fmt.Errorf("signature verification failed: %w", err)
    }

    // Step 3: Verify certificate identity
    if err := cv.verifyCertificateIdentity(cert); err != nil {
        return fmt.Errorf("identity verification failed: %w", err)
    }

    // Step 4: Verify certificate expiration
    if err := cv.verifyCertificateExpiration(cert); err != nil {
        return fmt.Errorf("expiration check failed: %w", err)
    }

    // Step 5: Verify key pair
    if err := cv.verifyKeyPair(cert, keyPEM); err != nil {
        return fmt.Errorf("key pair verification failed: %w", err)
    }

    cv.log.Infof("Pending certificate validation successful for device %q", cv.deviceName)
    return nil
}
```

2. **Add NewCertificateValidator constructor:**
```go
// NewCertificateValidator creates a new certificate validator.
func NewCertificateValidator(caBundlePath string, deviceName string, log provider.Logger) *CertificateValidator {
    return &CertificateValidator{
        caBundlePath: caBundlePath,
        deviceName:   deviceName,
        log:          log,
    }
}
```

**Testing:**
- Test ValidatePendingCertificate with valid certificate
- Test ValidatePendingCertificate with invalid signature
- Test ValidatePendingCertificate with wrong identity
- Test ValidatePendingCertificate with expired certificate
- Test ValidatePendingCertificate with mismatched key pair

---

### Task 8: Integrate Validation into Certificate Manager

**File:** `internal/agent/device/certmanager/manager.go` (modify)

**Objective:** Integrate certificate validation before activation.

**Implementation Steps:**

1. **Add validation before activation:**
```go
// In ensureCertificate_do, after writing to pending:
if isRenewal {
    // For renewals, write to pending location first
    if err := cert.Storage.WritePending(crt, keyBytes); err != nil {
        return nil, fmt.Errorf("failed to write pending certificate: %w", err)
    }
    cm.log.Infof("Certificate %q/%q written to pending location for validation", providerName, cert.Name)
    
    // NEW: Validate pending certificate before activation
    // Get CA bundle path from config
    caBundlePath := cm.getCABundlePath(cfg)
    validator := NewCertificateValidator(caBundlePath, cert.Name, cm.log)
    
    // Load pending certificate and key for validation
    pendingCert, err := cert.Storage.LoadPendingCertificate(ctx)
    if err != nil {
        _ = cert.Storage.CleanupPending(ctx)
        return nil, fmt.Errorf("failed to load pending certificate: %w", err)
    }
    
    pendingKey, err := cert.Storage.LoadPendingKey(ctx)
    if err != nil {
        _ = cert.Storage.CleanupPending(ctx)
        return nil, fmt.Errorf("failed to load pending key: %w", err)
    }
    
    // Validate pending certificate
    if err := validator.ValidatePendingCertificate(ctx, pendingCert, pendingKey, cm.readWriter); err != nil {
        _ = cert.Storage.CleanupPending(ctx)
        return nil, fmt.Errorf("pending certificate validation failed: %w", err)
    }
    
    cm.log.Infof("Pending certificate validation successful for %q/%q", providerName, cert.Name)
    // Certificate will be activated in next story (atomic swap)
}
```

2. **Add getCABundlePath helper:**
```go
// getCABundlePath returns the CA bundle path from configuration.
func (cm *CertManager) getCABundlePath(cfg *provider.CertificateConfig) string {
    // Get CA bundle path from management service config
    if cm.config != nil {
        return cm.config.ManagementService.Config.Service.CertificateAuthority
    }
    // Default fallback
    return "/etc/flightctl/certs/ca.crt"
}
```

**Testing:**
- Test validation is called for renewals
- Test validation failure cleans up pending files
- Test validation success allows activation
- Test CA bundle path is retrieved correctly

---

## Unit Tests

### Test File: `internal/agent/device/certmanager/swap_test.go` (new)

**Test Cases:**

1. **TestLoadCABundle:**
   - Loads CA bundle correctly
   - Handles missing file
   - Handles invalid PEM

2. **TestVerifyCertificateSignature:**
   - Verifies valid signature
   - Rejects invalid signature
   - Rejects wrong CA

3. **TestVerifyCertificateIdentity:**
   - Verifies matching identity
   - Rejects mismatched identity
   - Handles empty CommonName

4. **TestVerifyCertificateExpiration:**
   - Verifies valid certificate
   - Rejects expired certificate
   - Rejects not-yet-valid certificate
   - Warns on soon-to-expire certificate

5. **TestVerifyKeyPair:**
   - Verifies matching key pair
   - Rejects mismatched key pair
   - Rejects invalid key

6. **TestValidatePendingCertificate:**
   - Validates valid certificate
   - Rejects invalid signature
   - Rejects wrong identity
   - Rejects expired certificate
   - Rejects mismatched key pair

---

## Integration Tests

### Test File: `test/integration/certificate_validation_test.go` (new)

**Test Cases:**

1. **TestCertificateValidationFlow:**
   - Pending certificate is validated
   - Valid certificate passes all checks
   - Invalid certificate is rejected

2. **TestValidationFailureHandling:**
   - Validation failure cleans up pending files
   - Active certificate is preserved
   - Error messages are clear

3. **TestValidationWithDifferentScenarios:**
   - Valid certificate passes
   - Expired certificate fails
   - Wrong identity fails
   - Wrong CA fails
   - Mismatched key pair fails

---

## Code Review Checklist

- [ ] CA bundle loading works correctly
- [ ] Signature verification is comprehensive
- [ ] Identity verification is accurate
- [ ] Expiration check is correct
- [ ] Key pair verification works
- [ ] All validation steps are integrated
- [ ] Error handling is comprehensive
- [ ] Logging is appropriate
- [ ] Unit tests achieve >80% coverage
- [ ] Integration tests cover main flows
- [ ] Code follows existing patterns

---

## Definition of Done

- [ ] CertificateValidator struct created
- [ ] CA bundle loading implemented
- [ ] Signature verification implemented
- [ ] Identity verification implemented
- [ ] Expiration check implemented
- [ ] Key pair verification implemented
- [ ] Complete validation method implemented
- [ ] Integration with certificate manager added
- [ ] Unit tests written and passing (>80% coverage)
- [ ] Integration tests written and passing
- [ ] Code reviewed and approved
- [ ] Documentation updated

---

## Related Files

- `internal/agent/device/certmanager/swap.go` - Certificate swap and validation
- `internal/agent/device/certmanager/manager.go` - Certificate manager
- `internal/agent/device/certmanager/provider/storage/fs.go` - Certificate storage
- `pkg/crypto/` - Crypto utilities

---

## Dependencies

- **EDM-323-EPIC-3-STORY-1**: Pending Certificate Storage (must be completed)
  - Requires pending certificate storage to be implemented

---

## Notes

- **CA Bundle**: The CA bundle is loaded from the management service configuration. It should contain the root CA and any intermediate CAs needed to verify certificates.

- **Validation Order**: Validation steps are performed in a specific order: signature, identity, expiration, key pair. This allows early failure detection.

- **Error Messages**: Each validation step returns detailed error messages to help diagnose issues.

- **Key Pair Verification**: The key pair verification ensures the certificate and private key can actually be used together by attempting to create a TLS certificate.

- **Expiration Warnings**: The validator warns if a certificate expires soon (within 7 days) but still allows activation. This helps with monitoring.

- **Cleanup on Failure**: If validation fails, pending files are cleaned up to prevent accumulation of invalid certificates.

- **Performance**: Validation is performed synchronously before activation. This ensures invalid certificates are never activated.

---

**Document End**

