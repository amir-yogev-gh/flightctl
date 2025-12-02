# Test Story: Rollback Mechanism for Failed Swaps

**Story ID:** EDM-323-EPIC-3-STORY-4-TEST  
**Developer Story:** EDM-323-EPIC-3-STORY-4-DEV  
**Epic:** EDM-323-EPIC-3 (Atomic Certificate Swap)  
**Feature:** EDM-323 (Agent Certificate Rotation)  
**Story Points:** 3  
**Priority:** High

## Overview

Comprehensive test plan for rollback mechanism including backup restoration, pending cleanup, state updates, and recovery detection.

## Test Objectives

1. Verify rollback restores from backup correctly
2. Verify pending files are cleaned up
3. Verify state is updated on rollback
4. Verify recovery detection works
5. Verify startup recovery

## Test Scope

### In Scope
- RollbackSwap method
- Backup restoration
- Pending cleanup
- State updates
- Recovery detection
- Startup recovery

### Out of Scope
- Atomic swap details (covered in STORY-3)

## Unit Tests

### Test File: `internal/agent/device/certmanager/swap_rollback_test.go`

#### Test Suite 1: Rollback Operations

**Test Case 1.1: Rollback with Backup**
- **Setup:** Failed swap with backup available
- **Action:** Call RollbackSwap()
- **Expected:** Backup restored, pending cleaned up
- **Assertions:**
  - Backup restored
  - Pending files cleaned up
  - Old certificate active

**Test Case 1.2: Rollback without Backup**
- **Setup:** Failed swap without backup
- **Action:** Call RollbackSwap()
- **Expected:** Pending cleaned up, warning logged
- **Assertions:**
  - Pending files cleaned up
  - Warning logged
  - No error

**Test Case 1.3: Rollback Cleans Up Pending**
- **Setup:** Failed swap with pending files
- **Action:** Call RollbackSwap()
- **Expected:** Pending files removed
- **Assertions:**
  - Pending cert removed
  - Pending key removed
  - No pending files remain

---

#### Test Suite 2: Recovery Detection

**Test Case 2.1: Detect Incomplete Swap**
- **Setup:** Pending certificate exists
- **Action:** Call DetectAndRecoverIncompleteSwap()
- **Expected:** Incomplete swap detected
- **Assertions:**
  - Detection succeeds
  - Recovery attempted

**Test Case 2.2: Recover Missing Active Certificate**
- **Setup:** Active certificate missing, pending exists
- **Action:** Call recovery
- **Expected:** Certificate restored or pending activated
- **Assertions:**
  - Recovery succeeds
  - Certificate available

**Test Case 2.3: Recover Certificate/Key Mismatch**
- **Setup:** Certificate/key mismatch detected
- **Action:** Call recovery
- **Expected:** Mismatch resolved
- **Assertions:**
  - Recovery succeeds
  - Certificate/key match

---

## Integration Tests

### Test File: `test/integration/certificate_rollback_test.go`

#### Test Suite 3: Rollback Integration

**Test Case 3.1: Rollback After Failed Swap**
- **Setup:** Agent with failed swap
- **Action:** Trigger rollback
- **Expected:** Old certificate restored
- **Assertions:**
  - Rollback succeeds
  - Old certificate active
  - Device continues operating

**Test Case 3.2: Startup Recovery**
- **Setup:** Agent restart with incomplete swap
- **Action:** Agent starts
- **Expected:** Recovery detected and handled
- **Assertions:**
  - Recovery detected
  - Recovery succeeds
  - Agent starts normally

---

## Test Coverage Requirements

### Code Coverage Targets
- **Overall Coverage:** >80% for rollback code

---

## Definition of Done

- [ ] All unit tests written and passing
- [ ] All integration tests written and passing
- [ ] Code coverage >80% achieved
- [ ] Test documentation complete
- [ ] QA sign-off obtained

---

**Document End**

