# Flight Control - Agent Certificate Rotation: Brownfield Architecture

> **Document Type:** Brownfield Architecture  
> **Feature:** Agent Certificate Rotation  
> **Jira Reference:** [EDM-323](https://issues.redhat.com/browse/EDM-323)  
> **Version:** 1.0  
> **Last Updated:** November 29, 2025  
> **Status:** Design

## Table of Contents

- [Executive Summary](#executive-summary)
- [Current State Architecture](#current-state-architecture)
- [Target State Architecture](#target-state-architecture)
- [Integration Points](#integration-points)
- [Component Changes](#component-changes)
- [Data Flow Diagrams](#data-flow-diagrams)
- [Database Schema Changes](#database-schema-changes)
- [API Changes](#api-changes)
- [Configuration Changes](#configuration-changes)
- [Security Considerations](#security-considerations)
- [Migration Path](#migration-path)
- [Testing Strategy](#testing-strategy)

---

## Executive Summary

This document describes the brownfield architecture for implementing automatic certificate rotation in Flight Control. The feature integrates into the existing certificate management infrastructure, extending the current CSR-based provisioning system to support automatic renewal and expired certificate recovery.

### Key Integration Points

1. **Agent Certificate Manager** - Extends existing `certmanager` package with lifecycle management
2. **CSR Provisioner** - Reuses existing CSR workflow for renewal requests
3. **Service Certificate Handler** - Extends existing CSR approval logic for renewal scenarios
4. **TPM Identity Provider** - Leverages existing TPM attestation for recovery authentication
5. **Bootstrap Certificate** - Utilizes existing enrollment certificate as fallback

### Architectural Principles

- **Minimal Disruption**: Builds on existing certificate infrastructure
- **Backward Compatible**: Existing enrollment and certificate workflows remain unchanged
- **Incremental Enhancement**: New functionality added alongside existing code paths
- **Reuse Over Rebuild**: Leverages existing CSR, TPM, and storage mechanisms

---

## Current State Architecture

### Certificate Lifecycle (Before Feature)

```
┌─────────────────────────────────────────────────────────────┐
│                    Current Certificate Flow                 │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│  1. Enrollment Phase                                        │
│     ┌──────────────┐                                        │
│     │ Device Boot  │                                        │
│     └──────┬───────┘                                        │
│            │                                                 │
│            ▼                                                 │
│     ┌──────────────┐                                        │
│     │ Generate TPM │                                        │
│     │   Identity   │                                        │
│     └──────┬───────┘                                        │
│            │                                                 │
│            ▼                                                 │
│     ┌──────────────┐                                        │
│     │ Create CSR   │                                        │
│     │ (Enrollment) │                                        │
│     └──────┬───────┘                                        │
│            │                                                 │
│            ▼                                                 │
│     ┌──────────────┐                                        │
│     │ Submit ER    │                                        │
│     └──────┬───────┘                                        │
│            │                                                 │
│            ▼                                                 │
│     ┌──────────────┐                                        │
│     │ Admin        │                                        │
│     │ Approves     │                                        │
│     └──────┬───────┘                                        │
│            │                                                 │
│            ▼                                                 │
│     ┌──────────────┐                                        │
│     │ Service      │                                        │
│     │ Issues Cert  │                                        │
│     └──────┬───────┘                                        │
│            │                                                 │
│            ▼                                                 │
│     ┌──────────────┐                                        │
│     │ Device       │                                        │
│     │ Stores Cert  │                                        │
│     └──────┬───────┘                                        │
│            │                                                 │
│            ▼                                                 │
│  2. Operational Phase                                       │
│     ┌──────────────┐                                        │
│     │ Device Uses  │                                        │
│     │ Cert for mTLS│                                        │
│     └──────┬───────┘                                        │
│            │                                                 │
│            ▼                                                 │
│     ┌──────────────┐                                        │
│     │ Cert Valid   │                                        │
│     │ for 365 days │                                        │
│     └──────┬───────┘                                        │
│            │                                                 │
│            ▼                                                 │
│     ┌──────────────┐                                        │
│     │ Cert Expires │                                        │
│     └──────┬───────┘                                        │
│            │                                                 │
│            ▼                                                 │
│     ┌──────────────┐                                        │
│     │ Device       │                                        │
│     │ Loses Access │                                        │
│     └──────────────┘                                        │
│                                                              │
│  ❌ No automatic renewal                                    │
│  ❌ No expiration monitoring                                │
│  ❌ No recovery mechanism                                    │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

### Existing Components

#### Agent Certificate Manager (`internal/agent/device/certmanager`)

**Current Responsibilities:**
- Certificate provisioning via CSR
- Certificate storage management
- Certificate configuration from multiple providers
- Processing queue for async certificate operations

**Current Structure:**
```go
type CertManager struct {
    certificates *certStorage
    configs map[string]provider.ConfigProvider
    provisioners map[string]provider.ProvisionerFactory
    storages map[string]provider.StorageFactory
    processingQueue *CertificateProcessingQueue
}
```

**Current Limitations:**
- No expiration monitoring
- No automatic renewal triggers
- No expired certificate detection
- No atomic swap mechanism

#### CSR Provisioner (`internal/agent/device/certmanager/provider/provisioner/csr.go`)

**Current Capabilities:**
- Generate CSR with TPM-backed identity
- Submit CSR to service
- Poll for CSR approval
- Retrieve signed certificate

**Current Flow:**
1. Generate identity and CSR
2. Submit CSR to `/api/v1/agent/certificatesigningrequests`
3. Poll CSR status
4. Retrieve certificate when approved

#### Service CSR Handler (`internal/service/certificatesigningrequest.go`)

**Current Capabilities:**
- Accept CSR submissions
- Validate CSR signatures
- Verify TPM attestation (for enrollment)
- Auto-approve bootstrap CSRs
- Sign approved certificates

**Current Limitations:**
- No renewal-specific logic
- No expired certificate handling
- No renewal request validation

#### Certificate Storage (`internal/agent/device/certmanager/provider/storage/fs.go`)

**Current Structure:**
- Single certificate file per certificate
- No pending certificate support
- No atomic swap mechanism

---

## Target State Architecture

### Enhanced Certificate Lifecycle (After Feature)

```
┌─────────────────────────────────────────────────────────────┐
│              Enhanced Certificate Flow (With Rotation)      │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│  1. Enrollment Phase (Unchanged)                           │
│     [Same as current state]                                 │
│                                                              │
│  2. Operational Phase (Enhanced)                           │
│     ┌──────────────┐                                        │
│     │ Device Uses  │                                        │
│     │ Cert for mTLS│                                        │
│     └──────┬───────┘                                        │
│            │                                                 │
│            ▼                                                 │
│     ┌──────────────────────┐                                │
│     │ Certificate          │                                │
│     │ Lifecycle Manager    │                                │
│     │ (NEW)                │                                │
│     ├──────────────────────┤                                │
│     │ • Expiration Monitor │                                │
│     │ • Renewal Trigger    │                                │
│     │ • Recovery Handler   │                                │
│     └──────┬───────────────┘                                │
│            │                                                 │
│            ├─────────────────┐                               │
│            │                 │                               │
│            ▼                 ▼                               │
│     ┌──────────────┐  ┌──────────────┐                     │
│     │ Cert Valid   │  │ Cert Expiring│                     │
│     │ > 30 days    │  │ < 30 days    │                     │
│     └──────┬───────┘  └──────┬───────┘                     │
│            │                 │                               │
│            │                 ▼                               │
│            │         ┌──────────────┐                        │
│            │         │ Trigger      │                        │
│            │         │ Renewal      │                        │
│            │         └──────┬───────┘                        │
│            │                │                                 │
│            │                ▼                                 │
│            │         ┌──────────────┐                        │
│            │         │ Generate CSR │                        │
│            │         │ (Renewal)    │                        │
│            │         └──────┬───────┘                        │
│            │                │                                 │
│            │                ▼                                 │
│            │         ┌──────────────┐                        │
│            │         │ Submit to    │                        │
│            │         │ Service      │                        │
│            │         └──────┬───────┘                        │
│            │                │                                 │
│            │                ▼                                 │
│            │         ┌──────────────┐                        │
│            │         │ Receive New  │                        │
│            │         │ Certificate  │                        │
│            │         └──────┬───────┘                        │
│            │                │                                 │
│            │                ▼                                 │
│            │         ┌──────────────┐                        │
│            │         │ Atomic Swap  │                        │
│            │         │ (NEW)        │                        │
│            │         └──────┬───────┘                        │
│            │                │                                 │
│            └────────────────┘                               │
│                     │                                        │
│                     ▼                                        │
│            ┌──────────────┐                                 │
│            │ Continue with │                                 │
│            │ New Cert      │                                 │
│            └──────────────┘                                 │
│                                                              │
│  3. Recovery Phase (NEW)                                    │
│     ┌──────────────┐                                        │
│     │ Device       │                                        │
│     │ Offline      │                                        │
│     └──────┬───────┘                                        │
│            │                                                 │
│            ▼                                                 │
│     ┌──────────────┐                                        │
│     │ Cert Expires │                                        │
│     └──────┬───────┘                                        │
│            │                                                 │
│            ▼                                                 │
│     ┌──────────────┐                                        │
│     │ Device       │                                        │
│     │ Comes Online │                                        │
│     └──────┬───────┘                                        │
│            │                                                 │
│            ▼                                                 │
│     ┌──────────────┐                                        │
│     │ Detect       │                                        │
│     │ Expired Cert │                                        │
│     └──────┬───────┘                                        │
│            │                                                 │
│            ▼                                                 │
│     ┌──────────────┐                                        │
│     │ Fallback to  │                                        │
│     │ Bootstrap    │                                        │
│     │ Certificate  │                                        │
│     └──────┬───────┘                                        │
│            │                                                 │
│            ▼                                                 │
│     ┌──────────────┐                                        │
│     │ Submit Renewal│                                       │
│     │ with TPM      │                                        │
│     │ Attestation   │                                        │
│     └──────┬───────┘                                        │
│            │                                                 │
│            ▼                                                 │
│     ┌──────────────┐                                        │
│     │ Service      │                                        │
│     │ Validates    │                                        │
│     │ & Issues     │                                        │
│     └──────┬───────┘                                        │
│            │                                                 │
│            ▼                                                 │
│     ┌──────────────┐                                        │
│     │ Atomic Swap  │                                        │
│     │ & Resume     │                                        │
│     └──────────────┘                                        │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

### New Components

#### Certificate Lifecycle Manager (NEW)

**Location:** `internal/agent/device/certmanager/lifecycle.go`

**Responsibilities:**
- Monitor certificate expiration
- Trigger renewal at threshold
- Detect expired certificates
- Coordinate recovery flow
- Manage atomic swaps

**Interface:**
```go
type CertificateLifecycleManager interface {
    // Check if certificate needs renewal
    CheckRenewal(ctx context.Context, cert *certificate) (bool, time.Duration, error)
    
    // Perform certificate renewal
    RenewCertificate(ctx context.Context, cert *certificate) error
    
    // Detect and recover from expired certificate
    RecoverExpiredCertificate(ctx context.Context, cert *certificate) error
    
    // Atomically swap certificates
    SwapCertificate(ctx context.Context, cert *certificate, newCertPEM, newKeyPEM []byte) error
}
```

#### Expiration Monitor (NEW)

**Location:** `internal/agent/device/certmanager/expiration.go`

**Responsibilities:**
- Parse certificate expiration dates
- Calculate days until expiration
- Compare against renewal threshold
- Trigger renewal actions

#### Atomic Swap Coordinator (NEW)

**Location:** `internal/agent/device/certmanager/swap.go`

**Responsibilities:**
- Write new certificate to pending location
- Validate new certificate
- Perform atomic file operations
- Rollback on failure

---

## Integration Points

### 1. Agent Certificate Manager Integration

**Existing Component:** `internal/agent/device/certmanager/manager.go`

**Changes Required:**
- Add lifecycle manager to `CertManager` struct
- Integrate expiration monitoring into sync loop
- Add renewal trigger logic
- Extend certificate storage to support pending certificates

**Modified Structure:**
```go
type CertManager struct {
    // ... existing fields ...
    lifecycleManager *CertificateLifecycleManager  // NEW
    expirationMonitor *ExpirationMonitor          // NEW
    swapCoordinator *AtomicSwapCoordinator        // NEW
}
```

**Integration Points:**
```go
// In CertManager.Sync()
func (cm *CertManager) Sync(ctx context.Context, config *config.Config) error {
    // ... existing sync logic ...
    
    // NEW: Check for renewal needs
    for providerName, provider := range cm.configs {
        certificates := cm.certificates.GetCertificates(providerName)
        for _, cert := range certificates {
            // Check if renewal needed
            needsRenewal, daysUntilExpiry, err := cm.lifecycleManager.CheckRenewal(ctx, cert)
            if err != nil {
                cm.log.Errorf("Failed to check renewal for %s: %v", cert.Name, err)
                continue
            }
            
            if needsRenewal {
                cm.log.Infof("Certificate %s needs renewal (expires in %d days)", 
                    cert.Name, int(daysUntilExpiry.Hours()/24))
                if err := cm.lifecycleManager.RenewCertificate(ctx, cert); err != nil {
                    cm.log.Errorf("Failed to renew certificate %s: %v", cert.Name, err)
                }
            }
        }
    }
    
    return nil
}
```

### 2. CSR Provisioner Integration

**Existing Component:** `internal/agent/device/certmanager/provider/provisioner/csr.go`

**Changes Required:**
- Extend CSR submission to support renewal context
- Add renewal reason to CSR metadata
- Support expired certificate authentication fallback

**Enhanced CSR Submission:**
```go
// In CSRProvisioner.Provision()
func (p *CSRProvisioner) Provision(ctx context.Context, reason string) (bool, *x509.Certificate, []byte, error) {
    // ... existing CSR generation ...
    
    // NEW: Add renewal context to CSR
    req := api.CertificateSigningRequest{
        // ... existing fields ...
        Metadata: api.ObjectMeta{
            Name: &p.csrName,
            Labels: map[string]string{
                "flightctl.io/renewal-reason": reason,  // NEW
            },
        },
    }
    
    // ... rest of existing logic ...
}
```

### 3. Service CSR Handler Integration

**Existing Component:** `internal/service/certificatesigningrequest.go`

**Changes Required:**
- Add renewal request validation
- Support expired certificate authentication
- Auto-approve renewal requests from valid devices
- Track renewal events

**Enhanced Validation:**
```go
// NEW: Validate renewal request
func (h *ServiceHandler) validateRenewalRequest(ctx context.Context, orgId uuid.UUID, csr *api.CertificateSigningRequest) error {
    // Check if this is a renewal request
    reason := csr.Metadata.Labels["flightctl.io/renewal-reason"]
    if reason == "" {
        return nil // Not a renewal request, use normal validation
    }
    
    // Extract device name from CSR CN
    deviceName := extractDeviceNameFromCN(csr)
    
    // Verify device exists and was enrolled
    device, err := h.store.Device().Get(ctx, orgId, deviceName)
    if err != nil {
        return fmt.Errorf("device not found: %w", err)
    }
    
    // For expired certificates, validate bootstrap cert or TPM attestation
    if reason == "expired" {
        return h.validateExpiredCertificateRenewal(ctx, orgId, device, csr)
    }
    
    // For proactive renewals, validate current certificate
    if reason == "proactive" {
        return h.validateProactiveRenewal(ctx, orgId, device, csr)
    }
    
    return nil
}
```

### 4. Certificate Storage Integration

**Existing Component:** `internal/agent/device/certmanager/provider/storage/fs.go`

**Changes Required:**
- Support pending certificate storage
- Implement atomic swap operations
- Add rollback capability

**Enhanced Storage:**
```go
// NEW: Write pending certificate
func (s *FSStorage) WritePending(certPEM, keyPEM []byte) error {
    pendingCertPath := s.certPath + ".pending"
    pendingKeyPath := s.keyPath + ".pending"
    
    // Write to pending locations
    if err := os.WriteFile(pendingCertPath, certPEM, 0644); err != nil {
        return err
    }
    if err := os.WriteFile(pendingKeyPath, keyPEM, 0600); err != nil {
        return err
    }
    
    return nil
}

// NEW: Atomic swap
func (s *FSStorage) AtomicSwap() error {
    // Validate pending certificate first
    if err := s.validatePending(); err != nil {
        return fmt.Errorf("pending certificate validation failed: %w", err)
    }
    
    // Backup current certificate
    if err := s.backupCurrent(); err != nil {
        return err
    }
    
    // Atomic swap using rename (POSIX atomic operation)
    if err := os.Rename(s.certPath+".pending", s.certPath); err != nil {
        return err
    }
    if err := os.Rename(s.keyPath+".pending", s.keyPath); err != nil {
        // Rollback
        s.restoreBackup()
        return err
    }
    
    // Clean up backup
    s.cleanupBackup()
    
    return nil
}
```

### 5. TPM Identity Provider Integration

**Existing Component:** `internal/agent/identity/tpm.go`

**Changes Required:**
- Expose TPM attestation for renewal requests
- Support device fingerprint generation
- Enable attestation for expired certificate recovery

**Enhanced TPM Provider:**
```go
// NEW: Generate attestation for renewal
func (p *TPMProvider) GenerateRenewalAttestation(ctx context.Context) (*RenewalAttestation, error) {
    // Generate TPM quote
    quote, err := p.tpmClient.Quote(ctx)
    if err != nil {
        return nil, err
    }
    
    // Get PCR values
    pcrs, err := p.tpmClient.ReadPCRs(ctx)
    if err != nil {
        return nil, err
    }
    
    // Get device fingerprint
    fingerprint, err := p.GetDeviceFingerprint()
    if err != nil {
        return nil, err
    }
    
    return &RenewalAttestation{
        TPMQuote: quote,
        PCRValues: pcrs,
        DeviceFingerprint: fingerprint,
    }, nil
}
```

### 6. Bootstrap Certificate Integration

**Existing Component:** `internal/agent/device/bootstrap.go`

**Changes Required:**
- Expose bootstrap certificate for fallback authentication
- Support switching between management and bootstrap certificates
- Enable bootstrap certificate validation

**Enhanced Bootstrap:**
```go
// NEW: Get bootstrap certificate for fallback
func (b *Bootstrap) GetBootstrapCertificate() (*tls.Certificate, error) {
    bootstrapCertPath := b.config.BootstrapCertPath
    bootstrapKeyPath := b.config.BootstrapKeyPath
    
    cert, err := tls.LoadX509KeyPair(bootstrapCertPath, bootstrapKeyPath)
    if err != nil {
        return nil, fmt.Errorf("failed to load bootstrap certificate: %w", err)
    }
    
    // Validate certificate is not expired
    x509Cert, err := x509.ParseCertificate(cert.Certificate[0])
    if err != nil {
        return nil, err
    }
    
    if time.Now().After(x509Cert.NotAfter) {
        return nil, fmt.Errorf("bootstrap certificate expired")
    }
    
    return &cert, nil
}
```

---

## Component Changes

### Agent-Side Changes

#### 1. Certificate Manager (`internal/agent/device/certmanager/manager.go`)

**Additions:**
- Lifecycle manager integration
- Expiration monitoring loop
- Renewal trigger logic
- Recovery detection

**Modifications:**
- Extend `Sync()` method to check expiration
- Add periodic expiration check goroutine
- Integrate atomic swap into certificate provisioning

#### 2. Certificate Storage (`internal/agent/device/certmanager/provider/storage/fs.go`)

**Additions:**
- Pending certificate support
- Atomic swap operations
- Rollback mechanism
- Certificate validation

**Modifications:**
- Extend `Write()` to support pending mode
- Add `AtomicSwap()` method
- Add `ValidatePending()` method

#### 3. CSR Provisioner (`internal/agent/device/certmanager/provider/provisioner/csr.go`)

**Additions:**
- Renewal context support
- Expired certificate fallback
- Renewal reason tracking

**Modifications:**
- Extend `Provision()` to accept renewal reason
- Add fallback authentication logic
- Support bootstrap certificate for expired certs

#### 4. Agent Main Loop (`internal/agent/agent.go`)

**Additions:**
- Lifecycle manager initialization
- Expiration monitoring goroutine
- Renewal trigger integration

**Modifications:**
- Initialize lifecycle manager in agent setup
- Add periodic expiration check
- Integrate renewal into main loop

### Service-Side Changes

#### 1. CSR Service Handler (`internal/service/certificatesigningrequest.go`)

**Additions:**
- Renewal request validation
- Expired certificate handling
- Auto-approval for renewal requests
- Renewal event tracking

**Modifications:**
- Extend `CreateCertificateSigningRequest()` to handle renewals
- Add renewal validation logic
- Support bootstrap certificate authentication

#### 2. Certificate Signer (`internal/crypto/signer/`)

**Additions:**
- Renewal-specific signing logic
- Device identity validation for renewals

**Modifications:**
- Extend signers to support renewal context
- Validate device identity for renewal requests

#### 3. Store Layer (`internal/store/`)

**Additions:**
- Certificate renewal event tracking
- Device certificate metadata updates

**Modifications:**
- Add renewal event recording
- Update device certificate expiration tracking

---

## Data Flow Diagrams

### Proactive Renewal Flow

```
┌─────────────┐
│   Agent     │
│ CertManager │
└──────┬──────┘
       │
       │ 1. Periodic Check (daily)
       ▼
┌─────────────────────┐
│ Expiration Monitor  │
│ • Parse cert        │
│ • Calculate days    │
│ • Compare threshold │
└──────┬──────────────┘
       │
       │ 2. Needs Renewal? (30 days)
       ▼
┌─────────────────────┐
│ Lifecycle Manager   │
│ • Trigger Renewal   │
└──────┬──────────────┘
       │
       │ 3. Generate CSR
       ▼
┌─────────────────────┐
│ CSR Provisioner     │
│ • Create identity   │
│ • Generate CSR      │
│ • Add renewal label │
└──────┬──────────────┘
       │
       │ 4. Submit CSR
       ▼
┌─────────────┐
│   Service   │
│  API Server │
└──────┬──────┘
       │
       │ 5. Validate Renewal
       ▼
┌─────────────────────┐
│ CSR Handler         │
│ • Check device      │
│ • Validate cert     │
│ • Auto-approve      │
└──────┬──────────────┘
       │
       │ 6. Sign Certificate
       ▼
┌─────────────────────┐
│ Certificate Signer  │
│ • Issue new cert    │
│ • 365 day validity  │
└──────┬──────────────┘
       │
       │ 7. Return Certificate
       ▼
┌─────────────────────┐
│ CSR Provisioner     │
│ • Poll for cert     │
│ • Retrieve cert     │
└──────┬──────────────┘
       │
       │ 8. Receive Certificate
       ▼
┌─────────────────────┐
│ Atomic Swap         │
│ • Write pending     │
│ • Validate          │
│ • Atomic swap       │
└──────┬──────────────┘
       │
       │ 9. Complete
       ▼
┌─────────────┐
│   Agent     │
│ (New Cert)  │
└─────────────┘
```

### Expired Certificate Recovery Flow

```
┌─────────────┐
│   Agent     │
│ (Offline)   │
└──────┬──────┘
       │
       │ 1. Certificate Expires
       ▼
┌─────────────┐
│   Agent     │
│ (Comes      │
│  Online)    │
└──────┬──────┘
       │
       │ 2. Detect Expired Cert
       ▼
┌─────────────────────┐
│ Lifecycle Manager   │
│ • Check cert expiry │
│ • Detect expired    │
└──────┬──────────────┘
       │
       │ 3. Fallback to Bootstrap
       ▼
┌─────────────────────┐
│ Bootstrap Handler   │
│ • Load bootstrap    │
│ • Validate not exp. │
│ • Use for auth      │
└──────┬──────────────┘
       │
       │ 4. Generate Renewal CSR
       ▼
┌─────────────────────┐
│ TPM Provider        │
│ • Generate quote    │
│ • Get PCR values    │
│ • Get fingerprint   │
└──────┬──────────────┘
       │
       │ 5. Submit with Attestation
       ▼
┌─────────────┐
│   Service   │
│  API Server │
│ (Bootstrap  │
│  Auth)      │
└──────┬──────┘
       │
       │ 6. Validate Recovery Request
       ▼
┌─────────────────────┐
│ Security Validator  │
│ • Verify device     │
│ • Validate TPM      │
│ • Check fingerprint │
└──────┬──────────────┘
       │
       │ 7. Approve & Sign
       ▼
┌─────────────────────┐
│ Certificate Signer  │
│ • Issue new cert    │
└──────┬──────────────┘
       │
       │ 8. Return Certificate
       ▼
┌─────────────────────┐
│ Atomic Swap         │
│ • Write pending     │
│ • Validate          │
│ • Atomic swap       │
└──────┬──────────────┘
       │
       │ 9. Resume Operations
       ▼
┌─────────────┐
│   Agent     │
│ (Recovered) │
└─────────────┘
```

---

## Database Schema Changes

### Device Table Extensions

```sql
-- Add certificate tracking fields to devices table
ALTER TABLE devices 
ADD COLUMN certificate_expiration TIMESTAMP,
ADD COLUMN certificate_last_renewed TIMESTAMP,
ADD COLUMN certificate_renewal_count INTEGER DEFAULT 0,
ADD COLUMN certificate_fingerprint TEXT;

-- Add index for expiration monitoring
CREATE INDEX idx_devices_cert_expiration 
ON devices(certificate_expiration) 
WHERE certificate_expiration IS NOT NULL;

-- Add index for renewal tracking
CREATE INDEX idx_devices_cert_last_renewed 
ON devices(certificate_last_renewed) 
WHERE certificate_last_renewed IS NOT NULL;
```

### Certificate Renewal Events Table

```sql
-- Create table for tracking renewal events
CREATE TABLE certificate_renewal_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    device_id UUID NOT NULL REFERENCES devices(id) ON DELETE CASCADE,
    org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    event_type TEXT NOT NULL CHECK (event_type IN (
        'renewal_start',
        'renewal_success',
        'renewal_failed',
        'recovery_start',
        'recovery_success',
        'recovery_failed'
    )),
    reason TEXT CHECK (reason IN ('proactive', 'expired')),
    old_cert_expiration TIMESTAMP,
    new_cert_expiration TIMESTAMP,
    error_message TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- Indexes for efficient querying
CREATE INDEX idx_cert_renewal_events_device 
ON certificate_renewal_events(device_id);

CREATE INDEX idx_cert_renewal_events_org 
ON certificate_renewal_events(org_id);

CREATE INDEX idx_cert_renewal_events_created 
ON certificate_renewal_events(created_at);

CREATE INDEX idx_cert_renewal_events_type 
ON certificate_renewal_events(event_type);
```

### Migration Script

```sql
-- Migration: Add certificate rotation support
-- Version: 1.1.0
-- Date: 2025-11-29

BEGIN;

-- Add device certificate tracking
ALTER TABLE devices 
ADD COLUMN IF NOT EXISTS certificate_expiration TIMESTAMP,
ADD COLUMN IF NOT EXISTS certificate_last_renewed TIMESTAMP,
ADD COLUMN IF NOT EXISTS certificate_renewal_count INTEGER DEFAULT 0,
ADD COLUMN IF NOT EXISTS certificate_fingerprint TEXT;

-- Create renewal events table
CREATE TABLE IF NOT EXISTS certificate_renewal_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    device_id UUID NOT NULL REFERENCES devices(id) ON DELETE CASCADE,
    org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    event_type TEXT NOT NULL,
    reason TEXT,
    old_cert_expiration TIMESTAMP,
    new_cert_expiration TIMESTAMP,
    error_message TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- Create indexes
CREATE INDEX IF NOT EXISTS idx_devices_cert_expiration 
ON devices(certificate_expiration) 
WHERE certificate_expiration IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_cert_renewal_events_device 
ON certificate_renewal_events(device_id);

CREATE INDEX IF NOT EXISTS idx_cert_renewal_events_org 
ON certificate_renewal_events(org_id);

CREATE INDEX IF NOT EXISTS idx_cert_renewal_events_created 
ON certificate_renewal_events(created_at);

COMMIT;
```

---

## API Changes

### New Endpoint (Optional Enhancement)

While the feature can work with existing CSR endpoints, an optional dedicated renewal endpoint provides better semantics:

```yaml
# OpenAPI spec addition
/api/v1/agent/devices/{name}/certificaterenewal:
  post:
    summary: Request certificate renewal
    operationId: requestCertificateRenewal
    parameters:
      - name: name
        in: path
        required: true
        schema:
          type: string
    requestBody:
      content:
        application/json:
          schema:
            $ref: '#/components/schemas/CertificateRenewalRequest'
    responses:
      '200':
        description: Certificate renewal successful
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/CertificateRenewalResponse'
      '400':
        description: Invalid renewal request
      '401':
        description: Authentication failed
      '403':
        description: Renewal not authorized
```

### Enhanced CSR Metadata

CSR resources will include renewal context in labels:

```json
{
  "metadata": {
    "labels": {
      "flightctl.io/renewal-reason": "proactive",
      "flightctl.io/renewal-threshold-days": "30"
    }
  }
}
```

### Device Status Extensions

Device status will include certificate state:

```yaml
status:
  certificate:
    expiration: "2026-11-29T00:00:00Z"
    daysUntilExpiration: 45
    state: "normal"  # normal, expiring_soon, renewing, expired, recovering, renewal_failed
    lastRenewed: "2025-11-29T10:30:00Z"
    renewalCount: 1
```

---

## Configuration Changes

### Agent Configuration

```yaml
# /etc/flightctl/agent-config.yaml
certificate:
  # Renewal settings
  renewal:
    enabled: true
    threshold_days: 30          # Trigger renewal N days before expiry
    check_interval: 24h        # How often to check expiration
    retry_interval: 1h          # Retry interval on failure
    max_retries: 10             # Max retry attempts
    backoff_multiplier: 2.0     # Exponential backoff multiplier
    max_backoff: 24h           # Maximum backoff duration
  
  # Recovery settings
  recovery:
    enabled: true
    use_bootstrap_fallback: true
    use_tpm_attestation: true
  
  # Storage paths
  management:
    cert_path: /var/lib/flightctl/certs/management/cert.pem
    key_path: /var/lib/flightctl/certs/management/key.pem
    pending_cert_path: /var/lib/flightctl/certs/management/cert.pem.pending
    pending_key_path: /var/lib/flightctl/certs/management/key.pem.pending
  
  bootstrap:
    cert_path: /var/lib/flightctl/certs/bootstrap/cert.pem
    key_path: /var/lib/flightctl/certs/bootstrap/key.pem
```

### Service Configuration

```yaml
# Service configuration (no changes required, but optional enhancements)
certificate:
  renewal:
    auto_approve_renewals: true      # Auto-approve renewal CSRs
    require_tpm_attestation: true    # Require TPM attestation for expired certs
    max_renewal_attempts: 10        # Max renewal attempts per device
```

---

## Security Considerations

### Authentication Fallback Chain

The system implements a secure fallback chain for expired certificate recovery:

1. **Management Certificate** (Primary)
   - Used for normal operations
   - Valid for 365 days
   - TPM-backed private key

2. **Bootstrap Certificate** (Fallback)
   - Used when management cert expired
   - Longer validity (2+ years)
   - TPM-backed private key
   - Only used for renewal requests

3. **TPM Attestation** (Final Fallback)
   - Hardware-backed proof of device identity
   - Includes PCR values and device fingerprint
   - Validates device hasn't been tampered with

### Security Validations

**Proactive Renewal:**
- Device must have valid management certificate
- Certificate must match device identity
- Device must be enrolled and active

**Expired Certificate Recovery:**
- Device must exist in database
- Device must have been previously enrolled
- Bootstrap certificate must be valid (if used)
- TPM attestation must match device record
- Device fingerprint must match stored value
- Device must not be revoked/blacklisted

### Threat Mitigation

**Certificate Theft:**
- Private keys never leave TPM
- TPM attestation required for recovery
- Device fingerprint validation

**Replay Attacks:**
- CSR nonces prevent replay
- Timestamp validation
- One-time use CSRs

**Device Impersonation:**
- TPM attestation proves hardware identity
- Device fingerprint unique per device
- Certificate CN must match device name

---

## Migration Path

### Phase 1: Database Migration

1. Run database migration to add certificate tracking fields
2. Backfill certificate expiration dates from existing certificates
3. Create renewal events table

### Phase 2: Agent Deployment

1. Deploy agent with certificate lifecycle manager
2. Agents automatically start monitoring expiration
3. No immediate action required (certificates still valid)

### Phase 3: Service Deployment

1. Deploy service with renewal validation logic
2. Service starts accepting renewal requests
3. Existing CSR workflow continues to work

### Phase 4: Gradual Rollout

1. Monitor renewal events
2. Validate atomic swap operations
3. Verify recovery flows
4. Scale to full fleet

### Backward Compatibility

- Existing enrollment flow unchanged
- Existing CSR workflow continues to work
- Devices without new agent code continue to function
- Service supports both old and new certificate requests

---

## Testing Strategy

### Unit Tests

**Agent-Side:**
- Expiration calculation
- Threshold trigger logic
- Atomic swap operations
- Rollback mechanism
- TPM attestation generation

**Service-Side:**
- Renewal request validation
- TPM attestation verification
- Device fingerprint validation
- Auto-approval logic

### Integration Tests

**Full Renewal Flow:**
- Agent → Service → Agent renewal cycle
- Expired certificate recovery
- Bootstrap certificate fallback
- Atomic swap under failures
- Retry logic with backoff

### End-to-End Tests

**Scenarios:**
- Device enrollment → automatic renewal
- Device offline → certificate expires → recovery
- Network interruption during renewal
- Service unavailable during renewal
- Certificate validation failure

### Load Tests

**Scenarios:**
- 1,000 devices renewing simultaneously
- 10,000 devices with staggered renewals
- Service performance under renewal load
- Database performance with renewal events

---

## Appendix

### Component Dependencies

```
Agent Certificate Rotation
├── internal/agent/device/certmanager/
│   ├── manager.go (existing, modified)
│   ├── lifecycle.go (new)
│   ├── expiration.go (new)
│   ├── swap.go (new)
│   └── provider/
│       ├── provisioner/csr.go (existing, modified)
│       └── storage/fs.go (existing, modified)
├── internal/agent/identity/
│   └── tpm.go (existing, extended)
├── internal/service/
│   └── certificatesigningrequest.go (existing, modified)
├── internal/store/
│   └── device.go (existing, modified)
└── internal/crypto/
    └── signer/ (existing, extended)
```

### Key Files to Modify

**Agent:**
- `internal/agent/agent.go` - Initialize lifecycle manager
- `internal/agent/device/certmanager/manager.go` - Add expiration monitoring
- `internal/agent/device/certmanager/provider/provisioner/csr.go` - Add renewal support
- `internal/agent/device/certmanager/provider/storage/fs.go` - Add atomic swap

**Service:**
- `internal/service/certificatesigningrequest.go` - Add renewal validation
- `internal/store/device.go` - Add certificate tracking
- `internal/store/certificate_renewal_events.go` - New table operations

**Database:**
- `internal/store/migrations/` - Add migration scripts

### Metrics to Add

**Agent Metrics:**
- `flightctl_agent_certificate_expiration_timestamp`
- `flightctl_agent_certificate_days_until_expiration`
- `flightctl_agent_certificate_renewal_attempts_total`
- `flightctl_agent_certificate_renewal_success_total`
- `flightctl_agent_certificate_renewal_failures_total`
- `flightctl_agent_certificate_recovery_attempts_total`
- `flightctl_agent_certificate_renewal_duration_seconds`

**Service Metrics:**
- `flightctl_service_certificate_renewal_requests_total`
- `flightctl_service_certificate_renewal_issued_total`
- `flightctl_service_certificate_renewal_rejected_total`
- `flightctl_service_certificate_validation_failures_total`
- `flightctl_service_certificate_renewal_duration_seconds`

---

**Document End**

