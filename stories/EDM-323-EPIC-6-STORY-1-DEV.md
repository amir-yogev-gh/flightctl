# Developer Story: Unit Test Suite for Certificate Rotation

**Story ID:** EDM-323-EPIC-6-STORY-1  
**Epic:** EDM-323-EPIC-6 (Testing and Validation)  
**Feature:** EDM-323 (Agent Certificate Rotation)  
**Story Points:** 8  
**Priority:** High

## Overview

Create comprehensive unit tests for all certificate rotation components to ensure code quality, catch regressions, and achieve >80% code coverage.

## Implementation Tasks

### Task 1: Test Certificate Expiration Monitoring

**File:** `internal/agent/device/certmanager/expiration_test.go` (new)

**Objective:** Test certificate expiration monitoring logic.

**Test Cases:**
- Test ParseCertificateExpiration with valid certificate
- Test ParseCertificateExpiration with nil certificate
- Test CalculateDaysUntilExpiration with various expiration dates
- Test IsExpired with expired certificate
- Test IsExpired with valid certificate
- Test IsExpiringSoon with various thresholds
- Test edge cases (expiring today, expired yesterday, far future)

**Coverage Target:** >80%

---

### Task 2: Test Certificate Lifecycle Manager

**File:** `internal/agent/device/certmanager/lifecycle_test.go` (new)

**Objective:** Test certificate lifecycle management.

**Test Cases:**
- Test CheckRenewal with various expiration scenarios
- Test GetCertificateState with all state transitions
- Test UpdateCertificateState updates state correctly
- Test RecordError records errors correctly
- Test TriggerRenewal initiates renewal
- Test state transitions (normal → expiring_soon → renewing → normal)
- Test error state transitions

**Coverage Target:** >80%

---

### Task 3: Test CSR Generation for Renewal

**File:** `internal/agent/device/certmanager/provider/provisioner/csr_renewal_test.go` (new)

**Objective:** Test CSR generation with renewal context.

**Test Cases:**
- Test CSR generation includes renewal labels
- Test CSR generation includes renewal context
- Test CSR generation with TPM attestation
- Test CSR generation with bootstrap certificate
- Test CSR metadata is correct
- Test error handling

**Coverage Target:** >80%

---

### Task 4: Test Certificate Validation

**File:** `internal/agent/device/certmanager/swap_validation_test.go` (new)

**Objective:** Test certificate validation logic.

**Test Cases:**
- Test ValidatePendingCertificate with valid certificate
- Test ValidatePendingCertificate with expired certificate
- Test ValidatePendingCertificate with wrong identity
- Test ValidatePendingCertificate with invalid signature
- Test ValidatePendingCertificate with mismatched key pair
- Test each validation step independently
- Test error messages are clear

**Coverage Target:** >80%

---

### Task 5: Test Atomic Swap Operations

**File:** `internal/agent/device/certmanager/swap_atomic_test.go` (new)

**Objective:** Test atomic swap operations.

**Test Cases:**
- Test AtomicSwap succeeds
- Test AtomicSwap with certificate swap failure
- Test AtomicSwap with key swap failure
- Test AtomicSwap rollback on failure
- Test backup creation and restoration
- Test cleanup after successful swap
- Test concurrent swap attempts

**Coverage Target:** >80%

---

### Task 6: Test Rollback Mechanism

**File:** `internal/agent/device/certmanager/swap_rollback_test.go` (new)

**Objective:** Test rollback mechanism.

**Test Cases:**
- Test RollbackSwap restores from backup
- Test RollbackSwap handles missing backup
- Test RollbackSwap cleans up pending files
- Test DetectAndRecoverIncompleteSwap detects incomplete swaps
- Test recovery from missing active certificate
- Test recovery from certificate/key mismatch

**Coverage Target:** >80%

---

### Task 7: Test Expired Certificate Detection

**File:** `internal/agent/device/certmanager/lifecycle_expiration_test.go` (new)

**Objective:** Test expired certificate detection.

**Test Cases:**
- Test DetectExpiredCertificate with expired certificate
- Test DetectExpiredCertificate with expiring soon certificate
- Test DetectExpiredCertificate with normal certificate
- Test CheckExpiredCertificates checks all certificates
- Test CheckExpiredCertificates triggers recovery
- Test startup expiration check

**Coverage Target:** >80%

---

### Task 8: Test Bootstrap Certificate Fallback

**File:** `internal/agent/device/bootstrap_cert_test.go` (new)

**Objective:** Test bootstrap certificate fallback.

**Test Cases:**
- Test GetBootstrapCertificate loads certificate
- Test ValidateBootstrapCertificate with valid certificate
- Test ValidateBootstrapCertificate with expired certificate
- Test GetCertificateForAuth uses management cert when valid
- Test GetCertificateForAuth falls back to bootstrap
- Test HasValidBootstrapCertificate returns correct value

**Coverage Target:** >80%

---

### Task 9: Test TPM Attestation Generation

**File:** `internal/agent/identity/tpm_renewal_test.go` (new)

**Objective:** Test TPM attestation generation.

**Test Cases:**
- Test GenerateTPMQuote generates quote
- Test ReadPCRValues reads PCRs correctly
- Test GetDeviceFingerprint generates fingerprint
- Test GenerateRenewalAttestation generates complete attestation
- Test attestation includes all components
- Test error handling

**Coverage Target:** >80%

---

### Task 10: Test Recovery Validation

**File:** `internal/service/certificatesigningrequest_recovery_test.go` (new)

**Objective:** Test service-side recovery validation.

**Test Cases:**
- Test isRecoveryRequest detects recovery requests
- Test validateExpiredCertificateRenewal with valid recovery
- Test validateRecoveryPeerCertificate with bootstrap cert
- Test validateTPMAttestationForRecovery with valid attestation
- Test validateDeviceFingerprint matches device
- Test autoApproveRecovery sets approval condition
- Test validation rejects invalid requests

**Coverage Target:** >80%

---

### Task 11: Test Configuration

**File:** `internal/agent/config/certificate_config_test.go` (new)

**Objective:** Test certificate renewal configuration.

**Test Cases:**
- Test CertificateRenewalConfig defaults
- Test CertificateRenewalConfig validation
- Test configuration merging
- Test invalid configuration rejection
- Test configuration override

**Coverage Target:** >80%

---

### Task 12: Test Storage Operations

**File:** `internal/agent/device/certmanager/provider/storage/fs_pending_test.go` (new)

**Objective:** Test pending certificate storage.

**Test Cases:**
- Test WritePending writes to pending location
- Test LoadPendingCertificate loads correctly
- Test CleanupPending removes pending files
- Test HasPendingCertificate detects pending
- Test atomic operations

**Coverage Target:** >80%

---

## Code Coverage Requirements

- **Overall Coverage:** >80% for all certificate rotation code
- **Critical Paths:** 100% coverage for renewal and recovery flows
- **Error Handling:** >90% coverage for error paths
- **State Transitions:** 100% coverage for all state transitions

---

## Testing Best Practices

1. **Use Mocks:** Mock external dependencies (TPM, file system, network)
2. **Test Edge Cases:** Test boundary conditions (expiring today, expired yesterday)
3. **Test Error Cases:** Test all error paths
4. **Test State Transitions:** Test all state machine transitions
5. **Test Thread Safety:** Test concurrent operations where applicable
6. **Test Isolation:** Each test should be independent
7. **Test Naming:** Use descriptive test names

---

## Definition of Done

- [ ] Unit tests written for all components
- [ ] >80% code coverage achieved
- [ ] All unit tests passing
- [ ] Edge cases covered
- [ ] Error cases covered
- [ ] State transitions covered
- [ ] Code reviewed and approved
- [ ] Test documentation updated

---

## Related Files

- All certificate rotation implementation files
- Test files in corresponding test directories

---

## Dependencies

- **All Implementation Stories**: EPIC-1 through EPIC-5 must be completed

---

## Notes

- **Coverage Tools**: Use Go's built-in coverage tools (`go test -cover`)
- **Coverage Reports**: Generate coverage reports for review
- **Continuous Integration**: Ensure tests run in CI/CD pipeline
- **Test Maintenance**: Keep tests updated as code evolves

---

**Document End**

