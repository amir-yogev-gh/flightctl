# Test Story: Atomic Certificate Swap Operation

**Story ID:** EDM-323-EPIC-3-STORY-3-TEST  
**Developer Story:** EDM-323-EPIC-3-STORY-3-DEV  
**Epic:** EDM-323-EPIC-3 (Atomic Certificate Swap)  
**Feature:** EDM-323 (Agent Certificate Rotation)  
**Story Points:** 5  
**Priority:** High

## Overview

Comprehensive test plan for atomic certificate swap operations including POSIX atomic rename, backup creation, rollback, and power loss resilience.

## Test Objectives

1. Verify atomic swap uses POSIX rename correctly
2. Verify backup is created before swap
3. Verify rollback works on failure
4. Verify power loss resilience
5. Verify concurrent swap handling

## Test Scope

### In Scope
- AtomicSwap method
- Backup creation
- Rollback logic
- Power loss scenarios
- Concurrent operations

### Out of Scope
- Rollback mechanism details (covered in STORY-4)

## Unit Tests

### Test File: `internal/agent/device/certmanager/swap_atomic_test.go`

#### Test Suite 1: Atomic Swap

**Test Case 1.1: Successful Atomic Swap**
- **Setup:** Valid pending certificate
- **Action:** Call AtomicSwap()
- **Expected:** Certificate and key swapped atomically
- **Assertions:**
  - Pending cert becomes active
  - Pending key becomes active
  - Old files removed
  - No error

**Test Case 1.2: Certificate Swap Failure**
- **Setup:** Simulate certificate swap failure
- **Action:** Call AtomicSwap()
- **Expected:** Swap fails, old certificate preserved
- **Assertions:**
  - Error returned
  - Old certificate preserved
  - Pending certificate preserved

**Test Case 1.3: Key Swap Failure**
- **Setup:** Simulate key swap failure
- **Action:** Call AtomicSwap()
- **Expected:** Rollback occurs, old certificate restored
- **Assertions:**
  - Error returned
  - Rollback triggered
  - Old certificate restored

---

#### Test Suite 2: Backup Operations

**Test Case 2.1: Backup Created Before Swap**
- **Setup:** Active certificate exists
- **Action:** Call AtomicSwap()
- **Expected:** Backup created before swap
- **Assertions:**
  - Backup files created
  - Backup contains old certificate
  - Backup created before swap

**Test Case 2.2: No Backup for First Certificate**
- **Setup:** No active certificate (first certificate)
- **Action:** Call AtomicSwap()
- **Expected:** No backup created, swap succeeds
- **Assertions:**
  - No backup files
  - Swap succeeds
  - No error

**Test Case 2.3: Backup Cleanup After Success**
- **Setup:** Active certificate, successful swap
- **Action:** Call AtomicSwap()
- **Expected:** Backup cleaned up after success
- **Assertions:**
  - Backup files removed
  - Swap succeeds
  - No backup remains

---

#### Test Suite 3: Rollback

**Test Case 3.1: Rollback on Certificate Swap Failure**
- **Setup:** Certificate swap fails
- **Action:** Trigger rollback
- **Expected:** Old certificate restored
- **Assertions:**
  - Old certificate restored
  - Pending certificate preserved
  - No error

**Test Case 3.2: Rollback on Key Swap Failure**
- **Setup:** Key swap fails
- **Action:** Trigger rollback
- **Expected:** Both certificate and key restored
- **Assertions:**
  - Certificate restored
  - Key restored
  - State consistent

---

#### Test Suite 4: Power Loss Resilience

**Test Case 4.1: Power Loss During Certificate Swap**
- **Setup:** Simulate power loss during certificate rename
- **Action:** Check state after recovery
- **Expected:** Either old or new certificate is active
- **Assertions:**
  - At least one certificate active
  - Never both or neither
  - State is consistent

**Test Case 4.2: Power Loss During Key Swap**
- **Setup:** Simulate power loss during key rename
- **Action:** Check state after recovery
- **Expected:** Certificate/key mismatch detected
- **Assertions:**
  - Mismatch detected
  - Recovery possible
  - State handled correctly

---

## Integration Tests

### Test File: `test/integration/certificate_atomic_swap_test.go`

#### Test Suite 5: Atomic Swap Integration

**Test Case 5.1: Complete Atomic Swap Flow**
- **Setup:** Agent with validated pending certificate
- **Action:** Perform atomic swap
- **Expected:** Swap succeeds, new certificate active
- **Assertions:**
  - Swap succeeds
  - New certificate active
  - Old certificate removed
  - Device continues operating

---

## Test Coverage Requirements

### Code Coverage Targets
- **Overall Coverage:** >80% for atomic swap code

---

## Definition of Done

- [ ] All unit tests written and passing
- [ ] All integration tests written and passing
- [ ] Code coverage >80% achieved
- [ ] Power loss scenarios tested
- [ ] Test documentation complete
- [ ] QA sign-off obtained

---

**Document End**

