# Developer Story: End-to-End Test Scenarios

**Story ID:** EDM-323-EPIC-6-STORY-3  
**Epic:** EDM-323-EPIC-6 (Testing and Validation)  
**Feature:** EDM-323 (Agent Certificate Rotation)  
**Story Points:** 5  
**Priority:** High

## Overview

Create end-to-end tests for complete user scenarios including enrollment → automatic renewal, offline → expired → recovery, and network interruption scenarios.

## Implementation Tasks

### Task 1: Test Enrollment → Automatic Renewal

**File:** `test/e2e/enrollment_to_renewal_test.go` (new)

**Objective:** Test complete flow from enrollment to automatic renewal.

**Test Scenario:**
1. Enroll new device
2. Verify device is operational
3. Fast-forward time to near certificate expiration
4. Verify automatic renewal is triggered
5. Verify renewal completes successfully
6. Verify device continues operating with new certificate
7. Verify old certificate is replaced

**Test Cases:**
- Test enrollment → renewal flow completes
- Test renewal occurs before expiration
- Test device continues operating during renewal
- Test new certificate is active after renewal

---

### Task 2: Test Offline → Expired → Recovery

**File:** `test/e2e/offline_expired_recovery_test.go` (new)

**Objective:** Test device going offline, certificate expiring, then recovery.

**Test Scenario:**
1. Enroll device
2. Verify device is operational
3. Simulate device going offline
4. Fast-forward time to certificate expiration
5. Bring device online
6. Verify automatic recovery is triggered
7. Verify recovery completes successfully
8. Verify device resumes operation

**Test Cases:**
- Test offline → expired → recovery flow
- Test recovery uses bootstrap certificate
- Test recovery uses TPM attestation if bootstrap expired
- Test device resumes operation after recovery

---

### Task 3: Test Network Interruption During Renewal

**File:** `test/e2e/network_interruption_renewal_test.go` (new)

**Objective:** Test renewal continues after network recovery.

**Test Scenario:**
1. Enroll device
2. Trigger renewal
3. Interrupt network during renewal
4. Restore network
5. Verify renewal continues
6. Verify renewal completes successfully

**Test Cases:**
- Test renewal continues after network recovery
- Test retry logic works correctly
- Test renewal completes successfully
- Test no duplicate renewals occur

---

### Task 4: Test Service Unavailable During Renewal

**File:** `test/e2e/service_unavailable_renewal_test.go` (new)

**Objective:** Test retry behavior when service becomes available.

**Test Scenario:**
1. Enroll device
2. Trigger renewal
3. Make service unavailable
4. Verify retry attempts with exponential backoff
5. Restore service
6. Verify renewal completes successfully

**Test Cases:**
- Test retry on service unavailable
- Test exponential backoff timing
- Test renewal completes when service available
- Test max retries limit

---

### Task 5: Test Certificate Validation Failure

**File:** `test/e2e/validation_failure_rollback_test.go` (new)

**Objective:** Test rollback and retry on validation failure.

**Test Scenario:**
1. Enroll device
2. Trigger renewal with invalid certificate (simulated)
3. Verify validation fails
4. Verify rollback occurs
5. Verify old certificate is preserved
6. Trigger renewal with valid certificate
7. Verify renewal succeeds

**Test Cases:**
- Test validation failure triggers rollback
- Test old certificate is preserved
- Test retry with valid certificate succeeds
- Test device continues operating

---

### Task 6: Test Multiple Renewals

**File:** `test/e2e/multiple_renewals_test.go` (new)

**Objective:** Test multiple renewal cycles.

**Test Scenario:**
1. Enroll device
2. Trigger multiple renewal cycles
3. Verify each renewal completes successfully
4. Verify renewal count increments
5. Verify device continues operating

**Test Cases:**
- Test multiple renewals succeed
- Test renewal count increments
- Test device continues operating
- Test certificate tracking is updated

---

## E2E Test Infrastructure

### Test Environment
- Full test environment with agent and service
- Real or simulated TPM
- Network simulation capabilities
- Service availability simulation
- Time manipulation for expiration testing

### Test Utilities
- Time manipulation utilities
- Network simulation utilities
- Service availability simulation
- Certificate expiration simulation

---

## Definition of Done

- [ ] E2E tests written for all scenarios
- [ ] All E2E tests passing
- [ ] Tests use real components
- [ ] Failure scenarios tested
- [ ] Recovery scenarios tested
- [ ] Code reviewed and approved
- [ ] Test documentation updated

---

## Related Files

- `test/e2e/` - End-to-end test directory
- `test/harness/` - Test harness infrastructure

---

## Dependencies

- **All Implementation Stories**: EPIC-1 through EPIC-5 must be completed
- **E2E Test Infrastructure**: End-to-end test infrastructure

---

## Notes

- **Real Components**: E2E tests use real components (agent, service, database)
- **Time Manipulation**: Use time manipulation to test expiration scenarios
- **Network Simulation**: Simulate network interruptions and recovery
- **Service Simulation**: Simulate service unavailability and recovery
- **Test Duration**: E2E tests may take longer to run than unit/integration tests

---

**Document End**

