# Test Story: Certificate Renewal Event Store

**Story ID:** EDM-323-EPIC-6-STORY-1-TEST  
**Developer Story:** EDM-323-EPIC-6-STORY-1-DEV  
**Epic:** EDM-323-EPIC-6 (Audit Trail and Compliance)  
**Feature:** EDM-323 (Agent Certificate Rotation)  
**Story Points:** 3  
**Priority:** Medium

## Overview

Comprehensive test plan for certificate renewal event store including event creation, querying, filtering, and persistence.

## Test Objectives

1. Verify events are created correctly
2. Verify events are persisted to database
3. Verify querying works correctly
4. Verify filtering works correctly
5. Verify event data is accurate

## Test Scope

### In Scope
- Event creation
- Event persistence
- Event querying
- Event filtering
- Event data accuracy

### Out of Scope
- Event API (covered in STORY-2)
- Event aggregation (covered in STORY-3)

## Unit Tests

### Test File: `internal/store/certificate_renewal_event_test.go`

#### Test Suite 1: Event Creation

**Test Case 1.1: Create Renewal Event**
- **Setup:** Renewal event data
- **Action:** Create event
- **Expected:** Event created in database
- **Assertions:**
  - Event created
  - All fields set
  - No error

**Test Case 1.2: Create Recovery Event**
- **Setup:** Recovery event data
- **Action:** Create event
- **Expected:** Event created in database
- **Assertions:**
  - Event created
  - Event type correct
  - No error

---

#### Test Suite 2: Event Querying

**Test Case 2.1: Query Events by Device**
- **Setup:** Multiple events for different devices
- **Action:** Query events for specific device
- **Expected:** Only device events returned
- **Assertions:**
  - Correct events returned
  - Other events excluded
  - No error

**Test Case 2.2: Query Events by Type**
- **Setup:** Multiple event types
- **Action:** Query events by type
- **Expected:** Only matching events returned
- **Assertions:**
  - Correct events returned
  - Other events excluded
  - No error

**Test Case 2.3: Query Events by Date Range**
- **Setup:** Events across date range
- **Action:** Query events in range
- **Expected:** Only events in range returned
- **Assertions:**
  - Correct events returned
  - Events outside range excluded
  - No error

---

## Integration Tests

### Test File: `test/integration/certificate_event_store_test.go`

#### Test Suite 3: Event Store Integration

**Test Case 3.1: Complete Event Lifecycle**
- **Setup:** Service with event store
- **Action:** Create, query, filter events
- **Expected:** All operations work
- **Assertions:**
  - Events created
  - Events queried
  - Events filtered
  - Data accurate

---

## Test Coverage Requirements

### Code Coverage Targets
- **Overall Coverage:** >80% for event store code

---

## Definition of Done

- [ ] All unit tests written and passing
- [ ] All integration tests written and passing
- [ ] Code coverage >80% achieved
- [ ] Test documentation complete
- [ ] QA sign-off obtained

---

**Document End**

