# Test Story: Certificate Renewal Event Aggregation

**Story ID:** EDM-323-EPIC-6-STORY-3-TEST  
**Developer Story:** EDM-323-EPIC-6-STORY-3-DEV  
**Epic:** EDM-323-EPIC-6 (Audit Trail and Compliance)  
**Feature:** EDM-323 (Agent Certificate Rotation)  
**Story Points:** 3  
**Priority:** Medium

## Overview

Comprehensive test plan for certificate renewal event aggregation including aggregation queries, statistics, and reporting.

## Test Objectives

1. Verify aggregation queries work correctly
2. Verify statistics are calculated correctly
3. Verify reporting works correctly
4. Verify aggregation performance

## Test Scope

### In Scope
- Aggregation queries
- Statistics calculation
- Reporting
- Performance

### Out of Scope
- Event store (covered in STORY-1)
- Event API (covered in STORY-2)

## Unit Tests

### Test File: `internal/store/certificate_event_aggregation_test.go`

#### Test Suite 1: Aggregation Queries

**Test Case 1.1: Aggregate by Device**
- **Setup:** Events for multiple devices
- **Action:** Aggregate by device
- **Expected:** Aggregated results returned
- **Assertions:**
  - Results grouped by device
  - Counts accurate
  - No error

**Test Case 1.2: Aggregate by Type**
- **Setup:** Events of multiple types
- **Action:** Aggregate by type
- **Expected:** Aggregated results returned
- **Assertions:**
  - Results grouped by type
  - Counts accurate
  - No error

---

#### Test Suite 2: Statistics

**Test Case 2.1: Calculate Renewal Statistics**
- **Setup:** Multiple renewal events
- **Action:** Calculate statistics
- **Expected:** Statistics calculated correctly
- **Assertions:**
  - Total renewals correct
  - Success rate correct
  - Average duration correct

---

## Integration Tests

### Test File: `test/integration/certificate_event_aggregation_test.go`

#### Test Suite 3: Aggregation Integration

**Test Case 3.1: Complete Aggregation Flow**
- **Setup:** Service with events
- **Action:** Aggregate and report
- **Expected:** Aggregation works correctly
- **Assertions:**
  - Queries work
  - Statistics accurate
  - Reports generated

---

## Test Coverage Requirements

### Code Coverage Targets
- **Overall Coverage:** >80% for aggregation code

---

## Definition of Done

- [ ] All unit tests written and passing
- [ ] All integration tests written and passing
- [ ] Code coverage >80% achieved
- [ ] Test documentation complete
- [ ] QA sign-off obtained

---

**Document End**

