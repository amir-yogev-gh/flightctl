# Test Story: Agent Certificate Reception and Storage

**Story ID:** EDM-323-EPIC-2-STORY-5-TEST  
**Developer Story:** EDM-323-EPIC-2-STORY-5-DEV  
**Epic:** EDM-323-EPIC-2 (Proactive Certificate Renewal)  
**Feature:** EDM-323 (Agent Certificate Rotation)  
**Story Points:** 3  
**Priority:** High

## Overview

Comprehensive test plan for agent certificate reception and storage including certificate validation, state updates, and storage operations.

## Test Objectives

1. Verify certificates are validated before storage
2. Verify certificate state is updated after reception
3. Verify storage operations work correctly
4. Verify invalid certificates are rejected
5. Verify error handling

## Test Scope

### In Scope
- Certificate validation
- Certificate reception
- State updates
- Storage operations
- Error handling

### Out of Scope
- Atomic swap (covered in EPIC-3)
- Certificate activation (covered in EPIC-3)

## Unit Tests

### Test File: `internal/agent/device/certmanager/provider/provisioner/csr_validation_test.go`

#### Test Suite 1: Certificate Validation

**Test Case 1.1: Valid Certificate**
- **Setup:** Valid certificate and key
- **Action:** Validate certificate
- **Expected:** Validation succeeds
- **Assertions:**
  - No errors
  - Certificate valid

**Test Case 1.2: Expired Certificate**
- **Setup:** Expired certificate
- **Action:** Validate certificate
- **Expected:** Validation fails
- **Assertions:**
  - Error returned
  - Certificate rejected

**Test Case 1.3: Wrong Identity**
- **Setup:** Certificate with wrong CommonName
- **Action:** Validate certificate
- **Expected:** Validation fails
- **Assertions:**
  - Error returned
  - Certificate rejected

**Test Case 1.4: Invalid Signature**
- **Setup:** Certificate with invalid signature
- **Action:** Validate certificate
- **Expected:** Validation fails
- **Assertions:**
  - Error returned
  - Certificate rejected

**Test Case 1.5: Mismatched Key Pair**
- **Setup:** Certificate and key that don't match
- **Action:** Validate certificate
- **Expected:** Validation fails
- **Assertions:**
  - Error returned
  - Certificate rejected

---

#### Test Suite 2: Certificate Reception

**Test Case 2.1: Receive Valid Certificate**
- **Setup:** CSR provisioner with approved CSR
- **Action:** Poll for certificate
- **Expected:** Certificate received and validated
- **Assertions:**
  - Certificate received
  - Validation succeeds
  - State updated

**Test Case 2.2: Reject Invalid Certificate**
- **Setup:** Invalid certificate in CSR status
- **Action:** Poll for certificate
- **Expected:** Certificate rejected
- **Assertions:**
  - Certificate rejected
  - Error logged
  - State updated to failed

---

#### Test Suite 3: State Updates

**Test Case 3.1: Update State After Reception**
- **Setup:** Certificate received
- **Action:** Update certificate state
- **Expected:** State updated to normal
- **Assertions:**
  - State == CertificateStateNormal
  - Expiration info updated

---

## Integration Tests

### Test File: `test/integration/certificate_reception_test.go`

#### Test Suite 4: Certificate Reception Flow

**Test Case 4.1: Complete Reception Flow**
- **Setup:** Agent with renewal CSR submitted
- **Action:** Wait for certificate issuance
- **Expected:** Certificate received and validated
- **Assertions:**
  - Certificate received
  - Validation succeeds
  - State updated

---

## Test Coverage Requirements

### Code Coverage Targets
- **Overall Coverage:** >80% for reception code

---

## Definition of Done

- [ ] All unit tests written and passing
- [ ] All integration tests written and passing
- [ ] Code coverage >80% achieved
- [ ] Test documentation complete
- [ ] QA sign-off obtained

---

**Document End**

