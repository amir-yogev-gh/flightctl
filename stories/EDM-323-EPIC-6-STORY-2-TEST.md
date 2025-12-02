# Test Story: Certificate Renewal Event API

**Story ID:** EDM-323-EPIC-6-STORY-2-TEST  
**Developer Story:** EDM-323-EPIC-6-STORY-2-DEV  
**Epic:** EDM-323-EPIC-6 (Audit Trail and Compliance)  
**Feature:** EDM-323 (Agent Certificate Rotation)  
**Story Points:** 3  
**Priority:** Medium

## Overview

Comprehensive test plan for certificate renewal event API including API endpoints, query parameters, pagination, and authentication.

## Test Objectives

1. Verify API endpoints are exposed correctly
2. Verify query parameters work correctly
3. Verify pagination works correctly
4. Verify authentication works
5. Verify response format is correct

## Test Scope

### In Scope
- API endpoints
- Query parameters
- Pagination
- Authentication
- Response format

### Out of Scope
- Event store (covered in STORY-1)
- Event aggregation (covered in STORY-3)

## Unit Tests

### Test File: `internal/api_server/certificate_events_test.go`

#### Test Suite 1: API Endpoints

**Test Case 1.1: List Events Endpoint**
- **Setup:** API server with events
- **Action:** Call list events endpoint
- **Expected:** Events returned
- **Assertions:**
  - Endpoint accessible
  - Events returned
  - Response format correct

**Test Case 1.2: Get Event Endpoint**
- **Setup:** API server with event
- **Action:** Call get event endpoint
- **Expected:** Event returned
- **Assertions:**
  - Endpoint accessible
  - Event returned
  - Response format correct

---

#### Test Suite 2: Query Parameters

**Test Case 2.1: Filter by Device**
- **Setup:** API server with events
- **Action:** Query with device filter
- **Expected:** Filtered events returned
- **Assertions:**
  - Correct events returned
  - Filter applied correctly

**Test Case 2.2: Filter by Type**
- **Setup:** API server with events
- **Action:** Query with type filter
- **Expected:** Filtered events returned
- **Assertions:**
  - Correct events returned
  - Filter applied correctly

**Test Case 2.3: Filter by Date Range**
- **Setup:** API server with events
- **Action:** Query with date range
- **Expected:** Filtered events returned
- **Assertions:**
  - Correct events returned
  - Date range applied correctly

---

#### Test Suite 3: Pagination

**Test Case 3.1: Pagination Works**
- **Setup:** API server with many events
- **Action:** Query with pagination
- **Expected:** Paginated results returned
- **Assertions:**
  - Limit applied
  - Offset applied
  - Total count returned

---

## Integration Tests

### Test File: `test/integration/certificate_event_api_test.go`

#### Test Suite 4: API Integration

**Test Case 4.1: Complete API Flow**
- **Setup:** Service with events
- **Action:** Query events via API
- **Expected:** API works correctly
- **Assertions:**
  - Endpoints accessible
  - Querying works
  - Pagination works
  - Authentication works

---

## Test Coverage Requirements

### Code Coverage Targets
- **Overall Coverage:** >80% for API code

---

## Definition of Done

- [ ] All unit tests written and passing
- [ ] All integration tests written and passing
- [ ] Code coverage >80% achieved
- [ ] Test documentation complete
- [ ] QA sign-off obtained

---

**Document End**

