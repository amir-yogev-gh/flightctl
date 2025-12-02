# Developer Story: Store Layer Extensions for Certificate Tracking

**Story ID:** EDM-323-EPIC-7-STORY-2  
**Epic:** EDM-323-EPIC-7 (Database and API Enhancements)  
**Feature:** EDM-323 (Agent Certificate Rotation)  
**Story Points:** 3  
**Priority:** Medium

## Overview

Extend store layer with methods to update certificate tracking fields (expiration, last renewed, renewal count, fingerprint) to maintain certificate metadata in the database.

## Implementation Tasks

### Task 1: Add UpdateCertificateExpiration Method

**File:** `internal/store/device.go` (modify)

**Objective:** Add method to update certificate expiration.

**Implementation:**
```go
// UpdateCertificateExpiration updates the certificate expiration timestamp for a device.
func (s *DeviceStore) UpdateCertificateExpiration(ctx context.Context, orgID uuid.UUID, deviceName string, expiration time.Time) error {
    db := s.getDB(ctx)
    
    return db.WithContext(ctx).
        Model(&model.Device{}).
        Where("org_id = ? AND name = ?", orgID, deviceName).
        Update("certificate_expiration", expiration).Error
}
```

**Testing:**
- Test UpdateCertificateExpiration updates expiration
- Test UpdateCertificateExpiration handles non-existent device

---

### Task 2: Add UpdateCertificateRenewalInfo Method

**File:** `internal/store/device.go` (modify)

**Objective:** Add method to update renewal information.

**Implementation:**
```go
// UpdateCertificateRenewalInfo updates certificate renewal information for a device.
func (s *DeviceStore) UpdateCertificateRenewalInfo(ctx context.Context, orgID uuid.UUID, deviceName string, lastRenewed time.Time, renewalCount int) error {
    db := s.getDB(ctx)
    
    return db.WithContext(ctx).
        Model(&model.Device{}).
        Where("org_id = ? AND name = ?", orgID, deviceName).
        Updates(map[string]interface{}{
            "certificate_last_renewed": lastRenewed,
            "certificate_renewal_count": renewalCount,
        }).Error
}
```

**Testing:**
- Test UpdateCertificateRenewalInfo updates renewal info
- Test UpdateCertificateRenewalInfo handles non-existent device

---

### Task 3: Add UpdateCertificateFingerprint Method

**File:** `internal/store/device.go` (modify)

**Objective:** Add method to update certificate fingerprint.

**Implementation:**
```go
// UpdateCertificateFingerprint updates the certificate fingerprint for a device.
func (s *DeviceStore) UpdateCertificateFingerprint(ctx context.Context, orgID uuid.UUID, deviceName string, fingerprint string) error {
    db := s.getDB(ctx)
    
    return db.WithContext(ctx).
        Model(&model.Device{}).
        Where("org_id = ? AND name = ?", orgID, deviceName).
        Update("certificate_fingerprint", fingerprint).Error
}
```

**Testing:**
- Test UpdateCertificateFingerprint updates fingerprint
- Test UpdateCertificateFingerprint handles non-existent device

---

### Task 4: Integrate with Certificate Issuance

**File:** `internal/service/certificatesigningrequest.go` (modify)

**Objective:** Update certificate tracking when certificates are issued.

**Implementation:**
```go
// In signApprovedCertificateSigningRequest, after certificate is signed:
// Update certificate tracking fields
deviceName, _ := h.extractDeviceNameFromCSR(csr)
expiration := cert.NotAfter
fingerprint := calculateCertificateFingerprint(cert)

if err := h.store.Device().UpdateCertificateExpiration(ctx, orgId, deviceName, expiration); err != nil {
    h.log.Warnf("Failed to update certificate expiration: %v", err)
}

if err := h.store.Device().UpdateCertificateFingerprint(ctx, orgId, deviceName, fingerprint); err != nil {
    h.log.Warnf("Failed to update certificate fingerprint: %v", err)
}

// If this is a renewal, update renewal info
if h.isRenewalRequest(csr) {
    device, _ := h.store.Device().Get(ctx, orgId, deviceName)
    renewalCount := 0
    if device.CertificateRenewalCount != nil {
        renewalCount = *device.CertificateRenewalCount + 1
    }
    
    if err := h.store.Device().UpdateCertificateRenewalInfo(ctx, orgId, deviceName, time.Now(), renewalCount); err != nil {
        h.log.Warnf("Failed to update certificate renewal info: %v", err)
    }
}
```

**Testing:**
- Test certificate tracking is updated on issuance
- Test renewal info is updated on renewal

---

### Task 5: Add Transaction Support

**File:** `internal/store/device.go` (modify)

**Objective:** Ensure certificate tracking updates use transactions.

**Implementation:**
```go
// UpdateCertificateTracking updates all certificate tracking fields in a transaction.
func (s *DeviceStore) UpdateCertificateTracking(ctx context.Context, orgID uuid.UUID, deviceName string, expiration time.Time, fingerprint string, lastRenewed *time.Time, renewalCount *int) error {
    db := s.getDB(ctx)
    
    return db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
        updates := map[string]interface{}{
            "certificate_expiration": expiration,
            "certificate_fingerprint": fingerprint,
        }
        
        if lastRenewed != nil {
            updates["certificate_last_renewed"] = *lastRenewed
        }
        if renewalCount != nil {
            updates["certificate_renewal_count"] = *renewalCount
        }
        
        return tx.Model(&model.Device{}).
            Where("org_id = ? AND name = ?", orgID, deviceName).
            Updates(updates).Error
    })
}
```

**Testing:**
- Test UpdateCertificateTracking uses transaction
- Test transaction rollback on error

---

## Definition of Done

- [ ] Store layer extensions implemented
- [ ] Certificate tracking updates integrated
- [ ] Transaction support added
- [ ] Unit tests written and passing
- [ ] Code reviewed and approved
- [ ] Integration tests passing
- [ ] Documentation updated

---

## Related Files

- `internal/store/device.go` - Device store
- `internal/service/certificatesigningrequest.go` - Certificate issuance
- `internal/store/model/device.go` - Device model

---

## Dependencies

- **EDM-323-EPIC-1-STORY-4**: Database Schema (must be completed)
- **EDM-323-EPIC-7-STORY-1**: Renewal Events Table (must be completed)
- **Existing Store Infrastructure**: Uses existing store patterns

---

## Notes

- **Transaction Safety**: Certificate tracking updates use transactions to ensure data consistency
- **Efficient Updates**: Updates use appropriate indexes for performance
- **Error Handling**: Errors are logged but don't block certificate operations
- **Renewal Count**: Renewal count increments on each renewal
- **Fingerprint**: Fingerprint is calculated from certificate for tracking

---

**Document End**

