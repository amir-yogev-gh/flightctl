# Test Story: Certificate Validation Before Activation

**Story ID:** EDM-323-EPIC-3-STORY-2-TEST  
**Developer Story:** EDM-323-EPIC-3-STORY-2-DEV  
**Epic:** EDM-323-EPIC-3 (Atomic Certificate Swap)  
**Feature:** EDM-323 (Agent Certificate Rotation)  
**Story Points:** 5  
**Priority:** High

## Overview

Comprehensive test plan for certificate validation before activation including CA bundle loading, signature verification, identity verification, expiration checks, and key pair verification.

## Test Objectives

1. Verify CA bundle loading works correctly
2. Verify certificate signature verification
3. Verify certificate identity verification
4. Verify expiration checks
5. Verify key pair verification
6. Verify complete validation flow

## Test Scope

### In Scope
- CA bundle loading
- Signature verification
- Identity verification
- Expiration checks
- Key pair verification
- Complete validation

### Out of Scope
- Atomic swap (covered in STORY-3)
- Rollback (covered in STORY-4)

## Unit Tests

### Test File: `internal/agent/device/certmanager/swap_validation_test.go`

#### Test Suite 1: CA Bundle Loading

**Test Case 1.1: Load Valid CA Bundle**
- **Setup:** Valid CA bundle file
- **Action:** Call loadCABundle()
- **Expected:** CA pool created
- **Assertions:**
  - CA pool created
  - Certificates parsed
  - No error

**Test Case 1.2: Load Missing CA Bundle**
- **Setup:** CA bundle file doesn't exist
- **Action:** Call loadCABundle()
- **Expected:** Returns error
- **Assertions:**
  - Error returned
  - Error message clear

**Test Case 1.3: Load Invalid CA Bundle**
- **Setup:** Invalid CA bundle file
- **Action:** Call loadCABundle()
- **Expected:** Returns error
- **Assertions:**
  - Error returned
  - Error indicates parsing failure

---

#### Test Suite 2: Signature Verification

**Test Case 2.1: Valid Signature**
- **Setup:** Certificate signed by CA in bundle
- **Action:** Call verifyCertificateSignature()
- **Expected:** Verification succeeds
- **Assertions:**
  - No error
  - Signature valid

**Test Case 2.2: Invalid Signature**
- **Setup:** Certificate with invalid signature
- **Action:** Call verifyCertificateSignature()
- **Expected:** Verification fails
- **Assertions:**
  - Error returned
  - Error indicates signature failure

**Test Case 2.3: Wrong CA**
- **Setup:** Certificate signed by different CA
- **Action:** Call verifyCertificateSignature()
- **Expected:** Verification fails
- **Assertions:**
  - Error returned
  - Error indicates CA mismatch

---

#### Test Suite 3: Identity Verification

**Test Case 3.1: Matching Identity**
- **Setup:** Certificate with matching CommonName
- **Action:** Call verifyCertificateIdentity()
- **Expected:** Verification succeeds
- **Assertions:**
  - No error
  - Identity matches

**Test Case 3.2: Mismatched Identity**
- **Setup:** Certificate with different CommonName
- **Action:** Call verifyCertificateIdentity()
- **Expected:** Verification fails
- **Assertions:**
  - Error returned
  - Error indicates identity mismatch

---

#### Test Suite 4: Expiration Checks

**Test Case 4.1: Valid Certificate**
- **Setup:** Certificate not expired, valid
- **Action:** Call verifyCertificateExpiration()
- **Expected:** Verification succeeds
- **Assertions:**
  - No error
  - Certificate valid

**Test Case 4.2: Expired Certificate**
- **Setup:** Expired certificate
- **Action:** Call verifyCertificateExpiration()
- **Expected:** Verification fails
- **Assertions:**
  - Error returned
  - Error indicates expiration

**Test Case 4.3: Not Yet Valid Certificate**
- **Setup:** Certificate with future NotBefore
- **Action:** Call verifyCertificateExpiration()
- **Expected:** Verification fails
- **Assertions:**
  - Error returned
  - Error indicates not yet valid

---

#### Test Suite 5: Key Pair Verification

**Test Case 5.1: Matching Key Pair**
- **Setup:** Certificate and key that match
- **Action:** Call verifyKeyPair()
- **Expected:** Verification succeeds
- **Assertions:**
  - No error
  - Key pair matches

**Test Case 5.2: Mismatched Key Pair**
- **Setup:** Certificate and key that don't match
- **Action:** Call verifyKeyPair()
- **Expected:** Verification fails
- **Assertions:**
  - Error returned
  - Error indicates mismatch

---

#### Test Suite 6: Complete Validation

**Test Case 6.1: Valid Certificate Passes All Checks**
- **Setup:** Valid certificate, key, CA bundle
- **Action:** Call ValidatePendingCertificate()
- **Expected:** All checks pass
- **Assertions:**
  - No errors
  - All validations succeed

**Test Case 6.2: Invalid Certificate Fails Validation**
- **Setup:** Invalid certificate (any failure)
- **Action:** Call ValidatePendingCertificate()
- **Expected:** Validation fails
- **Assertions:**
  - Error returned
  - Appropriate error message

---

## Integration Tests

### Test File: `test/integration/certificate_validation_test.go`

#### Test Suite 7: Validation Integration

**Test Case 7.1: Validation Before Activation**
- **Setup:** Agent with pending certificate
- **Action:** Validate before activation
- **Expected:** Validation succeeds, activation proceeds
- **Assertions:**
  - Validation succeeds
  - Activation proceeds

**Test Case 7.2: Validation Failure Prevents Activation**
- **Setup:** Agent with invalid pending certificate
- **Action:** Validate before activation
- **Expected:** Validation fails, activation prevented
- **Assertions:**
  - Validation fails
  - Activation prevented
  - Pending files cleaned up

---

## Test Coverage Requirements

### Code Coverage Targets
- **Overall Coverage:** >80% for validation code
- **Validation Coverage:** 100% for all validation steps

---

## Definition of Done

- [ ] All unit tests written and passing
- [ ] All integration tests written and passing
- [ ] Code coverage >80% achieved
- [ ] Test documentation complete
- [ ] QA sign-off obtained

---

**Document End**

