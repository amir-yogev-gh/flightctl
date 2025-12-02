# Test Story: Certificate Issuance for Renewals

**Story ID:** EDM-323-EPIC-2-STORY-4-TEST  
**Developer Story:** EDM-323-EPIC-2-STORY-4-DEV  
**Epic:** EDM-323-EPIC-2 (Proactive Certificate Renewal)  
**Feature:** EDM-323 (Agent Certificate Rotation)  
**Story Points:** 3  
**Priority:** High

## Overview

Comprehensive test plan for certificate issuance for renewal requests including certificate signing, device tracking updates, and certificate parsing.

## Test Objectives

1. Verify certificates are signed correctly for renewals
2. Verify device tracking fields are updated
3. Verify certificate information is parsed correctly
4. Verify 365-day validity is enforced
5. Verify device identity is preserved

## Test Scope

### In Scope
- Certificate signing for renewals
- Certificate parsing
- Device tracking updates
- Validity period enforcement
- Identity preservation

### Out of Scope
- Certificate reception (covered in STORY-5)
- Atomic swap (covered in EPIC-3)

## Unit Tests

### Test File: `internal/service/certificatesigningrequest_renewal_issuance_test.go`

#### Test Suite 1: Certificate Signing

**Test Case 1.1: Sign Renewal Certificate**
- **Setup:** Approved renewal CSR
- **Action:** Sign certificate
- **Expected:** Certificate signed with 365-day validity
- **Assertions:**
  - Certificate signed
  - Validity == 365 days
  - Identity preserved

**Test Case 1.2: Parse Certificate**
- **Setup:** Signed certificate PEM
- **Action:** Parse certificate
- **Expected:** Certificate parsed correctly
- **Assertions:**
  - Certificate parsed
  - Expiration extracted
  - Fingerprint calculated

---

#### Test Suite 2: Device Tracking Updates

**Test Case 2.1: Update Certificate Expiration**
- **Setup:** Device with certificate issued
- **Action:** Update certificate tracking
- **Expected:** Expiration updated in database
- **Assertions:**
  - CertificateExpiration updated
  - Value matches certificate

**Test Case 2.2: Update Renewal Info**
- **Setup:** Device with certificate renewed
- **Action:** Update certificate tracking
- **Expected:** Renewal info updated
- **Assertions:**
  - CertificateLastRenewed updated
  - CertificateRenewalCount incremented

**Test Case 2.3: Update Fingerprint**
- **Setup:** Device with new certificate
- **Action:** Update certificate tracking
- **Expected:** Fingerprint updated
- **Assertions:**
  - CertificateFingerprint updated
  - Value matches certificate

---

## Integration Tests

### Test File: `test/integration/certificate_issuance_renewal_test.go`

#### Test Suite 3: Certificate Issuance Flow

**Test Case 3.1: Complete Renewal Issuance**
- **Setup:** Service with approved renewal CSR
- **Action:** Issue certificate
- **Expected:** Certificate issued and tracking updated
- **Assertions:**
  - Certificate issued
  - Tracking fields updated
  - Certificate valid

---

## Test Coverage Requirements

### Code Coverage Targets
- **Overall Coverage:** >80% for issuance code

---

## Definition of Done

- [ ] All unit tests written and passing
- [ ] All integration tests written and passing
- [ ] Code coverage >80% achieved
- [ ] Test documentation complete
- [ ] QA sign-off obtained

---

**Document End**

