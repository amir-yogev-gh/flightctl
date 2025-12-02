# Test Story: Certificate Lifecycle Metrics

**Story ID:** EDM-323-EPIC-5-STORY-1-TEST  
**Developer Story:** EDM-323-EPIC-5-STORY-1-DEV  
**Epic:** EDM-323-EPIC-5 (Observability and Monitoring)  
**Feature:** EDM-323 (Agent Certificate Rotation)  
**Story Points:** 5  
**Priority:** Medium

## Overview

Comprehensive test plan for certificate lifecycle metrics including Prometheus metric definitions, metric updates, and metric exposure.

## Test Objectives

1. Verify Prometheus metrics are defined correctly
2. Verify metrics are updated correctly
3. Verify metrics are exposed via HTTP endpoint
4. Verify metric values are accurate
5. Verify metric labels work correctly

## Test Scope

### In Scope
- Prometheus metric definitions
- Metric updates
- Metric exposure
- Metric accuracy
- Metric labels

### Out of Scope
- Device status (covered in STORY-2)
- Event logging (covered in STORY-3)

## Unit Tests

### Test File: `internal/agent/instrumentation/metrics/certificate_metrics_test.go`

#### Test Suite 1: Metric Definitions

**Test Case 1.1: Metric Registration**
- **Setup:** Initialize metrics
- **Action:** Register metrics
- **Expected:** All metrics registered
- **Assertions:**
  - All metrics registered
  - No errors

**Test Case 1.2: Metric Labels**
- **Setup:** Initialize metrics
- **Action:** Check metric labels
- **Expected:** Labels are correct
- **Assertions:**
  - Labels match specification
  - Label values valid

---

#### Test Suite 2: Metric Updates

**Test Case 2.1: Update Expiration Metrics**
- **Setup:** Certificate with expiration
- **Action:** Update expiration metrics
- **Expected:** Metrics updated correctly
- **Assertions:**
  - Expiration timestamp set
  - Days until expiration set
  - Values accurate

**Test Case 2.2: Update Renewal Metrics**
- **Setup:** Renewal triggered
- **Action:** Update renewal metrics
- **Expected:** Metrics updated correctly
- **Assertions:**
  - Renewal attempts incremented
  - Renewal duration recorded
  - Labels correct

**Test Case 2.3: Update Recovery Metrics**
- **Setup:** Recovery triggered
- **Action:** Update recovery metrics
- **Expected:** Metrics updated correctly
- **Assertions:**
  - Recovery attempts incremented
  - Recovery duration recorded

---

## Integration Tests

### Test File: `test/integration/certificate_metrics_test.go`

#### Test Suite 3: Metric Exposure

**Test Case 3.1: Metrics Exposed via HTTP**
- **Setup:** Agent with metrics enabled
- **Action:** Query metrics endpoint
- **Expected:** Metrics returned
- **Assertions:**
  - Metrics endpoint accessible
  - Metrics in Prometheus format
  - Values present

**Test Case 3.2: Metric Values Accurate**
- **Setup:** Agent with certificate operations
- **Action:** Query metrics, verify values
- **Expected:** Values match actual state
- **Assertions:**
  - Expiration metrics accurate
  - Renewal metrics accurate
  - Recovery metrics accurate

---

## Test Coverage Requirements

### Code Coverage Targets
- **Overall Coverage:** >80% for metrics code

---

## Definition of Done

- [ ] All unit tests written and passing
- [ ] All integration tests written and passing
- [ ] Code coverage >80% achieved
- [ ] Test documentation complete
- [ ] QA sign-off obtained

---

**Document End**

