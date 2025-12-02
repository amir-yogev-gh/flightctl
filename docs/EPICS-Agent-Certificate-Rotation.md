# Agent Certificate Rotation - Epic Breakdown

> **Feature:** Agent Certificate Rotation  
> **Jira Reference:** [EDM-323](https://issues.redhat.com/browse/EDM-323)  
> **Version:** 1.0  
> **Last Updated:** November 29, 2025

## Overview

This document provides a suggested epic breakdown for implementing automatic certificate rotation in Flight Control. Epics are organized to align with the implementation phases and user stories defined in the PRD.

---

## Epic 1: Certificate Lifecycle Foundation

**Epic ID:** EDM-323-EPIC-1  
**Title:** Certificate Lifecycle Foundation and Expiration Monitoring  
**Priority:** High  
**Estimated Effort:** 2-3 weeks  
**Dependencies:** None

### Description
Establish the foundational infrastructure for certificate lifecycle management, including expiration monitoring, configuration schema, and database tracking capabilities.

### User Stories Covered
- US-4: Certificate Status Visibility (partial)

### Functional Requirements
- FR-1: Certificate Expiration Monitoring
- FR-7: Configuration
- FR-8: Observability (partial)

### Deliverables
- Certificate lifecycle manager structure
- Expiration monitoring logic
- Configuration schema and validation
- Database schema updates and migrations
- Basic certificate status tracking

### Key Components
- `internal/agent/device/certmanager/lifecycle.go` (new)
- `internal/agent/device/certmanager/expiration.go` (new)
- Database migrations for certificate tracking
- Agent configuration schema extensions

### Acceptance Criteria
- [ ] Certificate expiration dates are parsed and tracked
- [ ] Expiration monitoring runs on agent startup and periodically
- [ ] Days until expiration are calculated correctly
- [ ] Configuration schema supports renewal settings
- [ ] Database tracks certificate expiration per device
- [ ] Unit tests cover expiration calculation logic

---

## Epic 2: Proactive Certificate Renewal

**Epic ID:** EDM-323-EPIC-2  
**Title:** Automatic Proactive Certificate Renewal  
**Priority:** High  
**Estimated Effort:** 2-3 weeks  
**Dependencies:** Epic 1

### Description
Implement automatic certificate renewal for devices with valid certificates that are approaching expiration. This includes CSR generation, service-side renewal handling, and certificate issuance.

### User Stories Covered
- US-1: Automatic Certificate Renewal
- US-4: Certificate Status Visibility (partial)

### Functional Requirements
- FR-2: Automatic Renewal
- FR-6: Atomic Certificate Swap (partial - basic swap)
- FR-7: Configuration
- FR-8: Observability (partial)

### Deliverables
- CSR generation for renewal requests
- Service-side renewal request handler
- Auto-approval logic for renewal CSRs
- Certificate issuance for renewals
- Basic metrics and logging

### Key Components
- Enhanced `internal/agent/device/certmanager/provider/provisioner/csr.go`
- Enhanced `internal/service/certificatesigningrequest.go`
- Renewal validation logic
- Metrics collection

### Acceptance Criteria
- [ ] Agent triggers renewal when certificate expires within threshold (default: 30 days)
- [ ] Agent generates CSR with renewal context
- [ ] Service accepts and validates renewal requests
- [ ] Service auto-approves valid renewal requests
- [ ] Service issues new certificates for approved renewals
- [ ] Agent receives and stores new certificates
- [ ] Renewal completes without disrupting device operations
- [ ] Metrics track renewal attempts and successes
- [ ] Integration tests validate full renewal flow

---

## Epic 3: Atomic Certificate Swap

**Epic ID:** EDM-323-EPIC-3  
**Title:** Atomic Certificate Rotation with Rollback  
**Priority:** High  
**Estimated Effort:** 2 weeks  
**Dependencies:** Epic 2

### Description
Implement atomic certificate swap mechanism to ensure devices never lose valid certificates during rotation, even in case of power loss or network interruption.

### User Stories Covered
- US-3: Atomic Certificate Rotation

### Functional Requirements
- FR-6: Atomic Certificate Swap (complete)
- NFR-1: Reliability

### Deliverables
- Pending certificate storage mechanism
- Certificate validation before activation
- Atomic file swap operations
- Rollback mechanism on failure
- Failure scenario handling

### Key Components
- `internal/agent/device/certmanager/swap.go` (new)
- Enhanced `internal/agent/device/certmanager/provider/storage/fs.go`
- Atomic file operations
- Certificate validation logic

### Acceptance Criteria
- [ ] New certificates are written to pending location before activation
- [ ] Certificates are validated before being made active
- [ ] Atomic swap operations use POSIX atomic file operations
- [ ] Old certificate is preserved until new certificate is validated
- [ ] Rollback mechanism restores old certificate on validation failure
- [ ] Device always has at least one valid certificate during rotation
- [ ] Power loss during rotation doesn't leave device without certificate
- [ ] Network interruption during rotation is handled gracefully
- [ ] Unit tests cover atomic swap operations
- [ ] Integration tests validate failure scenarios

---

## Epic 4: Expired Certificate Recovery

**Epic ID:** EDM-323-EPIC-4  
**Title:** Expired Certificate Detection and Recovery  
**Priority:** High  
**Estimated Effort:** 2-3 weeks  
**Dependencies:** Epic 3

### Description
Enable devices with expired certificates to automatically recover by detecting expiration, falling back to bootstrap certificates or TPM attestation, and obtaining new certificates.

### User Stories Covered
- US-2: Expired Certificate Recovery
- US-4: Certificate Status Visibility (partial)

### Functional Requirements
- FR-3: Expired Certificate Detection
- FR-4: Bootstrap Certificate Fallback
- FR-5: TPM-Based Recovery
- FR-8: Observability (partial)

### Deliverables
- Expired certificate detection logic
- Bootstrap certificate fallback handler
- TPM attestation generation for recovery
- Service-side recovery request validation
- Device fingerprint validation
- Recovery flow implementation

### Key Components
- Enhanced `internal/agent/device/certmanager/lifecycle.go`
- `internal/agent/device/bootstrap.go` (enhanced)
- Enhanced `internal/agent/identity/tpm.go`
- Enhanced `internal/service/certificatesigningrequest.go`
- Security validation logic

### Acceptance Criteria
- [ ] Agent detects expired management certificate on startup
- [ ] Agent detects expired certificate during periodic checks
- [ ] Agent differentiates between "expiring soon" and "already expired"
- [ ] Agent falls back to bootstrap certificate when management cert expired
- [ ] Agent generates TPM attestation for recovery requests
- [ ] Service accepts connections authenticated with bootstrap certificates
- [ ] Service validates TPM attestation for expired certificate renewals
- [ ] Service validates device fingerprint matches device record
- [ ] Service issues new certificates for validated recovery requests
- [ ] Agent installs new certificate and resumes operations
- [ ] Recovery completes automatically without user intervention
- [ ] Integration tests validate full recovery flow
- [ ] E2E tests validate offline → expired → recovery scenario

---

## Epic 5: Configuration and Observability

**Epic ID:** EDM-323-EPIC-5  
**Title:** Configuration Management and Observability  
**Priority:** Medium  
**Estimated Effort:** 1-2 weeks  
**Dependencies:** Epic 2, Epic 4

### Description
Complete configuration management, comprehensive metrics, enhanced logging, device status indicators, and operational documentation.

### User Stories Covered
- US-4: Certificate Status Visibility (complete)
- US-5: Configurable Renewal Behavior

### Functional Requirements
- FR-7: Configuration (complete)
- FR-8: Observability (complete)
- NFR-4: Usability

### Deliverables
- Complete configuration schema with all options
- Comprehensive Prometheus metrics
- Structured logging for all operations
- Device status certificate state indicators
- Configuration validation
- User documentation
- Operational runbooks

### Key Components
- Configuration schema and validation
- Metrics instrumentation
- Logging enhancements
- Device status extensions
- Documentation

### Acceptance Criteria
- [ ] All renewal settings are configurable via device spec
- [ ] Renewal threshold is configurable (default: 30 days)
- [ ] Expiration check interval is configurable (default: daily)
- [ ] Retry policy is fully configurable
- [ ] All metrics are exposed via Prometheus
- [ ] Structured logging covers all renewal operations
- [ ] Device status shows certificate state (normal, renewing, expired, etc.)
- [ ] Device status shows certificate expiration date
- [ ] Fleet-level metrics show certificate health distribution
- [ ] User documentation explains configuration options
- [ ] Operational runbooks cover troubleshooting scenarios

---

## Epic 6: Testing and Validation

**Epic ID:** EDM-323-EPIC-6  
**Title:** Comprehensive Testing and Validation  
**Priority:** High  
**Estimated Effort:** 2-3 weeks  
**Dependencies:** Epic 2, Epic 3, Epic 4

### Description
Comprehensive testing strategy including unit tests, integration tests, end-to-end tests, load tests, and failure scenario validation.

### Functional Requirements
- All requirements (validation)

### Deliverables
- Unit test suite for all components
- Integration test suite
- End-to-end test scenarios
- Load test scenarios
- Failure scenario tests
- Test documentation

### Key Test Scenarios
- Certificate expiration calculation
- Renewal trigger logic
- Atomic swap operations
- Rollback mechanism
- TPM attestation validation
- Device fingerprint validation
- Full renewal flow (agent → service → agent)
- Expired certificate recovery
- Bootstrap certificate fallback
- Network interruption scenarios
- Power loss scenarios
- Service unavailable scenarios
- Concurrent renewals from multiple devices
- Load testing with 1,000+ devices

### Acceptance Criteria
- [ ] Unit tests achieve >80% code coverage
- [ ] Integration tests cover all renewal flows
- [ ] E2E tests validate complete user scenarios
- [ ] Load tests validate system under renewal load
- [ ] Failure scenario tests validate resilience
- [ ] All tests pass consistently
- [ ] Test documentation is complete

---

## Epic 7: Database and API Enhancements

**Epic ID:** EDM-323-EPIC-7  
**Title:** Database Schema and API Enhancements  
**Priority:** Medium  
**Estimated Effort:** 1 week  
**Dependencies:** Epic 1

### Description
Database schema updates for certificate tracking, renewal event logging, and optional API enhancements for better renewal semantics.

### Functional Requirements
- Database schema changes
- Optional API enhancements

### Deliverables
- Database migration scripts
- Certificate renewal events table
- Device certificate tracking fields
- Optional renewal API endpoint
- API documentation updates

### Key Components
- Database migrations
- Store layer extensions
- Optional API endpoint
- OpenAPI spec updates

### Acceptance Criteria
- [ ] Database migrations are created and tested
- [ ] Certificate expiration tracking fields added to devices table
- [ ] Certificate renewal events table created
- [ ] Indexes are created for efficient queries
- [ ] Migration scripts are backward compatible
- [ ] Store layer supports certificate tracking
- [ ] Optional API endpoint is documented (if implemented)
- [ ] API documentation is updated

---

## Epic Dependencies Graph

```
Epic 1: Foundation
    │
    ├──► Epic 2: Proactive Renewal
    │       │
    │       ├──► Epic 3: Atomic Swap
    │       │       │
    │       │       └──► Epic 4: Expired Recovery
    │       │
    │       └──► Epic 5: Observability
    │
    └──► Epic 7: Database/API
            │
            └──► Epic 2: Proactive Renewal

Epic 2, Epic 3, Epic 4
    │
    └──► Epic 6: Testing
```

---

## Epic Priority and Sequencing

### Phase 1: Foundation (Weeks 1-3)
1. **Epic 1:** Certificate Lifecycle Foundation
2. **Epic 7:** Database and API Enhancements

### Phase 2: Core Functionality (Weeks 4-7)
3. **Epic 2:** Proactive Certificate Renewal
4. **Epic 3:** Atomic Certificate Swap
5. **Epic 4:** Expired Certificate Recovery

### Phase 3: Polish and Validation (Weeks 8-10)
6. **Epic 5:** Configuration and Observability
7. **Epic 6:** Testing and Validation

---

## Epic Sizing Estimates

| Epic | Story Points | Duration | Team Size |
|------|-------------|-----------|-----------|
| Epic 1: Foundation | 13 | 2-3 weeks | 2-3 engineers |
| Epic 2: Proactive Renewal | 13 | 2-3 weeks | 2-3 engineers |
| Epic 3: Atomic Swap | 8 | 2 weeks | 2 engineers |
| Epic 4: Expired Recovery | 13 | 2-3 weeks | 2-3 engineers |
| Epic 5: Observability | 5 | 1-2 weeks | 1-2 engineers |
| Epic 6: Testing | 13 | 2-3 weeks | 2-3 engineers |
| Epic 7: Database/API | 5 | 1 week | 1-2 engineers |

**Total Estimated Effort:** 70 story points, 10-12 weeks with 2-3 engineers

---

## Risk Assessment by Epic

### Epic 1: Foundation
- **Risk:** Low
- **Mitigation:** Well-defined requirements, existing certificate parsing code

### Epic 2: Proactive Renewal
- **Risk:** Medium
- **Mitigation:** Reuses existing CSR workflow, incremental implementation

### Epic 3: Atomic Swap
- **Risk:** High
- **Mitigation:** Extensive testing, POSIX atomic operations, rollback mechanism

### Epic 4: Expired Recovery
- **Risk:** Medium-High
- **Mitigation:** TPM attestation validation, device fingerprint checks

### Epic 5: Observability
- **Risk:** Low
- **Mitigation:** Standard patterns, existing metrics infrastructure

### Epic 6: Testing
- **Risk:** Low
- **Mitigation:** Comprehensive test plan, early test development

### Epic 7: Database/API
- **Risk:** Low
- **Mitigation:** Standard migration patterns, backward compatible changes

---

## Success Criteria by Epic

### Epic 1: Foundation
- Certificate expiration monitoring operational
- Configuration schema supports all renewal settings
- Database tracks certificate metadata

### Epic 2: Proactive Renewal
- >99% of certificates renewed automatically before expiration
- Renewal completes within 60 seconds
- Zero service disruption during renewal

### Epic 3: Atomic Swap
- Zero devices left without valid certificate due to rotation failure
- Atomic swap handles all failure scenarios
- Rollback mechanism tested and validated

### Epic 4: Expired Recovery
- >95% of expired certificates recovered automatically
- Recovery works with bootstrap certificate fallback
- TPM attestation validation successful

### Epic 5: Observability
- All metrics exposed and documented
- Device status shows certificate state
- Configuration options validated and documented

### Epic 6: Testing
- >80% code coverage
- All test scenarios pass
- Load tests validate system performance

### Epic 7: Database/API
- Migrations run successfully
- Database queries performant
- API documentation complete

---

## Notes

- Epics can be worked on in parallel where dependencies allow
- Epic 6 (Testing) should run continuously alongside development
- Consider splitting Epic 4 into two epics if TPM attestation complexity is high
- Epic 5 can be started early and completed incrementally
- Regular epic reviews should validate alignment with PRD requirements

---

**Document End**

