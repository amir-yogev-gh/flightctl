# Developer Story: Device Status Certificate State Indicators

**Story ID:** EDM-323-EPIC-5-STORY-2  
**Epic:** EDM-323-EPIC-5 (Configuration and Observability)  
**Feature:** EDM-323 (Agent Certificate Rotation)  
**Story Points:** 3  
**Priority:** Medium

## Overview

Add certificate state information to device status reporting. This enables fleet administrators to quickly identify devices with certificate issues through the device status API.

## Implementation Tasks

### Task 1: Add CertificateStatus to DeviceStatus

**File:** `api/v1beta1/types.go` (modify)

**Objective:** Add CertificateStatus struct to DeviceStatus.

**Implementation Steps:**

1. **Add CertificateStatus struct:**
```go
// CertificateStatus contains certificate lifecycle information for device status.
type CertificateStatus struct {
    // Expiration is when the certificate expires
    Expiration *time.Time `json:"expiration,omitempty"`
    
    // DaysUntilExpiration is the number of days until expiration (negative if expired)
    DaysUntilExpiration *int `json:"daysUntilExpiration,omitempty"`
    
    // State is the current certificate lifecycle state
    // Values: normal, expiring_soon, renewing, expired, recovering, renewal_failed
    State string `json:"state,omitempty"`
    
    // LastRenewed is when the certificate was last renewed
    LastRenewed *time.Time `json:"lastRenewed,omitempty"`
    
    // RenewalCount is the number of times the certificate has been renewed
    RenewalCount *int `json:"renewalCount,omitempty"`
}
```

2. **Add CertificateStatus to DeviceStatus:**
```go
// In DeviceStatus struct, add:
type DeviceStatus struct {
    // ... existing fields ...
    
    // Certificate contains certificate lifecycle status
    Certificate *CertificateStatus `json:"certificate,omitempty"`
}
```

**Testing:**
- Test CertificateStatus struct can be marshaled
- Test CertificateStatus is included in DeviceStatus

---

### Task 2: Create Certificate Status Exporter

**File:** `internal/agent/device/status/certificate.go` (new)

**Objective:** Create status exporter for certificate information.

**Implementation Steps:**

1. **Create certificate.go file:**
```go
package status

import (
    "context"
    "time"

    "github.com/flightctl/flightctl/api/v1beta1"
    "github.com/flightctl/flightctl/internal/agent/device/certmanager"
    "github.com/flightctl/flightctl/pkg/log"
)

// CertificateExporter exports certificate status information.
type CertificateExporter struct {
    certManager *certmanager.CertManager
    log         *log.PrefixLogger
}

// NewCertificateExporter creates a new certificate status exporter.
func NewCertificateExporter(certManager *certmanager.CertManager, log *log.PrefixLogger) *CertificateExporter {
    return &CertificateExporter{
        certManager: certManager,
        log:         log,
    }
}

// Status implements status.Exporter.
func (ce *CertificateExporter) Status(ctx context.Context, deviceStatus *v1beta1.DeviceStatus, opts ...CollectorOpt) error {
    if ce.certManager == nil {
        return nil // No certificate manager - skip
    }

    // Get certificate information from certificate manager
    certStatus, err := ce.getCertificateStatus(ctx)
    if err != nil {
        ce.log.Warnf("Failed to get certificate status: %v", err)
        // Continue with empty status
        return nil
    }

    deviceStatus.Certificate = certStatus
    return nil
}

// getCertificateStatus collects certificate status information.
func (ce *CertificateExporter) getCertificateStatus(ctx context.Context) (*v1beta1.CertificateStatus, error) {
    // Get management certificate status
    // This requires access to certificate manager's lifecycle manager
    
    // For now, placeholder implementation
    // Full implementation will query certificate manager for status
    
    status := &v1beta1.CertificateStatus{
        State: "normal",
    }
    
    return status, nil
}
```

**Testing:**
- Test CertificateExporter is created correctly
- Test Status method collects information

---

### Task 3: Implement Certificate Status Collection

**File:** `internal/agent/device/status/certificate.go` (modify)

**Objective:** Implement certificate status collection from certificate manager.

**Implementation Steps:**

1. **Implement getCertificateStatus:**
```go
// getCertificateStatus collects certificate status information.
func (ce *CertificateExporter) getCertificateStatus(ctx context.Context) (*v1beta1.CertificateStatus, error) {
    // Get certificate manager's lifecycle manager
    lifecycleManager := ce.certManager.GetLifecycleManager()
    if lifecycleManager == nil {
        return nil, nil // No lifecycle manager - return nil
    }

    // Get management certificate status
    // Assume certificate name is "management" and provider is "builtin"
    providerName := "builtin"
    certName := "management"

    // Get certificate state
    state, days, expiration, err := lifecycleManager.GetCertificateStatus(ctx, providerName, certName)
    if err != nil {
        return nil, fmt.Errorf("failed to get certificate state: %w", err)
    }

    status := &v1beta1.CertificateStatus{
        State: string(state),
    }

    // Add expiration information
    if expiration != nil {
        status.Expiration = expiration
        if days != nil {
            daysVal := *days
            status.DaysUntilExpiration = &daysVal
        }
    }

    // Add renewal information (if available from database)
    // This requires access to device store or certificate tracking
    // For now, leave as nil
    
    return status, nil
}
```

**Note:** This requires adding methods to LifecycleManager to get certificate status.

**Testing:**
- Test getCertificateStatus collects correct information
- Test getCertificateStatus handles errors

---

### Task 4: Add GetCertificateStatus to LifecycleManager

**File:** `internal/agent/device/certmanager/lifecycle.go` (modify)

**Objective:** Add method to get certificate status for status reporting.

**Implementation Steps:**

1. **Add GetCertificateStatus method:**
```go
// GetCertificateStatus returns certificate status information for status reporting.
func (lm *LifecycleManager) GetCertificateStatus(ctx context.Context, providerName string, certName string) (CertificateState, *int, *time.Time, error) {
    // Get certificate
    cert, err := lm.certManager.certificates.ReadCertificate(providerName, certName)
    if err != nil {
        return CertificateStateNormal, nil, nil, fmt.Errorf("failed to read certificate: %w", err)
    }

    cert.mu.RLock()
    defer cert.mu.RUnlock()

    // Get state
    thresholdDays := 30
    if lm.config != nil {
        thresholdDays = lm.config.Certificate.Renewal.ThresholdDays
        if thresholdDays == 0 {
            thresholdDays = 30
        }
    }

    state := lm.GetCertificateState(cert, thresholdDays)

    // Calculate days until expiration
    var days *int
    var expiration *time.Time
    
    if cert.Info.NotAfter != nil {
        expiration = cert.Info.NotAfter
        now := time.Now()
        daysVal := int(cert.Info.NotAfter.Sub(now).Hours() / 24)
        days = &daysVal
    }

    return state, days, expiration, nil
}
```

**Testing:**
- Test GetCertificateStatus returns correct state
- Test GetCertificateStatus returns correct days
- Test GetCertificateStatus returns correct expiration

---

### Task 5: Add Renewal Information to Status

**File:** `internal/agent/device/status/certificate.go` (modify)

**Objective:** Add renewal count and last renewed timestamp to status.

**Implementation Steps:**

1. **Add renewal information collection:**
```go
// In getCertificateStatus, add renewal information:
// Get renewal information from device store (if available)
// This requires access to device store or certificate tracking
// For now, get from certificate lifecycle state

if cert.Lifecycle != nil {
    // Get last renewed from lifecycle state (if stored)
    // This may need to be added to lifecycle state
    
    // Get renewal count from lifecycle state (if stored)
    // This may need to be added to lifecycle state
}

// Alternative: Get from device database record
// This requires access to device store
// device, err := ce.deviceStore.Get(ctx, orgId, deviceName)
// if err == nil && device.CertificateLastRenewed != nil {
//     status.LastRenewed = device.CertificateLastRenewed
// }
// if err == nil && device.CertificateRenewalCount != nil {
//     status.RenewalCount = device.CertificateRenewalCount
// }
```

**Note:** Full implementation depends on how renewal information is stored. It may come from the database (EDM-323-EPIC-1-STORY-4) or from in-memory state.

**Testing:**
- Test renewal information is collected
- Test renewal information is included in status

---

### Task 6: Register Certificate Exporter

**File:** `internal/agent/agent.go` (modify)

**Objective:** Register certificate status exporter with status manager.

**Implementation Steps:**

1. **Register certificate exporter:**
```go
// In agent initialization, after certificate manager is created:
certExporter := status.NewCertificateExporter(certManager, a.log)
statusManager.RegisterStatusExporter(certExporter)
```

**Testing:**
- Test certificate exporter is registered
- Test certificate status is collected

---

### Task 7: Update Status Collection

**File:** `internal/agent/device/status/certificate.go` (modify)

**Objective:** Ensure certificate status is collected during status sync.

**Implementation Steps:**

1. **Status collection is automatic:**
```go
// The Status method is called automatically by StatusManager during Sync
// No additional changes needed
```

**Testing:**
- Test certificate status is included in device status
- Test certificate status is updated during sync

---

## Unit Tests

### Test File: `internal/agent/device/status/certificate_test.go` (new)

**Test Cases:**

1. **TestCertificateExporter:**
   - Exporter is created correctly
   - Status method collects information

2. **TestGetCertificateStatus:**
   - Collects correct state
   - Collects correct expiration
   - Collects correct days until expiration
   - Handles errors

3. **TestCertificateStatusInDeviceStatus:**
   - Certificate status is included in device status
   - Certificate status is updated correctly

---

## Integration Tests

### Test File: `test/integration/device_status_certificate_test.go` (new)

**Test Cases:**

1. **TestCertificateStatusReporting:**
   - Certificate status is reported in device status
   - Status reflects current certificate state
   - Status updates during renewal/recovery

2. **TestCertificateStatusStates:**
   - Normal state is reported correctly
   - Expiring soon state is reported correctly
   - Expired state is reported correctly
   - Renewing state is reported correctly
   - Recovering state is reported correctly

---

## Code Review Checklist

- [ ] CertificateStatus struct is well-defined
- [ ] CertificateStatus is added to DeviceStatus
- [ ] CertificateExporter is implemented
- [ ] Status collection works correctly
- [ ] Renewal information is included
- [ ] Exporter is registered
- [ ] Unit tests achieve >80% coverage
- [ ] Integration tests cover main flows
- [ ] Code follows existing patterns

---

## Definition of Done

- [ ] CertificateStatus struct added
- [ ] CertificateStatus added to DeviceStatus
- [ ] CertificateExporter created
- [ ] Certificate status collection implemented
- [ ] GetCertificateStatus method added
- [ ] Renewal information added
- [ ] Exporter registered
- [ ] Unit tests written and passing (>80% coverage)
- [ ] Integration tests written and passing
- [ ] Code reviewed and approved
- [ ] API documentation updated
- [ ] Documentation updated

---

## Related Files

- `api/v1beta1/types.go` - API types
- `internal/agent/device/status/certificate.go` - Certificate status exporter
- `internal/agent/device/certmanager/lifecycle.go` - Lifecycle manager
- `internal/agent/agent.go` - Agent initialization

---

## Dependencies

- **EDM-323-EPIC-1**: Foundation (must be completed)
  - Requires certificate lifecycle manager
  
- **EDM-323-EPIC-2**: Proactive Renewal (must be completed)
  - Requires renewal operations
  
- **EDM-323-EPIC-4**: Expired Recovery (must be completed)
  - Requires recovery operations

- **Existing Status Infrastructure**: Uses existing device status infrastructure

---

## Notes

- **Status Structure**: Certificate status is added to DeviceStatus to enable fleet-wide monitoring.

- **State Values**: State values match certificate lifecycle states: normal, expiring_soon, renewing, expired, recovering, renewal_failed.

- **Renewal Information**: Renewal count and last renewed timestamp may come from database (if available) or in-memory state.

- **Status Updates**: Certificate status is updated during status sync, ensuring it reflects current state.

- **Error Handling**: If certificate status collection fails, the error is logged but doesn't block status collection.

- **Optional Fields**: All certificate status fields are optional to handle cases where information is not available.

---

**Document End**

