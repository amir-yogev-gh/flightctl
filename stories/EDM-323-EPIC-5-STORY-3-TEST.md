# Test Story: Certificate Renewal Event Logging

**Story ID:** EDM-323-EPIC-5-STORY-3-TEST  
**Developer Story:** EDM-323-EPIC-5-STORY-3-DEV  
**Epic:** EDM-323-EPIC-5 (Observability and Monitoring)  
**Feature:** EDM-323 (Agent Certificate Rotation)  
**Story Points:** 3  
**Priority:** Medium

## Overview

Comprehensive test plan for certificate renewal event logging including structured logging, log levels, and log content.

## Test Objectives

1. Verify renewal events are logged correctly
2. Verify log levels are appropriate
3. Verify log content is complete
4. Verify structured logging format
5. Verify error events are logged

## Test Scope

### In Scope
- Renewal event logging
- Recovery event logging
- Error event logging
- Structured logging
- Log levels

### Out of Scope
- Metrics (covered in STORY-1)
- Device status (covered in STORY-2)

## Unit Tests

### Test File: `internal/agent/device/certmanager/logging_test.go`

#### Test Suite 1: Renewal Event Logging

**Test Case 1.1: Log Renewal Trigger**
- **Setup:** Renewal triggered
- **Action:** Log renewal event
- **Expected:** Event logged with correct level
- **Assertions:**
  - Event logged
  - Log level appropriate (info)
  - Content complete

**Test Case 1.2: Log Renewal Success**
- **Setup:** Renewal succeeds
- **Action:** Log success event
- **Expected:** Event logged
- **Assertions:**
  - Event logged
  - Log level appropriate (info)
  - Content complete

**Test Case 1.3: Log Renewal Failure**
- **Setup:** Renewal fails
- **Action:** Log failure event
- **Expected:** Event logged with error
- **Assertions:**
  - Event logged
  - Log level appropriate (error)
  - Error details included

---

#### Test Suite 2: Recovery Event Logging

**Test Case 2.1: Log Recovery Trigger**
- **Setup:** Recovery triggered
- **Action:** Log recovery event
- **Expected:** Event logged
- **Assertions:**
  - Event logged
  - Log level appropriate (warn)
  - Content complete

**Test Case 2.2: Log Recovery Success**
- **Setup:** Recovery succeeds
- **Action:** Log success event
- **Expected:** Event logged
- **Assertions:**
  - Event logged
  - Log level appropriate (info)
  - Content complete

---

#### Test Suite 3: Structured Logging

**Test Case 3.1: Structured Log Format**
- **Setup:** Renewal event
- **Action:** Log event
- **Expected:** Log in structured format
- **Assertions:**
  - JSON format (if configured)
  - Fields present
  - Values correct

---

## Integration Tests

### Test File: `test/integration/certificate_logging_test.go`

#### Test Suite 4: Logging Integration

**Test Case 4.1: Complete Renewal Logging**
- **Setup:** Agent with certificate renewal
- **Action:** Perform renewal
- **Expected:** All events logged
- **Assertions:**
  - Trigger logged
  - Success logged
  - All events present

---

## Test Coverage Requirements

### Code Coverage Targets
- **Overall Coverage:** >80% for logging code

---

## Definition of Done

- [ ] All unit tests written and passing
- [ ] All integration tests written and passing
- [ ] Code coverage >80% achieved
- [ ] Test documentation complete
- [ ] QA sign-off obtained

---

**Document End**

