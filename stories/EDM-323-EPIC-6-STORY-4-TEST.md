# Test Story: Certificate Renewal Event Retention

**Story ID:** EDM-323-EPIC-6-STORY-4-TEST  
**Developer Story:** EDM-323-EPIC-6-STORY-4-DEV  
**Epic:** EDM-323-EPIC-6 (Audit Trail and Compliance)  
**Feature:** EDM-323 (Agent Certificate Rotation)  
**Story Points:** 3  
**Priority:** Medium

## Overview

Comprehensive test plan for certificate renewal event retention including retention policies, cleanup jobs, and retention enforcement.

## Test Objectives

1. Verify retention policies work correctly
2. Verify cleanup jobs run correctly
3. Verify retention enforcement works
4. Verify old events are deleted
5. Verify retention configuration

## Test Scope

### In Scope
- Retention policies
- Cleanup jobs
- Retention enforcement
- Event deletion
- Retention configuration

### Out of Scope
- Event store (covered in STORY-1)
- Event API (covered in STORY-2)

## Unit Tests

### Test File: `internal/store/certificate_event_retention_test.go`

#### Test Suite 1: Retention Policies

**Test Case 1.1: Apply Retention Policy**
- **Setup:** Events with various ages
- **Action:** Apply retention policy
- **Expected:** Old events identified
- **Assertions:**
  - Old events identified
  - Policy applied correctly
  - No error

**Test Case 1.2: Retention Configuration**
- **Setup:** Retention configuration
- **Action:** Apply retention
- **Expected:** Configuration respected
- **Assertions:**
  - Retention period respected
  - Events older than period identified

---

#### Test Suite 2: Cleanup Jobs

**Test Case 2.1: Cleanup Old Events**
- **Setup:** Events older than retention period
- **Action:** Run cleanup job
- **Expected:** Old events deleted
- **Assertions:**
  - Old events deleted
  - Recent events preserved
  - No error

**Test Case 2.2: Cleanup Job Scheduling**
- **Setup:** Cleanup job configured
- **Action:** Wait for scheduled run
- **Expected:** Job runs at scheduled time
- **Assertions:**
  - Job runs
  - Cleanup performed
  - Schedule respected

---

## Integration Tests

### Test File: `test/integration/certificate_event_retention_test.go`

#### Test Suite 3: Retention Integration

**Test Case 3.1: Complete Retention Flow**
- **Setup:** Service with events
- **Action:** Apply retention, run cleanup
- **Expected:** Retention works correctly
- **Assertions:**
  - Old events deleted
  - Recent events preserved
  - Retention policy enforced

---

## Test Coverage Requirements

### Code Coverage Targets
- **Overall Coverage:** >80% for retention code

---

## Definition of Done

- [ ] All unit tests written and passing
- [ ] All integration tests written and passing
- [ ] Code coverage >80% achieved
- [ ] Test documentation complete
- [ ] QA sign-off obtained

---

**Document End**

