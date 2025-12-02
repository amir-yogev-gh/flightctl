# Test Story: Service-Side Renewal Request Validation

**Story ID:** EDM-323-EPIC-2-STORY-3-TEST  
**Developer Story:** EDM-323-EPIC-2-STORY-3-DEV  
**Epic:** EDM-323-EPIC-2 (Proactive Certificate Renewal)  
**Feature:** EDM-323 (Agent Certificate Rotation)  
**Story Points:** 5  
**Priority:** High

## Overview

Comprehensive test plan for service-side renewal request validation including request detection, device verification, certificate validation, and auto-approval.

## Test Objectives

1. Verify renewal request detection works correctly
2. Verify device verification logic
3. Verify certificate validation
4. Verify auto-approval works
5. Verify invalid requests are rejected

## Test Scope

### In Scope
- isRenewalRequest method
- getRenewalContext method
- extractDeviceNameFromCSR method
- validateRenewalRequest method
- autoApproveRenewal method

### Out of Scope
- Certificate issuance (covered in STORY-4)
- Recovery validation (covered in EPIC-4)

## Unit Tests

### Test File: `internal/service/certificatesigningrequest_renewal_test.go`

#### Test Suite 1: Renewal Request Detection

**Test Case 1.1: Detect Proactive Renewal**
- **Setup:** CSR with renewal-reason=proactive label
- **Action:** Call isRenewalRequest()
- **Expected:** Returns true
- **Assertions:**
  - Result == true

**Test Case 1.2: Detect Expired Renewal**
- **Setup:** CSR with renewal-reason=expired label
- **Action:** Call isRenewalRequest()
- **Expected:** Returns true
- **Assertions:**
  - Result == true

**Test Case 1.3: Non-Renewal Request**
- **Setup:** CSR without renewal labels
- **Action:** Call isRenewalRequest()
- **Expected:** Returns false
- **Assertions:**
  - Result == false

**Test Case 1.4: Get Renewal Context**
- **Setup:** CSR with renewal labels
- **Action:** Call getRenewalContext()
- **Expected:** Context extracted correctly
- **Assertions:**
  - Reason extracted
  - Threshold days extracted
  - Days until expiration extracted

---

#### Test Suite 2: Device Verification

**Test Case 2.1: Valid Device**
- **Setup:** CSR from existing enrolled device
- **Action:** Validate renewal request
- **Expected:** Device verified
- **Assertions:**
  - Device found
  - Device enrolled
  - Verification succeeds

**Test Case 2.2: Non-Existent Device**
- **Setup:** CSR from non-existent device
- **Action:** Validate renewal request
- **Expected:** Validation fails
- **Assertions:**
  - Error returned
  - Request rejected

**Test Case 2.3: Unenrolled Device**
- **Setup:** CSR from unenrolled device
- **Action:** Validate renewal request
- **Expected:** Validation fails
- **Assertions:**
  - Error returned
  - Request rejected

---

#### Test Suite 3: Certificate Validation

**Test Case 3.1: Valid Certificate**
- **Setup:** CSR with valid peer certificate
- **Action:** Validate renewal request
- **Expected:** Certificate validated
- **Assertions:**
  - Certificate valid
  - Identity matches
  - Validation succeeds

**Test Case 3.2: Expired Certificate**
- **Setup:** CSR with expired peer certificate
- **Action:** Validate renewal request
- **Expected:** Expired certificate accepted for renewal
- **Assertions:**
  - Expired certificate accepted
  - Validation succeeds

**Test Case 3.3: Wrong Identity**
- **Setup:** CSR with certificate from different device
- **Action:** Validate renewal request
- **Expected:** Validation fails
- **Assertions:**
  - Error returned
  - Request rejected

---

#### Test Suite 4: Auto-Approval

**Test Case 4.1: Auto-Approve Valid Renewal**
- **Setup:** Valid renewal request
- **Action:** Process renewal request
- **Expected:** Request auto-approved
- **Assertions:**
  - Approval condition set
  - Status updated

**Test Case 4.2: Reject Invalid Renewal**
- **Setup:** Invalid renewal request
- **Action:** Process renewal request
- **Expected:** Request rejected
- **Assertions:**
  - Rejection condition set
  - Error message included

---

## Integration Tests

### Test File: `test/integration/service_renewal_validation_test.go`

#### Test Suite 5: Renewal Validation Flow

**Test Case 5.1: Complete Renewal Validation**
- **Setup:** Service with enrolled device
- **Action:** Submit renewal CSR
- **Expected:** CSR validated and approved
- **Assertions:**
  - Validation succeeds
  - Auto-approval works
  - CSR ready for signing

---

## Test Coverage Requirements

### Code Coverage Targets
- **Overall Coverage:** >80% for renewal validation code

---

## Definition of Done

- [ ] All unit tests written and passing
- [ ] All integration tests written and passing
- [ ] Code coverage >80% achieved
- [ ] Test documentation complete
- [ ] QA sign-off obtained

---

**Document End**

