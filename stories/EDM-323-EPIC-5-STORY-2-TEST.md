# Test Story: Device Status Certificate Information

**Story ID:** EDM-323-EPIC-5-STORY-2-TEST  
**Developer Story:** EDM-323-EPIC-5-STORY-2-DEV  
**Epic:** EDM-323-EPIC-5 (Observability and Monitoring)  
**Feature:** EDM-323 (Agent Certificate Rotation)  
**Story Points:** 3  
**Priority:** Medium

## Overview

Comprehensive test plan for device status certificate information including status updates, API exposure, and status accuracy.

## Test Objectives

1. Verify certificate status is updated correctly
2. Verify status is exposed via API
3. Verify status values are accurate
4. Verify status updates on certificate changes

## Test Scope

### In Scope
- Certificate status updates
- API exposure
- Status accuracy
- Status updates

### Out of Scope
- Metrics (covered in STORY-1)
- Event logging (covered in STORY-3)

## Unit Tests

### Test File: `api/v1beta1/device_status_test.go`

#### Test Suite 1: Certificate Status Updates

**Test Case 1.1: Update Certificate Status**
- **Setup:** Device with certificate
- **Action:** Update certificate status
- **Expected:** Status updated correctly
- **Assertions:**
  - Expiration set
  - Days until expiration set
  - State set
  - Last renewed set
  - Renewal count set

**Test Case 1.2: Status on Certificate Renewal**
- **Setup:** Device with certificate renewed
- **Action:** Update certificate status
- **Expected:** Status reflects renewal
- **Assertions:**
  - Last renewed updated
  - Renewal count incremented
  - Expiration updated

---

#### Test Suite 2: API Exposure

**Test Case 2.1: Status Exposed via API**
- **Setup:** Device with certificate status
- **Action:** Query device status API
- **Expected:** Certificate status returned
- **Assertions:**
  - Status in API response
  - All fields present
  - Values accurate

**Test Case 2.2: Status JSON Format**
- **Setup:** Device with certificate status
- **Action:** Query device status API
- **Expected:** Status in correct JSON format
- **Assertions:**
  - JSON valid
  - Fields match schema
  - Values correct

---

## Integration Tests

### Test File: `test/integration/device_status_certificate_test.go`

#### Test Suite 3: Status Integration

**Test Case 3.1: Status Updates on Certificate Changes**
- **Setup:** Device with certificate
- **Action:** Renew certificate
- **Expected:** Status updated automatically
- **Assertions:**
  - Status updated
  - Values accurate
  - API reflects changes

---

## Test Coverage Requirements

### Code Coverage Targets
- **Overall Coverage:** >80% for status code

---

## Definition of Done

- [ ] All unit tests written and passing
- [ ] All integration tests written and passing
- [ ] Code coverage >80% achieved
- [ ] Test documentation complete
- [ ] QA sign-off obtained

---

**Document End**

