# Developer Story: Rollback Mechanism for Failed Swaps

**Story ID:** EDM-323-EPIC-3-STORY-4  
**Epic:** EDM-323-EPIC-3 (Atomic Certificate Swap)  
**Feature:** EDM-323 (Agent Certificate Rotation)  
**Story Points:** 3  
**Priority:** High

## Overview

Implement comprehensive rollback mechanism for failed certificate swaps. When a swap fails, the system should restore the old certificate from backup, clean up pending files, update certificate state, and ensure the device continues operating with a valid certificate.

## Implementation Tasks

### Task 1: Enhance Rollback Logic

**File:** `internal/agent/device/certmanager/swap.go` (modify)

**Objective:** Enhance rollback logic to handle all failure scenarios.

**Implementation Steps:**

1. **Add comprehensive RollbackSwap method:**
```go
// RollbackSwap performs a complete rollback of a failed certificate swap.
// It restores the old certificate from backup, cleans up pending files, and updates state.
func (cv *CertificateValidator) RollbackSwap(ctx context.Context, storage provider.StorageProvider, swapError error) error {
    cv.log.Warnf("Rolling back failed certificate swap for device %q: %v", cv.deviceName, swapError)

    fsStorage, ok := storage.(*storage.FileSystemStorage)
    if !ok {
        return fmt.Errorf("rollback only supported for filesystem storage")
    }

    activeCertPath := fsStorage.GetActiveCertPath()
    activeKeyPath := fsStorage.GetActiveKeyPath()
    backupCertPath := activeCertPath + ".backup"
    backupKeyPath := activeKeyPath + ".backup"

    // Step 1: Check if backup exists
    backupExists, err := fsStorage.deviceReadWriter.PathExists(backupCertPath)
    if err != nil {
        cv.log.Errorf("Failed to check backup existence: %v", err)
        // Continue with cleanup even if backup check fails
    }

    // Step 2: Restore from backup if available
    if backupExists {
        cv.log.Infof("Restoring certificate from backup")
        
        // Restore certificate
        if err := os.Rename(backupCertPath, activeCertPath); err != nil {
            cv.log.Errorf("Failed to restore certificate from backup: %v", err)
            // Continue with cleanup even if restore fails
        } else {
            cv.log.Debugf("Certificate restored from backup")
        }

        // Restore key
        keyBackupExists, _ := fsStorage.deviceReadWriter.PathExists(backupKeyPath)
        if keyBackupExists {
            if err := os.Rename(backupKeyPath, activeKeyPath); err != nil {
                cv.log.Errorf("Failed to restore key from backup: %v", err)
            } else {
                cv.log.Debugf("Key restored from backup")
            }
        }
    } else {
        cv.log.Warnf("No backup available for rollback - device may need manual intervention")
    }

    // Step 3: Clean up pending files
    if cleanupErr := storage.CleanupPending(ctx); cleanupErr != nil {
        cv.log.Warnf("Failed to cleanup pending files during rollback: %v", cleanupErr)
        // Don't fail rollback if cleanup fails
    }

    // Step 4: Clean up backup files (after restore or if no backup)
    if backupExists {
        _ = fsStorage.deviceReadWriter.RemoveFile(backupCertPath)
        _ = fsStorage.deviceReadWriter.RemoveFile(backupKeyPath)
    }

    cv.log.Infof("Rollback completed for device %q", cv.deviceName)
    return nil
}
```

**Testing:**
- Test RollbackSwap restores from backup
- Test RollbackSwap handles missing backup
- Test RollbackSwap cleans up pending files
- Test RollbackSwap handles errors gracefully

---

### Task 2: Integrate Rollback into Atomic Swap

**File:** `internal/agent/device/certmanager/swap.go` (modify)

**Objective:** Integrate rollback into atomic swap error handling.

**Implementation Steps:**

1. **Update AtomicSwap to call rollback on failure:**
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

    // Step 2: Perform atomic swap
    certSwapErr := os.Rename(pendingCertPath, activeCertPath)
    keySwapErr := os.Rename(pendingKeyPath, activeKeyPath)

    // Step 3: Handle swap results
    if certSwapErr != nil {
        // Certificate swap failed - no rollback needed (nothing changed)
        _ = storage.CleanupPending(ctx)
        return fmt.Errorf("failed to atomically swap certificate: %w", certSwapErr)
    }

    if keySwapErr != nil {
        // Certificate swap succeeded but key swap failed
        // Rollback certificate swap
        cv.log.Errorf("Key swap failed, rolling back: %v", keySwapErr)
        if rollbackErr := cv.RollbackSwap(ctx, storage, keySwapErr); rollbackErr != nil {
            cv.log.Errorf("Rollback failed: %v", rollbackErr)
            // Return original error even if rollback fails
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
- Test rollback is called on swap failure
- Test rollback restores old certificate
- Test rollback cleans up pending files

---

### Task 3: Update Certificate State on Rollback

**File:** `internal/agent/device/certmanager/manager.go` (modify)

**Objective:** Update certificate lifecycle state when rollback occurs.

**Implementation Steps:**

1. **Update state in ensureCertificate_do:**
```go
// In ensureCertificate_do, when swap fails:
if isRenewal {
    // ... write to pending and validate ...
    
    // Perform atomic swap after validation
    if err := cert.Storage.AtomicSwap(ctx); err != nil {
        // Rollback already handled in AtomicSwap, but update state
        _ = cert.Storage.CleanupPending(ctx)
        
        // Update lifecycle state to reflect failure
        if cm.lifecycleManager != nil {
            _ = cm.lifecycleManager.RecordError(ctx, providerName, cert.Name, err)
            // State will be set to renewal_failed by RecordError
        }
        
        return nil, fmt.Errorf("failed to atomically swap certificate: %w", err)
    }
    
    // Update state to normal after successful swap
    if cm.lifecycleManager != nil {
        days, expiration, err := cm.lifecycleManager.CheckRenewal(ctx, providerName, cert.Name, 365)
        if err == nil {
            _ = cm.lifecycleManager.UpdateCertificateState(ctx, providerName, cert.Name, 
                CertificateStateNormal, days, expiration)
        }
    }
    
    cm.log.Infof("Certificate %q/%q atomically swapped to active location", providerName, cert.Name)
}
```

**Testing:**
- Test state is updated on rollback
- Test state reflects renewal failure
- Test state is updated on success

---

### Task 4: Add Rollback to Storage Provider Interface

**File:** `internal/agent/device/certmanager/provider/provider.go` (modify)

**Objective:** Add rollback method to storage provider interface.

**Implementation Steps:**

1. **Add RollbackSwap to StorageProvider interface:**
```go
// StorageProvider defines the interface for certificate storage operations.
type StorageProvider interface {
    // ... existing methods ...
    
    // RollbackSwap performs a rollback of a failed certificate swap.
    // It restores the old certificate from backup and cleans up pending files.
    RollbackSwap(ctx context.Context, swapError error) error
}
```

2. **Implement RollbackSwap in FileSystemStorage:**
```go
// RollbackSwap performs a rollback of a failed certificate swap.
func (fs *FileSystemStorage) RollbackSwap(ctx context.Context, swapError error) error {
    activeCertPath := fs.CertPath
    activeKeyPath := fs.KeyPath
    backupCertPath := activeCertPath + ".backup"
    backupKeyPath := activeKeyPath + ".backup"

    fs.log.Warnf("Rolling back failed certificate swap: %v", swapError)

    // Check if backup exists
    backupExists, err := fs.deviceReadWriter.PathExists(backupCertPath)
    if err != nil {
        fs.log.Errorf("Failed to check backup: %v", err)
    }

    // Restore from backup if available
    if backupExists {
        if err := os.Rename(backupCertPath, activeCertPath); err != nil {
            fs.log.Errorf("Failed to restore certificate from backup: %v", err)
        }
        
        keyBackupExists, _ := fs.deviceReadWriter.PathExists(backupKeyPath)
        if keyBackupExists {
            if err := os.Rename(backupKeyPath, activeKeyPath); err != nil {
                fs.log.Errorf("Failed to restore key from backup: %v", err)
            }
        }
    }

    // Clean up pending files
    _ = fs.CleanupPending(ctx)

    // Clean up backup files
    if backupExists {
        _ = fs.deviceReadWriter.RemoveFile(backupCertPath)
        _ = fs.deviceReadWriter.RemoveFile(backupKeyPath)
    }

    return nil
}
```

**Testing:**
- Test RollbackSwap in FileSystemStorage
- Test RollbackSwap handles all scenarios
- Test interface is properly implemented

---

### Task 5: Add Recovery Detection

**File:** `internal/agent/device/certmanager/swap.go` (modify)

**Objective:** Add method to detect and recover from incomplete swaps.

**Implementation Steps:**

1. **Add DetectAndRecoverIncompleteSwap method:**
```go
// DetectAndRecoverIncompleteSwap detects and recovers from incomplete certificate swaps.
// This is useful after device restart to handle power loss during swap.
func (cv *CertificateValidator) DetectAndRecoverIncompleteSwap(ctx context.Context, storage provider.StorageProvider) error {
    fsStorage, ok := storage.(*storage.FileSystemStorage)
    if !ok {
        return fmt.Errorf("recovery only supported for filesystem storage")
    }

    // Check if pending certificate exists (incomplete swap)
    hasPending, err := storage.HasPendingCertificate(ctx)
    if err != nil {
        return fmt.Errorf("failed to check pending certificate: %w", err)
    }

    if !hasPending {
        // No incomplete swap detected
        return nil
    }

    cv.log.Warnf("Detected incomplete certificate swap (pending certificate exists)")

    // Check if active certificate exists
    activeExists, err := fsStorage.deviceReadWriter.PathExists(fsStorage.GetActiveCertPath())
    if err != nil {
        return fmt.Errorf("failed to check active certificate: %w", err)
    }

    if !activeExists {
        // Active certificate missing - restore from backup or use pending
        cv.log.Warnf("Active certificate missing, attempting recovery")
        return cv.recoverMissingActiveCertificate(ctx, storage)
    }

    // Active certificate exists - validate it matches key
    if err := cv.validateActiveCertificateKeyPair(ctx, storage); err != nil {
        cv.log.Warnf("Active certificate/key mismatch detected: %v", err)
        // Attempt recovery
        return cv.recoverCertificateKeyMismatch(ctx, storage)
    }

    // Active certificate is valid - clean up pending
    cv.log.Infof("Active certificate is valid, cleaning up pending certificate")
    return storage.CleanupPending(ctx)
}

// recoverMissingActiveCertificate recovers when active certificate is missing.
func (cv *CertificateValidator) recoverMissingActiveCertificate(ctx context.Context, storage provider.StorageProvider) error {
    // Try to restore from backup first
    if err := cv.RollbackSwap(ctx, storage, fmt.Errorf("active certificate missing")); err == nil {
        return nil
    }

    // If no backup, try to use pending certificate (if valid)
    cv.log.Warnf("No backup available, attempting to use pending certificate")
    // Validate pending certificate
    // If valid, swap it to active
    // If invalid, return error
    
    return fmt.Errorf("unable to recover missing active certificate")
}

// validateActiveCertificateKeyPair validates that active certificate and key match.
func (cv *CertificateValidator) validateActiveCertificateKeyPair(ctx context.Context, storage provider.StorageProvider) error {
    cert, err := storage.LoadCertificate(ctx)
    if err != nil {
        return fmt.Errorf("failed to load certificate: %w", err)
    }

    // Load key (implementation depends on storage)
    // Verify they match
    
    return nil
}
```

**Testing:**
- Test DetectAndRecoverIncompleteSwap detects incomplete swaps
- Test recovery from missing active certificate
- Test recovery from certificate/key mismatch

---

### Task 6: Add Logging and Error Reporting

**File:** `internal/agent/device/certmanager/swap.go` (modify)

**Objective:** Add comprehensive logging for rollback operations.

**Implementation Steps:**

1. **Enhance logging in RollbackSwap:**
```go
// In RollbackSwap, add detailed logging:
cv.log.Warnf("Rolling back failed certificate swap for device %q: %v", cv.deviceName, swapError)
cv.log.Infof("Restoring certificate from backup")
cv.log.Debugf("Certificate restored from backup")
cv.log.Debugf("Key restored from backup")
cv.log.Warnf("No backup available for rollback - device may need manual intervention")
cv.log.Infof("Rollback completed for device %q", cv.deviceName)
```

2. **Add metrics (if available):**
```go
// In RollbackSwap, add metrics:
if cv.metricsCallback != nil {
    cv.metricsCallback("certificate_swap_rollback_total", 1.0, swapError)
}
```

**Testing:**
- Test logging includes relevant information
- Test log levels are appropriate

---

### Task 7: Add Startup Recovery Check

**File:** `internal/agent/device/certmanager/manager.go` (modify)

**Objective:** Check for incomplete swaps on startup.

**Implementation Steps:**

1. **Add startup recovery check:**
```go
// CheckIncompleteSwaps checks for and recovers from incomplete certificate swaps.
// This should be called on agent startup.
func (cm *CertManager) CheckIncompleteSwaps(ctx context.Context) error {
    cm.log.Debug("Checking for incomplete certificate swaps")

    // Iterate through all certificates
    for providerName, provider := range cm.certificates.providers {
        for certName, cert := range provider.Certificates {
            // Check if certificate has pending files
            hasPending, err := cert.Storage.HasPendingCertificate(ctx)
            if err != nil {
                cm.log.Warnf("Failed to check pending certificate for %q/%q: %v", providerName, certName, err)
                continue
            }

            if hasPending {
                cm.log.Warnf("Detected incomplete swap for certificate %q/%q", providerName, certName)
                
                // Create validator and attempt recovery
                caBundlePath := cm.getCABundlePath(cert.Config)
                validator := NewCertificateValidator(caBundlePath, certName, cm.log)
                
                if err := validator.DetectAndRecoverIncompleteSwap(ctx, cert.Storage); err != nil {
                    cm.log.Errorf("Failed to recover incomplete swap for %q/%q: %v", providerName, certName, err)
                    // Continue with other certificates
                } else {
                    cm.log.Infof("Recovered incomplete swap for certificate %q/%q", providerName, certName)
                }
            }
        }
    }

    return nil
}
```

2. **Call from agent startup:**
```go
// In agent initialization, after CertManager is created:
if err := certManager.CheckIncompleteSwaps(ctx); err != nil {
    log.Warnf("Failed to check incomplete swaps: %v", err)
}
```

**Testing:**
- Test CheckIncompleteSwaps detects incomplete swaps
- Test CheckIncompleteSwaps recovers correctly
- Test startup recovery works

---

## Unit Tests

### Test File: `internal/agent/device/certmanager/swap_rollback_test.go` (new)

**Test Cases:**

1. **TestRollbackSwap:**
   - RollbackSwap restores from backup
   - RollbackSwap cleans up pending files
   - RollbackSwap handles missing backup
   - RollbackSwap handles errors gracefully

2. **TestRollbackIntegration:**
   - Rollback is called on swap failure
   - Old certificate is restored
   - Device continues operating

3. **TestDetectAndRecoverIncompleteSwap:**
   - Detects incomplete swaps
   - Recovers from missing active certificate
   - Recovers from certificate/key mismatch
   - Cleans up pending if active is valid

4. **TestStartupRecovery:**
   - Startup recovery detects incomplete swaps
   - Startup recovery recovers correctly
   - Startup recovery handles errors

---

## Integration Tests

### Test File: `test/integration/certificate_rollback_test.go` (new)

**Test Cases:**

1. **TestRollbackFlow:**
   - Swap failure triggers rollback
   - Old certificate is restored
   - Device continues operating
   - Pending files are cleaned up

2. **TestStartupRecovery:**
   - Incomplete swap detected on startup
   - Recovery restores certificate
   - Device continues operating

3. **TestRollbackScenarios:**
   - Rollback with backup available
   - Rollback without backup
   - Rollback with partial swap

---

## Code Review Checklist

- [ ] Rollback logic is comprehensive
- [ ] Backup restoration works correctly
- [ ] Pending files are cleaned up
- [ ] Certificate state is updated
- [ ] Recovery detection works
- [ ] Startup recovery is implemented
- [ ] Error handling is comprehensive
- [ ] Logging is appropriate
- [ ] Unit tests achieve >80% coverage
- [ ] Integration tests cover main flows
- [ ] Code follows existing patterns

---

## Definition of Done

- [ ] RollbackSwap method implemented
- [ ] Rollback integrated into atomic swap
- [ ] Certificate state update on rollback
- [ ] Recovery detection implemented
- [ ] Startup recovery check added
- [ ] Logging and error reporting added
- [ ] Unit tests written and passing (>80% coverage)
- [ ] Integration tests written and passing
- [ ] Code reviewed and approved
- [ ] Documentation updated

---

## Related Files

- `internal/agent/device/certmanager/swap.go` - Certificate swap and rollback
- `internal/agent/device/certmanager/manager.go` - Certificate manager
- `internal/agent/device/certmanager/provider/storage/fs.go` - Certificate storage
- `internal/agent/agent.go` - Agent startup

---

## Dependencies

- **EDM-323-EPIC-3-STORY-3**: Atomic Certificate Swap (must be completed)
  - Requires atomic swap to be implemented

---

## Notes

- **Rollback Strategy**: Rollback restores the old certificate from backup and cleans up pending files. This ensures the device always has a valid certificate.

- **Backup Availability**: If no backup is available (e.g., first certificate), rollback may not be possible. In this case, the device may need manual intervention.

- **State Updates**: Certificate lifecycle state is updated to reflect rollback. The state transitions to "renewal_failed" to indicate the renewal attempt failed.

- **Recovery Detection**: On startup, the agent checks for incomplete swaps and attempts recovery. This handles power loss scenarios.

- **Error Handling**: Rollback operations are best-effort. If rollback fails, errors are logged but the operation continues to ensure cleanup happens.

- **Cleanup**: Pending files are always cleaned up during rollback, even if backup restoration fails. This prevents accumulation of invalid certificates.

- **Device Continuity**: After rollback, the device continues operating with the old certificate. The renewal can be retried later.

---

**Document End**

