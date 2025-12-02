# Developer Story: Bootstrap Certificate Fallback Handler

**Story ID:** EDM-323-EPIC-4-STORY-2  
**Epic:** EDM-323-EPIC-4 (Expired Certificate Recovery)  
**Feature:** EDM-323 (Agent Certificate Rotation)  
**Story Points:** 5  
**Priority:** High

## Overview

Implement bootstrap certificate fallback mechanism for expired certificate recovery. When the management certificate is expired, the agent should fall back to using the bootstrap (enrollment) certificate for authentication to enable certificate renewal requests.

## Implementation Tasks

### Task 1: Create Bootstrap Certificate Handler

**File:** `internal/agent/device/bootstrap_cert.go` (new)

**Objective:** Create new file for bootstrap certificate handling.

**Implementation Steps:**

1. **Create bootstrap_cert.go file:**
```go
package device

import (
    "context"
    "crypto/tls"
    "crypto/x509"
    "fmt"
    "path/filepath"
    "time"

    agent_config "github.com/flightctl/flightctl/internal/agent/config"
    "github.com/flightctl/flightctl/internal/agent/device/fileio"
    fccrypto "github.com/flightctl/flightctl/pkg/crypto"
    "github.com/flightctl/flightctl/pkg/log"
)

// BootstrapCertificateHandler handles bootstrap certificate operations for recovery.
type BootstrapCertificateHandler struct {
    certPath string
    keyPath  string
    rw       fileio.ReadWriter
    log      *log.PrefixLogger
}

// NewBootstrapCertificateHandler creates a new bootstrap certificate handler.
func NewBootstrapCertificateHandler(dataDir string, rw fileio.ReadWriter, log *log.PrefixLogger) *BootstrapCertificateHandler {
    certPath := filepath.Join(dataDir, agent_config.DefaultCertsDirName, agent_config.EnrollmentCertFile)
    keyPath := filepath.Join(dataDir, agent_config.DefaultCertsDirName, agent_config.EnrollmentKeyFile)
    
    return &BootstrapCertificateHandler{
        certPath: certPath,
        keyPath:  keyPath,
        rw:       rw,
        log:      log,
    }
}
```

**Testing:**
- Test file structure is correct
- Test paths are generated correctly

---

### Task 2: Implement Bootstrap Certificate Loading

**File:** `internal/agent/device/bootstrap_cert.go` (modify)

**Objective:** Add method to load bootstrap certificate.

**Implementation Steps:**

1. **Add GetBootstrapCertificate method:**
```go
// GetBootstrapCertificate loads and returns the bootstrap certificate and key.
// Returns error if certificate is missing or invalid.
func (b *BootstrapCertificateHandler) GetBootstrapCertificate(ctx context.Context) (*x509.Certificate, []byte, error) {
    // Check if certificate file exists
    exists, err := b.rw.PathExists(b.certPath)
    if err != nil {
        return nil, nil, fmt.Errorf("failed to check bootstrap certificate existence: %w", err)
    }
    if !exists {
        return nil, nil, fmt.Errorf("bootstrap certificate not found at %s", b.certPath)
    }

    // Load certificate
    certPEM, err := b.rw.ReadFile(b.certPath)
    if err != nil {
        return nil, nil, fmt.Errorf("failed to read bootstrap certificate: %w", err)
    }

    cert, err := fccrypto.ParsePEMCertificate(certPEM)
    if err != nil {
        return nil, nil, fmt.Errorf("failed to parse bootstrap certificate: %w", err)
    }

    // Load key
    keyPEM, err := b.rw.ReadFile(b.keyPath)
    if err != nil {
        return nil, nil, fmt.Errorf("failed to read bootstrap key: %w", err)
    }

    return cert, keyPEM, nil
}
```

**Testing:**
- Test GetBootstrapCertificate loads certificate correctly
- Test GetBootstrapCertificate handles missing certificate
- Test GetBootstrapCertificate handles invalid certificate

---

### Task 3: Implement Bootstrap Certificate Validation

**File:** `internal/agent/device/bootstrap_cert.go` (modify)

**Objective:** Validate bootstrap certificate is not expired.

**Implementation Steps:**

1. **Add ValidateBootstrapCertificate method:**
```go
// ValidateBootstrapCertificate validates that the bootstrap certificate is not expired.
func (b *BootstrapCertificateHandler) ValidateBootstrapCertificate(cert *x509.Certificate) error {
    now := time.Now()
    
    // Check if certificate is expired
    if cert.NotAfter.Before(now) {
        return fmt.Errorf("bootstrap certificate is expired (expired at %v)", cert.NotAfter)
    }
    
    // Check if certificate is not yet valid
    if cert.NotBefore.After(now) {
        return fmt.Errorf("bootstrap certificate is not yet valid (valid from %v)", cert.NotBefore)
    }
    
    return nil
}
```

2. **Add HasValidBootstrapCertificate method:**
```go
// HasValidBootstrapCertificate checks if a valid bootstrap certificate exists.
func (b *BootstrapCertificateHandler) HasValidBootstrapCertificate(ctx context.Context) (bool, error) {
    cert, _, err := b.GetBootstrapCertificate(ctx)
    if err != nil {
        return false, nil // Certificate doesn't exist or can't be loaded
    }
    
    if err := b.ValidateBootstrapCertificate(cert); err != nil {
        b.log.Warnf("Bootstrap certificate exists but is invalid: %v", err)
        return false, nil
    }
    
    return true, nil
}
```

**Testing:**
- Test ValidateBootstrapCertificate with valid certificate
- Test ValidateBootstrapCertificate with expired certificate
- Test HasValidBootstrapCertificate returns correct value

---

### Task 4: Implement Certificate Switching Logic

**File:** `internal/agent/device/bootstrap_cert.go` (modify)

**Objective:** Add logic to switch between management and bootstrap certificates.

**Implementation Steps:**

1. **Add GetCertificateForAuth method:**
```go
// GetCertificateForAuth returns the appropriate certificate for authentication.
// If management certificate is expired, falls back to bootstrap certificate.
func (b *BootstrapCertificateHandler) GetCertificateForAuth(ctx context.Context, managementCertPath, managementKeyPath string) (*tls.Certificate, error) {
    // First, try to load management certificate
    mgmtCertExists, err := b.rw.PathExists(managementCertPath)
    if err == nil && mgmtCertExists {
        mgmtCertPEM, err := b.rw.ReadFile(managementCertPath)
        if err == nil {
            mgmtCert, err := fccrypto.ParsePEMCertificate(mgmtCertPEM)
            if err == nil {
                // Check if management certificate is expired
                if time.Now().Before(mgmtCert.NotAfter) {
                    // Management certificate is valid - use it
                    mgmtKeyPEM, err := b.rw.ReadFile(managementKeyPath)
                    if err == nil {
                        tlsCert, err := tls.X509KeyPair(mgmtCertPEM, mgmtKeyPEM)
                        if err == nil {
                            b.log.Debug("Using management certificate for authentication")
                            return &tlsCert, nil
                        }
                    }
                } else {
                    b.log.Warnf("Management certificate is expired (expired at %v), falling back to bootstrap", mgmtCert.NotAfter)
                }
            }
        }
    }

    // Management certificate is expired or missing - fall back to bootstrap
    b.log.Infof("Falling back to bootstrap certificate for authentication")
    cert, keyPEM, err := b.GetBootstrapCertificate(ctx)
    if err != nil {
        return nil, fmt.Errorf("failed to get bootstrap certificate: %w", err)
    }

    if err := b.ValidateBootstrapCertificate(cert); err != nil {
        return nil, fmt.Errorf("bootstrap certificate validation failed: %w", err)
    }

    // Create TLS certificate from bootstrap cert and key
    certPEM, err := fccrypto.EncodeCertificatePEM(cert)
    if err != nil {
        return nil, fmt.Errorf("failed to encode bootstrap certificate: %w", err)
    }

    tlsCert, err := tls.X509KeyPair(certPEM, keyPEM)
    if err != nil {
        return nil, fmt.Errorf("failed to create TLS certificate from bootstrap cert: %w", err)
    }

    b.log.Infof("Using bootstrap certificate for authentication")
    return &tlsCert, nil
}
```

**Testing:**
- Test GetCertificateForAuth uses management cert when valid
- Test GetCertificateForAuth falls back to bootstrap when expired
- Test GetCertificateForAuth handles missing certificates

---

### Task 5: Integrate with Management Client

**File:** `internal/agent/client/management.go` (modify)

**Objective:** Use bootstrap certificate fallback in management client.

**Implementation Steps:**

1. **Add method to create client with certificate fallback:**
```go
// NewManagementWithCertificateFallback creates a management client with certificate fallback.
// If management certificate is expired, it falls back to bootstrap certificate.
func NewManagementWithCertificateFallback(
    client *client.ClientWithResponses,
    cb RPCMetricsCallback,
    bootstrapHandler *device.BootstrapCertificateHandler,
    managementCertPath, managementKeyPath string,
) (Management, error) {
    // Get appropriate certificate for authentication
    tlsCert, err := bootstrapHandler.GetCertificateForAuth(context.Background(), managementCertPath, managementKeyPath)
    if err != nil {
        return nil, fmt.Errorf("failed to get certificate for authentication: %w", err)
    }

    // Create HTTP client with certificate
    httpClient := &http.Client{
        Transport: &http.Transport{
            TLSClientConfig: &tls.Config{
                Certificates: []tls.Certificate{*tlsCert},
            },
        },
    }

    // Create client with certificate
    // Note: This is a simplified example - actual implementation may vary
    return NewManagement(client, cb), nil
}
```

**Note:** This is a placeholder - actual integration depends on how the management client is created.

**Testing:**
- Test client creation with bootstrap fallback
- Test client uses correct certificate

---

### Task 6: Integrate with Certificate Manager

**File:** `internal/agent/device/certmanager/lifecycle.go` (modify)

**Objective:** Use bootstrap certificate in recovery flow.

**Implementation Steps:**

1. **Add bootstrap handler to LifecycleManager:**
```go
// In LifecycleManager struct, add:
type LifecycleManager struct {
    // ... existing fields ...
    bootstrapHandler *device.BootstrapCertificateHandler
}

// In NewLifecycleManager, add bootstrap handler:
func NewLifecycleManager(..., bootstrapHandler *device.BootstrapCertificateHandler) *LifecycleManager {
    return &LifecycleManager{
        // ... existing fields ...
        bootstrapHandler: bootstrapHandler,
    }
}
```

2. **Use bootstrap certificate in recovery:**
```go
// In TriggerRecovery, check for bootstrap certificate:
func (lm *LifecycleManager) TriggerRecovery(ctx context.Context, providerName string, certName string) error {
    // Check if bootstrap certificate is available
    hasBootstrap, err := lm.bootstrapHandler.HasValidBootstrapCertificate(ctx)
    if err != nil {
        lm.log.Warnf("Failed to check bootstrap certificate: %v", err)
    }

    if hasBootstrap {
        lm.log.Infof("Using bootstrap certificate for recovery of %q/%q", providerName, certName)
        // Use bootstrap certificate for authentication
        // Full recovery flow will be implemented in EDM-323-EPIC-4-STORY-5
    } else {
        lm.log.Warnf("Bootstrap certificate not available, will use TPM attestation for recovery")
        // Fall back to TPM attestation (next story)
    }

    return nil
}
```

**Testing:**
- Test bootstrap handler is used in recovery
- Test fallback to TPM when bootstrap unavailable

---

### Task 7: Add Error Handling and Logging

**File:** `internal/agent/device/bootstrap_cert.go` (modify)

**Objective:** Add comprehensive error handling and logging.

**Implementation Steps:**

1. **Add logging in GetBootstrapCertificate:**
```go
// In GetBootstrapCertificate, add logging:
b.log.Debugf("Loading bootstrap certificate from %s", b.certPath)
```

2. **Add logging in GetCertificateForAuth:**
```go
// In GetCertificateForAuth, add logging:
b.log.Infof("Falling back to bootstrap certificate for authentication")
b.log.Warnf("Management certificate is expired, using bootstrap certificate")
```

**Testing:**
- Test logging includes relevant information
- Test log levels are appropriate

---

## Unit Tests

### Test File: `internal/agent/device/bootstrap_cert_test.go` (new)

**Test Cases:**

1. **TestGetBootstrapCertificate:**
   - Loads certificate correctly
   - Handles missing certificate
   - Handles invalid certificate

2. **TestValidateBootstrapCertificate:**
   - Validates valid certificate
   - Rejects expired certificate
   - Rejects not-yet-valid certificate

3. **TestHasValidBootstrapCertificate:**
   - Returns true for valid certificate
   - Returns false for missing certificate
   - Returns false for expired certificate

4. **TestGetCertificateForAuth:**
   - Uses management cert when valid
   - Falls back to bootstrap when expired
   - Handles missing certificates

---

## Integration Tests

### Test File: `test/integration/bootstrap_certificate_fallback_test.go` (new)

**Test Cases:**

1. **TestBootstrapCertificateFallback:**
   - Agent uses bootstrap certificate when management cert expired
   - Agent can authenticate with bootstrap certificate
   - Agent can submit renewal requests

2. **TestBootstrapCertificateValidation:**
   - Valid bootstrap certificate is accepted
   - Expired bootstrap certificate is rejected
   - Missing bootstrap certificate triggers TPM fallback

---

## Code Review Checklist

- [ ] Bootstrap certificate loading works correctly
- [ ] Bootstrap certificate validation is accurate
- [ ] Certificate switching logic works
- [ ] Integration with management client works
- [ ] Error handling is comprehensive
- [ ] Logging is appropriate
- [ ] Unit tests achieve >80% coverage
- [ ] Integration tests cover main flows
- [ ] Code follows existing patterns

---

## Definition of Done

- [ ] BootstrapCertificateHandler created
- [ ] GetBootstrapCertificate implemented
- [ ] ValidateBootstrapCertificate implemented
- [ ] GetCertificateForAuth implemented
- [ ] Integration with management client added
- [ ] Integration with certificate manager added
- [ ] Error handling and logging added
- [ ] Unit tests written and passing (>80% coverage)
- [ ] Integration tests written and passing
- [ ] Code reviewed and approved
- [ ] Documentation updated

---

## Related Files

- `internal/agent/device/bootstrap_cert.go` - Bootstrap certificate handler
- `internal/agent/device/certmanager/lifecycle.go` - Lifecycle manager
- `internal/agent/client/management.go` - Management client
- `internal/agent/config/config.go` - Configuration

---

## Dependencies

- **EDM-323-EPIC-4-STORY-1**: Expired Certificate Detection (must be completed)
  - Requires expiration detection to trigger recovery

---

## Notes

- **Bootstrap Certificate**: The bootstrap certificate is the enrollment certificate (`client-enrollment.crt`). It's used during initial enrollment and can be used for recovery.

- **Certificate Switching**: The agent automatically switches between management and bootstrap certificates based on expiration status. This is transparent to the application.

- **Fallback Chain**: The fallback chain is: management certificate → bootstrap certificate → TPM attestation. Each fallback is tried if the previous one is unavailable or expired.

- **Certificate Validation**: Bootstrap certificates are validated before use to ensure they're not expired. Expired bootstrap certificates trigger TPM attestation fallback.

- **Authentication**: The bootstrap certificate is used for mTLS authentication to the management service, allowing the agent to submit renewal requests even when the management certificate is expired.

- **Error Handling**: If bootstrap certificate loading or validation fails, the error is logged and the agent falls back to TPM attestation (next story).

---

**Document End**

