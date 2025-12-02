# Test Story: Recovery CSR Generation and Submission

**Story ID:** EDM-323-EPIC-4-STORY-5-TEST  
**Developer Story:** EDM-323-EPIC-4-STORY-5-DEV  
**Epic:** EDM-323-EPIC-4 (Expired Certificate Recovery)  
**Feature:** EDM-323 (Agent Certificate Rotation)  
**Story Points:** 3  
**Priority:** High

## Overview

Comprehensive test plan for recovery CSR generation and submission including recovery context, TPM attestation inclusion, bootstrap authentication, and CSR submission.

## Test Objectives

1. Verify recovery CSR includes recovery context
2. Verify TPM attestation is included
3. Verify bootstrap certificate authentication
4. Verify CSR submission works
5. Verify error handling

## Test Scope

### In Scope
- Recovery CSR generation
- Recovery context inclusion
- TPM attestation inclusion
- Bootstrap authentication
- CSR submission

### Out of Scope
- Service-side validation (covered in STORY-4)
- Certificate reception (covered in EPIC-2)

## Unit Tests

### Test File: `internal/agent/device/certmanager/provider/provisioner/csr_recovery_test.go`

#### Test Suite 1: Recovery CSR Generation

**Test Case 1.1: Generate Recovery CSR**
- **Setup:** Recovery context, TPM attestation
- **Action:** Generate recovery CSR
- **Expected:** CSR generated with recovery labels
- **Assertions:**
  - CSR generated
  - Recovery labels present
  - TPM attestation included

**Test Case 1.2: Recovery Context in CSR**
- **Setup:** Recovery context
- **Action:** Generate recovery CSR
- **Expected:** Recovery context in labels
- **Assertions:**
  - Recovery reason label present
  - Recovery metadata present

---

#### Test Suite 2: TPM Attestation Inclusion

**Test Case 2.1: Include TPM Attestation**
- **Setup:** TPM attestation available
- **Action:** Generate recovery CSR
- **Expected:** Attestation included in CSR
- **Assertions:**
  - TPM quote included
  - PCR values included
  - Device fingerprint included

**Test Case 2.2: Missing TPM Attestation**
- **Setup:** No TPM attestation
- **Action:** Generate recovery CSR
- **Expected:** Returns error or handles gracefully
- **Assertions:**
  - Error returned or handled
  - CSR not generated

---

#### Test Suite 3: Bootstrap Authentication

**Test Case 3.1: Authenticate with Bootstrap Certificate**
- **Setup:** Bootstrap certificate available
- **Action:** Submit recovery CSR
- **Expected:** CSR authenticated with bootstrap
- **Assertions:**
  - Bootstrap certificate used
  - Authentication succeeds
  - No error

**Test Case 3.2: Bootstrap Certificate Missing**
- **Setup:** No bootstrap certificate
- **Action:** Submit recovery CSR
- **Expected:** Returns error
- **Assertions:**
  - Error returned
  - Error message clear

---

## Integration Tests

### Test File: `test/integration/recovery_csr_test.go`

#### Test Suite 4: Recovery CSR Flow

**Test Case 4.1: Complete Recovery CSR Flow**
- **Setup:** Agent with expired certificate
- **Action:** Generate and submit recovery CSR
- **Expected:** CSR generated and submitted
- **Assertions:**
  - CSR generated
  - CSR submitted
  - Authentication works

---

## Test Coverage Requirements

### Code Coverage Targets
- **Overall Coverage:** >80% for recovery CSR code

---

## Definition of Done

- [ ] All unit tests written and passing
- [ ] All integration tests written and passing
- [ ] Code coverage >80% achieved
- [ ] Test documentation complete
- [ ] QA sign-off obtained

---

**Document End**

