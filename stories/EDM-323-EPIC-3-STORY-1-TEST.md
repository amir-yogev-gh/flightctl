# Test Story: Pending Certificate Storage Mechanism

**Story ID:** EDM-323-EPIC-3-STORY-1-TEST  
**Developer Story:** EDM-323-EPIC-3-STORY-1-DEV  
**Epic:** EDM-323-EPIC-3 (Atomic Certificate Swap)  
**Feature:** EDM-323 (Agent Certificate Rotation)  
**Story Points:** 3  
**Priority:** High

## Overview

Comprehensive test plan for pending certificate storage mechanism including pending path methods, write operations, loading, cleanup, and error handling.

## Test Objectives

1. Verify pending certificate paths are generated correctly
2. Verify WritePending writes to pending locations
3. Verify LoadPendingCertificate loads correctly
4. Verify CleanupPending removes pending files
5. Verify active certificate is preserved
6. Verify error handling works correctly

## Test Scope

### In Scope
- Pending path methods
- WritePending method
- LoadPendingCertificate method
- LoadPendingKey method
- HasPendingCertificate method
- CleanupPending method
- Error handling

### Out of Scope
- Certificate validation (covered in STORY-2)
- Atomic swap (covered in STORY-3)

## Unit Tests

### Test File: `internal/agent/device/certmanager/provider/storage/fs_pending_test.go`

#### Test Suite 1: Pending Path Methods

**Test Case 1.1: GetPendingCertPath**
- **Setup:** Create FileSystemStorage with cert path
- **Action:** Call getPendingCertPath()
- **Expected:** Returns path with .pending suffix
- **Assertions:**
  - Path ends with .pending
  - Path is correct

**Test Case 1.2: GetPendingKeyPath**
- **Setup:** Create FileSystemStorage with key path
- **Action:** Call getPendingKeyPath()
- **Expected:** Returns path with .pending suffix
- **Assertions:**
  - Path ends with .pending
  - Path is correct

---

#### Test Suite 2: WritePending

**Test Case 2.1: Write to Pending Location**
- **Setup:** Create FileSystemStorage, valid certificate and key
- **Action:** Call WritePending()
- **Expected:** Files written to pending locations
- **Assertions:**
  - Pending cert file exists
  - Pending key file exists
  - Active cert file unchanged

**Test Case 2.2: Create Directories**
- **Setup:** Create FileSystemStorage with non-existent directories
- **Action:** Call WritePending()
- **Expected:** Directories created
- **Assertions:**
  - Directories created
  - Permissions correct (0700)

**Test Case 2.3: File Permissions**
- **Setup:** Create FileSystemStorage
- **Action:** Call WritePending()
- **Expected:** Files have correct permissions
- **Assertions:**
  - Cert file: 0644
  - Key file: 0600

**Test Case 2.4: Cleanup on Cert Write Failure**
- **Setup:** Create FileSystemStorage, simulate cert write failure
- **Action:** Call WritePending()
- **Expected:** Pending cert file cleaned up
- **Assertions:**
  - Pending cert file removed
  - Error returned

**Test Case 2.5: Cleanup on Key Write Failure**
- **Setup:** Create FileSystemStorage, simulate key write failure
- **Action:** Call WritePending()
- **Expected:** Both pending files cleaned up
- **Assertions:**
  - Pending cert file removed
  - Pending key file removed
  - Error returned

**Test Case 2.6: Active Certificate Preserved**
- **Setup:** Create FileSystemStorage with existing active certificate
- **Action:** Call WritePending()
- **Expected:** Active certificate unchanged
- **Assertions:**
  - Active cert file unchanged
  - Active key file unchanged

---

#### Test Suite 3: LoadPendingCertificate

**Test Case 3.1: Load Valid Pending Certificate**
- **Setup:** Create FileSystemStorage, write pending certificate
- **Action:** Call LoadPendingCertificate()
- **Expected:** Certificate loaded and parsed
- **Assertions:**
  - Certificate loaded
  - Certificate parsed correctly
  - No error

**Test Case 3.2: Load Non-Existent Pending Certificate**
- **Setup:** Create FileSystemStorage, no pending certificate
- **Action:** Call LoadPendingCertificate()
- **Expected:** Returns error
- **Assertions:**
  - Error returned
  - Error message clear

**Test Case 3.3: Load Invalid Pending Certificate**
- **Setup:** Create FileSystemStorage, write invalid certificate
- **Action:** Call LoadPendingCertificate()
- **Expected:** Returns error
- **Assertions:**
  - Error returned
  - Error indicates parsing failure

---

#### Test Suite 4: LoadPendingKey

**Test Case 4.1: Load Valid Pending Key**
- **Setup:** Create FileSystemStorage, write pending key
- **Action:** Call LoadPendingKey()
- **Expected:** Key loaded
- **Assertions:**
  - Key loaded
  - Key bytes correct
  - No error

**Test Case 4.2: Load Non-Existent Pending Key**
- **Setup:** Create FileSystemStorage, no pending key
- **Action:** Call LoadPendingKey()
- **Expected:** Returns error
- **Assertions:**
  - Error returned

---

#### Test Suite 5: HasPendingCertificate

**Test Case 5.1: Pending Certificate Exists**
- **Setup:** Create FileSystemStorage, write pending certificate
- **Action:** Call HasPendingCertificate()
- **Expected:** Returns true
- **Assertions:**
  - Result == true
  - No error

**Test Case 5.2: No Pending Certificate**
- **Setup:** Create FileSystemStorage, no pending certificate
- **Action:** Call HasPendingCertificate()
- **Expected:** Returns false
- **Assertions:**
  - Result == false
  - No error

---

#### Test Suite 6: CleanupPending

**Test Case 6.1: Cleanup Pending Files**
- **Setup:** Create FileSystemStorage, write pending files
- **Action:** Call CleanupPending()
- **Expected:** Pending files removed
- **Assertions:**
  - Pending cert file removed
  - Pending key file removed
  - No error

**Test Case 6.2: Cleanup Non-Existent Files**
- **Setup:** Create FileSystemStorage, no pending files
- **Action:** Call CleanupPending()
- **Expected:** No error (idempotent)
- **Assertions:**
  - No error
  - Operation succeeds

**Test Case 6.3: Cleanup Partial Files**
- **Setup:** Create FileSystemStorage, only cert file exists
- **Action:** Call CleanupPending()
- **Expected:** Existing file removed, no error for missing file
- **Assertions:**
  - Cert file removed
  - No error for missing key

---

## Integration Tests

### Test File: `test/integration/certificate_pending_storage_test.go`

#### Test Suite 7: Pending Storage Integration

**Test Case 7.1: Write and Load Pending Certificate**
- **Setup:** Create FileSystemStorage
- **Action:** WritePending, then LoadPendingCertificate
- **Expected:** Certificate written and loaded correctly
- **Assertions:**
  - Write succeeds
  - Load succeeds
  - Certificate matches

**Test Case 7.2: Active Certificate Preserved**
- **Setup:** Create FileSystemStorage with active certificate
- **Action:** WritePending, verify active certificate
- **Expected:** Active certificate unchanged
- **Assertions:**
  - Active cert unchanged
  - Active key unchanged

---

## Test Coverage Requirements

### Code Coverage Targets
- **Overall Coverage:** >80% for pending storage code
- **Function Coverage:** 100% for all methods

---

## Definition of Done

- [ ] All unit tests written and passing
- [ ] All integration tests written and passing
- [ ] Code coverage >80% achieved
- [ ] Test documentation complete
- [ ] QA sign-off obtained

---

**Document End**

