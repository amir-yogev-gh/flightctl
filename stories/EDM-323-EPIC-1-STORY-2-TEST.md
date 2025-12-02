# Test Story: Certificate Lifecycle Manager Structure

**Story ID:** EDM-323-EPIC-1-STORY-2-TEST  
**Developer Story:** EDM-323-EPIC-1-STORY-2-DEV  
**Epic:** EDM-323-EPIC-1 (Certificate Lifecycle Foundation)  
**Feature:** EDM-323 (Agent Certificate Rotation)  
**Story Points:** 5  
**Priority:** High

## Overview

Comprehensive test plan for certificate lifecycle manager structure including state definitions, state management, renewal checking, and integration with certificate manager.

## Test Objectives

1. Verify certificate lifecycle states are correctly defined and validated
2. Verify state transitions work correctly
3. Verify renewal checking logic determines renewal need accurately
4. Verify state updates persist correctly
5. Verify thread safety of state management
6. Verify integration with certificate manager

## Test Scope

### In Scope
- CertificateState enum and validation
- CertificateLifecycleState struct and methods
- LifecycleManager implementation
- Renewal checking logic
- State update operations
- Thread safety

### Out of Scope
- Actual renewal triggering (covered in EPIC-2)
- Certificate recovery (covered in EPIC-4)

## Test Environment Setup

### Prerequisites
- Go 1.21+ installed
- Test certificates with various expiration dates
- Mock logger implementation
- Test expiration monitor

## Unit Tests

### Test File: `internal/agent/device/certmanager/lifecycle_test.go`

#### Test Suite 1: CertificateState Enum

**Test Case 1.1: Valid States**
- **Setup:** Test all defined state constants
- **Action:** Check IsValidState() for each state
- **Expected:** All defined states return true
- **Assertions:**
  - CertificateStateNormal.IsValidState() == true
  - CertificateStateExpiringSoon.IsValidState() == true
  - CertificateStateExpired.IsValidState() == true
  - CertificateStateRenewing.IsValidState() == true
  - CertificateStateRecovering.IsValidState() == true
  - CertificateStateRenewalFailed.IsValidState() == true

**Test Case 1.2: Invalid State**
- **Setup:** Create invalid state value
- **Action:** Call IsValidState() on invalid state
- **Expected:** Returns false
- **Assertions:**
  - CertificateState("invalid").IsValidState() == false

**Test Case 1.3: String Representation**
- **Setup:** Test all state constants
- **Action:** Call String() on each state
- **Expected:** Returns correct string value
- **Assertions:**
  - CertificateStateNormal.String() == "normal"
  - CertificateStateExpiringSoon.String() == "expiring_soon"
  - CertificateStateExpired.String() == "expired"
  - CertificateStateRenewing.String() == "renewing"
  - CertificateStateRecovering.String() == "recovering"
  - CertificateStateRenewalFailed.String() == "renewal_failed"

---

#### Test Suite 2: CertificateLifecycleState Methods

**Test Case 2.1: NewCertificateLifecycleState**
- **Setup:** Create new lifecycle state
- **Action:** Call NewCertificateLifecycleState()
- **Expected:** Returns state with default values
- **Assertions:**
  - State == CertificateStateNormal
  - DaysUntilExpiration == 0
  - ExpirationTime == nil
  - LastError == ""

**Test Case 2.2: SetState and GetState**
- **Setup:** Create lifecycle state
- **Action:** Set state to CertificateStateExpiringSoon, then get state
- **Expected:** State is set and retrieved correctly
- **Assertions:**
  - SetState(CertificateStateExpiringSoon)
  - GetState() == CertificateStateExpiringSoon

**Test Case 2.3: SetDaysUntilExpiration and GetDaysUntilExpiration**
- **Setup:** Create lifecycle state
- **Action:** Set days to 25, then get days
- **Expected:** Days are set and retrieved correctly
- **Assertions:**
  - SetDaysUntilExpiration(25)
  - GetDaysUntilExpiration() == 25

**Test Case 2.4: SetExpirationTime and GetExpirationTime**
- **Setup:** Create lifecycle state, create expiration time
- **Action:** Set expiration time, then get expiration time
- **Expected:** Expiration time is set and retrieved correctly
- **Assertions:**
  - expirationTime := time.Now().Add(30 * 24 * time.Hour)
  - SetExpirationTime(expirationTime)
  - GetExpirationTime() == &expirationTime

**Test Case 2.5: SetLastChecked and GetLastChecked**
- **Setup:** Create lifecycle state
- **Action:** Set last checked time, then get last checked time
- **Expected:** Last checked time is set and retrieved correctly
- **Assertions:**
  - now := time.Now()
  - SetLastChecked(now)
  - GetLastChecked() == now

**Test Case 2.6: SetLastError and GetLastError**
- **Setup:** Create lifecycle state
- **Action:** Set error message, then get error message
- **Expected:** Error message is set and retrieved correctly
- **Assertions:**
  - SetLastError("test error")
  - GetLastError() == "test error"

**Test Case 2.7: Thread Safety - Concurrent Access**
- **Setup:** Create lifecycle state, start multiple goroutines
- **Action:** Concurrently set and get state values
- **Expected:** No race conditions, values are consistent
- **Assertions:**
  - No race conditions detected
  - Values are consistent after concurrent operations

---

#### Test Suite 3: CheckRenewal

**Test Case 3.1: Certificate Needs Renewal - Within Threshold**
- **Setup:** Certificate expiring in 25 days, threshold = 30
- **Action:** Call CheckRenewal(ctx, cert, 30)
- **Expected:** Returns true, no error
- **Assertions:**
  - Result == true
  - No error

**Test Case 3.2: Certificate Does Not Need Renewal - Beyond Threshold**
- **Setup:** Certificate expiring in 35 days, threshold = 30
- **Action:** Call CheckRenewal(ctx, cert, 30)
- **Expected:** Returns false, no error
- **Assertions:**
  - Result == false
  - No error

**Test Case 3.3: Certificate Needs Renewal - Exactly at Threshold**
- **Setup:** Certificate expiring in exactly 30 days, threshold = 30
- **Action:** Call CheckRenewal(ctx, cert, 30)
- **Expected:** Returns true, no error
- **Assertions:**
  - Result == true
  - No error

**Test Case 3.4: Expired Certificate**
- **Setup:** Certificate expired 10 days ago, threshold = 30
- **Action:** Call CheckRenewal(ctx, cert, 30)
- **Expected:** Returns false (expired, not renewal case), no error
- **Assertions:**
  - Result == false
  - No error

**Test Case 3.5: Nil Certificate**
- **Setup:** Pass nil certificate
- **Action:** Call CheckRenewal(ctx, nil, 30)
- **Expected:** Returns false, error
- **Assertions:**
  - Error returned
  - Result == false

**Test Case 3.6: Certificate with No Expiration Info**
- **Setup:** Certificate with nil NotAfter
- **Action:** Call CheckRenewal(ctx, cert, 30)
- **Expected:** Returns false, error
- **Assertions:**
  - Error returned
  - Result == false

---

#### Test Suite 4: GetCertificateState

**Test Case 4.1: Normal State**
- **Setup:** Certificate expiring in 60 days, threshold = 30
- **Action:** Call GetCertificateState(cert, 30)
- **Expected:** Returns CertificateStateNormal
- **Assertions:**
  - State == CertificateStateNormal

**Test Case 4.2: Expiring Soon State**
- **Setup:** Certificate expiring in 25 days, threshold = 30
- **Action:** Call GetCertificateState(cert, 30)
- **Expected:** Returns CertificateStateExpiringSoon
- **Assertions:**
  - State == CertificateStateExpiringSoon

**Test Case 4.3: Expired State**
- **Setup:** Certificate expired 10 days ago
- **Action:** Call GetCertificateState(cert, 30)
- **Expected:** Returns CertificateStateExpired
- **Assertions:**
  - State == CertificateStateExpired

**Test Case 4.4: Renewing State**
- **Setup:** Certificate with lifecycle state set to renewing
- **Action:** Call GetCertificateState(cert, 30)
- **Expected:** Returns CertificateStateRenewing
- **Assertions:**
  - State == CertificateStateRenewing

**Test Case 4.5: Recovering State**
- **Setup:** Certificate with lifecycle state set to recovering
- **Action:** Call GetCertificateState(cert, 30)
- **Expected:** Returns CertificateStateRecovering
- **Assertions:**
  - State == CertificateStateRecovering

**Test Case 4.6: Nil Certificate**
- **Setup:** Pass nil certificate
- **Action:** Call GetCertificateState(nil, 30)
- **Expected:** Returns CertificateStateRenewalFailed
- **Assertions:**
  - State == CertificateStateRenewalFailed

---

#### Test Suite 5: UpdateCertificateState

**Test Case 5.1: Update State with All Fields**
- **Setup:** Create certificate, create lifecycle manager
- **Action:** Call UpdateCertificateState with all fields
- **Expected:** All fields updated correctly
- **Assertions:**
  - State updated
  - DaysUntilExpiration updated
  - ExpirationTime updated
  - LastChecked updated
  - LastError updated (if provided)

**Test Case 5.2: Update State with Nil Expiration Time**
- **Setup:** Create certificate
- **Action:** Call UpdateCertificateState with nil expiration time
- **Expected:** ExpirationTime remains nil
- **Assertions:**
  - ExpirationTime == nil

**Test Case 5.3: Update State with No Error**
- **Setup:** Create certificate
- **Action:** Call UpdateCertificateState with nil error
- **Expected:** LastError is cleared
- **Assertions:**
  - LastError == ""

**Test Case 5.4: Update State Creates Lifecycle if Nil**
- **Setup:** Create certificate without lifecycle state
- **Action:** Call UpdateCertificateState
- **Expected:** Lifecycle state is created
- **Assertions:**
  - cert.Lifecycle != nil
  - State is set correctly

---

#### Test Suite 6: RecordError

**Test Case 6.1: Record Error**
- **Setup:** Create certificate, create error
- **Action:** Call RecordError(cert, err)
- **Expected:** Error recorded, state set to renewal_failed
- **Assertions:**
  - State == CertificateStateRenewalFailed
  - LastError contains error message
  - LastChecked is updated

**Test Case 6.2: Record Error Creates Lifecycle if Nil**
- **Setup:** Create certificate without lifecycle state
- **Action:** Call RecordError(cert, err)
- **Expected:** Lifecycle state is created
- **Assertions:**
  - cert.Lifecycle != nil
  - State == CertificateStateRenewalFailed

---

## Integration Tests

### Test File: `test/integration/certificate_lifecycle_test.go`

#### Test Suite 7: Lifecycle Manager Integration

**Test Case 7.1: Lifecycle Manager Initialization**
- **Setup:** Create certificate manager with lifecycle manager
- **Action:** Initialize lifecycle manager
- **Expected:** Lifecycle manager initialized correctly
- **Assertions:**
  - Lifecycle manager != nil
  - Expiration monitor initialized
  - Logger set correctly

**Test Case 7.2: State Transitions During Certificate Operations**
- **Setup:** Create certificate manager with lifecycle manager
- **Action:** 
  - Load certificate (normal state)
  - Certificate expires soon (expiring_soon state)
  - Trigger renewal (renewing state)
  - Renewal succeeds (normal state)
- **Expected:** States transition correctly
- **Assertions:**
  - State transitions follow expected flow
  - Each state is set correctly

**Test Case 7.3: Concurrent State Updates**
- **Setup:** Create certificate manager with lifecycle manager
- **Action:** Concurrently update state from multiple goroutines
- **Expected:** No race conditions, state is consistent
- **Assertions:**
  - No race conditions
  - Final state is consistent

---

## Test Coverage Requirements

### Code Coverage Targets
- **Overall Coverage:** >80% for lifecycle.go
- **Function Coverage:** 100% for all public functions
- **State Transition Coverage:** 100% for all state transitions
- **Error Path Coverage:** >90% for error handling

---

## Definition of Done

- [ ] All unit tests written and passing
- [ ] All integration tests written and passing
- [ ] Code coverage >80% achieved
- [ ] Thread safety verified
- [ ] State transitions tested
- [ ] Test documentation complete
- [ ] QA sign-off obtained

---

**Document End**

