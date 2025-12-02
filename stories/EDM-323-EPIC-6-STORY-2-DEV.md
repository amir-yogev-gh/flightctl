# Developer Story: Integration Test Suite for Certificate Rotation

**Story ID:** EDM-323-EPIC-6-STORY-2  
**Epic:** EDM-323-EPIC-6 (Testing and Validation)  
**Feature:** EDM-323 (Agent Certificate Rotation)  
**Story Points:** 8  
**Priority:** High

## Overview

Create comprehensive integration tests for certificate rotation flows to validate end-to-end certificate operations with real components.

## Implementation Tasks

### Task 1: Test Proactive Renewal Flow

**File:** `test/integration/certificate_renewal_flow_test.go` (new)

**Objective:** Test complete proactive renewal flow.

**Test Scenario:**
1. Enroll device
2. Set certificate expiration to near future
3. Trigger renewal check
4. Verify CSR is generated
5. Verify CSR is submitted
6. Verify service validates and approves
7. Verify certificate is received
8. Verify certificate is validated
9. Verify atomic swap occurs
10. Verify device continues operating

**Test Cases:**
- Test renewal flow completes successfully
- Test renewal flow with validation failure
- Test renewal flow with service rejection
- Test renewal flow with network interruption

---

### Task 2: Test Expired Certificate Recovery

**File:** `test/integration/certificate_recovery_flow_test.go` (new)

**Objective:** Test complete expired certificate recovery flow.

**Test Scenario:**
1. Enroll device
2. Expire management certificate
3. Trigger recovery
4. Verify bootstrap certificate fallback
5. Verify recovery CSR is generated
6. Verify TPM attestation is included (if needed)
7. Verify service validates recovery
8. Verify certificate is issued
9. Verify certificate is installed
10. Verify device resumes operation

**Test Cases:**
- Test recovery with bootstrap certificate
- Test recovery with TPM attestation
- Test recovery with expired bootstrap (TPM only)
- Test recovery validation failure
- Test recovery with service rejection

---

### Task 3: Test Bootstrap Certificate Fallback

**File:** `test/integration/bootstrap_certificate_fallback_test.go` (new)

**Objective:** Test bootstrap certificate fallback mechanism.

**Test Scenario:**
1. Enroll device
2. Expire management certificate
3. Verify bootstrap certificate is loaded
4. Verify bootstrap certificate is validated
5. Verify authentication uses bootstrap certificate
6. Verify renewal requests succeed

**Test Cases:**
- Test fallback to bootstrap when management expired
- Test bootstrap certificate validation
- Test authentication with bootstrap certificate
- Test fallback when bootstrap also expired

---

### Task 4: Test Atomic Swap Operations

**File:** `test/integration/certificate_atomic_swap_test.go` (new)

**Objective:** Test atomic swap under various conditions.

**Test Scenarios:**
1. Normal swap
2. Swap with validation failure
3. Swap with rollback
4. Swap with power loss simulation

**Test Cases:**
- Test atomic swap succeeds
- Test atomic swap with validation failure
- Test atomic swap rollback
- Test atomic swap after power loss
- Test concurrent swap attempts

---

### Task 5: Test Retry Logic

**File:** `test/integration/certificate_retry_test.go` (new)

**Objective:** Test retry logic with exponential backoff.

**Test Scenarios:**
1. Network interruption during renewal
2. Service unavailable during renewal
3. Retry with exponential backoff
4. Retry success after service recovery

**Test Cases:**
- Test retry on network error
- Test retry on service unavailable
- Test exponential backoff timing
- Test retry success
- Test max retries limit

---

### Task 6: Test Certificate Validation

**File:** `test/integration/certificate_validation_test.go` (new)

**Objective:** Test certificate validation in integration.

**Test Scenarios:**
1. Valid certificate passes validation
2. Expired certificate fails validation
3. Wrong identity fails validation
4. Invalid signature fails validation
5. Mismatched key pair fails validation

**Test Cases:**
- Test validation with valid certificate
- Test validation with expired certificate
- Test validation with wrong identity
- Test validation with invalid signature
- Test validation with mismatched key pair

---

### Task 7: Test TPM Attestation

**File:** `test/integration/tpm_attestation_test.go` (new)

**Objective:** Test TPM attestation generation and validation.

**Test Scenarios:**
1. Generate TPM attestation
2. Include attestation in CSR
3. Service validates attestation
4. Certificate is issued

**Test Cases:**
- Test TPM attestation generation
- Test attestation in CSR
- Test service-side attestation validation
- Test attestation with TPM simulator

---

### Task 8: Test Service-Side Validation

**File:** `test/integration/service_renewal_validation_test.go` (new)

**Objective:** Test service-side renewal and recovery validation.

**Test Scenarios:**
1. Proactive renewal validation
2. Expired certificate recovery validation
3. Bootstrap certificate validation
4. TPM attestation validation
5. Auto-approval logic

**Test Cases:**
- Test proactive renewal validation
- Test recovery validation
- Test bootstrap certificate acceptance
- Test TPM attestation validation
- Test auto-approval
- Test validation rejection

---

## Test Infrastructure

### Test Harness Setup
- Use existing test harness for agent/service setup
- Create test certificates with configurable expiration
- Use TPM simulator for TPM tests
- Mock network interruptions
- Mock service unavailability

### Test Data
- Test certificates (valid, expired, expiring soon)
- Test device configurations
- Test TPM attestation data

---

## Definition of Done

- [ ] Integration tests written for all flows
- [ ] All integration tests passing
- [ ] Tests use real components (not just mocks)
- [ ] Error scenarios tested
- [ ] Retry logic tested
- [ ] Code reviewed and approved
- [ ] Test documentation updated

---

## Related Files

- `test/integration/` - Integration test directory
- `test/harness/` - Test harness infrastructure

---

## Dependencies

- **All Implementation Stories**: EPIC-1 through EPIC-5 must be completed
- **Test Infrastructure**: Existing test harness and infrastructure

---

## Notes

- **Real Components**: Integration tests should use real components where possible (not just mocks)
- **Test Isolation**: Each test should be independent and clean up after itself
- **Test Data**: Use test certificates and configurations
- **TPM Simulator**: Use TPM simulator for TPM-related tests
- **Network Simulation**: Simulate network interruptions and service unavailability

---

**Document End**

