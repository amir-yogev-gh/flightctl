# Developer Story: Pending Certificate Storage Mechanism

**Story ID:** EDM-323-EPIC-3-STORY-1  
**Epic:** EDM-323-EPIC-3 (Atomic Certificate Swap)  
**Feature:** EDM-323 (Agent Certificate Rotation)  
**Story Points:** 3  
**Priority:** High

## Overview

Implement a mechanism to store new certificates in a pending location before activation. This allows validation of certificates before making them active, ensuring the device always has a valid certificate available.

## Implementation Tasks

### Task 1: Add Pending Path Methods to FileSystemStorage

**File:** `internal/agent/device/certmanager/provider/storage/fs.go` (modify)

**Objective:** Add methods to generate pending file paths and check for pending certificates.

**Implementation Steps:**

1. **Add pending path helper methods:**
```go
// getPendingCertPath returns the pending certificate path for a given certificate path.
func (fs *FileSystemStorage) getPendingCertPath() string {
    return fs.CertPath + ".pending"
}

// getPendingKeyPath returns the pending key path for a given key path.
func (fs *FileSystemStorage) getPendingKeyPath() string {
    return fs.KeyPath + ".pending"
}

// HasPendingCertificate checks if a pending certificate exists.
func (fs *FileSystemStorage) HasPendingCertificate(ctx context.Context) (bool, error) {
    pendingCertPath := fs.getPendingCertPath()
    exists, err := fs.deviceReadWriter.PathExists(pendingCertPath)
    if err != nil {
        return false, fmt.Errorf("failed to check pending certificate existence: %w", err)
    }
    return exists, nil
}
```

**Testing:**
- Test getPendingCertPath returns correct path
- Test getPendingKeyPath returns correct path
- Test HasPendingCertificate detects pending certificates

---

### Task 2: Implement WritePending Method

**File:** `internal/agent/device/certmanager/provider/storage/fs.go` (modify)

**Objective:** Implement method to write certificates to pending locations.

**Implementation Steps:**

1. **Add WritePending method:**
```go
// WritePending stores a certificate and private key to pending locations.
// This allows validation before activation. The old certificate remains in the active location.
func (fs *FileSystemStorage) WritePending(cert *x509.Certificate, keyPEM []byte) error {
    certPEM, err := oscrypto.EncodeCertificates(cert)
    if err != nil {
        return fmt.Errorf("failed to encode certificate: %w", err)
    }

    pendingCertPath := fs.getPendingCertPath()
    pendingKeyPath := fs.getPendingKeyPath()

    // Create directories for pending files
    if err := fs.deviceReadWriter.MkdirAll(filepath.Dir(pendingCertPath), 0700); err != nil {
        return fmt.Errorf("mkdir for pending cert path: %w", err)
    }
    if err := fs.deviceReadWriter.MkdirAll(filepath.Dir(pendingKeyPath), 0700); err != nil {
        return fmt.Errorf("mkdir for pending key path: %w", err)
    }

    // Write pending certificate (0644)
    // Use atomic write to ensure consistency
    if err := fs.deviceReadWriter.WriteFile(pendingCertPath, certPEM, fileio.DefaultFilePermissions); err != nil {
        fs.log.Errorf("failed to write pending cert to %s: %v", pendingCertPath, err)
        // Clean up on failure
        _ = fs.deviceReadWriter.RemoveFile(pendingCertPath)
        return fmt.Errorf("write pending cert: %w", err)
    }

    // Write pending key (0600)
    // Use atomic write to ensure consistency
    if err := fs.deviceReadWriter.WriteFile(pendingKeyPath, keyPEM, 0o600); err != nil {
        fs.log.Errorf("failed to write pending key to %s: %v", pendingKeyPath, err)
        // Clean up both files on failure
        _ = fs.deviceReadWriter.RemoveFile(pendingCertPath)
        _ = fs.deviceReadWriter.RemoveFile(pendingKeyPath)
        return fmt.Errorf("write pending key: %w", err)
    }

    fs.log.Debugf("Successfully wrote pending cert and key to %s and %s", pendingCertPath, pendingKeyPath)
    return nil
}
```

**Testing:**
- Test WritePending writes to pending locations
- Test WritePending creates directories
- Test WritePending cleans up on failure
- Test WritePending preserves old certificate

---

### Task 3: Add LoadPendingCertificate Method

**File:** `internal/agent/device/certmanager/provider/storage/fs.go` (modify)

**Objective:** Add method to load pending certificate for validation.

**Implementation Steps:**

1. **Add LoadPendingCertificate method:**
```go
// LoadPendingCertificate loads a pending certificate from the filesystem.
// It reads the pending certificate file and parses it as a PEM-encoded X.509 certificate.
func (fs *FileSystemStorage) LoadPendingCertificate(ctx context.Context) (*x509.Certificate, error) {
    pendingCertPath := fs.getPendingCertPath()
    certPEM, err := fs.deviceReadWriter.ReadFile(pendingCertPath)
    if err != nil {
        return nil, fmt.Errorf("reading pending cert file: %w", err)
    }

    cert, err := fccrypto.ParsePEMCertificate(certPEM)
    if err != nil {
        return nil, fmt.Errorf("failed to parse PEM certificate: %w", err)
    }
    return cert, nil
}

// LoadPendingKey loads the pending private key from the filesystem.
func (fs *FileSystemStorage) LoadPendingKey(ctx context.Context) ([]byte, error) {
    pendingKeyPath := fs.getPendingKeyPath()
    keyPEM, err := fs.deviceReadWriter.ReadFile(pendingKeyPath)
    if err != nil {
        return nil, fmt.Errorf("reading pending key file: %w", err)
    }
    return keyPEM, nil
}
```

**Testing:**
- Test LoadPendingCertificate loads certificate correctly
- Test LoadPendingKey loads key correctly
- Test methods handle missing files

---

### Task 4: Add CleanupPending Method

**File:** `internal/agent/device/certmanager/provider/storage/fs.go` (modify)

**Objective:** Add method to clean up pending certificate files.

**Implementation Steps:**

1. **Add CleanupPending method:**
```go
// CleanupPending removes pending certificate and key files.
// This is used when validation fails or after successful activation.
func (fs *FileSystemStorage) CleanupPending(ctx context.Context) error {
    pendingCertPath := fs.getPendingCertPath()
    pendingKeyPath := fs.getPendingKeyPath()
    
    var errors []error
    
    if err := fs.deviceReadWriter.RemoveFile(pendingCertPath); err != nil {
        if !os.IsNotExist(err) {
            errors = append(errors, fmt.Errorf("failed to remove pending cert: %w", err))
            fs.log.Warnf("failed to remove pending cert file %s: %v", pendingCertPath, err)
        }
    }
    
    if err := fs.deviceReadWriter.RemoveFile(pendingKeyPath); err != nil {
        if !os.IsNotExist(err) {
            errors = append(errors, fmt.Errorf("failed to remove pending key: %w", err))
            fs.log.Warnf("failed to remove pending key file %s: %v", pendingKeyPath, err)
        }
    }
    
    if len(errors) > 0 {
        return fmt.Errorf("cleanup pending files failed: %v", errors)
    }
    
    fs.log.Debugf("Cleaned up pending certificate files")
    return nil
}
```

**Testing:**
- Test CleanupPending removes pending files
- Test CleanupPending handles missing files gracefully
- Test CleanupPending logs errors appropriately

---

### Task 5: Update Storage Provider Interface

**File:** `internal/agent/device/certmanager/provider/provider.go` (modify)

**Objective:** Add pending certificate methods to the StorageProvider interface.

**Implementation Steps:**

1. **Add methods to StorageProvider interface:**
```go
// StorageProvider defines the interface for certificate storage operations.
type StorageProvider interface {
    // LoadCertificate loads the active certificate from storage.
    LoadCertificate(ctx context.Context) (*x509.Certificate, error)
    
    // Write stores a certificate and private key to the active location.
    Write(cert *x509.Certificate, keyPEM []byte) error
    
    // WritePending stores a certificate and private key to pending locations.
    WritePending(cert *x509.Certificate, keyPEM []byte) error
    
    // LoadPendingCertificate loads a pending certificate from storage.
    LoadPendingCertificate(ctx context.Context) (*x509.Certificate, error)
    
    // LoadPendingKey loads the pending private key from storage.
    LoadPendingKey(ctx context.Context) ([]byte, error)
    
    // HasPendingCertificate checks if a pending certificate exists.
    HasPendingCertificate(ctx context.Context) (bool, error)
    
    // CleanupPending removes pending certificate and key files.
    CleanupPending(ctx context.Context) error
    
    // Delete removes certificate and private key files from storage.
    Delete(ctx context.Context) error
}
```

**Testing:**
- Test interface is properly defined
- Test all implementations satisfy interface

---

### Task 6: Integrate WritePending into Certificate Manager

**File:** `internal/agent/device/certmanager/manager.go` (modify)

**Objective:** Use WritePending when storing renewal certificates.

**Implementation Steps:**

1. **Modify ensureCertificate_do to use WritePending for renewals:**
```go
// ensureCertificate_do performs the actual certificate provisioning work.
func (cm *CertManager) ensureCertificate_do(ctx context.Context, providerName string, cert *certificate, cfg *provider.CertificateConfig) (*time.Duration, error) {
    // ... existing code ...
    
    ready, crt, keyBytes, err := cert.Provisioner.Provision(ctx)
    if err != nil {
        return nil, err
    }

    if !ready {
        return &cm.requeueDelay, nil
    }

    if ctx.Err() != nil {
        return nil, ctx.Err()
    }

    // check storage drift
    if !cert.Config.Storage.Equal(cfg.Storage) {
        cm.log.Debugf("Certificate %q for provider %q: storage configuration changed, deleting old storage", certName, providerName)
        if err := cm.purgeStorage(ctx, providerName, cert); err != nil {
            cm.log.Error(err.Error())
        }
    }

    // NEW: Check if this is a renewal (certificate already exists)
    isRenewal := cert.Info.NotAfter != nil && cert.Info.NotBefore != nil
    
    if isRenewal {
        // For renewals, write to pending location first
        if err := cert.Storage.WritePending(crt, keyBytes); err != nil {
            return nil, fmt.Errorf("failed to write pending certificate: %w", err)
        }
        cm.log.Infof("Certificate %q/%q written to pending location for validation", providerName, cert.Name)
        // Certificate will be activated after validation in next story
        // For now, we just write to pending
    } else {
        // For initial provisioning, write directly to active location
        if err := cert.Storage.Write(crt, keyBytes); err != nil {
            return nil, err
        }
    }

    cm.addCertificateInfo(cert, crt)

    // ... rest of method ...
}
```

**Testing:**
- Test WritePending is used for renewals
- Test Write is used for initial provisioning
- Test error handling

---

### Task 7: Add Error Handling and Logging

**File:** `internal/agent/device/certmanager/provider/storage/fs.go` (modify)

**Objective:** Add comprehensive error handling and logging.

**Implementation Steps:**

1. **Enhance error handling in WritePending:**
```go
// In WritePending, add detailed error messages:
if err := fs.deviceReadWriter.WriteFile(pendingCertPath, certPEM, fileio.DefaultFilePermissions); err != nil {
    fs.log.Errorf("failed to write pending cert to %s: %v", pendingCertPath, err)
    // Clean up on failure
    if cleanupErr := fs.deviceReadWriter.RemoveFile(pendingCertPath); cleanupErr != nil {
        fs.log.Warnf("failed to cleanup pending cert after write failure: %v", cleanupErr)
    }
    return fmt.Errorf("write pending cert: %w", err)
}
```

2. **Add logging for pending operations:**
```go
// In WritePending, add success logging:
fs.log.Debugf("Successfully wrote pending cert and key to %s and %s", pendingCertPath, pendingKeyPath)

// In CleanupPending, add logging:
fs.log.Debugf("Cleaned up pending certificate files at %s and %s", pendingCertPath, pendingKeyPath)
```

**Testing:**
- Test error messages are clear
- Test logging includes relevant information

---

## Unit Tests

### Test File: `internal/agent/device/certmanager/provider/storage/fs_pending_test.go` (new)

**Test Cases:**

1. **TestGetPendingPaths:**
   - getPendingCertPath returns correct path
   - getPendingKeyPath returns correct path
   - Paths have .pending suffix

2. **TestWritePending:**
   - WritePending writes to pending locations
   - WritePending creates directories
   - WritePending uses correct permissions
   - WritePending cleans up on failure
   - WritePending preserves old certificate

3. **TestLoadPendingCertificate:**
   - LoadPendingCertificate loads certificate correctly
   - LoadPendingKey loads key correctly
   - Methods handle missing files

4. **TestHasPendingCertificate:**
   - HasPendingCertificate detects pending certificates
   - HasPendingCertificate returns false when no pending
   - HasPendingCertificate handles errors

5. **TestCleanupPending:**
   - CleanupPending removes pending files
   - CleanupPending handles missing files gracefully
   - CleanupPending logs errors appropriately

6. **TestPendingStorageIntegration:**
   - WritePending followed by LoadPendingCertificate works
   - CleanupPending removes pending files
   - Active certificate is preserved during pending operations

---

## Integration Tests

### Test File: `test/integration/certificate_pending_storage_test.go` (new)

**Test Cases:**

1. **TestPendingCertificateStorage:**
   - Certificate written to pending location
   - Active certificate remains unchanged
   - Pending certificate can be loaded

2. **TestPendingStorageErrorHandling:**
   - Write failure cleans up pending files
   - Active certificate preserved on failure
   - Error messages are clear

3. **TestPendingStorageCleanup:**
   - CleanupPending removes pending files
   - CleanupPending is idempotent
   - Active certificate unaffected by cleanup

---

## Code Review Checklist

- [ ] Pending paths are generated correctly
- [ ] WritePending uses atomic writes
- [ ] WritePending cleans up on failure
- [ ] Active certificate is preserved
- [ ] Error handling is comprehensive
- [ ] Logging is appropriate
- [ ] Interface is properly extended
- [ ] Unit tests achieve >80% coverage
- [ ] Integration tests cover main flows
- [ ] Code follows existing patterns

---

## Definition of Done

- [ ] Pending path methods implemented
- [ ] WritePending method implemented
- [ ] LoadPendingCertificate method implemented
- [ ] CleanupPending method implemented
- [ ] StorageProvider interface updated
- [ ] Certificate manager integration added
- [ ] Error handling and logging added
- [ ] Unit tests written and passing (>80% coverage)
- [ ] Integration tests written and passing
- [ ] Code reviewed and approved
- [ ] Documentation updated

---

## Related Files

- `internal/agent/device/certmanager/provider/storage/fs.go` - File system storage
- `internal/agent/device/certmanager/provider/provider.go` - Storage provider interface
- `internal/agent/device/certmanager/manager.go` - Certificate manager
- `internal/agent/device/fileio/writer.go` - File I/O operations

---

## Dependencies

- **EDM-323-EPIC-2-STORY-5**: Agent Certificate Reception and Storage (must be completed)
  - Requires certificate reception to be implemented

---

## Notes

- **Atomic Writes**: The `fileio.WriteFile` method already uses atomic writes (via `writeFileAtomically`), so pending certificate writes are atomic.

- **Pending Location**: Pending certificates are stored with `.pending` suffix to clearly distinguish them from active certificates.

- **Active Certificate Preservation**: The old certificate remains in the active location until the new certificate is validated and activated. This ensures the device always has a valid certificate.

- **Error Handling**: If writing to pending location fails, the pending files are cleaned up and the error is returned. The active certificate is never modified during pending write operations.

- **Directory Creation**: Directories are created with 0700 permissions for security. Certificate files use 0644 and key files use 0600.

- **Backward Compatibility**: The existing `Write` method continues to work for initial provisioning. `WritePending` is used only for renewals.

---

**Document End**

