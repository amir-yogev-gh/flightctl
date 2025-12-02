# Developer Story: Atomic Certificate Swap Operation

**Story ID:** EDM-323-EPIC-3-STORY-3  
**Epic:** EDM-323-EPIC-3 (Atomic Certificate Swap)  
**Feature:** EDM-323 (Agent Certificate Rotation)  
**Story Points:** 5  
**Priority:** High

## Overview

Implement atomic certificate swap using POSIX atomic rename operations. This ensures devices never lose valid certificates even during power loss, as the swap operation is atomic at the filesystem level.

## Implementation Tasks

### Task 1: Implement Atomic Swap Method

**File:** `internal/agent/device/certmanager/swap.go` (modify)

**Objective:** Implement atomic swap operation using POSIX rename.

**Implementation Steps:**

1. **Add AtomicSwap method to CertificateValidator:**
```go
// AtomicSwap performs an atomic swap of pending certificate to active location.
// It uses POSIX atomic rename operations to ensure the swap is atomic.
// Returns error if swap fails, in which case old certificate is preserved.
func (cv *CertificateValidator) AtomicSwap(ctx context.Context, storage provider.StorageProvider) error {
    cv.log.Debugf("Performing atomic certificate swap for device %q", cv.deviceName)

    // Get pending and active paths from storage
    // We need to access the underlying FileSystemStorage to get paths
    fsStorage, ok := storage.(*storage.FileSystemStorage)
    if !ok {
        return fmt.Errorf("atomic swap only supported for filesystem storage")
    }

    pendingCertPath := fsStorage.GetPendingCertPath()
    activeCertPath := fsStorage.CertPath
    pendingKeyPath := fsStorage.GetPendingKeyPath()
    activeKeyPath := fsStorage.KeyPath

    // Verify pending files exist
    hasPending, err := storage.HasPendingCertificate(ctx)
    if err != nil {
        return fmt.Errorf("failed to check pending certificate: %w", err)
    }
    if !hasPending {
        return fmt.Errorf("no pending certificate to swap")
    }

    // Use POSIX atomic rename for certificate
    // os.Rename is atomic on POSIX systems
    if err := os.Rename(pendingCertPath, activeCertPath); err != nil {
        return fmt.Errorf("failed to atomically swap certificate: %w", err)
    }

    // Use POSIX atomic rename for key
    // If this fails, we need to rollback the certificate swap
    if err := os.Rename(pendingKeyPath, activeKeyPath); err != nil {
        // Rollback: try to restore certificate from backup or pending
        // Note: This is a best-effort rollback
        cv.log.Errorf("Failed to swap key, attempting rollback: %v", err)
        if rollbackErr := cv.rollbackCertificateSwap(ctx, storage, pendingCertPath, activeCertPath); rollbackErr != nil {
            cv.log.Errorf("Rollback failed: %v", rollbackErr)
        }
        return fmt.Errorf("failed to atomically swap key: %w", err)
    }

    cv.log.Infof("Atomic certificate swap completed successfully for device %q", cv.deviceName)
    return nil
}
```

**Note:** This requires adding methods to FileSystemStorage to expose paths. We'll add those in a separate task.

**Testing:**
- Test AtomicSwap swaps certificate and key
- Test AtomicSwap is atomic
- Test AtomicSwap rollback on key failure

---

### Task 2: Add Path Access Methods to FileSystemStorage

**File:** `internal/agent/device/certmanager/provider/storage/fs.go` (modify)

**Objective:** Add methods to access file paths for atomic operations.

**Implementation Steps:**

1. **Add path access methods:**
```go
// GetPendingCertPath returns the pending certificate path.
// This is exposed for atomic swap operations.
func (fs *FileSystemStorage) GetPendingCertPath() string {
    return fs.getPendingCertPath()
}

// GetPendingKeyPath returns the pending key path.
// This is exposed for atomic swap operations.
func (fs *FileSystemStorage) GetPendingKeyPath() string {
    return fs.getPendingKeyPath()
}

// GetActiveCertPath returns the active certificate path.
func (fs *FileSystemStorage) GetActiveCertPath() string {
    return fs.CertPath
}

// GetActiveKeyPath returns the active key path.
func (fs *FileSystemStorage) GetActiveKeyPath() string {
    return fs.KeyPath
}
```

**Testing:**
- Test path methods return correct paths
- Test paths are accessible

---

### Task 3: Implement Backup Before Swap

**File:** `internal/agent/device/certmanager/swap.go` (modify)

**Objective:** Create backup of current certificate before swap for rollback.

**Implementation Steps:**

1. **Add backup methods:**
```go
// backupActiveCertificate creates a backup of the active certificate and key.
func (cv *CertificateValidator) backupActiveCertificate(ctx context.Context, storage provider.StorageProvider) (string, string, error) {
    fsStorage, ok := storage.(*storage.FileSystemStorage)
    if !ok {
        return "", "", fmt.Errorf("backup only supported for filesystem storage")
    }

    activeCertPath := fsStorage.GetActiveCertPath()
    activeKeyPath := fsStorage.GetActiveKeyPath()
    backupCertPath := activeCertPath + ".backup"
    backupKeyPath := activeKeyPath + ".backup"

    // Check if active certificate exists
    exists, err := fsStorage.deviceReadWriter.PathExists(activeCertPath)
    if err != nil {
        return "", "", fmt.Errorf("failed to check active certificate: %w", err)
    }
    if !exists {
        // No active certificate to backup (first certificate)
        return "", "", nil
    }

    // Copy active certificate to backup
    if err := cv.copyFile(activeCertPath, backupCertPath, fsStorage.deviceReadWriter); err != nil {
        return "", "", fmt.Errorf("failed to backup certificate: %w", err)
    }

    // Copy active key to backup
    if err := cv.copyFile(activeKeyPath, backupKeyPath, fsStorage.deviceReadWriter); err != nil {
        // Clean up certificate backup on failure
        _ = fsStorage.deviceReadWriter.RemoveFile(backupCertPath)
        return "", "", fmt.Errorf("failed to backup key: %w", err)
    }

    cv.log.Debugf("Created backup of active certificate at %s and %s", backupCertPath, backupKeyPath)
    return backupCertPath, backupKeyPath, nil
}

// copyFile copies a file from source to destination.
func (cv *CertificateValidator) copyFile(src, dst string, rw fileio.ReadWriter) error {
    data, err := rw.ReadFile(src)
    if err != nil {
        return fmt.Errorf("failed to read source file: %w", err)
    }

    // Get source file permissions
    // For simplicity, use default permissions (will be preserved by WriteFile)
    if err := rw.WriteFile(dst, data, 0o600); err != nil {
        return fmt.Errorf("failed to write destination file: %w", err)
    }

    return nil
}
```

**Testing:**
- Test backupActiveCertificate creates backup
- Test backupActiveCertificate handles missing active certificate
- Test copyFile copies files correctly

---

### Task 4: Implement Rollback Logic

**File:** `internal/agent/device/certmanager/swap.go` (modify)

**Objective:** Implement rollback logic to restore backup if swap fails.

**Implementation Steps:**

1. **Add rollbackCertificateSwap method:**
```go
// rollbackCertificateSwap attempts to rollback a failed certificate swap.
func (cv *CertificateValidator) rollbackCertificateSwap(ctx context.Context, storage provider.StorageProvider, pendingCertPath, activeCertPath string) error {
    fsStorage, ok := storage.(*storage.FileSystemStorage)
    if !ok {
        return fmt.Errorf("rollback only supported for filesystem storage")
    }

    backupCertPath := activeCertPath + ".backup"
    backupKeyPath := fsStorage.GetActiveKeyPath() + ".backup"

    // Check if backup exists
    backupExists, err := fsStorage.deviceReadWriter.PathExists(backupCertPath)
    if err != nil {
        return fmt.Errorf("failed to check backup: %w", err)
    }

    if backupExists {
        // Restore from backup
        cv.log.Warnf("Restoring certificate from backup due to swap failure")
        if err := os.Rename(backupCertPath, activeCertPath); err != nil {
            return fmt.Errorf("failed to restore certificate from backup: %w", err)
        }
        if err := os.Rename(backupKeyPath, fsStorage.GetActiveKeyPath()); err != nil {
            return fmt.Errorf("failed to restore key from backup: %w", err)
        }
        cv.log.Infof("Certificate restored from backup")
    } else {
        // No backup available, try to restore from pending if it still exists
        pendingExists, err := fsStorage.deviceReadWriter.PathExists(pendingCertPath)
        if err == nil && pendingExists {
            cv.log.Warnf("No backup available, attempting to restore from pending")
            // This is a best-effort recovery
            // The pending certificate may have been partially swapped
        }
    }

    return nil
}
```

**Testing:**
- Test rollbackCertificateSwap restores from backup
- Test rollbackCertificateSwap handles missing backup
- Test rollbackCertificateSwap handles errors

---

### Task 5: Implement Complete Atomic Swap with Backup

**File:** `internal/agent/device/certmanager/swap.go` (modify)

**Objective:** Combine backup and swap into a complete atomic operation.

**Implementation Steps:**

1. **Enhance AtomicSwap with backup:**
```go
// AtomicSwap performs an atomic swap of pending certificate to active location.
func (cv *CertificateValidator) AtomicSwap(ctx context.Context, storage provider.StorageProvider) error {
    cv.log.Debugf("Performing atomic certificate swap for device %q", cv.deviceName)

    fsStorage, ok := storage.(*storage.FileSystemStorage)
    if !ok {
        return fmt.Errorf("atomic swap only supported for filesystem storage")
    }

    pendingCertPath := fsStorage.GetPendingCertPath()
    activeCertPath := fsStorage.GetActiveCertPath()
    pendingKeyPath := fsStorage.GetPendingKeyPath()
    activeKeyPath := fsStorage.GetActiveKeyPath()

    // Verify pending files exist
    hasPending, err := storage.HasPendingCertificate(ctx)
    if err != nil {
        return fmt.Errorf("failed to check pending certificate: %w", err)
    }
    if !hasPending {
        return fmt.Errorf("no pending certificate to swap")
    }

    // Step 1: Create backup of active certificate (if exists)
    backupCertPath, backupKeyPath, err := cv.backupActiveCertificate(ctx, storage)
    if err != nil {
        return fmt.Errorf("failed to backup active certificate: %w", err)
    }

    // Step 2: Perform atomic swap using POSIX rename
    // os.Rename is atomic on POSIX systems
    certSwapErr := os.Rename(pendingCertPath, activeCertPath)
    keySwapErr := os.Rename(pendingKeyPath, activeKeyPath)

    // Step 3: Handle swap results
    if certSwapErr != nil {
        return fmt.Errorf("failed to atomically swap certificate: %w", certSwapErr)
    }

    if keySwapErr != nil {
        // Certificate swap succeeded but key swap failed
        // Rollback certificate swap
        cv.log.Errorf("Key swap failed, rolling back certificate swap: %v", keySwapErr)
        if rollbackErr := cv.rollbackCertificateSwap(ctx, storage, pendingCertPath, activeCertPath); rollbackErr != nil {
            cv.log.Errorf("Rollback failed: %v", rollbackErr)
        }
        return fmt.Errorf("failed to atomically swap key: %w", keySwapErr)
    }

    // Step 4: Clean up backup after successful swap
    if backupCertPath != "" {
        _ = fsStorage.deviceReadWriter.RemoveFile(backupCertPath)
        _ = fsStorage.deviceReadWriter.RemoveFile(backupKeyPath)
        cv.log.Debugf("Cleaned up backup files after successful swap")
    }

    cv.log.Infof("Atomic certificate swap completed successfully for device %q", cv.deviceName)
    return nil
}
```

**Testing:**
- Test complete atomic swap with backup
- Test swap failure triggers rollback
- Test backup cleanup after success

---

### Task 6: Add Atomic Swap to Storage Provider Interface

**File:** `internal/agent/device/certmanager/provider/provider.go` (modify)

**Objective:** Add atomic swap method to storage provider interface.

**Implementation Steps:**

1. **Add AtomicSwap to StorageProvider interface:**
```go
// StorageProvider defines the interface for certificate storage operations.
type StorageProvider interface {
    // ... existing methods ...
    
    // AtomicSwap performs an atomic swap of pending certificate to active location.
    // It uses POSIX atomic rename operations to ensure the swap is atomic.
    AtomicSwap(ctx context.Context) error
}
```

2. **Implement AtomicSwap in FileSystemStorage:**
```go
// AtomicSwap performs an atomic swap of pending certificate to active location.
func (fs *FileSystemStorage) AtomicSwap(ctx context.Context) error {
    pendingCertPath := fs.getPendingCertPath()
    activeCertPath := fs.CertPath
    pendingKeyPath := fs.getPendingKeyPath()
    activeKeyPath := fs.KeyPath

    // Verify pending files exist
    hasPending, err := fs.HasPendingCertificate(ctx)
    if err != nil {
        return fmt.Errorf("failed to check pending certificate: %w", err)
    }
    if !hasPending {
        return fmt.Errorf("no pending certificate to swap")
    }

    // Use POSIX atomic rename
    if err := os.Rename(pendingCertPath, activeCertPath); err != nil {
        return fmt.Errorf("failed to atomically swap certificate: %w", err)
    }

    if err := os.Rename(pendingKeyPath, activeKeyPath); err != nil {
        // Rollback certificate swap
        _ = os.Rename(activeCertPath, pendingCertPath)
        return fmt.Errorf("failed to atomically swap key: %w", err)
    }

    fs.log.Debugf("Atomic swap completed: %s -> %s, %s -> %s", 
        pendingCertPath, activeCertPath, pendingKeyPath, activeKeyPath)
    return nil
}
```

**Testing:**
- Test AtomicSwap in FileSystemStorage
- Test AtomicSwap rollback on failure
- Test interface is properly implemented

---

### Task 7: Integrate Atomic Swap into Certificate Manager

**File:** `internal/agent/device/certmanager/manager.go` (modify)

**Objective:** Integrate atomic swap after validation.

**Implementation Steps:**

1. **Add atomic swap after validation:**
```go
// In ensureCertificate_do, after validation:
if isRenewal {
    // ... write to pending and validate ...
    
    // NEW: Perform atomic swap after validation
    if err := cert.Storage.AtomicSwap(ctx); err != nil {
        _ = cert.Storage.CleanupPending(ctx)
        return nil, fmt.Errorf("failed to atomically swap certificate: %w", err)
    }
    
    cm.log.Infof("Certificate %q/%q atomically swapped to active location", providerName, cert.Name)
    
    // Clean up pending files (they're now active)
    // Actually, they're already moved, so no cleanup needed
}
```

**Testing:**
- Test atomic swap is called after validation
- Test atomic swap failure handling
- Test certificate is active after swap

---

### Task 8: Add Power Loss Resilience Testing

**File:** `internal/agent/device/certmanager/swap.go` (modify)

**Objective:** Ensure swap is resilient to power loss.

**Implementation Steps:**

1. **Document power loss resilience:**
```go
// AtomicSwap performs an atomic swap of pending certificate to active location.
// 
// Power Loss Resilience:
// - os.Rename is atomic on POSIX systems (single filesystem operation)
// - If power is lost during certificate rename: either old or new certificate is active
// - If power is lost during key rename: certificate may be swapped but key not
// - In this case, the device will fail to authenticate, but certificate is preserved
// - On restart, the device can detect the mismatch and recover
//
// To improve resilience:
// - Both renames should complete or both should fail
// - We use rollback if key swap fails
func (cv *CertificateValidator) AtomicSwap(ctx context.Context, storage provider.StorageProvider) error {
    // ... implementation ...
}
```

2. **Add recovery detection (for future story):**
```go
// CheckCertificateKeyMatch verifies that the active certificate and key match.
// This can be used on startup to detect power loss during swap.
func (cv *CertificateValidator) CheckCertificateKeyMatch(ctx context.Context, storage provider.StorageProvider) error {
    // Load active certificate and key
    cert, err := storage.LoadCertificate(ctx)
    if err != nil {
        return fmt.Errorf("failed to load certificate: %w", err)
    }

    // Load key (implementation depends on storage type)
    // Verify they match
    // Return error if mismatch detected
    
    return nil
}
```

**Testing:**
- Test power loss scenarios (simulated)
- Test certificate/key mismatch detection
- Test recovery after power loss

---

## Unit Tests

### Test File: `internal/agent/device/certmanager/swap_atomic_test.go` (new)

**Test Cases:**

1. **TestAtomicSwap:**
   - Atomic swap succeeds
   - Certificate and key are swapped
   - Pending files become active

2. **TestAtomicSwapWithBackup:**
   - Backup is created before swap
   - Backup is cleaned up after success
   - Backup is used for rollback on failure

3. **TestAtomicSwapRollback:**
   - Rollback on certificate swap failure
   - Rollback on key swap failure
   - Backup is restored correctly

4. **TestAtomicSwapPowerLoss:**
   - Simulate power loss during swap
   - Verify either old or new certificate is active
   - Verify device can recover

5. **TestAtomicSwapConcurrency:**
   - Multiple swap attempts are handled
   - Concurrent swaps don't corrupt state
   - Only one swap succeeds at a time

---

## Integration Tests

### Test File: `test/integration/certificate_atomic_swap_test.go` (new)

**Test Cases:**

1. **TestAtomicSwapFlow:**
   - Complete flow: pending -> validate -> swap -> active
   - Certificate is active after swap
   - Old certificate is replaced

2. **TestAtomicSwapFailure:**
   - Swap failure triggers rollback
   - Old certificate is preserved
   - Pending files are cleaned up

3. **TestAtomicSwapPowerLoss:**
   - Simulate power loss scenarios
   - Verify device can recover
   - Verify certificate state is consistent

---

## Code Review Checklist

- [ ] Atomic swap uses POSIX rename
- [ ] Backup is created before swap
- [ ] Rollback works correctly
- [ ] Power loss resilience is considered
- [ ] Error handling is comprehensive
- [ ] Logging is appropriate
- [ ] Unit tests achieve >80% coverage
- [ ] Integration tests cover main flows
- [ ] Power loss scenarios are tested
- [ ] Code follows existing patterns

---

## Definition of Done

- [ ] AtomicSwap method implemented
- [ ] Backup mechanism implemented
- [ ] Rollback logic implemented
- [ ] Path access methods added
- [ ] Storage provider interface updated
- [ ] Integration with certificate manager added
- [ ] Power loss resilience documented
- [ ] Unit tests written and passing (>80% coverage)
- [ ] Integration tests written and passing
- [ ] Power loss scenarios tested
- [ ] Code reviewed and approved
- [ ] Documentation updated

---

## Related Files

- `internal/agent/device/certmanager/swap.go` - Certificate swap operations
- `internal/agent/device/certmanager/provider/storage/fs.go` - File system storage
- `internal/agent/device/certmanager/manager.go` - Certificate manager
- `internal/agent/device/fileio/writer.go` - File I/O operations

---

## Dependencies

- **EDM-323-EPIC-3-STORY-2**: Certificate Validation Before Activation (must be completed)
  - Requires certificate validation to be implemented

---

## Notes

- **POSIX Atomic Rename**: The `os.Rename` operation is atomic on POSIX-compliant filesystems. This means the rename either completes fully or not at all, even during power loss.

- **Two-Step Swap**: The swap involves two rename operations (certificate and key). If the second fails, we rollback the first. This ensures consistency.

- **Backup Strategy**: A backup is created before swap to enable rollback. The backup is cleaned up after successful swap.

- **Power Loss Resilience**: If power is lost during swap:
  - If lost during certificate rename: either old or new certificate is active (atomic)
  - If lost during key rename: certificate may be swapped but key not (requires recovery)
  - On restart, the device can detect mismatches and recover

- **Recovery**: Future stories may add recovery logic to detect and fix certificate/key mismatches after power loss.

- **Concurrency**: The certificate manager's processing queue ensures only one certificate operation happens at a time, preventing concurrent swaps.

- **File Permissions**: File permissions are preserved during rename operations on POSIX systems.

---

**Document End**

