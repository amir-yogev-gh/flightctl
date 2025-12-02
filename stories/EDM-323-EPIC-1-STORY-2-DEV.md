# Developer Story: Certificate Lifecycle Manager Structure

**Story ID:** EDM-323-EPIC-1-STORY-2  
**Epic:** EDM-323-EPIC-1 (Certificate Lifecycle Foundation)  
**Feature:** EDM-323 (Agent Certificate Rotation)  
**Story Points:** 5  
**Priority:** High

## Overview

Implement a certificate lifecycle manager that coordinates certificate operations, tracks certificate lifecycle states, and determines when renewal is needed based on expiration thresholds.

## Implementation Tasks

### Task 1: Define Certificate Lifecycle States

**File:** `internal/agent/device/certmanager/lifecycle.go` (new)

**Objective:** Define the certificate lifecycle states and state management types.

**Implementation Steps:**

1. **Create the file structure with state definitions:**
```go
package certmanager

import (
    "context"
    "crypto/x509"
    "fmt"
    "sync"
    "time"
    
    "github.com/flightctl/flightctl/internal/agent/device/certmanager/provider"
)

// CertificateState represents the current lifecycle state of a certificate.
type CertificateState string

const (
    // CertificateStateNormal indicates the certificate is valid and not expiring soon
    CertificateStateNormal CertificateState = "normal"
    
    // CertificateStateExpiringSoon indicates the certificate is expiring within the threshold
    CertificateStateExpiringSoon CertificateState = "expiring_soon"
    
    // CertificateStateExpired indicates the certificate has expired
    CertificateStateExpired CertificateState = "expired"
    
    // CertificateStateRenewing indicates the certificate renewal is in progress
    CertificateStateRenewing CertificateState = "renewing"
    
    // CertificateStateRecovering indicates expired certificate recovery is in progress
    CertificateStateRecovering CertificateState = "recovering"
    
    // CertificateStateRenewalFailed indicates the last renewal attempt failed
    CertificateStateRenewalFailed CertificateState = "renewal_failed"
)

// String returns the string representation of the certificate state.
func (s CertificateState) String() string {
    return string(s)
}

// IsValidState checks if the state is a valid certificate state.
func (s CertificateState) IsValidState() bool {
    switch s {
    case CertificateStateNormal,
         CertificateStateExpiringSoon,
         CertificateStateExpired,
         CertificateStateRenewing,
         CertificateStateRecovering,
         CertificateStateRenewalFailed:
        return true
    default:
        return false
    }
}
```

2. **Define lifecycle state information:**
```go
// CertificateLifecycleState holds the lifecycle state and related information for a certificate.
type CertificateLifecycleState struct {
    // State is the current lifecycle state
    State CertificateState `json:"state"`
    
    // DaysUntilExpiration is the number of days until expiration (negative if expired)
    DaysUntilExpiration int `json:"days_until_expiration,omitempty"`
    
    // ExpirationTime is when the certificate expires
    ExpirationTime *time.Time `json:"expiration_time,omitempty"`
    
    // LastChecked is when the state was last checked
    LastChecked time.Time `json:"last_checked,omitempty"`
    
    // LastError is the last error encountered during lifecycle operations
    LastError string `json:"last_error,omitempty"`
    
    // Mutex for thread-safe access
    mu sync.RWMutex `json:"-"`
}

// NewCertificateLifecycleState creates a new lifecycle state with the given state.
func NewCertificateLifecycleState(state CertificateState) *CertificateLifecycleState {
    return &CertificateLifecycleState{
        State:       state,
        LastChecked: time.Now().UTC(),
    }
}

// GetState returns the current state (thread-safe).
func (cls *CertificateLifecycleState) GetState() CertificateState {
    cls.mu.RLock()
    defer cls.mu.RUnlock()
    return cls.State
}

// SetState updates the state (thread-safe).
func (cls *CertificateLifecycleState) SetState(state CertificateState) {
    cls.mu.Lock()
    defer cls.mu.Unlock()
    cls.State = state
    cls.LastChecked = time.Now().UTC()
}

// Update updates the lifecycle state with new information.
func (cls *CertificateLifecycleState) Update(state CertificateState, daysUntilExpiration int, expirationTime *time.Time) {
    cls.mu.Lock()
    defer cls.mu.Unlock()
    cls.State = state
    cls.DaysUntilExpiration = daysUntilExpiration
    cls.ExpirationTime = expirationTime
    cls.LastChecked = time.Now().UTC()
    cls.LastError = "" // Clear error on successful update
}

// SetError records an error in the lifecycle state.
func (cls *CertificateLifecycleState) SetError(err error) {
    cls.mu.Lock()
    defer cls.mu.Unlock()
    if err != nil {
        cls.LastError = err.Error()
    } else {
        cls.LastError = ""
    }
}
```

**Testing:**
- Unit tests for state definitions
- Unit tests for state validation
- Unit tests for CertificateLifecycleState methods
- Test thread safety of state updates

---

### Task 2: Create Certificate Lifecycle Manager Interface

**File:** `internal/agent/device/certmanager/lifecycle.go` (continue)

**Objective:** Define the interface for certificate lifecycle management.

**Implementation Steps:**

1. **Define the CertificateLifecycleManager interface:**
```go
// CertificateLifecycleManager defines the interface for managing certificate lifecycle.
type CertificateLifecycleManager interface {
    // CheckRenewal checks if a certificate needs renewal based on expiration threshold.
    // Returns true if renewal is needed, days until expiration, and any error.
    CheckRenewal(ctx context.Context, providerName, certName string, thresholdDays int) (bool, int, error)
    
    // GetCertificateState returns the current lifecycle state of a certificate.
    GetCertificateState(ctx context.Context, providerName, certName string) (*CertificateLifecycleState, error)
    
    // SetCertificateState updates the lifecycle state of a certificate.
    SetCertificateState(ctx context.Context, providerName, certName string, state CertificateState) error
    
    // UpdateCertificateState updates the lifecycle state with full information.
    UpdateCertificateState(ctx context.Context, providerName, certName string, state CertificateState, daysUntilExpiration int, expirationTime *time.Time) error
    
    // RecordError records an error for a certificate's lifecycle operations.
    RecordError(ctx context.Context, providerName, certName string, err error) error
}
```

---

### Task 3: Implement Lifecycle Manager

**File:** `internal/agent/device/certmanager/lifecycle.go` (continue)

**Objective:** Implement the lifecycle manager that integrates with CertManager and ExpirationMonitor.

**Implementation Steps:**

1. **Create the LifecycleManager struct:**
```go
// LifecycleManager implements CertificateLifecycleManager.
// It coordinates certificate lifecycle operations and state management.
type LifecycleManager struct {
    // certManager is the certificate manager to interact with
    certManager *CertManager
    
    // expirationMonitor is used to check certificate expiration
    expirationMonitor *ExpirationMonitor
    
    // lifecycleStates tracks the lifecycle state for each certificate
    // Key format: "providerName/certName"
    lifecycleStates map[string]*CertificateLifecycleState
    
    // Mutex for thread-safe access to lifecycle states
    mu sync.RWMutex
    
    // Logger for lifecycle operations
    log provider.Logger
}

// NewLifecycleManager creates a new lifecycle manager.
func NewLifecycleManager(certManager *CertManager, expirationMonitor *ExpirationMonitor, log provider.Logger) *LifecycleManager {
    return &LifecycleManager{
        certManager:      certManager,
        expirationMonitor: expirationMonitor,
        lifecycleStates:  make(map[string]*CertificateLifecycleState),
        log:              log,
    }
}

// stateKey generates a key for the lifecycle state map.
func (lm *LifecycleManager) stateKey(providerName, certName string) string {
    return fmt.Sprintf("%s/%s", providerName, certName)
}
```

2. **Implement CheckRenewal:**
```go
// CheckRenewal checks if a certificate needs renewal based on expiration threshold.
func (lm *LifecycleManager) CheckRenewal(ctx context.Context, providerName, certName string, thresholdDays int) (bool, int, error) {
    if thresholdDays < 0 {
        return false, 0, fmt.Errorf("threshold days must be non-negative, got %d", thresholdDays)
    }
    
    // Get certificate from cert manager
    cert, err := lm.certManager.certificates.ReadCertificate(providerName, certName)
    if err != nil {
        return false, 0, fmt.Errorf("failed to read certificate: %w", err)
    }
    
    cert.mu.RLock()
    certInfo := cert.Info
    cert.mu.RUnlock()
    
    // Check if certificate has expiration info
    if certInfo.NotAfter == nil {
        lm.log.Debugf("Certificate %q/%q has no expiration info, cannot check renewal", providerName, certName)
        return false, 0, fmt.Errorf("certificate has no expiration date")
    }
    
    // Load certificate from storage to get full X.509 certificate
    storage, err := lm.certManager.initStorageProvider(cert.Config)
    if err != nil {
        return false, 0, fmt.Errorf("failed to init storage: %w", err)
    }
    
    x509Cert, err := storage.LoadCertificate(ctx)
    if err != nil {
        return false, 0, fmt.Errorf("failed to load certificate: %w", err)
    }
    
    // Calculate days until expiration
    days, err := lm.expirationMonitor.CalculateDaysUntilExpiration(x509Cert)
    if err != nil {
        return false, 0, fmt.Errorf("failed to calculate days until expiration: %w", err)
    }
    
    // Check if expired
    isExpired, err := lm.expirationMonitor.IsExpired(x509Cert)
    if err != nil {
        return false, days, fmt.Errorf("failed to check if expired: %w", err)
    }
    
    if isExpired {
        lm.log.Debugf("Certificate %q/%q is expired (%d days ago)", providerName, certName, -days)
        // Update state to expired
        _ = lm.UpdateCertificateState(ctx, providerName, certName, CertificateStateExpired, days, certInfo.NotAfter)
        return true, days, nil // Expired certificates need renewal (recovery)
    }
    
    // Check if expiring soon
    isExpiringSoon, err := lm.expirationMonitor.IsExpiringSoon(x509Cert, thresholdDays)
    if err != nil {
        return false, days, fmt.Errorf("failed to check if expiring soon: %w", err)
    }
    
    if isExpiringSoon {
        lm.log.Debugf("Certificate %q/%q is expiring soon (%d days until expiration)", providerName, certName, days)
        // Update state to expiring_soon
        _ = lm.UpdateCertificateState(ctx, providerName, certName, CertificateStateExpiringSoon, days, certInfo.NotAfter)
        return true, days, nil // Needs renewal
    }
    
    // Certificate is normal (not expiring soon)
    _ = lm.UpdateCertificateState(ctx, providerName, certName, CertificateStateNormal, days, certInfo.NotAfter)
    return false, days, nil
}
```

3. **Implement GetCertificateState:**
```go
// GetCertificateState returns the current lifecycle state of a certificate.
func (lm *LifecycleManager) GetCertificateState(ctx context.Context, providerName, certName string) (*CertificateLifecycleState, error) {
    key := lm.stateKey(providerName, certName)
    
    lm.mu.RLock()
    state, exists := lm.lifecycleStates[key]
    lm.mu.RUnlock()
    
    if !exists {
        // Return default state if not tracked yet
        return NewCertificateLifecycleState(CertificateStateNormal), nil
    }
    
    // Return a copy to avoid race conditions
    state.mu.RLock()
    copy := &CertificateLifecycleState{
        State:              state.State,
        DaysUntilExpiration: state.DaysUntilExpiration,
        ExpirationTime:     state.ExpirationTime,
        LastChecked:        state.LastChecked,
        LastError:          state.LastError,
    }
    state.mu.RUnlock()
    
    return copy, nil
}
```

4. **Implement SetCertificateState:**
```go
// SetCertificateState updates the lifecycle state of a certificate.
func (lm *LifecycleManager) SetCertificateState(ctx context.Context, providerName, certName string, newState CertificateState) error {
    if !newState.IsValidState() {
        return fmt.Errorf("invalid certificate state: %s", newState)
    }
    
    key := lm.stateKey(providerName, certName)
    
    lm.mu.Lock()
    state, exists := lm.lifecycleStates[key]
    if !exists {
        state = NewCertificateLifecycleState(newState)
        lm.lifecycleStates[key] = state
    } else {
        state.SetState(newState)
    }
    lm.mu.Unlock()
    
    lm.log.Debugf("Certificate %q/%q state updated to %s", providerName, certName, newState)
    return nil
}
```

5. **Implement UpdateCertificateState:**
```go
// UpdateCertificateState updates the lifecycle state with full information.
func (lm *LifecycleManager) UpdateCertificateState(ctx context.Context, providerName, certName string, state CertificateState, daysUntilExpiration int, expirationTime *time.Time) error {
    if !state.IsValidState() {
        return fmt.Errorf("invalid certificate state: %s", state)
    }
    
    key := lm.stateKey(providerName, certName)
    
    lm.mu.Lock()
    lifecycleState, exists := lm.lifecycleStates[key]
    if !exists {
        lifecycleState = NewCertificateLifecycleState(state)
        lm.lifecycleStates[key] = lifecycleState
    }
    lifecycleState.Update(state, daysUntilExpiration, expirationTime)
    lm.mu.Unlock()
    
    lm.log.Debugf("Certificate %q/%q state updated: %s (expires in %d days)", providerName, certName, state, daysUntilExpiration)
    return nil
}
```

6. **Implement RecordError:**
```go
// RecordError records an error for a certificate's lifecycle operations.
func (lm *LifecycleManager) RecordError(ctx context.Context, providerName, certName string, err error) error {
    key := lm.stateKey(providerName, certName)
    
    lm.mu.Lock()
    state, exists := lm.lifecycleStates[key]
    if !exists {
        state = NewCertificateLifecycleState(CertificateStateNormal)
        lm.lifecycleStates[key] = state
    }
    state.SetError(err)
    
    // Update state to renewal_failed if currently renewing
    if state.GetState() == CertificateStateRenewing {
        state.SetState(CertificateStateRenewalFailed)
    }
    lm.mu.Unlock()
    
    if err != nil {
        lm.log.Warnf("Certificate %q/%q lifecycle error: %v", providerName, certName, err)
    }
    
    return nil
}
```

**Testing:**
- Unit tests for all lifecycle manager methods
- Test state transitions
- Test error handling
- Test thread safety
- Test with non-existent certificates

---

### Task 4: Extend Certificate Struct with Lifecycle State

**File:** `internal/agent/device/certmanager/certificate.go` (modify)

**Objective:** Add lifecycle state tracking to the certificate struct.

**Implementation Steps:**

1. **Add lifecycle state to CertificateInfo:**
```go
// CertificateInfo contains parsed certificate metadata.
type CertificateInfo struct {
    // Certificate validity start time
    NotBefore *time.Time `json:"not_before,omitempty"`
    // Certificate validity end time (expiration)
    NotAfter *time.Time `json:"not_after,omitempty"`
    // LifecycleState is the current lifecycle state of the certificate
    LifecycleState CertificateState `json:"lifecycle_state,omitempty"`
}
```

2. **Add method to get lifecycle state from certificate:**
```go
// GetLifecycleState returns the lifecycle state from certificate info.
func (c *certificate) GetLifecycleState() CertificateState {
    c.mu.RLock()
    defer c.mu.RUnlock()
    return c.Info.LifecycleState
}

// SetLifecycleState updates the lifecycle state in certificate info.
func (c *certificate) SetLifecycleState(state CertificateState) {
    c.mu.Lock()
    defer c.mu.Unlock()
    c.Info.LifecycleState = state
}
```

**Testing:**
- Unit tests for lifecycle state getter/setter
- Test thread safety

---

### Task 5: Integrate Lifecycle Manager with CertManager

**File:** `internal/agent/device/certmanager/manager.go` (modify)

**Objective:** Integrate the lifecycle manager into CertManager.

**Implementation Steps:**

1. **Add LifecycleManager to CertManager struct:**
```go
type CertManager struct {
    log provider.Logger
    // ... existing fields ...
    expirationMonitor *ExpirationMonitor
    lifecycleManager  *LifecycleManager  // NEW
    // ... rest of fields ...
}
```

2. **Initialize LifecycleManager in NewManager:**
```go
func NewManager(ctx context.Context, log provider.Logger, opts ...ManagerOption) (*CertManager, error) {
    if log == nil {
        return nil, fmt.Errorf("logger is nil")
    }

    cm := &CertManager{
        log:          log,
        configs:      make(map[string]provider.ConfigProvider),
        provisioners: make(map[string]provider.ProvisionerFactory),
        storages:     make(map[string]provider.StorageFactory),
        certificates: newCertStorage(),
    }

    // ... apply options ...

    if cm.requeueDelay == 0 {
        cm.requeueDelay = DefaultRequeueDelay
    }

    // Initialize expiration monitor
    if cm.expirationMonitor == nil {
        cm.expirationMonitor = NewExpirationMonitor(log)
    }
    
    // Initialize lifecycle manager
    cm.lifecycleManager = NewLifecycleManager(cm, cm.expirationMonitor, log)

    cm.processingQueue = NewCertificateProcessingQueue(cm.ensureCertificate)
    go cm.processingQueue.Run(ctx)
    return cm, nil
}
```

3. **Add method to get lifecycle manager:**
```go
// GetLifecycleManager returns the lifecycle manager.
func (cm *CertManager) GetLifecycleManager() *LifecycleManager {
    return cm.lifecycleManager
}
```

4. **Add method to check renewal for a certificate:**
```go
// CheckCertificateRenewal checks if a certificate needs renewal.
// This is a convenience method that delegates to the lifecycle manager.
func (cm *CertManager) CheckCertificateRenewal(ctx context.Context, providerName, certName string, thresholdDays int) (bool, int, error) {
    if cm.lifecycleManager == nil {
        return false, 0, fmt.Errorf("lifecycle manager not initialized")
    }
    return cm.lifecycleManager.CheckRenewal(ctx, providerName, certName, thresholdDays)
}
```

**Testing:**
- Unit tests for lifecycle manager integration
- Test initialization
- Test delegation methods

---

### Task 6: Update Certificate State During Operations

**File:** `internal/agent/device/certmanager/manager.go` (modify)

**Objective:** Update certificate lifecycle state during certificate operations.

**Implementation Steps:**

1. **Update state when certificate is provisioned:**
In `ensureCertificate_do`, after successful certificate write:

```go
// After successful certificate write in ensureCertificate_do:
if err := cert.Storage.Write(crt, keyBytes); err != nil {
    return nil, err
}

cm.addCertificateInfo(cert, crt)

// Update lifecycle state to normal after successful provisioning
if cm.lifecycleManager != nil {
    _ = cm.lifecycleManager.SetCertificateState(ctx, providerName, cert.Name, CertificateStateNormal)
}

cert.Config = config
// ... rest of method ...
```

2. **Update state when renewal starts:**
Add a method to mark certificate as renewing:

```go
// MarkCertificateRenewing marks a certificate as being renewed.
func (cm *CertManager) MarkCertificateRenewing(ctx context.Context, providerName, certName string) error {
    if cm.lifecycleManager == nil {
        return fmt.Errorf("lifecycle manager not initialized")
    }
    return cm.lifecycleManager.SetCertificateState(ctx, providerName, certName, CertificateStateRenewing)
}
```

3. **Update state on errors:**
In error handling paths, record errors:

```go
// In ensureCertificate, on error:
if err != nil {
    cm.log.Errorf("failed to ensure certificate %q from provider %q: %v", cert.Name, providerName, err)
    
    // Record error in lifecycle manager
    if cm.lifecycleManager != nil {
        _ = cm.lifecycleManager.RecordError(ctx, providerName, cert.Name, err)
    }
    
    // ... rest of error handling ...
}
```

**Testing:**
- Test state updates during provisioning
- Test state updates during renewal
- Test error recording
- Test state persistence

---

### Task 7: Add Lifecycle State Checking to Sync

**File:** `internal/agent/device/certmanager/manager.go` (modify)

**Objective:** Check certificate lifecycle state during sync operations.

**Implementation Steps:**

1. **Add lifecycle state check to syncCertificate:**
```go
// In syncCertificate, after checking if provisioning is needed:
if !cm.shouldprovisionCertificate(providerName, cert, cfg) {
    cert.Config = cfg
    
    // Check if renewal is needed based on expiration
    if cm.lifecycleManager != nil {
        thresholdDays := 30 // Default, should come from config
        needsRenewal, days, err := cm.lifecycleManager.CheckRenewal(ctx, providerName, certName, thresholdDays)
        if err != nil {
            cm.log.Warnf("Failed to check renewal for certificate %q/%q: %v", providerName, certName, err)
        } else if needsRenewal {
            cm.log.Infof("Certificate %q/%q needs renewal (expires in %d days)", providerName, certName, days)
            // Renewal will be triggered in a future story
        }
    }
    
    cm.log.Debugf("Certificate %q for provider %q: no provision required", certName, providerName)
    return nil
}
```

**Testing:**
- Test lifecycle check during sync
- Test renewal detection
- Test error handling during lifecycle check

---

## Unit Tests

### Test File: `internal/agent/device/certmanager/lifecycle_test.go` (new)

**Test Cases:**

1. **TestCertificateState:**
   - Valid states
   - Invalid states
   - String representation
   - IsValidState method

2. **TestCertificateLifecycleState:**
   - State creation
   - State updates
   - Error recording
   - Thread safety

3. **TestLifecycleManager_CheckRenewal:**
   - Certificate expiring soon (needs renewal)
   - Certificate expired (needs renewal)
   - Certificate normal (no renewal)
   - Certificate not found (error)
   - Invalid threshold (error)

4. **TestLifecycleManager_GetCertificateState:**
   - Existing state
   - Non-existent certificate (default state)
   - State copy (no race conditions)

5. **TestLifecycleManager_SetCertificateState:**
   - Valid state transitions
   - Invalid state (error)
   - State persistence

6. **TestLifecycleManager_UpdateCertificateState:**
   - Full state update
   - Invalid state (error)
   - State persistence

7. **TestLifecycleManager_RecordError:**
   - Error recording
   - State transition to renewal_failed
   - Error clearing

8. **TestLifecycleManager_StateTransitions:**
   - Normal → ExpiringSoon
   - ExpiringSoon → Renewing
   - Renewing → Normal (success)
   - Renewing → RenewalFailed (error)
   - Expired → Recovering
   - Recovering → Normal (success)

### Test File: `internal/agent/device/certmanager/manager_lifecycle_test.go` (new)

**Test Cases:**

1. **TestCertManager_LifecycleManagerIntegration:**
   - Lifecycle manager initialization
   - GetLifecycleManager method
   - CheckCertificateRenewal delegation

2. **TestCertManager_StateUpdates:**
   - State update during provisioning
   - State update during renewal
   - Error recording

3. **TestCertManager_SyncWithLifecycleCheck:**
   - Lifecycle check during sync
   - Renewal detection
   - Error handling

---

## Integration Tests

### Test File: `test/integration/certificate_lifecycle_test.go` (new)

**Test Cases:**

1. **TestLifecycleStateTracking:**
   - Certificate lifecycle state is tracked
   - State persists across operations
   - State transitions correctly

2. **TestRenewalDetection:**
   - Expiring certificate detected
   - Expired certificate detected
   - Normal certificate not flagged

3. **TestStateTransitions:**
   - Complete state transition flow
   - Error handling during transitions
   - State recovery after errors

---

## Code Review Checklist

- [ ] All state transitions are valid
- [ ] Thread safety is maintained (mutex usage)
- [ ] Error handling is comprehensive
- [ ] Logging is appropriate (debug for normal, warn for issues)
- [ ] Lifecycle manager integrates properly with CertManager
- [ ] State persistence works correctly
- [ ] Unit tests achieve >80% coverage
- [ ] Integration tests cover main flows
- [ ] Code follows Go best practices
- [ ] Documentation comments are complete

---

## Definition of Done

- [ ] `lifecycle.go` file created with all types and methods
- [ ] CertificateState enum defined with all states
- [ ] CertificateLifecycleState struct implemented
- [ ] CertificateLifecycleManager interface defined
- [ ] LifecycleManager implementation complete
- [ ] Certificate struct extended with lifecycle state
- [ ] CertManager integrates lifecycle manager
- [ ] State updates during certificate operations
- [ ] Lifecycle check integrated into sync
- [ ] Unit tests written and passing (>80% coverage)
- [ ] Integration tests written and passing
- [ ] Code reviewed and approved
- [ ] Documentation updated (code comments, README if needed)

---

## Related Files

- `internal/agent/device/certmanager/manager.go` - Certificate manager
- `internal/agent/device/certmanager/certificate.go` - Certificate struct
- `internal/agent/device/certmanager/expiration.go` - Expiration monitor (from STORY-1)
- `internal/agent/config/config.go` - Agent configuration

---

## Dependencies

- **EDM-323-EPIC-1-STORY-1**: Certificate Expiration Monitoring (must be completed first)
  - Requires `ExpirationMonitor` to be available
  - Uses expiration checking methods

---

## Notes

- Lifecycle states are tracked in memory; consider persistence if needed for recovery
- State transitions should be logged for debugging
- Errors should not prevent state updates (log and continue)
- Thread safety is critical - use mutexes appropriately
- State should be checked periodically, not just on sync
- Consider adding metrics for state distribution across fleet

---

**Document End**

