# Test Story: Device Store Certificate Tracking Methods

**Story ID:** EDM-323-EPIC-7-STORY-2-TEST  
**Developer Story:** EDM-323-EPIC-7-STORY-2-DEV  
**Epic:** EDM-323-EPIC-7 (Database and API Enhancements)  
**Feature:** EDM-323 (Agent Certificate Rotation)  
**Story Points:** 3  
**Priority:** Medium

## Overview

Comprehensive test plan for device store certificate tracking methods including update methods, query methods, and data accuracy.

## Test Objectives

1. Verify update methods work correctly
2. Verify query methods work correctly
3. Verify data accuracy
4. Verify transaction safety
5. Verify error handling

## Test Scope

### In Scope
- Update methods
- Query methods
- Data accuracy
- Transaction safety
- Error handling

### Out of Scope
- Database schema (covered in EPIC-1)
- Device model (covered in EPIC-1)

## Unit Tests

### Test File: `internal/store/device_certificate_tracking_test.go`

#### Test Suite 1: Update Methods

**Test Case 1.1: Update Certificate Expiration**
- **Setup:** Device in database
- **Action:** Update certificate expiration
- **Expected:** Expiration updated
- **Assertions:**
  - Expiration updated
  - Other fields unchanged
  - No error

**Test Case 1.2: Update Renewal Info**
- **Setup:** Device in database
- **Action:** Update renewal info
- **Expected:** Renewal info updated
- **Assertions:**
  - Last renewed updated
  - Renewal count incremented
  - No error

**Test Case 1.3: Update Fingerprint**
- **Setup:** Device in database
- **Action:** Update fingerprint
- **Expected:** Fingerprint updated
- **Assertions:**
  - Fingerprint updated
  - Other fields unchanged
  - No error

---

#### Test Suite 2: Query Methods

**Test Case 2.1: Query Devices by Expiration**
- **Setup:** Devices with various expiration dates
- **Action:** Query devices expiring soon
- **Expected:** Correct devices returned
- **Assertions:**
  - Correct devices returned
  - Results accurate
  - No error

**Test Case 2.2: Query Devices by Renewal Count**
- **Setup:** Devices with various renewal counts
- **Action:** Query devices by renewal count
- **Expected:** Correct devices returned
- **Assertions:**
  - Correct devices returned
  - Results accurate
  - No error

---

## Integration Tests

### Test File: `test/integration/device_certificate_tracking_test.go`

#### Test Suite 3: Tracking Integration

**Test Case 3.1: Complete Tracking Flow**
- **Setup:** Service with devices
- **Action:** Update and query certificate tracking
- **Expected:** Tracking works correctly
- **Assertions:**
  - Updates work
  - Queries work
  - Data accurate

---

## Test Coverage Requirements

### Code Coverage Targets
- **Overall Coverage:** >80% for tracking code

---

## Definition of Done

- [ ] All unit tests written and passing
- [ ] All integration tests written and passing
- [ ] Code coverage >80% achieved
- [ ] Test documentation complete
- [ ] QA sign-off obtained

---

**Document End**

