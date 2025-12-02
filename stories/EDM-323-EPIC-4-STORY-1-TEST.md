# Test Story: Expired Certificate Detection and Recovery Trigger

**Story ID:** EDM-323-EPIC-4-STORY-1-TEST  
**Developer Story:** EDM-323-EPIC-4-STORY-1-DEV  
**Epic:** EDM-323-EPIC-4 (Expired Certificate Recovery)  
**Feature:** EDM-323 (Agent Certificate Rotation)  
**Story Points:** 3  
**Priority:** High

## Overview

Comprehensive test plan for expired certificate detection and recovery trigger including detection logic, recovery state transitions, and recovery initiation.

## Test Objectives

1. Verify expired certificate detection works correctly
2. Verify recovery state is set correctly
3. Verify recovery is triggered automatically
4. Verify recovery retry logic
5. Verify error handling

## Test Scope

### In Scope
- Expired certificate detection
- Recovery state transitions
- Recovery trigger
- Retry logic
- Error handling

### Out of Scope
- Bootstrap certificate (covered in STORY-2)
- TPM attestation (covered in STORY-3)

## Unit Tests

### Test File: `internal/agent/device/certmanager/recovery_detection_test.go`

#### Test Suite 1: Expired Certificate Detection

**Test Case 1.1: Detect Expired Certificate**
- **Setup:** Certificate expired 10 days ago
- **Action:** Check expiration
- **Expected:** Expired detected
- **Assertions:**
  - Expired == true
  - State set to expired

**Test Case 1.2: Detect Recently Expired Certificate**
- **Setup:** Certificate expired 1 hour ago
- **Action:** Check expiration
- **Expected:** Expired detected
- **Assertions:**
  - Expired == true
  - State set to expired

**Test Case 1.3: Valid Certificate Not Detected as Expired**
- **Setup:** Certificate expires in 30 days
- **Action:** Check expiration
- **Expected:** Not expired
- **Assertions:**
  - Expired == false
  - State remains normal

---

#### Test Suite 2: Recovery Trigger

**Test Case 2.1: Trigger Recovery for Expired Certificate**
- **Setup:** Expired certificate detected
- **Action:** Trigger recovery
- **Expected:** Recovery initiated
- **Assertions:**
  - State set to recovering
  - Recovery process started

**Test Case 2.2: Recovery Not Triggered for Valid Certificate**
- **Setup:** Valid certificate
- **Action:** Trigger recovery
- **Expected:** Recovery not triggered
- **Assertions:**
  - Recovery not triggered
  - State unchanged

---

## Integration Tests

### Test File: `test/integration/certificate_recovery_trigger_test.go`

#### Test Suite 3: Recovery Trigger Integration

**Test Case 3.1: Automatic Recovery Trigger**
- **Setup:** Agent with expired certificate
- **Action:** Wait for detection
- **Expected:** Recovery automatically triggered
- **Assertions:**
  - Recovery triggered
  - State set to recovering

---

## Test Coverage Requirements

### Code Coverage Targets
- **Overall Coverage:** >80% for recovery detection code

---

## Definition of Done

- [ ] All unit tests written and passing
- [ ] All integration tests written and passing
- [ ] Code coverage >80% achieved
- [ ] Test documentation complete
- [ ] QA sign-off obtained

---

**Document End**

