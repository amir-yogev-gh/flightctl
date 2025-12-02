# Test Story: Certificate Renewal Configuration Schema

**Story ID:** EDM-323-EPIC-1-STORY-3-TEST  
**Developer Story:** EDM-323-EPIC-1-STORY-3-DEV  
**Epic:** EDM-323-EPIC-1 (Certificate Lifecycle Foundation)  
**Feature:** EDM-323 (Agent Certificate Rotation)  
**Story Points:** 3  
**Priority:** High

## Overview

Comprehensive test plan for certificate renewal configuration schema including constants, struct definitions, validation, defaults, and configuration merging.

## Test Objectives

1. Verify configuration constants are correctly defined
2. Verify configuration structs are correctly defined
3. Verify validation logic rejects invalid configurations
4. Verify default values are applied correctly
5. Verify configuration merging works correctly
6. Verify configuration is integrated with agent config

## Test Scope

### In Scope
- Configuration constants
- CertificateRenewalConfig struct
- CertificateConfig struct
- Validation logic
- Default values
- Configuration merging
- Integration with agent config

### Out of Scope
- Actual renewal operations (covered in EPIC-2)
- Certificate operations using config (covered in other stories)

## Unit Tests

### Test File: `internal/agent/config/certificate_config_test.go`

#### Test Suite 1: Configuration Constants

**Test Case 1.1: Default Values**
- **Setup:** Test all default constants
- **Action:** Verify constant values
- **Expected:** All defaults match specification
- **Assertions:**
  - DefaultCertificateRenewalEnabled == true
  - DefaultCertificateRenewalThresholdDays == 30
  - DefaultCertificateRenewalCheckInterval == 24*time.Hour
  - DefaultCertificateRenewalRetryInterval == 1*time.Hour
  - DefaultCertificateRenewalMaxRetries == 10
  - DefaultCertificateRenewalBackoffMultiplier == 2.0
  - DefaultCertificateRenewalMaxBackoff == 24*time.Hour

**Test Case 1.2: Minimum Values**
- **Setup:** Test minimum constants
- **Action:** Verify minimum values
- **Expected:** All minimums match specification
- **Assertions:**
  - MinCertificateRenewalThresholdDays == 1
  - MinCertificateRenewalCheckInterval == 1*time.Hour
  - MinCertificateRenewalRetryInterval == 1*time.Minute
  - MinCertificateRenewalBackoffMultiplier == 1.0

**Test Case 1.3: Maximum Values**
- **Setup:** Test maximum constants
- **Action:** Verify maximum values
- **Expected:** All maximums match specification
- **Assertions:**
  - MaxCertificateRenewalThresholdDays == 365

---

#### Test Suite 2: CertificateRenewalConfig Struct

**Test Case 2.1: Struct Creation**
- **Setup:** Create CertificateRenewalConfig
- **Action:** Create config with all fields
- **Expected:** All fields are set correctly
- **Assertions:**
  - All fields accessible
  - Values match input

**Test Case 2.2: JSON Marshaling**
- **Setup:** Create config with values
- **Action:** Marshal to JSON
- **Expected:** JSON is valid and contains all fields
- **Assertions:**
  - JSON is valid
  - All fields present in JSON

**Test Case 2.3: JSON Unmarshaling**
- **Setup:** Create JSON config string
- **Action:** Unmarshal JSON into struct
- **Expected:** Struct populated correctly
- **Assertions:**
  - All fields populated
  - Values match JSON

**Test Case 2.4: Zero Values**
- **Setup:** Create config with zero values
- **Action:** Check zero value behavior
- **Expected:** Zero values handled correctly
- **Assertions:**
  - Zero values don't cause errors
  - Defaults applied where appropriate

---

#### Test Suite 3: Configuration Validation

**Test Case 3.1: Valid Configuration**
- **Setup:** Create config with all valid values
- **Action:** Call Validate()
- **Expected:** Returns no error
- **Assertions:**
  - No error returned
  - All values accepted

**Test Case 3.2: Invalid CheckInterval - Too Short**
- **Setup:** Create config with CheckInterval < 1 hour
- **Action:** Call Validate()
- **Expected:** Returns error
- **Assertions:**
  - Error message contains "check-interval must be at least 1 hour"
  - Validation fails

**Test Case 3.3: Invalid ThresholdDays - Too Small**
- **Setup:** Create config with ThresholdDays < 1
- **Action:** Call Validate()
- **Expected:** Returns error
- **Assertions:**
  - Error message contains "threshold-days must be between 1 and 365"
  - Validation fails

**Test Case 3.4: Invalid ThresholdDays - Too Large**
- **Setup:** Create config with ThresholdDays > 365
- **Action:** Call Validate()
- **Expected:** Returns error
- **Assertions:**
  - Error message contains "threshold-days must be between 1 and 365"
  - Validation fails

**Test Case 3.5: Invalid RetryInterval - Too Short**
- **Setup:** Create config with RetryInterval < 1 minute
- **Action:** Call Validate()
- **Expected:** Returns error
- **Assertions:**
  - Error message contains "retry-interval must be at least 1 minute"
  - Validation fails

**Test Case 3.6: Invalid MaxRetries - Negative**
- **Setup:** Create config with MaxRetries < 0
- **Action:** Call Validate()
- **Expected:** Returns error
- **Assertions:**
  - Error message contains "max-retries must be non-negative"
  - Validation fails

**Test Case 3.7: Invalid BackoffMultiplier - Too Small**
- **Setup:** Create config with BackoffMultiplier < 1.0
- **Action:** Call Validate()
- **Expected:** Returns error
- **Assertions:**
  - Error message contains "backoff-multiplier must be at least 1.0"
  - Validation fails

**Test Case 3.8: Invalid MaxBackoff - Too Short**
- **Setup:** Create config with MaxBackoff < RetryInterval
- **Action:** Call Validate()
- **Expected:** Returns error
- **Assertions:**
  - Error message contains "max-backoff must be at least retry-interval"
  - Validation fails

---

#### Test Suite 4: Default Values

**Test Case 4.1: NewDefaultCertificateRenewalConfig**
- **Setup:** Call NewDefaultCertificateRenewalConfig()
- **Action:** Check returned config
- **Expected:** All defaults applied
- **Assertions:**
  - Enabled == true
  - ThresholdDays == 30
  - CheckInterval == 24*time.Hour
  - RetryInterval == 1*time.Hour
  - MaxRetries == 10
  - BackoffMultiplier == 2.0
  - MaxBackoff == 24*time.Hour

**Test Case 4.2: Partial Configuration Uses Defaults**
- **Setup:** Create config with only Enabled set
- **Action:** Apply defaults
- **Expected:** Missing fields use defaults
- **Assertions:**
  - Enabled == provided value
  - Other fields == defaults

---

#### Test Suite 5: Configuration Merging

**Test Case 5.1: Merge Base and Override**
- **Setup:** Base config + override config
- **Action:** Merge configurations
- **Expected:** Override values take precedence
- **Assertions:**
  - Override values applied
  - Base values used where override missing

**Test Case 5.2: Merge with Empty Override**
- **Setup:** Base config + empty override
- **Action:** Merge configurations
- **Expected:** Base values used
- **Assertions:**
  - All values from base
  - No values from override

**Test Case 5.3: Merge with Partial Override**
- **Setup:** Base config + partial override
- **Action:** Merge configurations
- **Expected:** Override values + base defaults
- **Assertions:**
  - Override values applied
  - Base values for non-overridden fields

---

#### Test Suite 6: Integration with Agent Config

**Test Case 6.1: Config Loading with Certificate Section**
- **Setup:** Create agent config JSON with certificate section
- **Action:** Load config
- **Expected:** Certificate config loaded correctly
- **Assertions:**
  - Certificate.Renewal.Enabled loaded
  - Certificate.Renewal.ThresholdDays loaded
  - All fields loaded correctly

**Test Case 6.2: Config Loading without Certificate Section**
- **Setup:** Create agent config JSON without certificate section
- **Action:** Load config
- **Expected:** Default certificate config applied
- **Assertions:**
  - Default values used
  - No errors

**Test Case 6.3: Config Validation in Agent Config**
- **Setup:** Create agent config with invalid certificate config
- **Action:** Validate agent config
- **Expected:** Validation fails
- **Assertions:**
  - Error returned
  - Error references certificate config

---

## Integration Tests

### Test File: `test/integration/certificate_config_test.go`

#### Test Suite 7: Configuration Integration

**Test Case 7.1: Agent Starts with Valid Config**
- **Setup:** Create agent config file with valid certificate config
- **Action:** Start agent
- **Expected:** Agent starts successfully
- **Assertions:**
  - Agent starts
  - Certificate config loaded
  - No errors

**Test Case 7.2: Agent Starts with Invalid Config**
- **Setup:** Create agent config file with invalid certificate config
- **Action:** Start agent
- **Expected:** Agent fails to start with validation error
- **Assertions:**
  - Agent fails to start
  - Error message indicates config issue

**Test Case 7.3: Config Override File**
- **Setup:** Create base config + override config file
- **Action:** Start agent
- **Expected:** Override values applied
- **Assertions:**
  - Override values used
  - Base values for non-overridden fields

---

## Test Coverage Requirements

### Code Coverage Targets
- **Overall Coverage:** >80% for certificate config code
- **Validation Coverage:** 100% for all validation paths
- **Default Coverage:** 100% for default value logic

---

## Definition of Done

- [ ] All unit tests written and passing
- [ ] All integration tests written and passing
- [ ] Code coverage >80% achieved
- [ ] All validation paths tested
- [ ] Configuration merging tested
- [ ] Test documentation complete
- [ ] QA sign-off obtained

---

**Document End**

