# Test Story: Bootstrap Certificate Fallback

**Story ID:** EDM-323-EPIC-4-STORY-2-TEST  
**Developer Story:** EDM-323-EPIC-4-STORY-2-DEV  
**Epic:** EDM-323-EPIC-4 (Expired Certificate Recovery)  
**Feature:** EDM-323 (Agent Certificate Rotation)  
**Story Points:** 5  
**Priority:** High

## Overview

Comprehensive test plan for bootstrap certificate fallback including bootstrap certificate loading, validation, client switching, and fallback logic.

## Test Objectives

1. Verify bootstrap certificate loading works correctly
2. Verify bootstrap certificate validation
3. Verify client switching to bootstrap certificate
4. Verify fallback logic works
5. Verify error handling

## Test Scope

### In Scope
- Bootstrap certificate loading
- Bootstrap certificate validation
- Client switching
- Fallback logic
- Error handling

### Out of Scope
- TPM attestation (covered in STORY-3)
- Recovery CSR (covered in STORY-4)

## Unit Tests

### Test File: `internal/agent/device/bootstrap_certificate_test.go`

#### Test Suite 1: Bootstrap Certificate Loading

**Test Case 1.1: Load Valid Bootstrap Certificate**
- **Setup:** Valid bootstrap certificate file
- **Action:** Call GetBootstrapCertificate()
- **Expected:** Certificate loaded
- **Assertions:**
  - Certificate loaded
  - Key loaded
  - No error

**Test Case 1.2: Load Missing Bootstrap Certificate**
- **Setup:** Bootstrap certificate file doesn't exist
- **Action:** Call GetBootstrapCertificate()
- **Expected:** Returns error
- **Assertions:**
  - Error returned
  - Error message clear

**Test Case 1.3: Load Invalid Bootstrap Certificate**
- **Setup:** Invalid bootstrap certificate file
- **Action:** Call GetBootstrapCertificate()
- **Expected:** Returns error
- **Assertions:**
  - Error returned
  - Error indicates parsing failure

---

#### Test Suite 2: Bootstrap Certificate Validation

**Test Case 2.1: Valid Bootstrap Certificate**
- **Setup:** Valid bootstrap certificate
- **Action:** Validate certificate
- **Expected:** Validation succeeds
- **Assertions:**
  - No error
  - Certificate valid

**Test Case 2.2: Expired Bootstrap Certificate**
- **Setup:** Expired bootstrap certificate
- **Action:** Validate certificate
- **Expected:** Validation fails
- **Assertions:**
  - Error returned
  - Error indicates expiration

---

#### Test Suite 3: Client Switching

**Test Case 3.1: Switch to Bootstrap Client**
- **Setup:** Management client with expired certificate
- **Action:** Call UseBootstrapClient()
- **Expected:** Client switched to bootstrap certificate
- **Assertions:**
  - Client uses bootstrap certificate
  - TLS config updated
  - No error

**Test Case 3.2: Switch Fails with Invalid Bootstrap**
- **Setup:** Invalid bootstrap certificate
- **Action:** Call UseBootstrapClient()
- **Expected:** Returns error
- **Assertions:**
  - Error returned
  - Client unchanged

---

## Integration Tests

### Test File: `test/integration/bootstrap_fallback_test.go`

#### Test Suite 4: Bootstrap Fallback Integration

**Test Case 4.1: Fallback to Bootstrap on Expired Certificate**
- **Setup:** Agent with expired certificate
- **Action:** Attempt to connect
- **Expected:** Falls back to bootstrap certificate
- **Assertions:**
  - Bootstrap certificate used
  - Connection succeeds
  - Recovery can proceed

---

## Test Coverage Requirements

### Code Coverage Targets
- **Overall Coverage:** >80% for bootstrap fallback code

---

## Definition of Done

- [ ] All unit tests written and passing
- [ ] All integration tests written and passing
- [ ] Code coverage >80% achieved
- [ ] Test documentation complete
- [ ] QA sign-off obtained

---

**Document End**

