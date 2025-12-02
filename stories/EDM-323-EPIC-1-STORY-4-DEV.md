# Developer Story: Database Schema for Certificate Tracking

**Story ID:** EDM-323-EPIC-1-STORY-4  
**Epic:** EDM-323-EPIC-1 (Certificate Lifecycle Foundation)  
**Feature:** EDM-323 (Agent Certificate Rotation)  
**Story Points:** 3  
**Priority:** High

## Overview

Add database schema changes to track certificate expiration and renewal history in the devices table. Create indexes for efficient querying and update the store layer to handle certificate tracking fields.

## Implementation Tasks

### Task 1: Add Certificate Tracking Fields to Device Model

**File:** `internal/store/model/device.go` (modify)

**Objective:** Add certificate tracking fields to the Device struct.

**Implementation Steps:**

1. **Add certificate tracking fields to Device struct:**
```go
type Device struct {
    Resource

    // ... existing fields ...

    // Certificate tracking fields
    // CertificateExpiration is when the device's management certificate expires
    CertificateExpiration *time.Time `gorm:"index:idx_devices_cert_expiration" json:"certificate_expiration,omitempty"`
    
    // CertificateLastRenewed is when the certificate was last renewed
    CertificateLastRenewed *time.Time `gorm:"index:idx_devices_cert_last_renewed" json:"certificate_last_renewed,omitempty"`
    
    // CertificateRenewalCount is the number of times the certificate has been renewed
    CertificateRenewalCount int `json:"certificate_renewal_count,omitempty"`
    
    // CertificateFingerprint is the fingerprint of the current certificate
    CertificateFingerprint *string `json:"certificate_fingerprint,omitempty"`

    // ... rest of fields ...
}
```

**Note:** The indexes will be created in the migration, not via GORM tags. The `gorm:"index"` tags are for documentation, but we'll create partial indexes manually.

**Testing:**
- Test struct can be marshaled/unmarshaled
- Test fields are accessible
- Test default values (zero values)

---

### Task 2: Create Migration Methods for Certificate Fields

**File:** `internal/store/device.go` (modify)

**Objective:** Add migration methods to create certificate tracking columns and indexes.

**Implementation Steps:**

1. **Add certificate fields migration method:**
```go
// addCertificateTrackingFields adds certificate tracking columns to the devices table.
func (s *DeviceStore) addCertificateTrackingFields(db *gorm.DB) error {
    if db.Dialector.Name() != "postgres" {
        // For non-PostgreSQL databases, use GORM AutoMigrate
        // GORM will handle the migration automatically
        return nil
    }

    // Check if columns already exist to make migration idempotent
    hasExpiration := db.Migrator().HasColumn(&model.Device{}, "certificate_expiration")
    hasLastRenewed := db.Migrator().HasColumn(&model.Device{}, "certificate_last_renewed")
    hasRenewalCount := db.Migrator().HasColumn(&model.Device{}, "certificate_renewal_count")
    hasFingerprint := db.Migrator().HasColumn(&model.Device{}, "certificate_fingerprint")

    if !hasExpiration {
        if err := db.Exec("ALTER TABLE devices ADD COLUMN certificate_expiration TIMESTAMP").Error; err != nil {
            return fmt.Errorf("failed to add certificate_expiration column: %w", err)
        }
        s.log.Info("Added certificate_expiration column to devices table")
    }

    if !hasLastRenewed {
        if err := db.Exec("ALTER TABLE devices ADD COLUMN certificate_last_renewed TIMESTAMP").Error; err != nil {
            return fmt.Errorf("failed to add certificate_last_renewed column: %w", err)
        }
        s.log.Info("Added certificate_last_renewed column to devices table")
    }

    if !hasRenewalCount {
        if err := db.Exec("ALTER TABLE devices ADD COLUMN certificate_renewal_count INTEGER DEFAULT 0").Error; err != nil {
            return fmt.Errorf("failed to add certificate_renewal_count column: %w", err)
        }
        s.log.Info("Added certificate_renewal_count column to devices table")
    }

    if !hasFingerprint {
        if err := db.Exec("ALTER TABLE devices ADD COLUMN certificate_fingerprint TEXT").Error; err != nil {
            return fmt.Errorf("failed to add certificate_fingerprint column: %w", err)
        }
        s.log.Info("Added certificate_fingerprint column to devices table")
    }

    return nil
}
```

2. **Add index creation method:**
```go
// createCertificateExpirationIndex creates an index on certificate_expiration for efficient queries.
func (s *DeviceStore) createCertificateExpirationIndex(db *gorm.DB) error {
    if db.Dialector.Name() != "postgres" {
        // For non-PostgreSQL, use GORM index creation
        if !db.Migrator().HasIndex(&model.Device{}, "idx_devices_cert_expiration") {
            return db.Migrator().CreateIndex(&model.Device{}, "CertificateExpiration")
        }
        return nil
    }

    // Create partial index for PostgreSQL (only indexes non-null values)
    if !db.Migrator().HasIndex(&model.Device{}, "idx_devices_cert_expiration") {
        if err := db.Exec(`
            CREATE INDEX idx_devices_cert_expiration 
            ON devices(certificate_expiration) 
            WHERE certificate_expiration IS NOT NULL
        `).Error; err != nil {
            return fmt.Errorf("failed to create certificate expiration index: %w", err)
        }
        s.log.Info("Created idx_devices_cert_expiration index")
    }
    return nil
}

// createCertificateLastRenewedIndex creates an index on certificate_last_renewed for efficient queries.
func (s *DeviceStore) createCertificateLastRenewedIndex(db *gorm.DB) error {
    if db.Dialector.Name() != "postgres" {
        // For non-PostgreSQL, use GORM index creation
        if !db.Migrator().HasIndex(&model.Device{}, "idx_devices_cert_last_renewed") {
            return db.Migrator().CreateIndex(&model.Device{}, "CertificateLastRenewed")
        }
        return nil
    }

    // Create partial index for PostgreSQL (only indexes non-null values)
    if !db.Migrator().HasIndex(&model.Device{}, "idx_devices_cert_last_renewed") {
        if err := db.Exec(`
            CREATE INDEX idx_devices_cert_last_renewed 
            ON devices(certificate_last_renewed) 
            WHERE certificate_last_renewed IS NOT NULL
        `).Error; err != nil {
            return fmt.Errorf("failed to create certificate last renewed index: %w", err)
        }
        s.log.Info("Created idx_devices_cert_last_renewed index")
    }
    return nil
}
```

3. **Update InitialMigration to include certificate tracking:**
```go
func (s *DeviceStore) InitialMigration(ctx context.Context) error {
    db := s.getDB(ctx)

    if err := db.AutoMigrate(&model.Device{}, &model.DeviceLabel{}, &model.DeviceTimestamp{}); err != nil {
        return err
    }

    // ... existing index creation methods ...

    // Add certificate tracking fields
    if err := s.addCertificateTrackingFields(db); err != nil {
        return err
    }

    // Create certificate tracking indexes
    if err := s.createCertificateExpirationIndex(db); err != nil {
        return err
    }

    if err := s.createCertificateLastRenewedIndex(db); err != nil {
        return err
    }

    // ... rest of existing migration code ...

    return nil
}
```

**Testing:**
- Test migration runs successfully
- Test migration is idempotent (can run multiple times)
- Test columns are created with correct types
- Test indexes are created
- Test migration handles existing columns gracefully

---

### Task 3: Add Helper Methods to Update Certificate Fields

**File:** `internal/store/device.go` (modify)

**Objective:** Add methods to update certificate tracking fields.

**Implementation Steps:**

1. **Add UpdateCertificateExpiration method:**
```go
// UpdateCertificateExpiration updates the certificate expiration date for a device.
func (s *DeviceStore) UpdateCertificateExpiration(ctx context.Context, orgId uuid.UUID, deviceName string, expiration *time.Time) error {
    db := s.getDB(ctx)
    
    result := db.Model(&model.Device{}).
        Where("org_id = ? AND name = ?", orgId, deviceName).
        Update("certificate_expiration", expiration)
    
    if result.Error != nil {
        return ErrorFromGormError(result.Error)
    }
    
    if result.RowsAffected == 0 {
        return flterrors.ErrResourceNotFound
    }
    
    return nil
}
```

2. **Add UpdateCertificateRenewalInfo method:**
```go
// UpdateCertificateRenewalInfo updates certificate renewal tracking information.
func (s *DeviceStore) UpdateCertificateRenewalInfo(ctx context.Context, orgId uuid.UUID, deviceName string, lastRenewed *time.Time, renewalCount int, fingerprint *string) error {
    db := s.getDB(ctx)
    
    updates := map[string]interface{}{
        "certificate_last_renewed": lastRenewed,
        "certificate_renewal_count": renewalCount,
    }
    
    if fingerprint != nil {
        updates["certificate_fingerprint"] = fingerprint
    }
    
    result := db.Model(&model.Device{}).
        Where("org_id = ? AND name = ?", orgId, deviceName).
        Updates(updates)
    
    if result.Error != nil {
        return ErrorFromGormError(result.Error)
    }
    
    if result.RowsAffected == 0 {
        return flterrors.ErrResourceNotFound
    }
    
    return nil
}
```

3. **Add GetCertificateExpiration method:**
```go
// GetCertificateExpiration retrieves the certificate expiration date for a device.
func (s *DeviceStore) GetCertificateExpiration(ctx context.Context, orgId uuid.UUID, deviceName string) (*time.Time, error) {
    db := s.getDB(ctx)
    
    var device model.Device
    result := db.Select("certificate_expiration").
        Where("org_id = ? AND name = ?", orgId, deviceName).
        First(&device)
    
    if result.Error != nil {
        if errors.Is(result.Error, gorm.ErrRecordNotFound) {
            return nil, flterrors.ErrResourceNotFound
        }
        return nil, ErrorFromGormError(result.Error)
    }
    
    return device.CertificateExpiration, nil
}
```

**Testing:**
- Test UpdateCertificateExpiration updates correctly
- Test UpdateCertificateRenewalInfo updates all fields
- Test GetCertificateExpiration retrieves correctly
- Test methods handle non-existent devices
- Test methods handle nil values correctly

---

### Task 4: Update Device Interface

**File:** `internal/store/device.go` (modify)

**Objective:** Add certificate tracking methods to the Device interface.

**Implementation Steps:**

1. **Add methods to Device interface:**
```go
type Device interface {
    InitialMigration(ctx context.Context) error

    // ... existing methods ...

    // Certificate tracking methods
    UpdateCertificateExpiration(ctx context.Context, orgId uuid.UUID, deviceName string, expiration *time.Time) error
    UpdateCertificateRenewalInfo(ctx context.Context, orgId uuid.UUID, deviceName string, lastRenewed *time.Time, renewalCount int, fingerprint *string) error
    GetCertificateExpiration(ctx context.Context, orgId uuid.UUID, deviceName string) (*time.Time, error)
}
```

**Testing:**
- Test interface is properly implemented
- Test interface methods are accessible

---

### Task 5: Add Query Methods for Certificate Expiration

**File:** `internal/store/device.go` (modify)

**Objective:** Add methods to query devices by certificate expiration.

**Implementation Steps:**

1. **Add ListDevicesExpiringSoon method:**
```go
// ListDevicesExpiringSoon lists devices with certificates expiring within the specified threshold.
func (s *DeviceStore) ListDevicesExpiringSoon(ctx context.Context, orgId uuid.UUID, thresholdDate time.Time) ([]*api.Device, error) {
    db := s.getDB(ctx)
    
    var devices []model.Device
    result := db.Where("org_id = ? AND certificate_expiration IS NOT NULL AND certificate_expiration <= ?", 
        orgId, thresholdDate).
        Find(&devices)
    
    if result.Error != nil {
        return nil, ErrorFromGormError(result.Error)
    }
    
    apiDevices := make([]*api.Device, len(devices))
    for i, device := range devices {
        apiDevice, err := device.ToApiResource()
        if err != nil {
            return nil, fmt.Errorf("failed to convert device %s to API resource: %w", device.Name, err)
        }
        apiDevices[i] = apiDevice
    }
    
    return apiDevices, nil
}
```

2. **Add ListDevicesWithExpiredCertificates method:**
```go
// ListDevicesWithExpiredCertificates lists devices with expired certificates.
func (s *DeviceStore) ListDevicesWithExpiredCertificates(ctx context.Context, orgId uuid.UUID) ([]*api.Device, error) {
    db := s.getDB(ctx)
    now := time.Now().UTC()
    
    var devices []model.Device
    result := db.Where("org_id = ? AND certificate_expiration IS NOT NULL AND certificate_expiration < ?", 
        orgId, now).
        Find(&devices)
    
    if result.Error != nil {
        return nil, ErrorFromGormError(result.Error)
    }
    
    apiDevices := make([]*api.Device, len(devices))
    for i, device := range devices {
        apiDevice, err := device.ToApiResource()
        if err != nil {
            return nil, fmt.Errorf("failed to convert device %s to API resource: %w", device.Name, err)
        }
        apiDevices[i] = apiDevice
    }
    
    return apiDevices, nil
}
```

**Testing:**
- Test ListDevicesExpiringSoon returns correct devices
- Test ListDevicesWithExpiredCertificates returns correct devices
- Test queries use indexes efficiently
- Test queries handle empty results
- Test queries handle nil expiration dates

---

### Task 6: Update Device Creation/Update to Handle Certificate Fields

**File:** `internal/store/device.go` (modify)

**Objective:** Ensure certificate fields are preserved during device operations.

**Implementation Steps:**

1. **Verify AutoMigrate handles new fields:**
GORM's `AutoMigrate` should automatically handle the new fields when the model is updated. However, we need to ensure:
   - Fields are not lost during updates
   - Fields can be set during device creation
   - Fields are preserved during device updates

2. **Check Update method preserves certificate fields:**
The existing `Update` method should preserve certificate fields unless explicitly unset. Verify this works correctly.

3. **Add certificate field handling in Create/Update if needed:**
If certificate fields need special handling, add logic to preserve them:

```go
// In Update method, ensure certificate fields are preserved if not being updated
// The generic store should handle this, but verify
```

**Testing:**
- Test device creation preserves certificate fields
- Test device update preserves certificate fields
- Test certificate fields can be updated independently
- Test certificate fields are not lost during other updates

---

### Task 7: Add Backfill Logic (Optional)

**File:** `internal/store/device.go` (modify)

**Objective:** Optionally backfill certificate expiration from existing certificates.

**Implementation Steps:**

1. **Add backfill method (if needed):**
```go
// backfillCertificateExpiration attempts to extract certificate expiration from device status.
// This is optional and may not be possible if certificates are not stored in status.
func (s *DeviceStore) backfillCertificateExpiration(ctx context.Context, db *gorm.DB) error {
    // This would require parsing certificates from device status
    // For now, we'll leave this as a placeholder
    // Certificate expiration will be populated when certificates are renewed
    s.log.Info("Certificate expiration backfill skipped - will be populated on next renewal")
    return nil
}
```

**Note:** This is optional since we can't reliably extract expiration dates from existing data. Certificate expiration will be populated when certificates are issued or renewed.

**Testing:**
- Test backfill method (if implemented)
- Test backfill handles missing data gracefully

---

## Unit Tests

### Test File: `internal/store/device_certificate_test.go` (new)

**Test Cases:**

1. **TestDeviceModel_CertificateFields:**
   - Certificate fields are accessible
   - Fields can be set and retrieved
   - Nil values are handled correctly

2. **TestDeviceStore_AddCertificateTrackingFields:**
   - Migration adds columns correctly
   - Migration is idempotent
   - Columns have correct types
   - Default values are set correctly

3. **TestDeviceStore_CertificateIndexes:**
   - Indexes are created
   - Indexes are partial (only non-null values)
   - Indexes improve query performance

4. **TestDeviceStore_UpdateCertificateExpiration:**
   - Updates expiration correctly
   - Handles nil values
   - Returns error for non-existent device
   - Updates are persisted

5. **TestDeviceStore_UpdateCertificateRenewalInfo:**
   - Updates all renewal fields correctly
   - Handles nil values
   - Increments renewal count
   - Updates fingerprint

6. **TestDeviceStore_GetCertificateExpiration:**
   - Retrieves expiration correctly
   - Returns nil for devices without expiration
   - Returns error for non-existent device

7. **TestDeviceStore_ListDevicesExpiringSoon:**
   - Returns devices expiring before threshold
   - Excludes devices with nil expiration
   - Excludes devices expiring after threshold
   - Uses index efficiently

8. **TestDeviceStore_ListDevicesWithExpiredCertificates:**
   - Returns only expired certificates
   - Excludes devices with nil expiration
   - Excludes devices with valid certificates
   - Uses index efficiently

---

## Integration Tests

### Test File: `test/integration/store_certificate_tracking_test.go` (new)

**Test Cases:**

1. **TestCertificateTrackingMigration:**
   - Migration runs successfully
   - Columns are created
   - Indexes are created
   - Migration is idempotent

2. **TestCertificateTrackingOperations:**
   - Certificate expiration can be set
   - Certificate renewal info can be updated
   - Certificate expiration can be retrieved
   - Fields persist across operations

3. **TestCertificateExpirationQueries:**
   - Expiring devices query works
   - Expired devices query works
   - Queries use indexes
   - Queries handle empty results

4. **TestCertificateTrackingPerformance:**
   - Indexes improve query performance
   - Queries scale with device count
   - Partial indexes reduce index size

---

## Migration Testing

### Manual Testing Steps

1. **Test Migration on Fresh Database:**
```bash
# Start fresh database
# Run migration
flightctl-db-migrate

# Verify columns exist
psql -c "\d devices" | grep certificate

# Verify indexes exist
psql -c "\d devices" | grep idx_devices_cert
```

2. **Test Migration on Existing Database:**
```bash
# Use existing database with devices
# Run migration
flightctl-db-migrate

# Verify columns added
# Verify existing data preserved
# Verify indexes created
```

3. **Test Migration Idempotency:**
```bash
# Run migration twice
flightctl-db-migrate
flightctl-db-migrate

# Verify no errors on second run
# Verify no duplicate columns/indexes
```

4. **Test Dry Run:**
```bash
# Test migration validation
flightctl-db-migrate --dry-run

# Verify no changes applied
# Verify validation passes
```

---

## Code Review Checklist

- [ ] Migration is idempotent (can run multiple times)
- [ ] Migration handles existing columns gracefully
- [ ] Indexes are created correctly (partial indexes for PostgreSQL)
- [ ] Store methods handle nil values correctly
- [ ] Store methods return appropriate errors
- [ ] Queries use indexes efficiently
- [ ] Unit tests achieve >80% coverage
- [ ] Integration tests cover migration and operations
- [ ] Migration is backward compatible
- [ ] Documentation is updated

---

## Definition of Done

- [ ] Certificate tracking fields added to Device model
- [ ] Migration methods implemented
- [ ] Indexes created (partial indexes for PostgreSQL)
- [ ] Store methods for certificate tracking implemented
- [ ] Device interface updated
- [ ] Query methods for expiration implemented
- [ ] Unit tests written and passing (>80% coverage)
- [ ] Integration tests written and passing
- [ ] Migration tested on fresh database
- [ ] Migration tested on existing database
- [ ] Migration tested for idempotency
- [ ] Code reviewed and approved
- [ ] Migration documented

---

## Related Files

- `internal/store/model/device.go` - Device model
- `internal/store/device.go` - Device store implementation
- `internal/store/store.go` - Store initialization
- `cmd/flightctl-db-migrate/main.go` - Migration command

---

## Dependencies

- None (can be done in parallel)
- Uses existing store infrastructure
- Uses existing migration patterns

---

## Notes

- **Partial Indexes**: PostgreSQL partial indexes (WHERE clause) are more efficient for nullable columns. They only index non-null values, reducing index size and improving performance.

- **Migration Idempotency**: All migrations must be idempotent. Check for column/index existence before creating to allow safe re-runs.

- **Backward Compatibility**: Existing devices will have NULL values for certificate fields. This is expected and acceptable.

- **Certificate Fingerprint**: The fingerprint can be calculated from the certificate (SHA256 hash). This will be populated when certificates are issued/renewed.

- **Performance**: Partial indexes on nullable columns are more efficient than full indexes. They reduce index size and improve query performance.

- **GORM AutoMigrate**: GORM's AutoMigrate will handle adding columns, but we use explicit SQL for better control and to create partial indexes.

- **Database Dialects**: The migration handles PostgreSQL specifically. For other databases, GORM's AutoMigrate will handle column creation, but partial indexes may not be supported.

---

## SQL Migration Script (Reference)

For reference, here's the SQL that will be executed:

```sql
-- Add certificate tracking columns
ALTER TABLE devices 
ADD COLUMN IF NOT EXISTS certificate_expiration TIMESTAMP,
ADD COLUMN IF NOT EXISTS certificate_last_renewed TIMESTAMP,
ADD COLUMN IF NOT EXISTS certificate_renewal_count INTEGER DEFAULT 0,
ADD COLUMN IF NOT EXISTS certificate_fingerprint TEXT;

-- Create partial index for expiration (only indexes non-null values)
CREATE INDEX IF NOT EXISTS idx_devices_cert_expiration 
ON devices(certificate_expiration) 
WHERE certificate_expiration IS NOT NULL;

-- Create partial index for last renewed (only indexes non-null values)
CREATE INDEX IF NOT EXISTS idx_devices_cert_last_renewed 
ON devices(certificate_last_renewed) 
WHERE certificate_last_renewed IS NOT NULL;
```

**Note:** The actual implementation uses GORM's Migrator to check for existence and create conditionally, rather than using `IF NOT EXISTS` (which may not be available in all PostgreSQL versions).

---

**Document End**

