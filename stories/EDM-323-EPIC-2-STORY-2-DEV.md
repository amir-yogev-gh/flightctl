# Developer Story: CSR Generation for Certificate Renewal

**Story ID:** EDM-323-EPIC-2-STORY-2  
**Epic:** EDM-323-EPIC-2 (Proactive Certificate Renewal)  
**Feature:** EDM-323 (Agent Certificate Rotation)  
**Story Points:** 5  
**Priority:** High

## Overview

Extend the CSR provisioner to support certificate renewal by adding renewal context to CSR metadata and ensuring the same device identity is used. The renewal CSR should be authenticated using the current valid certificate.

## Implementation Tasks

### Task 1: Add Renewal Context to CSR Provisioner

**File:** `internal/agent/device/certmanager/provider/provisioner/csr.go` (modify)

**Objective:** Add renewal context information to CSR provisioner configuration and requests.

**Implementation Steps:**

1. **Add renewal context to CSRProvisionerConfig:**
```go
// CSRProvisionerConfig defines configuration for Certificate Signing Request (CSR) based provisioning.
type CSRProvisionerConfig struct {
    // ... existing fields ...
    
    // RenewalContext contains information about certificate renewal (if this is a renewal request)
    RenewalContext *RenewalContext `json:"renewal-context,omitempty"`
}

// RenewalContext contains context information for certificate renewal requests.
type RenewalContext struct {
    // Reason indicates why the certificate is being renewed
    // Values: "proactive" (renewed before expiration), "expired" (renewed after expiration)
    Reason string `json:"reason"`
    
    // ThresholdDays is the number of days before expiration that triggered renewal
    ThresholdDays int `json:"threshold-days,omitempty"`
    
    // DaysUntilExpiration is the number of days until expiration when renewal was triggered
    DaysUntilExpiration int `json:"days-until-expiration,omitempty"`
}
```

2. **Add renewal context to CSRProvisioner:**
```go
type CSRProvisioner struct {
    // ... existing fields ...
    
    // Renewal context (if this is a renewal request)
    renewalContext *RenewalContext
}
```

3. **Update NewCSRProvisioner to accept renewal context:**
```go
// NewCSRProvisioner creates a new CSR provisioner with the specified configuration.
func NewCSRProvisioner(deviceName string, csrClient csrClient, identityProvider identity.ExportableProvider, cfg *CSRProvisionerConfig) (*CSRProvisioner, error) {
    return &CSRProvisioner{
        deviceName:       deviceName,
        csrClient:        csrClient,
        cfg:              cfg,
        identityProvider: identityProvider,
        renewalContext:   cfg.RenewalContext,
    }, nil
}
```

**Testing:**
- Test RenewalContext struct can be marshaled/unmarshaled
- Test renewal context is stored in provisioner
- Test nil renewal context is handled

---

### Task 2: Add Renewal Labels to CSR Request

**File:** `internal/agent/device/certmanager/provider/provisioner/csr.go` (modify)

**Objective:** Add renewal metadata/labels to the CSR request when it's a renewal.

**Implementation Steps:**

1. **Modify Provision method to add renewal labels:**
```go
// Provision attempts to provision a certificate through the CSR workflow.
func (p *CSRProvisioner) Provision(ctx context.Context) (bool, *x509.Certificate, []byte, error) {
    if p.csrName != "" {
        return p.check(ctx)
    }

    if p.cfg.CommonName == "" {
        return false, nil, nil, fmt.Errorf("commonName must be set")
    }

    // Generate unique CSR object name for Kubernetes resource
    p.csrName = fmt.Sprintf("%s-%s", p.cfg.CommonName, uuid.NewString()[:8])

    // Generate private key and CSR using the configured CommonName (without suffix)
    id, err := p.identityProvider.NewExportable(p.cfg.CommonName)
    if err != nil {
        return false, nil, nil, fmt.Errorf("new identity: %w", err)
    }
    csr, err := id.CSR()
    if err != nil {
        return false, nil, nil, fmt.Errorf("create CSR: %w", err)
    }

    p.identity = id

    usages := []string{
        "clientAuth",
        "CA:false",
    }

    if len(p.cfg.Usages) > 0 {
        usages = append(usages, p.cfg.Usages...)
    }

    // Build metadata with renewal labels if this is a renewal
    metadata := api.ObjectMeta{
        Name: &p.csrName,
    }
    
    if p.renewalContext != nil {
        labels := make(map[string]string)
        labels["flightctl.io/renewal-reason"] = p.renewalContext.Reason
        if p.renewalContext.ThresholdDays > 0 {
            labels["flightctl.io/renewal-threshold-days"] = fmt.Sprintf("%d", p.renewalContext.ThresholdDays)
        }
        if p.renewalContext.DaysUntilExpiration != 0 {
            labels["flightctl.io/renewal-days-until-expiration"] = fmt.Sprintf("%d", p.renewalContext.DaysUntilExpiration)
        }
        metadata.Labels = &labels
    }

    req := api.CertificateSigningRequest{
        ApiVersion: api.CertificateSigningRequestAPIVersion,
        Kind:       api.CertificateSigningRequestKind,
        Metadata:   metadata,
        Spec: api.CertificateSigningRequestSpec{
            ExpirationSeconds: p.cfg.ExpirationSeconds,
            Request:           csr,
            SignerName:        p.cfg.Signer,
            Usages:            &usages,
        },
    }
    
    _, statusCode, err := p.csrClient.CreateCertificateSigningRequest(ctx, req)
    if err != nil {
        return false, nil, nil, fmt.Errorf("create csr: %w", err)
    }

    switch statusCode {
    case http.StatusOK, http.StatusCreated:
        return false, nil, nil, nil
    default:
        return false, nil, nil, fmt.Errorf("%w: unexpected status code %d", errors.ErrCreateCertificateSigningRequest, statusCode)
    }
}
```

**Testing:**
- Test renewal labels are added when renewal context exists
- Test labels have correct values
- Test no labels are added when not a renewal
- Test label values are formatted correctly

---

### Task 3: Pass Renewal Context from Certificate Manager

**File:** `internal/agent/device/certmanager/manager.go` (modify)

**Objective:** Pass renewal context to the CSR provisioner when triggering renewal.

**Implementation Steps:**

1. **Add method to create renewal context:**
```go
// createRenewalContext creates a renewal context for certificate renewal.
func (cm *CertManager) createRenewalContext(ctx context.Context, providerName string, cert *certificate, thresholdDays int) (*provisioner.RenewalContext, error) {
    if cm.lifecycleManager == nil {
        return nil, fmt.Errorf("lifecycle manager not initialized")
    }
    
    // Get days until expiration
    _, days, err := cm.lifecycleManager.CheckRenewal(ctx, providerName, cert.Name, thresholdDays)
    if err != nil {
        return nil, fmt.Errorf("failed to get expiration info: %w", err)
    }
    
    // Determine renewal reason
    reason := "proactive"
    if days < 0 {
        reason = "expired"
    }
    
    return &provisioner.RenewalContext{
        Reason:              reason,
        ThresholdDays:       thresholdDays,
        DaysUntilExpiration: days,
    }, nil
}
```

2. **Modify provisionCertificate to accept renewal context:**
```go
// provisionCertificate queues a certificate for provisioning by adding it to the processing queue.
func (cm *CertManager) provisionCertificate(ctx context.Context, providerName string, cert *certificate, cfg provider.CertificateConfig, renewalContext *provisioner.RenewalContext) error {
    // If this is a renewal and the provisioner is CSR, add renewal context
    if renewalContext != nil && cfg.Provisioner.Type == provider.ProvisionerTypeCSR {
        // Decode CSR config and add renewal context
        var csrConfig provisioner.CSRProvisionerConfig
        if err := json.Unmarshal(cfg.Provisioner.Config, &csrConfig); err == nil {
            csrConfig.RenewalContext = renewalContext
            // Re-encode config
            if configBytes, err := json.Marshal(csrConfig); err == nil {
                cfg.Provisioner.Config = configBytes
            }
        }
    }
    
    return cm.processingQueue.Process(providerName, cert, cfg)
}
```

3. **Update triggerRenewal to create and pass renewal context:**
```go
// triggerRenewal initiates the certificate renewal process.
func (cm *CertManager) triggerRenewal(ctx context.Context, providerName string, cert *certificate, cfg provider.CertificateConfig) error {
    if cm.lifecycleManager == nil {
        return fmt.Errorf("lifecycle manager not initialized")
    }
    
    // Create renewal context
    thresholdDays := 30 // Default
    if cm.config != nil {
        thresholdDays = cm.config.Certificate.Renewal.ThresholdDays
        if thresholdDays == 0 {
            thresholdDays = 30
        }
    }
    
    renewalContext, err := cm.createRenewalContext(ctx, providerName, cert, thresholdDays)
    if err != nil {
        cm.log.Warnf("Failed to create renewal context for %q/%q: %v", providerName, cert.Name, err)
        // Continue without renewal context
        renewalContext = nil
    }
    
    // Set certificate state to "renewing"
    if err := cm.lifecycleManager.SetCertificateState(ctx, providerName, cert.Name, CertificateStateRenewing); err != nil {
        cm.log.Warnf("Failed to set certificate state to renewing for %q/%q: %v", providerName, cert.Name, err)
        // Continue anyway - state update failure shouldn't block renewal
    }
    
    // Queue certificate for renewal with renewal context
    if err := cm.provisionCertificate(ctx, providerName, cert, cfg, renewalContext); err != nil {
        // If queuing fails, reset state
        _ = cm.lifecycleManager.SetCertificateState(ctx, providerName, cert.Name, CertificateStateExpiringSoon)
        return fmt.Errorf("failed to queue certificate for renewal: %w", err)
    }
    
    cm.log.Infof("Triggered renewal for certificate %q/%q", providerName, cert.Name)
    return nil
}
```

4. **Update other calls to provisionCertificate:**
```go
// In syncCertificate, update calls to provisionCertificate:
if err := cm.provisionCertificate(ctx, providerName, cert, cfg, nil); err != nil {
    return fmt.Errorf("failed to provision certificate %q from provider %q: %w", cert.Name, providerName, err)
}

// In triggerRenewal, already updated above
```

**Testing:**
- Test renewal context is created correctly
- Test renewal context is passed to provisioner
- Test renewal context is added to CSR config
- Test non-renewal requests don't have renewal context
- Test error handling when creating renewal context

---

### Task 4: Ensure Same Device Identity for Renewal

**File:** `internal/agent/device/certmanager/provider/provisioner/csr.go` (modify)

**Objective:** Ensure renewal CSRs use the same device identity (CommonName) as the original certificate.

**Implementation Steps:**

1. **Verify CommonName is preserved:**
The CommonName is already set in the certificate config and used when creating the identity. For renewal, we need to ensure:
   - The same CommonName is used (already handled by config)
   - The same identity type is used (already handled by config)
   - For TPM identities, the same TPM key is used (handled by TPM provider)

2. **Add validation to ensure identity consistency:**
```go
// In CSRProvisioner.Provision, add validation:
// For renewal, verify we're using the same CommonName
if p.renewalContext != nil {
    // The CommonName should match the device identity
    // This is already ensured by using the same certificate config
    // Log for debugging
    // Note: We can't easily verify the key matches without storing the previous key
    // The TPM provider handles key reuse automatically for the same CommonName
}
```

3. **Document identity preservation:**
Add comments explaining that:
   - CommonName determines device identity
   - Same CommonName = same device identity
   - TPM provider reuses keys for same CommonName
   - Software provider creates new keys (which is acceptable for renewal)

**Testing:**
- Test CommonName is preserved in renewal CSR
- Test identity type is preserved
- Test TPM identities reuse keys (if applicable)
- Test software identities work correctly

---

### Task 5: Verify Authentication Uses Current Certificate

**File:** `internal/agent/device/certmanager/provider/provisioner/csr.go` (modify)

**Objective:** Ensure renewal CSR requests are authenticated using the current valid certificate.

**Implementation Steps:**

1. **Verify authentication is handled by management client:**
The management client (`csrClient`) is created with the identity provider, which uses the current certificate for mTLS authentication. This is already handled correctly.

2. **Add logging to confirm authentication:**
```go
// In Provision, when submitting renewal CSR:
if p.renewalContext != nil {
    // Log that we're using current certificate for authentication
    // The management client handles this automatically via mTLS
    // No code changes needed - authentication is transparent
}
```

3. **Document authentication flow:**
Add comments explaining:
   - Management client uses current certificate for mTLS
   - Authentication is handled transparently
   - No special handling needed for renewal

**Testing:**
- Test renewal CSR is authenticated correctly
- Test authentication uses current certificate
- Test authentication fails if certificate is invalid
- Test authentication works with TPM certificates

---

### Task 6: Update CSR Provisioner Factory for Renewal Context

**File:** `internal/agent/device/certmanager/provider/provisioner/csr.go` (modify)

**Objective:** Ensure CSR provisioner factory can handle renewal context in config.

**Implementation Steps:**

1. **Update New method to preserve renewal context:**
```go
// New creates a new CSRProvisioner based on the provided certificate config.
func (f *CSRProvisionerFactory) New(log provider.Logger, cc provider.CertificateConfig) (provider.ProvisionerProvider, error) {
    prov := cc.Provisioner

    var csrConfig CSRProvisionerConfig
    if err := json.Unmarshal(prov.Config, &csrConfig); err != nil {
        return nil, fmt.Errorf("failed to decode CSR provisioner config for certificate %q: %w", cc.Name, err)
    }

    // Renewal context is already in csrConfig if present
    // (set by certificate manager before calling this method)

    // ... rest of existing code ...
    
    return NewCSRProvisioner(f.deviceName, f.managementClient, identityProvider, &csrConfig)
}
```

**Testing:**
- Test factory preserves renewal context
- Test factory works without renewal context
- Test factory handles invalid renewal context

---

### Task 7: Add Logging for Renewal CSR Operations

**File:** `internal/agent/device/certmanager/provider/provisioner/csr.go` (modify)

**Objective:** Add appropriate logging for renewal CSR operations.

**Implementation Steps:**

1. **Add logging in Provision for renewal:**
```go
// In Provision, when creating renewal CSR:
if p.renewalContext != nil {
    // Log renewal CSR creation
    // Note: We don't have a logger in CSRProvisioner, so this would need to be added
    // Or log at the certificate manager level
}
```

2. **Add logging in certificate manager:**
```go
// In triggerRenewal, after creating renewal context:
if renewalContext != nil {
    cm.log.Infof("Creating renewal CSR for certificate %q/%q (reason: %s, days until expiration: %d)", 
        providerName, cert.Name, renewalContext.Reason, renewalContext.DaysUntilExpiration)
}
```

**Testing:**
- Test logging includes renewal information
- Test log levels are appropriate

---

## Unit Tests

### Test File: `internal/agent/device/certmanager/provider/provisioner/csr_renewal_test.go` (new)

**Test Cases:**

1. **TestRenewalContext:**
   - RenewalContext struct can be marshaled/unmarshaled
   - RenewalContext fields are correct
   - Nil renewal context is handled

2. **TestCSRProvisionerWithRenewal:**
   - Renewal context is stored in provisioner
   - Renewal labels are added to CSR request
   - Label values are correct
   - No labels added when not renewal

3. **TestRenewalLabels:**
   - Labels include renewal-reason
   - Labels include threshold-days when set
   - Labels include days-until-expiration
   - Labels are formatted correctly

4. **TestDeviceIdentityPreservation:**
   - CommonName is preserved in renewal
   - Identity type is preserved
   - Same device identity is used

5. **TestRenewalContextCreation:**
   - Renewal context created correctly
   - Reason is "proactive" for expiring certificates
   - Reason is "expired" for expired certificates
   - Days until expiration calculated correctly

6. **TestRenewalContextPassing:**
   - Renewal context passed to provisioner
   - Renewal context added to CSR config
   - Non-renewal requests don't have context

---

## Integration Tests

### Test File: `test/integration/certificate_renewal_csr_test.go` (new)

**Test Cases:**

1. **TestRenewalCSRGeneration:**
   - Renewal CSR is generated with correct labels
   - Renewal CSR uses same device identity
   - Renewal CSR is authenticated with current certificate

2. **TestRenewalCSRSubmission:**
   - Renewal CSR is submitted successfully
   - Renewal labels are present in submitted CSR
   - Server receives renewal context

3. **TestRenewalCSRWithTPM:**
   - TPM-backed renewal CSR works correctly
   - Same TPM key is used
   - Authentication works with TPM certificate

4. **TestRenewalCSRWithSoftwareKey:**
   - Software key renewal CSR works correctly
   - New key is generated (acceptable)
   - Authentication works with software certificate

---

## Code Review Checklist

- [ ] Renewal context is added to CSR config
- [ ] Renewal labels are added to CSR request
- [ ] Same device identity is used (CommonName preserved)
- [ ] Authentication uses current certificate (handled by client)
- [ ] Renewal context is passed from certificate manager
- [ ] Error handling is comprehensive
- [ ] Logging is appropriate
- [ ] Unit tests achieve >80% coverage
- [ ] Integration tests cover main flows
- [ ] Code follows existing patterns

---

## Definition of Done

- [ ] RenewalContext struct created
- [ ] Renewal context added to CSRProvisionerConfig
- [ ] Renewal labels added to CSR requests
- [ ] Renewal context passed from certificate manager
- [ ] Same device identity ensured
- [ ] Authentication verified (uses current certificate)
- [ ] Logging added for renewal operations
- [ ] Unit tests written and passing (>80% coverage)
- [ ] Integration tests written and passing
- [ ] Code reviewed and approved
- [ ] Documentation updated

---

## Related Files

- `internal/agent/device/certmanager/provider/provisioner/csr.go` - CSR provisioner
- `internal/agent/device/certmanager/manager.go` - Certificate manager
- `internal/agent/device/certmanager/lifecycle.go` - Lifecycle manager
- `internal/agent/client/management.go` - Management client
- `internal/agent/identity/` - Identity providers

---

## Dependencies

- **EDM-323-EPIC-2-STORY-1**: Agent-Side Certificate Renewal Trigger (must be completed)
  - Requires renewal trigger to be implemented
  
- **Existing CSR Provisioner**: Uses existing CSR provisioner infrastructure
  - No changes to core CSR workflow
  
- **Identity Providers**: Uses existing identity provider infrastructure
  - TPM and software identity providers

---

## Notes

- **Device Identity**: The device identity is determined by the CommonName, not the private key. Using the same CommonName ensures the same device identity, even if a new key is generated.

- **Key Reuse**: 
  - For TPM identities, the TPM provider typically reuses the same key for the same CommonName
  - For software identities, a new key may be generated, which is acceptable for renewal
  - Key rotation (new key) is a valid renewal strategy

- **Authentication**: The management client automatically uses the current certificate for mTLS authentication. No special handling is needed.

- **Renewal Labels**: Labels are added to CSR metadata to help the server identify and process renewal requests appropriately.

- **Renewal Reason**: 
  - "proactive" = renewed before expiration (normal renewal)
  - "expired" = renewed after expiration (recovery scenario)

- **Backward Compatibility**: Renewal context is optional. CSRs without renewal context work as before (initial provisioning).

---

**Document End**

