# Test Story: Agent-Side Certificate Renewal Trigger

**Story ID:** EDM-323-EPIC-2-STORY-1-TEST  
**Developer Story:** EDM-323-EPIC-2-STORY-1-DEV  
**Epic:** EDM-323-EPIC-2 (Proactive Certificate Renewal)  
**Feature:** EDM-323 (Agent Certificate Rotation)  
**Story Points:** 5  
**Priority:** High

## Overview

Comprehensive test plan for agent-side certificate renewal triggering including renewal checks, state transitions, and integration with sync flow.

## Test Objectives

1. Verify renewal check logic determines renewal need correctly
2. Verify renewal triggering sets state correctly
3. Verify renewal is queued for processing
4. Verify integration with sync flow works
5. Verify duplicate renewal prevention
6. Verify error handling

## Test Scope

### In Scope
- shouldRenewCertificate method
- triggerRenewal method
- Sync flow integration
- State transitions
- Duplicate prevention

### Out of Scope
- CSR generation (covered in STORY-2)
- Certificate issuance (covered in STORY-4)

## Unit Tests

### Test File: `internal/agent/device/certmanager/manager_renewal_test.go`

#### Test Suite 1: shouldRenewCertificate

**Test Case 1.1: Certificate Needs Renewal**
- **Setup:** Certificate expiring in 25 days, threshold = 30
- **Action:** Call shouldRenewCertificate()
- **Expected:** Returns true, days = 25
- **Assertions:**
  - Result == true
  - Days == 25
  - No error

**Test Case 1.2: Certificate Does Not Need Renewal**
- **Setup:** Certificate expiring in 35 days, threshold = 30
- **Action:** Call shouldRenewCertificate()
- **Expected:** Returns false, days = 35
- **Assertions:**
  - Result == false
  - Days == 35
  - No error

**Test Case 1.3: No Lifecycle Manager**
- **Setup:** CertManager without lifecycle manager
- **Action:** Call shouldRenewCertificate()
- **Expected:** Returns false, error
- **Assertions:**
  - Error returned
  - Result == false

**Test Case 1.4: Negative Threshold**
- **Setup:** Certificate with threshold = -1
- **Action:** Call shouldRenewCertificate()
- **Expected:** Returns false, error
- **Assertions:**
  - Error returned
  - Result == false

---

#### Test Suite 2: triggerRenewal

**Test Case 2.1: Successful Renewal Trigger**
- **Setup:** Certificate needing renewal
- **Action:** Call triggerRenewal()
- **Expected:** State set to renewing, certificate queued
- **Assertions:**
  - State == CertificateStateRenewing
  - Certificate queued
  - No error

**Test Case 2.2: Queuing Failure**
- **Setup:** Certificate needing renewal, queuing fails
- **Action:** Call triggerRenewal()
- **Expected:** State reset to expiring_soon, error returned
- **Assertions:**
  - State == CertificateStateExpiringSoon
  - Error returned

**Test Case 2.3: No Lifecycle Manager**
- **Setup:** CertManager without lifecycle manager
- **Action:** Call triggerRenewal()
- **Expected:** Returns error
- **Assertions:**
  - Error returned

---

#### Test Suite 3: Sync Flow Integration

**Test Case 3.1: Renewal Checked During Sync**
- **Setup:** Certificate expiring soon, sync triggered
- **Action:** Call sync()
- **Expected:** Renewal check performed
- **Assertions:**
  - Renewal check called
  - Renewal triggered if needed

**Test Case 3.2: Renewal Not Triggered if Already Renewing**
- **Setup:** Certificate in renewing state
- **Action:** Call sync()
- **Expected:** Renewal not triggered again
- **Assertions:**
  - Renewal not triggered
  - State remains renewing

**Test Case 3.3: Renewal Check Error Handling**
- **Setup:** Certificate with check error
- **Action:** Call sync()
- **Expected:** Error logged, sync continues
- **Assertions:**
  - Error logged
  - Sync continues
  - No panic

---

## Integration Tests

### Test File: `test/integration/certificate_renewal_trigger_test.go`

#### Test Suite 4: Renewal Trigger Flow

**Test Case 4.1: Automatic Renewal Trigger**
- **Setup:** Agent with certificate expiring soon
- **Action:** Wait for sync
- **Expected:** Renewal automatically triggered
- **Assertions:**
  - Renewal triggered
  - State set to renewing
  - Certificate queued

**Test Case 4.2: Renewal Not Triggered for Valid Certificate**
- **Setup:** Agent with certificate expiring in 60 days
- **Action:** Wait for sync
- **Expected:** Renewal not triggered
- **Assertions:**
  - Renewal not triggered
  - State remains normal

---

## Test Coverage Requirements

### Code Coverage Targets
- **Overall Coverage:** >80% for renewal trigger code
- **Function Coverage:** 100% for all public functions

---

## Definition of Done

- [ ] All unit tests written and passing
- [ ] All integration tests written and passing
- [ ] Code coverage >80% achieved
- [ ] Test documentation complete
- [ ] QA sign-off obtained

---

**Document End**

