# Test Story: Database Schema for Certificate Tracking

**Story ID:** EDM-323-EPIC-1-STORY-4-TEST  
**Developer Story:** EDM-323-EPIC-1-STORY-4-DEV  
**Epic:** EDM-323-EPIC-1 (Certificate Lifecycle Foundation)  
**Feature:** EDM-323 (Agent Certificate Rotation)  
**Story Points:** 3  
**Priority:** High

## Overview

Comprehensive test plan for database schema changes for certificate tracking including model updates, migrations, indexes, and store layer methods.

## Test Objectives

1. Verify certificate tracking fields are added to Device model
2. Verify database migrations create columns correctly
3. Verify indexes are created and functional
4. Verify store layer methods work correctly
5. Verify data persistence and retrieval
6. Verify migration idempotency

## Test Scope

### In Scope
- Device model fields
- Database migrations
- Index creation
- Store layer methods
- Data persistence
- Query methods

### Out of Scope
- Certificate renewal operations (covered in EPIC-2)
- Certificate recovery (covered in EPIC-4)

## Test Environment Setup

### Prerequisites
- PostgreSQL database (or test database)
- GORM setup
- Test database with devices table
- Migration utilities

## Unit Tests

### Test File: `internal/store/model/device_test.go`

#### Test Suite 1: Device Model Fields

**Test Case 1.1: Certificate Fields Exist**
- **Setup:** Create Device struct
- **Action:** Access certificate fields
- **Expected:** All fields accessible
- **Assertions:**
  - CertificateExpiration field exists
  - CertificateLastRenewed field exists
  - CertificateRenewalCount field exists
  - CertificateFingerprint field exists

**Test Case 1.2: Field Types**
- **Setup:** Create Device struct
- **Action:** Check field types
- **Expected:** Types are correct
- **Assertions:**
  - CertificateExpiration is *time.Time
  - CertificateLastRenewed is *time.Time
  - CertificateRenewalCount is int
  - CertificateFingerprint is *string

**Test Case 1.3: JSON Marshaling**
- **Setup:** Create Device with certificate fields
- **Action:** Marshal to JSON
- **Expected:** JSON contains certificate fields
- **Assertions:**
  - JSON includes certificate_expiration
  - JSON includes certificate_last_renewed
  - JSON includes certificate_renewal_count
  - JSON includes certificate_fingerprint

**Test Case 1.4: JSON Unmarshaling**
- **Setup:** Create JSON with certificate fields
- **Action:** Unmarshal into Device
- **Expected:** Fields populated correctly
- **Assertions:**
  - All fields populated
  - Values match JSON

**Test Case 1.5: Zero Values**
- **Setup:** Create Device with zero values
- **Action:** Check zero value behavior
- **Expected:** Zero values handled correctly
- **Assertions:**
  - Nil pointers for time fields
  - 0 for renewal count
  - Nil for fingerprint

---

### Test File: `internal/store/device_migration_test.go`

#### Test Suite 2: Database Migrations

**Test Case 2.1: Add Certificate Tracking Fields**
- **Setup:** Create test database, run migration
- **Action:** Call addCertificateTrackingFields()
- **Expected:** Columns added to devices table
- **Assertions:**
  - certificate_expiration column exists
  - certificate_last_renewed column exists
  - certificate_renewal_count column exists
  - certificate_fingerprint column exists

**Test Case 2.2: Migration Idempotency**
- **Setup:** Run migration twice
- **Action:** Call addCertificateTrackingFields() twice
- **Expected:** No errors on second call
- **Assertions:**
  - First call succeeds
  - Second call succeeds (idempotent)
  - No duplicate columns

**Test Case 2.3: Migration with Existing Columns**
- **Setup:** Manually add columns, then run migration
- **Action:** Call addCertificateTrackingFields()
- **Expected:** Migration detects existing columns
- **Assertions:**
  - No errors
  - Columns not duplicated

**Test Case 2.4: Column Types**
- **Setup:** Run migration
- **Action:** Check column types in database
- **Expected:** Types are correct
- **Assertions:**
  - certificate_expiration is TIMESTAMP
  - certificate_last_renewed is TIMESTAMP
  - certificate_renewal_count is INTEGER
  - certificate_fingerprint is TEXT

**Test Case 2.5: Default Values**
- **Setup:** Run migration, insert device
- **Action:** Insert device without certificate fields
- **Expected:** Default values applied
- **Assertions:**
  - certificate_renewal_count == 0
  - Other fields are NULL

---

#### Test Suite 3: Index Creation

**Test Case 3.1: Create Certificate Expiration Index**
- **Setup:** Run migration
- **Action:** Call createCertificateExpirationIndex()
- **Expected:** Index created on certificate_expiration
- **Assertions:**
  - Index exists
  - Index on correct column
  - Index is functional

**Test Case 3.2: Index Idempotency**
- **Setup:** Run migration twice
- **Action:** Call createCertificateExpirationIndex() twice
- **Expected:** No errors on second call
- **Assertions:**
  - First call succeeds
  - Second call succeeds (idempotent)
  - No duplicate indexes

**Test Case 3.3: Index Performance**
- **Setup:** Create devices with various expiration dates
- **Action:** Query devices by expiration
- **Expected:** Query uses index
- **Assertions:**
  - Query is fast
  - Index is used (EXPLAIN shows index usage)

**Test Case 3.4: Partial Index on Non-Null Expiration**
- **Setup:** Run migration
- **Action:** Check index definition
- **Expected:** Index is partial (only non-null values)
- **Assertions:**
  - Index WHERE clause excludes NULL
  - Index size is smaller

---

#### Test Suite 4: Store Layer Methods

**Test Case 4.1: UpdateCertificateExpiration**
- **Setup:** Create device in database
- **Action:** Call UpdateCertificateExpiration()
- **Expected:** Expiration updated in database
- **Assertions:**
  - Expiration value updated
  - Other fields unchanged
  - No errors

**Test Case 4.2: UpdateCertificateRenewalInfo**
- **Setup:** Create device in database
- **Action:** Call UpdateCertificateRenewalInfo()
- **Expected:** Renewal info updated
- **Assertions:**
  - LastRenewed updated
  - RenewalCount updated
  - Other fields unchanged

**Test Case 4.3: UpdateCertificateFingerprint**
- **Setup:** Create device in database
- **Action:** Call UpdateCertificateFingerprint()
- **Expected:** Fingerprint updated
- **Assertions:**
  - Fingerprint updated
  - Other fields unchanged

**Test Case 4.4: Update Non-Existent Device**
- **Setup:** Device doesn't exist
- **Action:** Call update methods
- **Expected:** No error (or appropriate error)
- **Assertions:**
  - Error handling appropriate
  - No database errors

**Test Case 4.5: Transaction Safety**
- **Setup:** Create device, start transaction
- **Action:** Update certificate fields, rollback
- **Expected:** Changes rolled back
- **Assertions:**
  - Changes not persisted
  - Original values restored

---

#### Test Suite 5: Query Methods

**Test Case 5.1: Query Devices by Expiration**
- **Setup:** Create devices with various expiration dates
- **Action:** Query devices expiring soon
- **Expected:** Correct devices returned
- **Assertions:**
  - Query returns correct devices
  - Results ordered correctly
  - Index used for performance

**Test Case 5.2: Query Devices by Renewal Count**
- **Setup:** Create devices with various renewal counts
- **Action:** Query devices by renewal count
- **Expected:** Correct devices returned
- **Assertions:**
  - Query returns correct devices
  - Filtering works correctly

**Test Case 5.3: Query with NULL Values**
- **Setup:** Create devices with NULL certificate fields
- **Action:** Query devices
- **Expected:** NULL values handled correctly
- **Assertions:**
  - NULL values don't cause errors
  - Filtering works with NULL

---

## Integration Tests

### Test File: `test/integration/certificate_tracking_test.go`

#### Test Suite 6: Database Integration

**Test Case 6.1: Full Migration Flow**
- **Setup:** Fresh database
- **Action:** Run all migrations
- **Expected:** All columns and indexes created
- **Assertions:**
  - All columns exist
  - All indexes exist
  - No errors

**Test Case 6.2: Device Creation with Certificate Fields**
- **Setup:** Run migrations
- **Action:** Create device with certificate fields
- **Expected:** Device created with certificate data
- **Assertions:**
  - Device created
  - Certificate fields stored
  - Can retrieve certificate fields

**Test Case 6.3: Device Update with Certificate Fields**
- **Setup:** Create device, run migrations
- **Action:** Update certificate fields
- **Expected:** Fields updated correctly
- **Assertions:**
  - Fields updated
  - Values persist
  - Can retrieve updated values

**Test Case 6.4: Migration Rollback**
- **Setup:** Run migrations
- **Action:** Attempt to rollback (if supported)
- **Expected:** Rollback works or handled gracefully
- **Assertions:**
  - No errors
  - State is consistent

---

## Test Coverage Requirements

### Code Coverage Targets
- **Overall Coverage:** >80% for migration and store code
- **Migration Coverage:** 100% for migration logic
- **Store Method Coverage:** 100% for all methods

---

## Definition of Done

- [ ] All unit tests written and passing
- [ ] All integration tests written and passing
- [ ] Code coverage >80% achieved
- [ ] Migrations tested for idempotency
- [ ] Indexes verified functional
- [ ] Test documentation complete
- [ ] QA sign-off obtained

---

**Document End**

