# Developer Story: Certificate Renewal Configuration Schema

**Story ID:** EDM-323-EPIC-1-STORY-3  
**Epic:** EDM-323-EPIC-1 (Certificate Lifecycle Foundation)  
**Feature:** EDM-323 (Agent Certificate Rotation)  
**Story Points:** 3  
**Priority:** High

## Overview

Implement configuration schema for certificate renewal settings, including renewal thresholds, check intervals, and retry policies. Add validation and default values following existing configuration patterns.

## Implementation Tasks

### Task 1: Define Certificate Renewal Configuration Constants

**File:** `internal/agent/config/config.go` (modify)

**Objective:** Add constants for default certificate renewal configuration values.

**Implementation Steps:**

1. **Add constants after existing constants:**
```go
const (
    // ... existing constants ...
    
    // DefaultCertificateRenewalEnabled controls whether certificate renewal is enabled by default
    DefaultCertificateRenewalEnabled = true
    
    // DefaultCertificateRenewalThresholdDays is the default number of days before expiration to trigger renewal
    DefaultCertificateRenewalThresholdDays = 30
    
    // DefaultCertificateRenewalCheckInterval is the default interval for checking certificate expiration
    DefaultCertificateRenewalCheckInterval = util.Duration(24 * time.Hour)
    
    // DefaultCertificateRenewalRetryInterval is the default interval between renewal retry attempts
    DefaultCertificateRenewalRetryInterval = util.Duration(1 * time.Hour)
    
    // DefaultCertificateRenewalMaxRetries is the default maximum number of renewal retry attempts
    DefaultCertificateRenewalMaxRetries = 10
    
    // DefaultCertificateRenewalBackoffMultiplier is the default exponential backoff multiplier
    DefaultCertificateRenewalBackoffMultiplier = 2.0
    
    // DefaultCertificateRenewalMaxBackoff is the default maximum backoff duration
    DefaultCertificateRenewalMaxBackoff = util.Duration(24 * time.Hour)
    
    // MinCertificateRenewalThresholdDays is the minimum allowed threshold days
    MinCertificateRenewalThresholdDays = 1
    
    // MaxCertificateRenewalThresholdDays is the maximum allowed threshold days
    MaxCertificateRenewalThresholdDays = 365
    
    // MinCertificateRenewalCheckInterval is the minimum allowed check interval
    MinCertificateRenewalCheckInterval = util.Duration(1 * time.Hour)
    
    // MinCertificateRenewalRetryInterval is the minimum allowed retry interval
    MinCertificateRenewalRetryInterval = util.Duration(1 * time.Minute)
    
    // MinCertificateRenewalBackoffMultiplier is the minimum allowed backoff multiplier
    MinCertificateRenewalBackoffMultiplier = 1.0
)
```

**Testing:**
- Verify constants are accessible
- Verify constant values are correct

---

### Task 2: Create Certificate Renewal Configuration Struct

**File:** `internal/agent/config/config.go` (modify)

**Objective:** Define the CertificateRenewalConfig struct with all renewal settings.

**Implementation Steps:**

1. **Add CertificateRenewalConfig struct after TPM struct:**
```go
// CertificateRenewalConfig holds configuration for certificate renewal.
type CertificateRenewalConfig struct {
    // Enabled controls whether certificate renewal is enabled
    Enabled bool `json:"enabled,omitempty"`
    
    // ThresholdDays is the number of days before expiration to trigger renewal
    ThresholdDays int `json:"threshold-days,omitempty"`
    
    // CheckInterval is how often to check certificate expiration
    CheckInterval util.Duration `json:"check-interval,omitempty"`
    
    // RetryInterval is the interval between renewal retry attempts
    RetryInterval util.Duration `json:"retry-interval,omitempty"`
    
    // MaxRetries is the maximum number of renewal retry attempts
    MaxRetries int `json:"max-retries,omitempty"`
    
    // BackoffMultiplier is the exponential backoff multiplier for retries
    BackoffMultiplier float64 `json:"backoff-multiplier,omitempty"`
    
    // MaxBackoff is the maximum backoff duration between retries
    MaxBackoff util.Duration `json:"max-backoff,omitempty"`
}

// DefaultCertificateRenewalConfig returns a CertificateRenewalConfig with default values.
func DefaultCertificateRenewalConfig() CertificateRenewalConfig {
    return CertificateRenewalConfig{
        Enabled:          DefaultCertificateRenewalEnabled,
        ThresholdDays:    DefaultCertificateRenewalThresholdDays,
        CheckInterval:    DefaultCertificateRenewalCheckInterval,
        RetryInterval:    DefaultCertificateRenewalRetryInterval,
        MaxRetries:       DefaultCertificateRenewalMaxRetries,
        BackoffMultiplier: DefaultCertificateRenewalBackoffMultiplier,
        MaxBackoff:       DefaultCertificateRenewalMaxBackoff,
    }
}
```

2. **Add CertificateConfig struct:**
```go
// CertificateConfig holds all certificate-related configuration.
type CertificateConfig struct {
    // Renewal holds certificate renewal configuration
    Renewal CertificateRenewalConfig `json:"renewal,omitempty"`
}
```

**Testing:**
- Test DefaultCertificateRenewalConfig returns correct defaults
- Test struct can be marshaled/unmarshaled to/from JSON/YAML

---

### Task 3: Add Certificate Field to Config Struct

**File:** `internal/agent/config/config.go` (modify)

**Objective:** Add Certificate field to the main Config struct.

**Implementation Steps:**

1. **Add Certificate field to Config struct:**
```go
type Config struct {
    config.ServiceConfig

    // ... existing fields ...
    
    // Certificate holds certificate management configuration
    Certificate CertificateConfig `json:"certificate,omitempty"`
    
    // ... rest of fields ...
}
```

**Testing:**
- Test Config struct includes Certificate field
- Test field can be serialized/deserialized

---

### Task 4: Set Default Values in NewDefault

**File:** `internal/agent/config/config.go` (modify)

**Objective:** Initialize certificate renewal configuration with defaults in NewDefault().

**Implementation Steps:**

1. **Add certificate config initialization in NewDefault():**
```go
func NewDefault() *Config {
    c := &Config{
        ConfigDir:            DefaultConfigDir,
        DataDir:              DefaultDataDir,
        StatusUpdateInterval: DefaultStatusUpdateInterval,
        SpecFetchInterval:    DefaultSpecFetchInterval,
        readWriter:           fileio.NewReadWriter(),
        LogLevel:             logrus.InfoLevel.String(),
        DefaultLabels:        make(map[string]string),
        ServiceConfig:        config.NewServiceConfig(),
        SystemInfo:           DefaultSystemInfo,
        SystemInfoTimeout:    DefaultSystemInfoTimeout,
        PullTimeout:          DefaultPullTimeout,
        PullRetrySteps:       DefaultPullRetrySteps,
        MetricsEnabled:       DefaultMetricsEnabled,
        ProfilingEnabled:     DefaultProfilingEnabled,
        TPM: TPM{
            Enabled:         false,
            AuthEnabled:     false,
            DevicePath:      DefaultTPMDevicePath,
            StorageFilePath: filepath.Join(DefaultDataDir, DefaultTPMKeyFile),
        },
        AuditLog: *audit.NewDefaultAuditConfig(),
        Certificate: CertificateConfig{  // NEW
            Renewal: DefaultCertificateRenewalConfig(),
        },
    }

    // ... rest of function ...
}
```

**Testing:**
- Test NewDefault() includes certificate config with defaults
- Test default values match expected constants

---

### Task 5: Implement Configuration Validation

**File:** `internal/agent/config/config.go` (modify)

**Objective:** Add validation for certificate renewal configuration.

**Implementation Steps:**

1. **Add Validate method to CertificateRenewalConfig:**
```go
// Validate validates the certificate renewal configuration.
func (c *CertificateRenewalConfig) Validate() error {
    if c.ThresholdDays < MinCertificateRenewalThresholdDays {
        return fmt.Errorf("threshold-days must be at least %d, got %d", 
            MinCertificateRenewalThresholdDays, c.ThresholdDays)
    }
    
    if c.ThresholdDays > MaxCertificateRenewalThresholdDays {
        return fmt.Errorf("threshold-days must be at most %d, got %d", 
            MaxCertificateRenewalThresholdDays, c.ThresholdDays)
    }
    
    if c.CheckInterval <= 0 {
        return fmt.Errorf("check-interval must be positive, got %v", c.CheckInterval)
    }
    
    if c.CheckInterval < MinCertificateRenewalCheckInterval {
        return fmt.Errorf("check-interval must be at least %v, got %v", 
            MinCertificateRenewalCheckInterval, c.CheckInterval)
    }
    
    if c.RetryInterval <= 0 {
        return fmt.Errorf("retry-interval must be positive, got %v", c.RetryInterval)
    }
    
    if c.RetryInterval < MinCertificateRenewalRetryInterval {
        return fmt.Errorf("retry-interval must be at least %v, got %v", 
            MinCertificateRenewalRetryInterval, c.RetryInterval)
    }
    
    if c.MaxRetries < 0 {
        return fmt.Errorf("max-retries must be non-negative, got %d", c.MaxRetries)
    }
    
    if c.BackoffMultiplier < MinCertificateRenewalBackoffMultiplier {
        return fmt.Errorf("backoff-multiplier must be at least %v, got %v", 
            MinCertificateRenewalBackoffMultiplier, c.BackoffMultiplier)
    }
    
    if c.MaxBackoff <= 0 {
        return fmt.Errorf("max-backoff must be positive, got %v", c.MaxBackoff)
    }
    
    // Ensure max backoff is not less than retry interval
    if c.MaxBackoff < c.RetryInterval {
        return fmt.Errorf("max-backoff (%v) must be at least retry-interval (%v)", 
            c.MaxBackoff, c.RetryInterval)
    }
    
    return nil
}
```

2. **Add validation call in Config.Validate():**
```go
// Validate checks that the required fields are set and ensures that the paths exist.
func (cfg *Config) Validate() error {
    if err := cfg.EnrollmentService.Validate(); err != nil {
        return err
    }
    if err := cfg.ManagementService.Validate(); err != nil {
        return err
    }
    if err := cfg.validateSyncIntervals(); err != nil {
        return err
    }

    if cfg.SystemInfoTimeout > MaxSystemInfoTimeout {
        return fmt.Errorf("system-info-timeout cannot exceed %s, got %s", MaxSystemInfoTimeout, cfg.SystemInfoTimeout)
    }

    if cfg.TPM.AuthEnabled && !cfg.TPM.Enabled {
        return fmt.Errorf("cannot enable TPM password authentication when TPM device identity is disabled")
    }

    // Validate certificate renewal configuration
    if err := cfg.Certificate.Renewal.Validate(); err != nil {
        return fmt.Errorf("certificate renewal configuration validation failed: %w", err)
    }

    // Validate audit log configuration
    if err := cfg.AuditLog.Validate(cfg.readWriter); err != nil {
        return fmt.Errorf("audit log configuration validation failed: %w", err)
    }

    // ... rest of validation ...
}
```

**Testing:**
- Test validation with valid configuration
- Test validation with invalid threshold_days (too low, too high)
- Test validation with invalid check_interval (zero, negative, too low)
- Test validation with invalid retry_interval (zero, negative, too low)
- Test validation with invalid max_retries (negative)
- Test validation with invalid backoff_multiplier (too low)
- Test validation with invalid max_backoff (zero, negative, less than retry_interval)
- Test validation error messages are clear

---

### Task 6: Add Configuration Completion Logic

**File:** `internal/agent/config/config.go` (modify)

**Objective:** Ensure certificate renewal configuration has defaults filled in during Complete().

**Implementation Steps:**

1. **Add completion logic in Complete():**
```go
// Complete fills in defaults for fields not set by the config file
func (cfg *Config) Complete() error {
    // ... existing completion logic ...
    
    // Complete certificate renewal configuration
    if cfg.Certificate.Renewal.ThresholdDays == 0 {
        cfg.Certificate.Renewal.ThresholdDays = DefaultCertificateRenewalThresholdDays
    }
    if cfg.Certificate.Renewal.CheckInterval == 0 {
        cfg.Certificate.Renewal.CheckInterval = DefaultCertificateRenewalCheckInterval
    }
    if cfg.Certificate.Renewal.RetryInterval == 0 {
        cfg.Certificate.Renewal.RetryInterval = DefaultCertificateRenewalRetryInterval
    }
    if cfg.Certificate.Renewal.MaxRetries == 0 && cfg.Certificate.Renewal.Enabled {
        // Only set default max retries if renewal is enabled
        // If max-retries is explicitly set to 0, that's valid (no retries)
        // We check if it's the zero value by checking if renewal is enabled
        // and max-retries wasn't explicitly set
        cfg.Certificate.Renewal.MaxRetries = DefaultCertificateRenewalMaxRetries
    }
    if cfg.Certificate.Renewal.BackoffMultiplier == 0 {
        cfg.Certificate.Renewal.BackoffMultiplier = DefaultCertificateRenewalBackoffMultiplier
    }
    if cfg.Certificate.Renewal.MaxBackoff == 0 {
        cfg.Certificate.Renewal.MaxBackoff = DefaultCertificateRenewalMaxBackoff
    }
    
    return nil
}
```

**Note:** The `Enabled` field doesn't need completion since it has a default value (false for bool), but we want it to default to true. We need to handle this differently.

2. **Better approach - use pointer or check in Complete:**
```go
// Complete fills in defaults for fields not set by the config file
func (cfg *Config) Complete() error {
    // ... existing completion logic ...
    
    // Complete certificate renewal configuration
    // Use a default config and merge non-zero values
    defaultRenewal := DefaultCertificateRenewalConfig()
    
    // If enabled is not explicitly set (false), use default (true)
    // We can't distinguish between "not set" and "set to false" for bool,
    // so we'll use the default if the entire renewal config looks uninitialized
    if cfg.Certificate.Renewal.CheckInterval == 0 && 
       cfg.Certificate.Renewal.ThresholdDays == 0 {
        // Looks like renewal config wasn't set at all, use defaults
        cfg.Certificate.Renewal = defaultRenewal
    } else {
        // Some fields were set, fill in missing ones
        if cfg.Certificate.Renewal.ThresholdDays == 0 {
            cfg.Certificate.Renewal.ThresholdDays = defaultRenewal.ThresholdDays
        }
        if cfg.Certificate.Renewal.CheckInterval == 0 {
            cfg.Certificate.Renewal.CheckInterval = defaultRenewal.CheckInterval
        }
        if cfg.Certificate.Renewal.RetryInterval == 0 {
            cfg.Certificate.Renewal.RetryInterval = defaultRenewal.RetryInterval
        }
        if cfg.Certificate.Renewal.MaxRetries == 0 {
            cfg.Certificate.Renewal.MaxRetries = defaultRenewal.MaxRetries
        }
        if cfg.Certificate.Renewal.BackoffMultiplier == 0 {
            cfg.Certificate.Renewal.BackoffMultiplier = defaultRenewal.BackoffMultiplier
        }
        if cfg.Certificate.Renewal.MaxBackoff == 0 {
            cfg.Certificate.Renewal.MaxBackoff = defaultRenewal.MaxBackoff
        }
        // Enabled defaults to true if not explicitly set
        // Since we can't distinguish, we'll default to true if other fields suggest config wasn't set
        // This is a limitation of bool in JSON - consider using *bool if this becomes an issue
    }
    
    return nil
}
```

**Testing:**
- Test Complete() fills in defaults when config is empty
- Test Complete() preserves explicitly set values
- Test Complete() fills in only missing values when some are set

---

### Task 7: Add Configuration Merging Support

**File:** `internal/agent/config/config.go` (modify)

**Objective:** Support merging certificate renewal config in override files.

**Implementation Steps:**

1. **Add certificate renewal merging to mergeConfigs():**
```go
func mergeConfigs(base, override *Config) {
    // ... existing merge logic ...
    
    // certificate renewal
    overrideIfNotEmpty(&base.Certificate.Renewal.Enabled, override.Certificate.Renewal.Enabled)
    overrideIfNotEmpty(&base.Certificate.Renewal.ThresholdDays, override.Certificate.Renewal.ThresholdDays)
    overrideIfNotEmpty(&base.Certificate.Renewal.CheckInterval, override.Certificate.Renewal.CheckInterval)
    overrideIfNotEmpty(&base.Certificate.Renewal.RetryInterval, override.Certificate.Renewal.RetryInterval)
    overrideIfNotEmpty(&base.Certificate.Renewal.MaxRetries, override.Certificate.Renewal.MaxRetries)
    overrideIfNotEmpty(&base.Certificate.Renewal.BackoffMultiplier, override.Certificate.Renewal.BackoffMultiplier)
    overrideIfNotEmpty(&base.Certificate.Renewal.MaxBackoff, override.Certificate.Renewal.MaxBackoff)
}
```

**Testing:**
- Test merging preserves base values when override is empty
- Test merging uses override values when set
- Test merging works with override config files

---

## Unit Tests

### Test File: `internal/agent/config/certificate_renewal_test.go` (new)

**Test Cases:**

1. **TestDefaultCertificateRenewalConfig:**
   - Default config has correct values
   - All fields are set to expected defaults

2. **TestCertificateRenewalConfig_Validate:**
   - Valid configuration passes validation
   - Invalid threshold_days (too low) fails validation
   - Invalid threshold_days (too high) fails validation
   - Invalid check_interval (zero) fails validation
   - Invalid check_interval (negative) fails validation
   - Invalid check_interval (too low) fails validation
   - Invalid retry_interval (zero) fails validation
   - Invalid retry_interval (negative) fails validation
   - Invalid retry_interval (too low) fails validation
   - Invalid max_retries (negative) fails validation
   - Invalid backoff_multiplier (too low) fails validation
   - Invalid max_backoff (zero) fails validation
   - Invalid max_backoff (negative) fails validation
   - Invalid max_backoff (less than retry_interval) fails validation
   - Error messages are clear and descriptive

3. **TestConfig_CertificateRenewalDefaults:**
   - NewDefault() includes certificate renewal config
   - Default values match expected constants
   - Complete() fills in missing values
   - Complete() preserves explicitly set values

4. **TestConfig_CertificateRenewalValidation:**
   - Config validation includes certificate renewal validation
   - Invalid renewal config causes validation failure
   - Valid renewal config passes validation

5. **TestConfig_CertificateRenewalMerging:**
   - Merging preserves base values
   - Merging uses override values
   - Merging works with partial overrides

6. **TestCertificateRenewalConfig_JSONYAML:**
   - Config can be marshaled to JSON
   - Config can be marshaled to YAML
   - Config can be unmarshaled from JSON
   - Config can be unmarshaled from YAML
   - Field names match JSON tags

---

## Integration Tests

### Test File: `test/integration/config_certificate_renewal_test.go` (new)

**Test Cases:**

1. **TestCertificateRenewalConfigLoading:**
   - Config file with certificate renewal settings loads correctly
   - Default values are applied when not specified
   - Explicit values override defaults

2. **TestCertificateRenewalConfigOverride:**
   - Override config file merges certificate renewal settings
   - Override values take precedence
   - Base values preserved when not overridden

3. **TestCertificateRenewalConfigValidation:**
   - Invalid config file is rejected with clear error
   - Valid config file passes validation
   - Agent startup fails with invalid renewal config

---

## Example Configuration Files

### Example 1: Minimal Configuration (uses defaults)
```yaml
# /etc/flightctl/config.yaml
certificate:
  renewal:
    enabled: true
```

### Example 2: Custom Threshold
```yaml
# /etc/flightctl/config.yaml
certificate:
  renewal:
    enabled: true
    threshold-days: 60
```

### Example 3: Full Configuration
```yaml
# /etc/flightctl/config.yaml
certificate:
  renewal:
    enabled: true
    threshold-days: 30
    check-interval: 24h
    retry-interval: 1h
    max-retries: 10
    backoff-multiplier: 2.0
    max-backoff: 24h
```

### Example 4: Override Configuration
```yaml
# /etc/flightctl/conf.d/renewal-override.yaml
certificate:
  renewal:
    threshold-days: 45
    check-interval: 12h
```

---

## Code Review Checklist

- [ ] Constants are properly named and documented
- [ ] Struct fields have appropriate JSON tags
- [ ] Default values match requirements
- [ ] Validation covers all fields and edge cases
- [ ] Validation error messages are clear
- [ ] Complete() logic handles all cases correctly
- [ ] Merging logic works correctly
- [ ] Unit tests achieve >80% coverage
- [ ] Integration tests cover config loading
- [ ] Code follows existing config patterns
- [ ] Documentation comments are complete

---

## Definition of Done

- [ ] Constants defined for default values
- [ ] CertificateRenewalConfig struct created
- [ ] CertificateConfig struct created
- [ ] Certificate field added to Config struct
- [ ] Default values set in NewDefault()
- [ ] Validation implemented and integrated
- [ ] Completion logic implemented
- [ ] Merging logic implemented
- [ ] Unit tests written and passing (>80% coverage)
- [ ] Integration tests written and passing
- [ ] Code reviewed and approved
- [ ] Configuration documented (examples, validation rules)

---

## Related Files

- `internal/agent/config/config.go` - Main configuration file
- `internal/agent/config/config_test.go` - Existing config tests
- `internal/util/duration.go` - Duration utility type (if exists)

---

## Dependencies

- None (can be done in parallel with other stories)
- Uses existing config infrastructure
- Uses existing util.Duration type

---

## Notes

- **Bool Default Handling**: Go's JSON unmarshaling doesn't distinguish between "not set" and "set to false" for bool fields. If this becomes an issue, consider using `*bool` (pointer to bool) to allow nil to mean "not set".

- **Zero Value Detection**: For int and float64, zero values (0) are ambiguous - they could mean "not set" or "explicitly set to 0". The current approach uses 0 as "not set" for most fields, but MaxRetries=0 is valid (no retries). Consider using pointers if this becomes problematic.

- **Duration Parsing**: The `util.Duration` type should handle parsing from strings like "24h", "1h", etc. Verify this works correctly.

- **Backward Compatibility**: Existing config files without certificate renewal settings should continue to work (renewal will use defaults).

- **Configuration Documentation**: Consider adding configuration documentation to user docs explaining each field and its purpose.

---

**Document End**

