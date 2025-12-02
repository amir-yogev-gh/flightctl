# Test Story: Service-Side Recovery Request Validation

**Story ID:** EDM-323-EPIC-4-STORY-4-TEST  
**Developer Story:** EDM-323-EPIC-4-STORY-4-DEV  
**Epic:** EDM-323-EPIC-4 (Expired Certificate Recovery)  
**Feature:** EDM-323 (Agent Certificate Rotation)  
**Story Points:** 5  
**Priority:** High

## Overview

Comprehensive test plan for service-side recovery request validation including recovery request detection, TPM attestation verification, device verification, and auto-approval.

## Test Objectives

1. Verify recovery request detection works correctly
2. Verify TPM attestation verification
3. Verify device verification
4. Verify auto-approval works
5. Verify invalid requests are rejected

## Test Scope

### In Scope
- Recovery request detection
- TPM attestation verification
- Device verification
- Auto-approval
- Error handling

### Out of Scope
- Certificate issuance (covered in EPIC-2)

## Unit Tests

### Test File: `internal/service/certificatesigningrequest_recovery_test.go`

#### Test Suite 1: Recovery Request Detection

**Test Case 1.1: Detect Recovery Request**
- **Setup:** CSR with recovery labels
- **Action:** Call isRecoveryRequest()
- **Expected:** Returns true
- **Assertions:**
  - Result == true

**Test Case 1.2: Non-Recovery Request**
- **Setup:** CSR without recovery labels
- **Action:** Call isRecoveryRequest()
- **Expected:** Returns false
- **Assertions:**
  - Result == false

---

#### Test Suite 2: TPM Attestation Verification

**Test Case 2.1: Verify Valid Attestation**
- **Setup:** CSR with valid TPM attestation
- **Action:** Verify attestation
- **Expected:** Verification succeeds
- **Assertions:**
  - Verification succeeds
  - No error

**Test Case 2.2: Verify Invalid Attestation**
- **Setup:** CSR with invalid TPM attestation
- **Action:** Verify attestation
- **Expected:** Verification fails
- **Assertions:**
  - Verification fails
  - Error returned

**Test Case 2.3: Verify Missing Attestation**
- **Setup:** CSR without TPM attestation
- **Action:** Verify attestation
- **Expected:** Verification fails
- **Assertions:**
  - Verification fails
  - Error returned

---

#### Test Suite 3: Device Verification

**Test Case 3.1: Verify Valid Device**
- **Setup:** CSR from existing enrolled device
- **Action:** Verify device
- **Expected:** Device verified
- **Assertions:**
  - Device found
  - Device verified
  - No error

**Test Case 3.2: Verify Non-Existent Device**
- **Setup:** CSR from non-existent device
- **Action:** Verify device
- **Expected:** Verification fails
- **Assertions:**
  - Verification fails
  - Error returned

---

#### Test Suite 4: Auto-Approval

**Test Case 4.1: Auto-Approve Valid Recovery**
- **Setup:** Valid recovery request
- **Action:** Process recovery request
- **Expected:** Request auto-approved
- **Assertions:**
  - Approval condition set
  - Status updated

**Test Case 4.2: Reject Invalid Recovery**
- **Setup:** Invalid recovery request
- **Action:** Process recovery request
- **Expected:** Request rejected
- **Assertions:**
  - Rejection condition set
  - Error message included

---

## Integration Tests

### Test File: `test/integration/service_recovery_validation_test.go`

#### Test Suite 5: Recovery Validation Integration

**Test Case 5.1: Complete Recovery Validation**
- **Setup:** Service with enrolled device
- **Action:** Submit recovery CSR
- **Expected:** CSR validated and approved
- **Assertions:**
  - Validation succeeds
  - Auto-approval works
  - CSR ready for signing

---

## Test Coverage Requirements

### Code Coverage Targets
- **Overall Coverage:** >80% for recovery validation code

---

## Definition of Done

- [ ] All unit tests written and passing
- [ ] All integration tests written and passing
- [ ] Code coverage >80% achieved
- [ ] Test documentation complete
- [ ] QA sign-off obtained

---

**Document End**

