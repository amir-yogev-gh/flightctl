# Developer Story: Certificate Expiration Monitoring Infrastructure

**Story ID:** EDM-323-EPIC-1-STORY-1  
**Epic:** EDM-323-EPIC-1 (Certificate Lifecycle Foundation)  
**Feature:** EDM-323 (Agent Certificate Rotation)  
**Story Points:** 5  
**Priority:** High

## Overview

Implement certificate expiration monitoring infrastructure that parses certificate expiration dates, calculates days until expiration, and provides this information for renewal decisions.

## Implementation Tasks

### Task 1: Create Expiration Monitor Package

**File:** `internal/agent/device/certmanager/expiration.go` (new)

**Objective:** Create a new package for certificate expiration monitoring with core functions.

**Implementation Steps:**

1. **Create the file structure:**
```go
package certmanager

import (
    "context"
    "crypto/x509"
    "time"
    
    "github.com/flightctl/flightctl/internal/agent/device/certmanager/provider"
)

// ExpirationMonitor handles certificate expiration monitoring and calculations.
type ExpirationMonitor struct {
    log provider.Logger
}

// NewExpirationMonitor creates a new expiration monitor.
func NewExpirationMonitor(log provider.Logger) *ExpirationMonitor {
    return &ExpirationMonitor{
        log: log,
    }
}
```

2. **Implement ParseCertificateExpiration:**
```go
// ParseCertificateExpiration extracts the expiration date from an X.509 certificate.
// Returns the expiration time (NotAfter) and any error encountered.
func (em *ExpirationMonitor) ParseCertificateExpiration(cert *x509.Certificate) (time.Time, error) {
    if cert == nil {
        return time.Time{}, fmt.Errorf("certificate is nil")
    }
    
    if cert.NotAfter.IsZero() {
        return time.Time{}, fmt.Errorf("certificate has no expiration date")
    }
    
    return cert.NotAfter, nil
}
```

3. **Implement CalculateDaysUntilExpiration:**
```go
// CalculateDaysUntilExpiration calculates the number of days until certificate expiration.
// Uses UTC timezone for consistent calculations across timezones.
// Returns negative days if certificate is already expired.
func (em *ExpirationMonitor) CalculateDaysUntilExpiration(cert *x509.Certificate) (int, error) {
    expiration, err := em.ParseCertificateExpiration(cert)
    if err != nil {
        return 0, err
    }
    
    now := time.Now().UTC()
    expirationUTC := expiration.UTC()
    
    // Calculate duration until expiration
    duration := expirationUTC.Sub(now)
    days := int(duration.Hours() / 24)
    
    return days, nil
}
```

4. **Implement IsExpired:**
```go
// IsExpired checks if a certificate has expired.
// Returns true if the current time is after the certificate's NotAfter time.
func (em *ExpirationMonitor) IsExpired(cert *x509.Certificate) (bool, error) {
    if cert == nil {
        return false, fmt.Errorf("certificate is nil")
    }
    
    if cert.NotAfter.IsZero() {
        return false, fmt.Errorf("certificate has no expiration date")
    }
    
    now := time.Now().UTC()
    expirationUTC := cert.NotAfter.UTC()
    
    return now.After(expirationUTC), nil
}
```

5. **Implement IsExpiringSoon:**
```go
// IsExpiringSoon checks if a certificate is expiring within the specified threshold.
// thresholdDays is the number of days before expiration to consider "soon".
// Returns true if certificate expires within thresholdDays.
func (em *ExpirationMonitor) IsExpiringSoon(cert *x509.Certificate, thresholdDays int) (bool, error) {
    if thresholdDays < 0 {
        return false, fmt.Errorf("threshold days must be non-negative")
    }
    
    daysUntilExpiration, err := em.CalculateDaysUntilExpiration(cert)
    if err != nil {
        return false, err
    }
    
    // Certificate is expiring soon if days until expiration <= threshold
    return daysUntilExpiration <= thresholdDays, nil
}
```

**Testing:**
- Unit tests for each function
- Test with nil certificate
- Test with zero expiration date
- Test with expired certificate
- Test with certificate expiring today
- Test with certificate expiring in future
- Test timezone handling

---

### Task 2: Integrate Expiration Monitor with Certificate Manager

**File:** `internal/agent/device/certmanager/manager.go` (modify)

**Objective:** Integrate expiration monitoring into the certificate manager to check certificates on startup and periodically.

**Implementation Steps:**

1. **Add ExpirationMonitor to CertManager struct:**
```go
type CertManager struct {
    log provider.Logger
    // ... existing fields ...
    expirationMonitor *ExpirationMonitor  // NEW
    // ... rest of fields ...
}
```

2. **Initialize ExpirationMonitor in NewManager:**
```go
func NewManager(ctx context.Context, log provider.Logger, opts ...ManagerOption) (*CertManager, error) {
    cm := &CertManager{
        log:               log,
        certificates:      newCertStorage(),
        configs:           make(map[string]provider.ConfigProvider),
        provisioners:      make(map[string]provider.ProvisionerFactory),
        storages:          make(map[string]provider.StorageFactory),
        processingQueue:   NewCertificateProcessingQueue(),
        requeueDelay:      DefaultRequeueDelay,
        expirationMonitor: NewExpirationMonitor(log),  // NEW
    }
    
    // ... rest of initialization ...
}
```

3. **Add method to check certificate expiration:**
```go
// CheckCertificateExpiration checks the expiration status of a certificate.
// Returns days until expiration, expiration time, and any error.
func (cm *CertManager) CheckCertificateExpiration(ctx context.Context, providerName, certName string) (int, *time.Time, error) {
    cert := cm.certificates.GetCertificate(providerName, certName)
    if cert == nil {
        return 0, nil, fmt.Errorf("certificate %q from provider %q not found", certName, providerName)
    }
    
    // Try to load certificate from storage if not already loaded
    if cert.Info.NotAfter == nil {
        if cert.Storage == nil {
            // Initialize storage if needed
            storage, err := cm.initStorageProvider(cert.Config)
            if err != nil {
                return 0, nil, fmt.Errorf("failed to init storage: %w", err)
            }
            cert.Storage = storage
        }
        
        // Load certificate from storage
        x509Cert, err := cert.Storage.LoadCertificate(ctx)
        if err != nil {
            return 0, nil, fmt.Errorf("failed to load certificate: %w", err)
        }
        
        // Update certificate info
        cert.Info.NotBefore = &x509Cert.NotBefore
        cert.Info.NotAfter = &x509Cert.NotAfter
        
        // Persist updated info
        if err := cm.certificates.StoreCertificate(providerName, cert); err != nil {
            cm.log.Warnf("failed to store certificate info: %v", err)
        }
    }
    
    // Parse certificate expiration
    if cert.Info.NotAfter == nil {
        return 0, nil, fmt.Errorf("certificate has no expiration date")
    }
    
    // Calculate days until expiration
    days, err := cm.expirationMonitor.CalculateDaysUntilExpiration(&x509.Certificate{
        NotAfter: *cert.Info.NotAfter,
    })
    if err != nil {
        return 0, nil, err
    }
    
    return days, cert.Info.NotAfter, nil
}
```

**Testing:**
- Unit tests for CheckCertificateExpiration
- Integration test with certificate manager
- Test with certificate that has expiration info
- Test with certificate that needs loading from storage

---

### Task 3: Add Periodic Expiration Checking

**File:** `internal/agent/device/certmanager/manager.go` (modify)

**Objective:** Add periodic expiration checking that runs at configurable intervals.

**Implementation Steps:**

1. **Add periodic check method:**
```go
// CheckAllCertificatesExpiration checks expiration for all managed certificates.
// This method is intended to be called periodically.
func (cm *CertManager) CheckAllCertificatesExpiration(ctx context.Context) error {
    cm.certificates.mu.RLock()
    defer cm.certificates.mu.RUnlock()
    
    for providerName, provider := range cm.certificates.providers {
        provider.mu.RLock()
        for certName, cert := range provider.Certificates {
            days, expiration, err := cm.CheckCertificateExpiration(ctx, providerName, certName)
            if err != nil {
                cm.log.Warnf("failed to check expiration for certificate %q from provider %q: %v", 
                    certName, providerName, err)
                continue
            }
            
            if days < 0 {
                cm.log.Warnf("certificate %q from provider %q expired %d days ago (expired: %v)", 
                    certName, providerName, -days, expiration)
            } else {
                cm.log.Debugf("certificate %q from provider %q expires in %d days (expires: %v)", 
                    certName, providerName, days, expiration)
            }
        }
        provider.mu.RUnlock()
    }
    
    return nil
}
```

2. **Add StartPeriodicExpirationCheck method:**
```go
// StartPeriodicExpirationCheck starts a goroutine that periodically checks certificate expiration.
// The check interval is configurable via the checkInterval parameter.
// The goroutine stops when ctx is cancelled.
func (cm *CertManager) StartPeriodicExpirationCheck(ctx context.Context, checkInterval time.Duration) {
    if checkInterval <= 0 {
        cm.log.Warnf("invalid check interval %v, using default 24h", checkInterval)
        checkInterval = 24 * time.Hour
    }
    
    ticker := time.NewTicker(checkInterval)
    defer ticker.Stop()
    
    // Check immediately on startup
    if err := cm.CheckAllCertificatesExpiration(ctx); err != nil {
        cm.log.Warnf("initial expiration check failed: %v", err)
    }
    
    go func() {
        for {
            select {
            case <-ctx.Done():
                cm.log.Debug("stopping periodic expiration check")
                return
            case <-ticker.C:
                if err := cm.CheckAllCertificatesExpiration(ctx); err != nil {
                    cm.log.Warnf("periodic expiration check failed: %v", err)
                }
            }
        }
    }()
}
```

**Testing:**
- Unit tests for CheckAllCertificatesExpiration
- Unit tests for StartPeriodicExpirationCheck
- Integration test with ticker
- Test context cancellation stops the goroutine

---

### Task 4: Add Configuration Support

**File:** `internal/agent/config/config.go` (modify)

**Objective:** Add configuration fields for certificate renewal check interval.

**Implementation Steps:**

1. **Add CertificateRenewalConfig struct:**
```go
// CertificateRenewalConfig holds configuration for certificate renewal.
type CertificateRenewalConfig struct {
    // Enabled controls whether certificate renewal is enabled
    Enabled bool `json:"enabled,omitempty"`
    // CheckInterval is how often to check certificate expiration (default: 24h)
    CheckInterval util.Duration `json:"check-interval,omitempty"`
    // ThresholdDays is the number of days before expiration to trigger renewal (default: 30)
    ThresholdDays int `json:"threshold-days,omitempty"`
}

// DefaultCertificateRenewalConfig returns default certificate renewal configuration.
func DefaultCertificateRenewalConfig() CertificateRenewalConfig {
    return CertificateRenewalConfig{
        Enabled:       true,
        CheckInterval: util.Duration(24 * time.Hour),
        ThresholdDays: 30,
    }
}
```

2. **Add Certificate field to Config struct:**
```go
type Config struct {
    config.ServiceConfig
    
    // ... existing fields ...
    
    // Certificate holds certificate management configuration
    Certificate CertificateConfig `json:"certificate,omitempty"`
}

// CertificateConfig holds all certificate-related configuration.
type CertificateConfig struct {
    // Renewal holds certificate renewal configuration
    Renewal CertificateRenewalConfig `json:"renewal,omitempty"`
}
```

3. **Add validation in config loading:**
```go
// ValidateCertificateRenewalConfig validates certificate renewal configuration.
func (c *CertificateRenewalConfig) Validate() error {
    if c.CheckInterval.Duration < time.Hour {
        return fmt.Errorf("check-interval must be at least 1 hour, got %v", c.CheckInterval)
    }
    
    if c.ThresholdDays < 1 || c.ThresholdDays > 365 {
        return fmt.Errorf("threshold-days must be between 1 and 365, got %d", c.ThresholdDays)
    }
    
    return nil
}
```

**Testing:**
- Unit tests for configuration parsing
- Unit tests for configuration validation
- Test default values
- Test invalid configurations are rejected

---

### Task 5: Integrate with Agent Main Loop

**File:** `internal/agent/agent.go` (modify)

**Objective:** Initialize expiration monitoring in the agent and start periodic checks.

**Implementation Steps:**

1. **Find where CertManager is initialized:**
Look for `certManager, err := certmanager.NewManager(...)` in `internal/agent/agent.go`

2. **Start periodic expiration check after CertManager initialization:**
```go
// Initialize certificate manager
certManager, err := certmanager.NewManager(
    ctx, a.log,
    certmanager.WithBuiltins(
        deviceName,
        bootstrap.ManagementClient(),
        deviceReadWriter,
        a.config,
        identity.NewExportableFactory(tpmClient, a.log),
    ),
)
if err != nil {
    return fmt.Errorf("failed to initialize certificate manager: %w", err)
}

if err := certManager.Sync(ctx, a.config); err != nil {
    a.log.Warnf("Failed to sync certificate manager: %v", err)
}

// Start periodic expiration checking
if a.config.Certificate.Renewal.Enabled {
    checkInterval := a.config.Certificate.Renewal.CheckInterval.Duration
    if checkInterval == 0 {
        checkInterval = 24 * time.Hour // Default
    }
    certManager.StartPeriodicExpirationCheck(ctx, checkInterval)
    a.log.Infof("Started periodic certificate expiration checking (interval: %v)", checkInterval)
}
```

**Testing:**
- Integration test with agent startup
- Test that periodic check starts when enabled
- Test that periodic check doesn't start when disabled
- Test with custom check interval

---

### Task 6: Update Certificate Info on Load

**File:** `internal/agent/device/certmanager/manager.go` (modify)

**Objective:** Ensure certificate expiration info is populated when certificates are loaded.

**Implementation Steps:**

1. **Modify ensureCertificate_do to populate expiration info:**
In the `ensureCertificate_do` method, after loading certificate from storage, update the certificate info:

```go
// In ensureCertificate_do method, after loading certificate:
if err := cert.Storage.Write(crt, keyBytes); err != nil {
    return nil, err
}

// Update certificate info with expiration dates
cm.addCertificateInfo(cert, crt)

// ... rest of method ...
```

The `addCertificateInfo` method already exists and should populate `cert.Info.NotBefore` and `cert.Info.NotAfter`.

2. **Ensure expiration info is loaded on startup:**
In the `createCertificate` method, when loading existing certificate:

```go
// Try to load existing certificate details from storage provider
storage, err := cm.initStorageProvider(cfg)
if err == nil {
    parsedCert, loadErr := storage.LoadCertificate(ctx)
    if loadErr == nil && parsedCert != nil {
        cm.addCertificateInfo(cert, parsedCert)
        // Expiration info is now populated
    } else if loadErr != nil {
        cm.log.Debugf("no existing cert loaded for %q/%q: %v", providerName, certName, loadErr)
    }
} else {
    cm.log.Errorf("failed to init storage provider for certificate %q from provider %q: %v", certName, providerName, err)
}
```

**Testing:**
- Test that expiration info is populated on certificate load
- Test that expiration info persists across agent restarts
- Test with certificates that don't exist yet

---

## Unit Tests

### Test File: `internal/agent/device/certmanager/expiration_test.go` (new)

**Test Cases:**

1. **TestParseCertificateExpiration:**
   - Valid certificate with expiration
   - Nil certificate (error)
   - Certificate with zero expiration (error)

2. **TestCalculateDaysUntilExpiration:**
   - Certificate expiring in 30 days
   - Certificate expiring today
   - Certificate expired 10 days ago (negative days)
   - Certificate expiring in 1 year
   - Timezone handling (UTC vs local)

3. **TestIsExpired:**
   - Expired certificate (true)
   - Valid certificate (false)
   - Certificate expiring today (false, not yet expired)
   - Nil certificate (error)

4. **TestIsExpiringSoon:**
   - Certificate expiring in 25 days with threshold 30 (true)
   - Certificate expiring in 35 days with threshold 30 (false)
   - Certificate expiring today with threshold 30 (true)
   - Certificate expired with threshold 30 (true)
   - Negative threshold (error)

### Test File: `internal/agent/device/certmanager/manager_expiration_test.go` (new)

**Test Cases:**

1. **TestCheckCertificateExpiration:**
   - Certificate with expiration info
   - Certificate needing load from storage
   - Certificate not found (error)
   - Certificate with no expiration date (error)

2. **TestCheckAllCertificatesExpiration:**
   - Multiple certificates with various expiration states
   - Certificates from multiple providers
   - Error handling for individual certificate failures

3. **TestStartPeriodicExpirationCheck:**
   - Periodic check runs at specified interval
   - Check runs immediately on startup
   - Context cancellation stops the goroutine
   - Invalid interval uses default

---

## Integration Tests

### Test File: `test/integration/certificate_expiration_test.go` (new)

**Test Cases:**

1. **TestExpirationMonitoringOnStartup:**
   - Agent starts with existing certificate
   - Expiration is checked on startup
   - Expiration info is logged

2. **TestPeriodicExpirationChecking:**
   - Agent runs with periodic checking enabled
   - Expiration checks occur at configured interval
   - Multiple checks over time

3. **TestExpirationInfoPersistence:**
   - Certificate expiration info is stored
   - Info persists across agent restarts
   - Info is updated when certificate changes

---

## Code Review Checklist

- [ ] All functions have proper error handling
- [ ] Timezone handling uses UTC consistently
- [ ] Logging is appropriate (debug for normal operations, warn for issues)
- [ ] Thread safety is maintained (mutex usage where needed)
- [ ] Configuration validation is comprehensive
- [ ] Unit tests achieve >80% coverage
- [ ] Integration tests cover main flows
- [ ] Code follows Go best practices
- [ ] Documentation comments are complete
- [ ] No hardcoded values (use configuration)

---

## Definition of Done

- [ ] `expiration.go` file created with all functions
- [ ] `CertManager` integrates expiration monitor
- [ ] Periodic checking implemented and started
- [ ] Configuration schema added and validated
- [ ] Agent main loop integrates expiration checking
- [ ] Certificate info populated on load
- [ ] Unit tests written and passing (>80% coverage)
- [ ] Integration tests written and passing
- [ ] Code reviewed and approved
- [ ] Documentation updated (code comments, README if needed)

---

## Related Files

- `internal/agent/device/certmanager/manager.go` - Certificate manager
- `internal/agent/device/certmanager/certificate.go` - Certificate struct and info
- `internal/agent/device/certmanager/provider/storage/fs.go` - Certificate storage
- `internal/agent/config/config.go` - Agent configuration
- `internal/agent/agent.go` - Agent main loop
- `pkg/crypto/crypto.go` - Certificate parsing utilities

---

## Notes

- Use UTC for all time calculations to avoid timezone issues
- Expiration info should be cached in memory but refreshed periodically
- Consider performance: don't load certificates from disk on every check if info is cached
- Log expiration status at appropriate levels (debug for normal, warn for expiring/expired)
- Ensure thread safety when accessing certificate info concurrently

---

**Document End**

