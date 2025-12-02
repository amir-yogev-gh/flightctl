# Product Requirements Document: Agent Certificate Rotation

> **Document Type:** Brownfield Feature PRD  
> **Jira Reference:** [EDM-323](https://issues.redhat.com/browse/EDM-323)  
> **Version:** 1.0  
> **Status:** To Do  
> **Last Updated:** November 29, 2025  
> **Target Release:** 1.1.0

---

## Executive Summary

This PRD defines the requirements for implementing automatic certificate rotation for Flight Control device management certificates. Currently, device certificates expire after 365 days with no automatic renewal mechanism, requiring manual re-enrollment that creates significant operational overhead for large fleets. This feature will enable devices to automatically renew certificates before expiration, recover from expired certificates after extended offline periods, and perform atomic certificate swaps to prevent service disruptions.

### Key Objectives

1. **Eliminate Manual Re-enrollment**: Remove the need for fleet administrators to manually track and re-enroll devices
2. **Enable Long-Term Autonomous Operation**: Allow devices to operate continuously without certificate-related interruptions
3. **Provide Offline Recovery**: Enable devices that have been offline beyond certificate expiry to automatically recover
4. **Ensure Zero-Downtime Rotation**: Guarantee atomic certificate swaps that maintain connectivity during rotation

---

## Problem Statement

### Current State

Device management certificates in Flight Control are issued once during enrollment with a fixed 365-day validity period. The system currently:

- Issues certificates only during initial enrollment
- Does not monitor certificate expiration dates
- Has no mechanism to proactively renew certificates
- Cannot recover from expired certificates without manual intervention
- Performs non-atomic certificate replacement operations

### Pain Points

#### 1. No Automatic Renewal
- **Problem**: Certificates are issued once and never renewed automatically
- **Impact**: Users must manually re-enroll every device before certificate expiry (365 days)
- **Scale Impact**: For a fleet of 10,000 devices, this means ~27 manual re-enrollments per day

#### 2. No Expiration Monitoring
- **Problem**: System tracks validity but doesn't check approaching expiration
- **Impact**: No proactive notification or action before certificates expire
- **Operational Impact**: Fleet administrators must maintain external tracking systems

#### 3. No Recovery from Expired Certificates
- **Problem**: Devices that go offline for extended periods cannot reconnect after certificate expiry
- **Impact**: Device becomes unreachable from management perspective until manual intervention
- **Use Case Impact**: Critical for edge deployments with intermittent connectivity (ships, remote locations, etc.)

#### 4. No Atomic Certificate Swap
- **Problem**: Certificate replacement is not atomic
- **Impact**: Power loss or network interruption during replacement can leave device without valid certificate
- **Failure Mode**: Device cannot authenticate to management service; requires physical access to recover

### Business Impact

- **Operational Overhead**: Manual tracking and re-enrollment burden scales linearly with fleet size
- **Service Disruption Risk**: Expired certificates cause immediate loss of device management capability
- **Deployment Constraints**: Limits long-term autonomous deployments in remote/harsh environments
- **Scalability Bottleneck**: Manual processes prevent efficient scaling beyond thousands of devices

---

## User Stories

### Primary User Stories

**US-1: Automatic Certificate Renewal**
> **As a** fleet administrator  
> **I want** my devices to automatically renew their management certificates before they expire  
> **So that** I don't have to manually track expiration dates and re-enroll thousands of devices

**Acceptance Criteria:**
- Device monitors certificate expiration continuously
- Renewal is triggered automatically at configurable threshold (default: 30 days before expiration)
- Renewal completes in background without disrupting device operations
- No user intervention required for successful renewal

---

**US-2: Expired Certificate Recovery**
> **As a** fleet administrator  
> **I want** devices that have been offline for extended periods to automatically recover and renew their expired certificates when they come back online  
> **So that** I don't have to physically access remote devices to restore management connectivity

**Acceptance Criteria:**
- Device detects expired management certificate upon coming online
- Device falls back to bootstrap/enrollment certificate or TPM credentials for authentication
- Device submits renewal request with security proof (TPM attestation/device fingerprint)
- Service validates security checks and issues new certificate
- Device automatically installs new certificate and resumes normal operations

---

**US-3: Atomic Certificate Rotation**
> **As a** fleet administrator  
> **I want** certificate rotation to be atomic  
> **So that** devices are never left without a valid certificate even if rotation is interrupted by power loss or network failure

**Acceptance Criteria:**
- New certificate is written atomically before old certificate is removed
- New certificate is validated before old certificate is removed
- Rollback mechanism preserves old certificate if validation fails
- Device always has at least one valid certificate at any point in rotation process

---

### Secondary User Stories

**US-4: Certificate Status Visibility**
> **As a** fleet administrator  
> **I want** to view certificate expiration status for my fleet  
> **So that** I can proactively identify devices that may need attention

**Acceptance Criteria:**
- Device status reports certificate expiration date
- Device status indicates renewal state (normal, renewing, expired, recovery)
- Fleet-level metrics show certificate health distribution

---

**US-5: Configurable Renewal Behavior**
> **As a** fleet administrator  
> **I want** to configure when certificate renewal is triggered  
> **So that** I can balance network load and security requirements

**Acceptance Criteria:**
- Renewal threshold is configurable (days before expiration)
- Retry policy is configurable (intervals, max attempts)
- Configuration can be set per-device via device spec

---

## Solution Overview

### High-Level Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                        Edge Device                          │
├─────────────────────────────────────────────────────────────┤
│  ┌──────────────────────────────────────────────────────┐  │
│  │         Certificate Lifecycle Manager                │  │
│  ├──────────────────────────────────────────────────────┤  │
│  │  • Expiration Monitor (continuous)                   │  │
│  │  • Renewal Trigger (30 days before expiry)           │  │
│  │  • Expired Certificate Detector                      │  │
│  │  • Bootstrap Cert Fallback Handler                   │  │
│  │  • Atomic Swap Coordinator                           │  │
│  └──────────────────────────────────────────────────────┘  │
│                          │                                  │
│                          ▼                                  │
│  ┌──────────────────────────────────────────────────────┐  │
│  │         Certificate Storage (TPM-backed)             │  │
│  │  • Management Certificate (active)                   │  │
│  │  • Management Certificate (pending)                  │  │
│  │  • Bootstrap/Enrollment Certificate                  │  │
│  └──────────────────────────────────────────────────────┘  │
└────────────────────────┬────────────────────────────────────┘
                         │ mTLS (cert auth)
                         │ or Bootstrap Cert (recovery)
                         ▼
┌─────────────────────────────────────────────────────────────┐
│                  Flight Control Service                     │
├─────────────────────────────────────────────────────────────┤
│  ┌──────────────────────────────────────────────────────┐  │
│  │         Certificate Renewal Handler                  │  │
│  ├──────────────────────────────────────────────────────┤  │
│  │  • Renewal Request Processor                         │  │
│  │  • Security Validator (TPM attestation)              │  │
│  │  • Expired Certificate Support                       │  │
│  │  • Certificate Signer (CA)                           │  │
│  └──────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────┘
```

### Solution Components

#### 1. Automatic Certificate Renewal

**Functionality:**
- Agent continuously monitors management certificate expiration date
- Triggers renewal when approaching expiration (configurable threshold, default: 30 days)
- Uses existing CSR mechanisms to request new certificate
- Performs renewal in background without disrupting operations

**Behavior:**
- Check expiration on agent startup
- Re-check periodically (e.g., daily)
- Generate CSR with same device identity
- Submit CSR to service via existing certificate
- Receive and validate new certificate
- Atomically swap to new certificate

---

#### 2. Expired Certificate Recovery

**Functionality:**
- Detects expired management certificate when device comes online
- Falls back to alternative authentication methods
- Submits renewal request with security proof
- Automatically installs new certificate

**Authentication Fallback Chain:**
1. **Management Certificate** (primary)
2. **Bootstrap/Enrollment Certificate** (if still valid and available)
3. **TPM Credentials** (device identity key, attestation)

**Recovery Process:**
```
1. Agent starts → detects expired management cert
2. Agent attempts connection with bootstrap cert
3. Service recognizes device (by device ID)
4. Agent submits renewal CSR + TPM attestation
5. Service validates:
   - Device was previously enrolled
   - TPM attestation matches device record
   - Device fingerprint matches
6. Service issues new certificate
7. Agent installs new certificate atomically
8. Agent resumes normal operations
```

---

#### 3. Atomic Certificate Rotation

**Functionality:**
- Writes new certificate before removing old certificate
- Validates new certificate before committing swap
- Implements rollback mechanism on failure
- Ensures device always has valid certificate

**Atomic Swap Process:**
```
1. Receive new certificate from service
2. Write new certificate to pending location
3. Validate new certificate:
   - Verify signature chain
   - Verify subject/SAN matches device identity
   - Verify not expired
   - Test connection with new cert (optional)
4. If validation succeeds:
   a. Atomically move pending → active (rename/symlink swap)
   b. Delete old certificate
5. If validation fails:
   a. Keep old certificate
   b. Delete pending certificate
   c. Log error and retry
```

**Failure Handling:**
- **Network interruption during download**: Retry on next sync
- **Power loss during write**: Pending cert incomplete, old cert still valid
- **Validation failure**: Old cert preserved, retry renewal
- **Service unavailable**: Exponential backoff retry

---

### User Experience

#### Normal Operation (Proactive Renewal)

**Timeline:**
- **Day 0**: Device enrolled, certificate valid for 365 days
- **Day 335** (30 days before expiry): Agent detects approaching expiration
- **Day 335**: Agent generates CSR, submits to service
- **Day 335**: Service issues new certificate
- **Day 335**: Agent receives, validates, and atomically swaps certificate
- **Day 335**: Device continues operating with new certificate (valid for 365 days)

**User Visibility:**
- No user action required
- Renewal happens transparently
- Device status may briefly show "Certificate Renewing" state
- Metrics track successful renewals

---

#### Recovery Scenario (Expired Certificate)

**Timeline:**
- **Day 0**: Device enrolled, goes offline
- **Day 365**: Management certificate expires (device still offline)
- **Day 400**: Device comes back online
- **Day 400**: Agent detects expired management certificate
- **Day 400**: Agent falls back to bootstrap certificate
- **Day 400**: Agent submits renewal request with TPM attestation
- **Day 400**: Service validates security proof, issues new certificate
- **Day 400**: Agent installs new certificate, resumes operations

**User Visibility:**
- Device status shows "Offline" while device is down
- Device status briefly shows "Certificate Recovery" when coming online
- Device status changes to "Online" after successful recovery
- Event log records recovery action

---

## Detailed Requirements

### Functional Requirements

#### FR-1: Certificate Expiration Monitoring
- **FR-1.1**: Agent MUST check management certificate expiration on startup
- **FR-1.2**: Agent MUST periodically check certificate expiration (configurable interval, default: daily)
- **FR-1.3**: Agent MUST calculate days until expiration
- **FR-1.4**: Agent MUST trigger renewal when expiration is within threshold (configurable, default: 30 days)

#### FR-2: Automatic Renewal
- **FR-2.1**: Agent MUST generate CSR with same device identity as current certificate
- **FR-2.2**: Agent MUST submit CSR to service using current valid certificate
- **FR-2.3**: Service MUST accept CSR from devices with valid certificates
- **FR-2.4**: Service MUST issue new certificate with standard validity period (365 days)
- **FR-2.5**: Agent MUST receive and store new certificate
- **FR-2.6**: Agent MUST validate new certificate before activation

#### FR-3: Expired Certificate Detection
- **FR-3.1**: Agent MUST detect expired management certificate on startup
- **FR-3.2**: Agent MUST detect expired management certificate during periodic checks
- **FR-3.3**: Agent MUST differentiate between "expiring soon" and "already expired"

#### FR-4: Bootstrap Certificate Fallback
- **FR-4.1**: Agent MUST maintain bootstrap/enrollment certificate separately from management certificate
- **FR-4.2**: Agent MUST attempt bootstrap certificate authentication when management certificate is expired
- **FR-4.3**: Service MUST accept connections authenticated with valid bootstrap certificates
- **FR-4.4**: Service MUST recognize devices by device ID when using bootstrap authentication

#### FR-5: TPM-Based Recovery
- **FR-5.1**: Agent MUST be able to generate TPM attestation for renewal requests
- **FR-5.2**: Agent MUST include device fingerprint in renewal requests
- **FR-5.3**: Service MUST validate TPM attestation against device records
- **FR-5.4**: Service MUST validate device fingerprint against device records
- **FR-5.5**: Service MUST accept renewal requests from devices with expired certificates if validation passes

#### FR-6: Atomic Certificate Swap
- **FR-6.1**: Agent MUST write new certificate to separate location before replacing active certificate
- **FR-6.2**: Agent MUST validate new certificate before making it active
- **FR-6.3**: Certificate validation MUST include:
  - Signature verification against CA chain
  - Subject/SAN match to device identity
  - Validity period check (not expired)
- **FR-6.4**: Agent MUST use atomic file operations (rename/move) to activate new certificate
- **FR-6.5**: Agent MUST delete old certificate only after new certificate is active
- **FR-6.6**: Agent MUST implement rollback: preserve old certificate if new certificate validation fails

#### FR-7: Configuration
- **FR-7.1**: Renewal threshold MUST be configurable (days before expiration)
- **FR-7.2**: Expiration check interval MUST be configurable
- **FR-7.3**: Retry policy MUST be configurable (intervals, max attempts)
- **FR-7.4**: Configuration SHOULD be settable via device spec

#### FR-8: Observability
- **FR-8.1**: Agent MUST log renewal initiation
- **FR-8.2**: Agent MUST log renewal completion
- **FR-8.3**: Agent MUST log renewal failures with error details
- **FR-8.4**: Agent MUST log recovery initiation
- **FR-8.5**: Agent MUST log recovery completion
- **FR-8.6**: Agent MUST expose metrics:
  - Certificate expiration date
  - Days until expiration
  - Renewal attempts
  - Renewal successes
  - Renewal failures
  - Recovery attempts
  - Recovery successes
- **FR-8.7**: Device status MUST indicate certificate state:
  - Normal
  - Expiring Soon
  - Renewing
  - Expired
  - Recovering
  - Renewal Failed

---

### Non-Functional Requirements

#### NFR-1: Reliability
- **NFR-1.1**: Certificate rotation MUST be idempotent (safe to retry)
- **NFR-1.2**: Certificate rotation MUST never leave device without valid certificate
- **NFR-1.3**: Certificate rotation failures MUST not affect device's ability to retry
- **NFR-1.4**: Certificate rotation MUST handle network interruptions gracefully

#### NFR-2: Performance
- **NFR-2.1**: Certificate expiration checks MUST NOT significantly impact agent performance
- **NFR-2.2**: Certificate renewal MUST NOT disrupt device operations
- **NFR-2.3**: Certificate rotation MUST complete within 60 seconds under normal conditions
- **NFR-2.4**: Service MUST handle concurrent renewal requests from multiple devices

#### NFR-3: Security
- **NFR-3.1**: New certificates MUST have same or stronger security properties as original
- **NFR-3.2**: Bootstrap certificates MUST only be used when management certificate is invalid
- **NFR-3.3**: TPM attestation MUST be validated before accepting renewal from expired certificate
- **NFR-3.4**: Device fingerprint MUST be validated before accepting renewal from expired certificate
- **NFR-3.5**: Renewal requests MUST not be accepted for devices that were never enrolled
- **NFR-3.6**: Old certificates MUST be securely deleted after successful rotation

#### NFR-4: Usability
- **NFR-4.1**: Certificate rotation MUST be transparent to users under normal operation
- **NFR-4.2**: Certificate recovery MUST be automatic without user intervention
- **NFR-4.3**: Certificate status MUST be easily viewable in device status
- **NFR-4.4**: Certificate metrics MUST be available via standard monitoring tools (Prometheus)

#### NFR-5: Compatibility
- **NFR-5.1**: Certificate rotation MUST work with existing enrollment process
- **NFR-5.2**: Certificate rotation MUST work with TPM-backed device identity
- **NFR-5.3**: Certificate rotation MUST work with existing CA infrastructure
- **NFR-5.4**: Certificate rotation MUST not break existing device communication

---

## Technical Design

### Agent-Side Components

#### Certificate Lifecycle Manager

**Responsibilities:**
- Monitor certificate expiration
- Trigger renewal at appropriate time
- Coordinate atomic swap
- Handle recovery scenarios

**Implementation Location:**
- Package: `internal/agent/certificate`
- Integration: Part of agent main loop

**Key Methods:**
```go
type CertificateLifecycleManager interface {
    // Check if certificate needs renewal
    CheckRenewal(ctx context.Context) (bool, error)
    
    // Perform certificate renewal
    RenewCertificate(ctx context.Context) error
    
    // Detect and recover from expired certificate
    RecoverExpiredCertificate(ctx context.Context) error
    
    // Atomically swap certificates
    SwapCertificate(ctx context.Context, newCert *Certificate) error
}
```

---

#### Expiration Monitor

**Responsibilities:**
- Parse certificate expiration date
- Calculate days until expiration
- Trigger renewal based on threshold

**Configuration:**
```yaml
certificate:
  renewal_threshold_days: 30      # Trigger renewal N days before expiry
  check_interval: 24h             # How often to check expiration
  retry_interval: 1h              # Retry interval on failure
  max_retries: 10                 # Max retry attempts
```

---

#### Bootstrap Certificate Handler

**Responsibilities:**
- Maintain separate bootstrap certificate
- Provide fallback authentication
- Switch between management and bootstrap certs

**Certificate Storage Layout:**
```
/var/lib/flightctl/certs/
├── management/
│   ├── cert.pem          # Active management certificate
│   ├── cert.pem.pending  # New certificate being validated
│   └── key.pem           # Private key (TPM-backed)
├── bootstrap/
│   ├── cert.pem          # Bootstrap/enrollment certificate
│   └── key.pem           # Bootstrap private key (TPM-backed)
└── ca/
    └── ca-bundle.pem     # CA certificate chain
```

---

#### Atomic Swap Coordinator

**Responsibilities:**
- Write new certificate to pending location
- Validate new certificate
- Atomically activate new certificate
- Rollback on failure

**Atomic Operations:**
- Use `rename()` system call for atomic file replacement (POSIX atomic)
- Use symlinks as alternative (atomic symlink swap)
- Validate before committing
- Keep old certificate until validation succeeds

---

### Service-Side Components

#### Certificate Renewal Handler

**Responsibilities:**
- Accept renewal CSR requests
- Validate device identity
- Issue new certificates
- Handle expired certificate renewal

**API Endpoint:**
```
POST /api/v1/agent/devices/{name}/certificaterenewal
```

**Request:**
```json
{
  "csr": "-----BEGIN CERTIFICATE REQUEST-----...",
  "attestation": {
    "tpm_quote": "...",
    "pcr_values": [...],
    "device_fingerprint": "..."
  },
  "reason": "proactive|expired"
}
```

**Response:**
```json
{
  "certificate": "-----BEGIN CERTIFICATE-----...",
  "validity_days": 365
}
```

---

#### Security Validator

**Responsibilities:**
- Validate TPM attestation
- Verify device fingerprint
- Check device enrollment status
- Authorize renewal request

**Validation Steps:**
1. Verify device exists in database
2. Verify device was previously enrolled
3. If certificate expired:
   - Validate bootstrap certificate OR TPM attestation
   - Validate device fingerprint matches device record
   - Check device not in revoked/blacklist state
4. Generate and sign new certificate

---

### Database Schema Changes

#### Device Table Extensions

```sql
-- Add certificate tracking fields
ALTER TABLE devices ADD COLUMN certificate_expiration TIMESTAMP;
ALTER TABLE devices ADD COLUMN certificate_last_renewed TIMESTAMP;
ALTER TABLE devices ADD COLUMN certificate_renewal_count INTEGER DEFAULT 0;
ALTER TABLE devices ADD COLUMN certificate_fingerprint TEXT;

-- Add index for expiration monitoring
CREATE INDEX idx_devices_cert_expiration ON devices(certificate_expiration);
```

#### Certificate Renewal Events Table

```sql
CREATE TABLE certificate_renewal_events (
    id UUID PRIMARY KEY,
    device_id UUID REFERENCES devices(id),
    event_type TEXT NOT NULL, -- 'renewal_start', 'renewal_success', 'renewal_failed', 'recovery_start', 'recovery_success'
    reason TEXT, -- 'proactive', 'expired'
    old_cert_expiration TIMESTAMP,
    new_cert_expiration TIMESTAMP,
    error_message TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_cert_renewal_events_device ON certificate_renewal_events(device_id);
CREATE INDEX idx_cert_renewal_events_created ON certificate_renewal_events(created_at);
```

---

### Configuration

#### Agent Configuration

```yaml
# /etc/flightctl/agent-config.yaml
certificate:
  # Renewal settings
  renewal_threshold_days: 30
  check_interval: 24h
  
  # Retry policy
  retry_interval: 1h
  max_retries: 10
  backoff_multiplier: 2.0
  max_backoff: 24h
  
  # Paths
  management_cert_path: /var/lib/flightctl/certs/management/cert.pem
  management_key_path: /var/lib/flightctl/certs/management/key.pem
  bootstrap_cert_path: /var/lib/flightctl/certs/bootstrap/cert.pem
  bootstrap_key_path: /var/lib/flightctl/certs/bootstrap/key.pem
  ca_bundle_path: /var/lib/flightctl/certs/ca/ca-bundle.pem
  
  # Recovery settings
  enable_expired_recovery: true
  use_tpm_attestation: true
```

---

### Metrics

#### Agent Metrics

```prometheus
# Certificate expiration
flightctl_agent_certificate_expiration_timestamp{type="management|bootstrap"}

# Days until expiration
flightctl_agent_certificate_days_until_expiration{type="management|bootstrap"}

# Renewal attempts/successes/failures
flightctl_agent_certificate_renewal_attempts_total{reason="proactive|expired"}
flightctl_agent_certificate_renewal_success_total{reason="proactive|expired"}
flightctl_agent_certificate_renewal_failures_total{reason="proactive|expired"}

# Recovery operations
flightctl_agent_certificate_recovery_attempts_total
flightctl_agent_certificate_recovery_success_total
flightctl_agent_certificate_recovery_failures_total

# Renewal duration
flightctl_agent_certificate_renewal_duration_seconds
```

#### Service Metrics

```prometheus
# Renewal requests
flightctl_service_certificate_renewal_requests_total{reason="proactive|expired"}
flightctl_service_certificate_renewal_issued_total{reason="proactive|expired"}
flightctl_service_certificate_renewal_rejected_total{reason="proactive|expired"}

# Validation failures
flightctl_service_certificate_validation_failures_total{reason="attestation|fingerprint|revoked"}

# Processing duration
flightctl_service_certificate_renewal_duration_seconds
```

---

### Testing Strategy

#### Unit Tests
- Certificate expiration calculation
- Threshold trigger logic
- Atomic swap operations
- Rollback mechanism
- TPM attestation validation
- Device fingerprint validation

#### Integration Tests
- Full renewal flow (agent → service → agent)
- Expired certificate recovery
- Bootstrap certificate fallback
- Atomic swap under simulated failures (power loss, network interruption)
- Retry logic with exponential backoff
- Concurrent renewals from multiple devices

#### End-to-End Tests
- Device enrollment → wait → automatic renewal
- Device goes offline → certificate expires → recovery on reconnect
- Network interruption during renewal
- Service unavailable during renewal
- Certificate validation failure scenarios

#### Load Tests
- 1,000 devices renewing simultaneously
- 10,000 devices with staggered renewal over 30 days
- Service performance under renewal load

---

## Implementation Plan

### Phase 1: Foundation (Week 1-2)
- **Deliverables:**
  - Certificate lifecycle manager structure
  - Expiration monitoring logic
  - Configuration schema
  - Database schema updates
  
- **Tasks:**
  - [ ] Create `internal/agent/certificate` package
  - [ ] Implement expiration checker
  - [ ] Add configuration fields
  - [ ] Create database migrations
  - [ ] Write unit tests

---

### Phase 2: Proactive Renewal (Week 3-4)
- **Deliverables:**
  - Automatic renewal for valid certificates
  - Service-side renewal handler
  - Basic metrics and logging
  
- **Tasks:**
  - [ ] Implement CSR generation for renewal
  - [ ] Create `/certificaterenewal` API endpoint
  - [ ] Implement certificate issuance
  - [ ] Add metrics collection
  - [ ] Write integration tests

---

### Phase 3: Atomic Swap (Week 5-6)
- **Deliverables:**
  - Atomic certificate swap
  - Rollback mechanism
  - Failure handling
  
- **Tasks:**
  - [ ] Implement pending certificate handling
  - [ ] Add certificate validation
  - [ ] Implement atomic file operations
  - [ ] Add rollback logic
  - [ ] Test failure scenarios

---

### Phase 4: Expired Recovery (Week 7-8)
- **Deliverables:**
  - Expired certificate detection
  - Bootstrap certificate fallback
  - TPM attestation validation
  - Recovery flow
  
- **Tasks:**
  - [ ] Implement expired detection
  - [ ] Add bootstrap certificate handler
  - [ ] Implement TPM attestation
  - [ ] Add device fingerprint validation
  - [ ] Implement recovery flow
  - [ ] Write recovery tests

---

### Phase 5: Observability & Polish (Week 9-10)
- **Deliverables:**
  - Complete metrics coverage
  - Enhanced logging
  - Device status indicators
  - Documentation
  
- **Tasks:**
  - [ ] Complete metrics implementation
  - [ ] Add structured logging
  - [ ] Update device status reporting
  - [ ] Write user documentation
  - [ ] Write operational runbooks
  - [ ] Perform load testing

---

## Success Metrics

### Primary Metrics

**Automatic Renewal Rate**
- **Target**: >99% of certificates renewed automatically before expiration
- **Measurement**: `renewal_success_total{reason="proactive"}` / `total_devices`

**Recovery Success Rate**
- **Target**: >95% of expired certificates recovered automatically
- **Measurement**: `recovery_success_total` / `recovery_attempts_total`

**Zero-Certificate Incidents**
- **Target**: 0 devices left without valid certificate due to rotation failure
- **Measurement**: Monitor devices with no valid certificate + audit logs

---

### Secondary Metrics

**Manual Intervention Rate**
- **Target**: <1% of devices require manual certificate intervention
- **Measurement**: Manual re-enrollment count / total device count

**Renewal Latency**
- **Target**: <60 seconds average renewal time
- **Measurement**: `renewal_duration_seconds` P50/P95/P99

**Service Availability During Renewals**
- **Target**: 100% device availability during certificate renewal
- **Measurement**: Monitor connection failures during renewal

---

## Risks and Mitigations

### Risk 1: Certificate Rotation Fails, Device Loses Connectivity
**Likelihood**: Medium  
**Impact**: High  
**Mitigation**: 
- Implement atomic swap with validation
- Implement rollback mechanism
- Extensive testing of failure scenarios
- Keep old certificate until new certificate validated

### Risk 2: Service Unavailable During Renewal Window
**Likelihood**: Low  
**Impact**: Medium  
**Mitigation**:
- Implement retry with exponential backoff
- Start renewal 30 days before expiration (ample retry time)
- Device continues operating with current certificate

### Risk 3: TPM Attestation Validation Fails
**Likelihood**: Low  
**Impact**: Medium  
**Mitigation**:
- Implement fallback to device fingerprint validation
- Allow manual approval override for edge cases
- Comprehensive logging for debugging

### Risk 4: Bootstrap Certificate Also Expired
**Likelihood**: Very Low  
**Impact**: High  
**Mitigation**:
- Bootstrap certificates have longer validity (e.g., 2+ years)
- TPM attestation as final fallback
- Alert when bootstrap certificate approaching expiration

### Risk 5: CA Private Key Rotation During Certificate Renewal
**Likelihood**: Low  
**Impact**: Low  
**Mitigation**:
- Document CA rotation procedures
- Ensure CA bundle updated on devices
- Test certificate chain validation

---

## Out of Scope

The following items are explicitly **NOT** part of this feature:

1. **Certificate Revocation**: Revoking certificates before expiration
2. **Certificate Shortening**: Reducing certificate validity periods (remains 365 days)
3. **Multiple Certificate Management**: Managing multiple certificates per device (beyond management + bootstrap)
4. **Certificate Chain Updates**: Updating CA certificates or certificate chains
5. **Manual Renewal Triggers**: API/CLI commands to manually trigger renewal
6. **Certificate Renewal Notifications**: Alerts/emails about upcoming renewals
7. **Certificate Renewal Scheduling**: User-controlled renewal timing
8. **Certificate Renewal Policies**: Per-device/fleet/org renewal policies
9. **Certificate Renewal History**: Detailed audit trail of all renewals
10. **Certificate Renewal Rollback**: Rolling back to previous certificate after renewal
11. **Non-Management Certificates**: Renewal of other certificate types
12. **Certificate Renewal for Unenrolled Devices**: Devices that were never enrolled
13. **Certificate Renewal During Enrollment**: Renewing during initial enrollment
14. **Certificate Renewal Metrics Dashboard**: Dedicated UI dashboard

---

## Appendix

### References

- **Jira Ticket**: [EDM-323 - Implement Agent Certificate Rotation](https://issues.redhat.com/browse/EDM-323)
- **Architecture Documentation**: [ARCHITECTURE.md](ARCHITECTURE.md)
- **Security Documentation**: [Security Architecture](ARCHITECTURE.md#security-architecture)

### Glossary

- **Bootstrap Certificate**: Initial certificate issued during enrollment, used as fallback
- **Management Certificate**: Primary certificate used for device-service communication
- **CSR**: Certificate Signing Request
- **TPM**: Trusted Platform Module - hardware security chip
- **Attestation**: Cryptographic proof of device identity and state
- **Atomic Swap**: Operation that cannot be partially completed (all-or-nothing)

### Contributors

- **Reporter**: Frank Zdarsky (fzdarsky@redhat.com)
- **Assignee**: Assaf Albo (aalbo@redhat.com)
- **Architect**: Avishay Traeger (atraeger@redhat.com)
- **QA Contact**: Samuel De La Cruz Lopez (sdelacru@redhat.com)

---

**Document End**

