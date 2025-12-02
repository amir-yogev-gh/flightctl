# Test Story: CSR Generation for Certificate Renewal

**Story ID:** EDM-323-EPIC-2-STORY-2-TEST  
**Developer Story:** EDM-323-EPIC-2-STORY-2-DEV  
**Epic:** EDM-323-EPIC-2 (Proactive Certificate Renewal)  
**Feature:** EDM-323 (Agent Certificate Rotation)  
**Story Points:** 5  
**Priority:** High

## Overview

Comprehensive test plan for CSR generation with renewal context including renewal labels, device identity preservation, and authentication.

## Test Objectives

1. Verify renewal context is added to CSR
2. Verify renewal labels are included in CSR metadata
3. Verify device identity is preserved
4. Verify CSR is authenticated with current certificate
5. Verify CSR submission works correctly

## Test Scope

### In Scope
- RenewalContext struct
- CSR metadata labels
- Device identity preservation
- CSR authentication
- CSR submission

### Out of Scope
- Service-side validation (covered in STORY-3)
- Certificate issuance (covered in STORY-4)

## Unit Tests

### Test File: `internal/agent/device/certmanager/provider/provisioner/csr_renewal_test.go`

#### Test Suite 1: RenewalContext

**Test Case 1.1: RenewalContext Creation**
- **Setup:** Create RenewalContext
- **Action:** Create context with all fields
- **Expected:** All fields set correctly
- **Assertions:**
  - Reason set correctly
  - ThresholdDays set correctly
  - DaysUntilExpiration set correctly

**Test Case 1.2: RenewalContext JSON**
- **Setup:** Create RenewalContext
- **Action:** Marshal/unmarshal JSON
- **Expected:** JSON round-trip works
- **Assertions:**
  - JSON valid
  - Values preserved

---

#### Test Suite 2: CSR with Renewal Labels

**Test Case 2.1: Proactive Renewal Labels**
- **Setup:** Create CSR with proactive renewal context
- **Action:** Generate CSR
- **Expected:** Labels include renewal-reason=proactive
- **Assertions:**
  - Label flightctl.io/renewal-reason == "proactive"
  - Threshold days label present
  - Days until expiration label present

**Test Case 2.2: Expired Renewal Labels**
- **Setup:** Create CSR with expired renewal context
- **Action:** Generate CSR
- **Expected:** Labels include renewal-reason=expired
- **Assertions:**
  - Label flightctl.io/renewal-reason == "expired"
  - Other labels present

**Test Case 2.3: No Renewal Context**
- **Setup:** Create CSR without renewal context
- **Action:** Generate CSR
- **Expected:** No renewal labels
- **Assertions:**
  - No renewal labels in metadata
  - CSR generated normally

---

#### Test Suite 3: Device Identity Preservation

**Test Case 3.1: Same CommonName**
- **Setup:** Create renewal CSR
- **Action:** Generate CSR
- **Expected:** CommonName matches original device
- **Assertions:**
  - CommonName matches device name
  - Identity preserved

**Test Case 3.2: Same Identity Provider**
- **Setup:** Create renewal CSR
- **Action:** Generate CSR
- **Expected:** Uses same identity provider
- **Assertions:**
  - Same identity used
  - Key matches device

---

## Integration Tests

### Test File: `test/integration/csr_renewal_test.go`

#### Test Suite 4: CSR Renewal Flow

**Test Case 4.1: Renewal CSR Submission**
- **Setup:** Agent with expiring certificate
- **Action:** Trigger renewal
- **Expected:** Renewal CSR submitted with labels
- **Assertions:**
  - CSR submitted
  - Labels present
  - Authentication works

---

## Test Coverage Requirements

### Code Coverage Targets
- **Overall Coverage:** >80% for CSR renewal code

---

## Definition of Done

- [ ] All unit tests written and passing
- [ ] All integration tests written and passing
- [ ] Code coverage >80% achieved
- [ ] Test documentation complete
- [ ] QA sign-off obtained

---

**Document End**

