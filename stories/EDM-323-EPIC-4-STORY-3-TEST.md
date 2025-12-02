# Test Story: TPM Attestation Generation for Recovery

**Story ID:** EDM-323-EPIC-4-STORY-3-TEST  
**Developer Story:** EDM-323-EPIC-4-STORY-3-DEV  
**Epic:** EDM-323-EPIC-4 (Expired Certificate Recovery)  
**Feature:** EDM-323 (Agent Certificate Rotation)  
**Story Points:** 5  
**Priority:** High

## Overview

Comprehensive test plan for TPM attestation generation including TPM quote generation, PCR value reading, device fingerprint extraction, and attestation data creation.

## Test Objectives

1. Verify TPM quote generation works correctly
2. Verify PCR values are read correctly
3. Verify device fingerprint extraction
4. Verify attestation data creation
5. Verify attestation validation

## Test Scope

### In Scope
- TPM quote generation
- PCR value reading
- Device fingerprint extraction
- Attestation data creation
- Attestation validation

### Out of Scope
- Service-side attestation verification (covered in STORY-4)
- Recovery CSR (covered in STORY-5)

## Unit Tests

### Test File: `internal/agent/identity/tpm_attestation_test.go`

#### Test Suite 1: TPM Quote Generation

**Test Case 1.1: Generate Valid TPM Quote**
- **Setup:** TPM client available
- **Action:** Call GenerateRenewalAttestation()
- **Expected:** TPM quote generated
- **Assertions:**
  - Quote generated
  - Quote valid
  - No error

**Test Case 1.2: TPM Quote with Nonce**
- **Setup:** TPM client, nonce provided
- **Action:** Generate quote with nonce
- **Expected:** Quote includes nonce
- **Assertions:**
  - Quote includes nonce
  - Quote valid
  - No error

**Test Case 1.3: TPM Quote Failure**
- **Setup:** TPM client unavailable
- **Action:** Call GenerateRenewalAttestation()
- **Expected:** Returns error
- **Assertions:**
  - Error returned
  - Error message clear

---

#### Test Suite 2: PCR Value Reading

**Test Case 2.1: Read PCR Values**
- **Setup:** TPM client available
- **Action:** Read PCR values
- **Expected:** PCR values read
- **Assertions:**
  - PCR values read
  - Values valid
  - No error

**Test Case 2.2: Read Specific PCRs**
- **Setup:** TPM client, specific PCR selection
- **Action:** Read selected PCRs
- **Expected:** Only selected PCRs read
- **Assertions:**
  - Selected PCRs read
  - Other PCRs not included
  - No error

---

#### Test Suite 3: Device Fingerprint

**Test Case 3.1: Extract Device Fingerprint**
- **Setup:** TPM client, device identity
- **Action:** Extract fingerprint
- **Expected:** Fingerprint extracted
- **Assertions:**
  - Fingerprint extracted
  - Fingerprint matches device
  - No error

---

#### Test Suite 4: Attestation Data Creation

**Test Case 4.1: Create Complete Attestation**
- **Setup:** TPM client, all components available
- **Action:** Call GenerateRenewalAttestation()
- **Expected:** Complete attestation created
- **Assertions:**
  - Quote present
  - PCR values present
  - Fingerprint present
  - Attestation key present
  - No error

---

## Integration Tests

### Test File: `test/integration/tpm_attestation_test.go`

#### Test Suite 5: TPM Attestation Integration

**Test Case 5.1: Generate Attestation for Recovery**
- **Setup:** Agent with expired certificate, TPM available
- **Action:** Generate attestation for recovery
- **Expected:** Attestation generated successfully
- **Assertions:**
  - Attestation generated
  - All components present
  - Ready for recovery CSR

---

## Test Coverage Requirements

### Code Coverage Targets
- **Overall Coverage:** >80% for TPM attestation code

---

## Definition of Done

- [ ] All unit tests written and passing
- [ ] All integration tests written and passing
- [ ] Code coverage >80% achieved
- [ ] Test documentation complete
- [ ] QA sign-off obtained

---

**Document End**

