package storage

import (
	"context"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/flightctl/flightctl/internal/agent/device/certmanager/provider"
	"github.com/flightctl/flightctl/internal/agent/device/fileio"
	fccrypto "github.com/flightctl/flightctl/pkg/crypto"
	oscrypto "github.com/openshift/library-go/pkg/crypto"
)

// FileSystemStorageConfig defines configuration for filesystem-based certificate storage.
// It specifies where certificates and private keys should be stored on the filesystem
// and what permissions should be applied to the files.
type FileSystemStorageConfig struct {
	// CertPath is the path where the certificate will be stored
	CertPath string `json:"cert-path"`
	// KeyPath is the path where the private key will be stored
	KeyPath string `json:"key-path"`
}

// FileSystemStorage handles certificate storage on the local filesystem.
// It stores certificates and private keys as managed files with appropriate permissions
// and supports loading existing certificates from the filesystem.
type FileSystemStorage struct {
	// Path where the certificate file will be stored
	CertPath string
	// Path where the private key file will be stored
	KeyPath string
	// File I/O interface for reading and writing files
	deviceReadWriter fileio.ReadWriter
	// Logger for storage operations
	log provider.Logger
}

// NewFileSystemStorage creates a new filesystem storage provider with the specified configuration.
// It uses the provided file I/O interface and logger for operations.
func NewFileSystemStorage(certPath, keyPath string, rw fileio.ReadWriter, log provider.Logger) *FileSystemStorage {
	return &FileSystemStorage{
		CertPath:         certPath,
		KeyPath:          keyPath,
		deviceReadWriter: rw,
		log:              log,
	}
}

// LoadCertificate loads a certificate from the filesystem.
// It reads the certificate file and parses it as a PEM-encoded X.509 certificate.

func (fs *FileSystemStorage) LoadCertificate(_ context.Context) (*x509.Certificate, error) {
	certPEM, err := fs.deviceReadWriter.ReadFile(fs.CertPath)
	if err != nil {
		return nil, fmt.Errorf("reading cert file: %w", err)
	}

	cert, err := fccrypto.ParsePEMCertificate(certPEM)
	if err != nil {
		return nil, fmt.Errorf("failed to parse PEM certificate: %w", err)
	}
	return cert, nil
}

// Write stores a certificate and private key to the filesystem.
// It creates the necessary directories and writes both files with appropriate permissions.
func (fs *FileSystemStorage) Write(cert *x509.Certificate, keyPEM []byte) error {
	certPEM, err := oscrypto.EncodeCertificates(cert)
	if err != nil {
		return err
	}

	if err := fs.deviceReadWriter.MkdirAll(filepath.Dir(fs.CertPath), 0700); err != nil {
		return fmt.Errorf("mkdir for cert path: %w", err)
	}
	if err := fs.deviceReadWriter.MkdirAll(filepath.Dir(fs.KeyPath), 0700); err != nil {
		return fmt.Errorf("mkdir for key path: %w", err)
	}

	// write certificate (0644)
	if err := fs.deviceReadWriter.WriteFile(fs.CertPath, certPEM, fileio.DefaultFilePermissions); err != nil {
		fs.log.Errorf("failed to write cert to %s: %v", fs.CertPath, err)
		return fmt.Errorf("write cert: %w", err)
	}

	// write private key (0600)
	if err := fs.deviceReadWriter.WriteFile(fs.KeyPath, keyPEM, 0o600); err != nil {
		fs.log.Errorf("failed to write key to %s: %v", fs.KeyPath, err)
		return fmt.Errorf("write key: %w", err)
	}

	fs.log.Debugf("Successfully wrote cert and key to %s and %s", fs.CertPath, fs.KeyPath)
	return nil
}

// getPendingCertPath returns the pending certificate path for a given certificate path.
func (fs *FileSystemStorage) getPendingCertPath() string {
	return fs.CertPath + ".pending"
}

// getPendingKeyPath returns the pending key path for a given key path.
func (fs *FileSystemStorage) getPendingKeyPath() string {
	return fs.KeyPath + ".pending"
}

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

// HasPendingCertificate checks if a pending certificate exists.
func (fs *FileSystemStorage) HasPendingCertificate(ctx context.Context) (bool, error) {
	pendingCertPath := fs.getPendingCertPath()
	exists, err := fs.deviceReadWriter.PathExists(pendingCertPath)
	if err != nil {
		return false, fmt.Errorf("failed to check pending certificate existence: %w", err)
	}
	return exists, nil
}

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
		if cleanupErr := fs.deviceReadWriter.RemoveFile(pendingCertPath); cleanupErr != nil {
			fs.log.Warnf("failed to cleanup pending cert after write failure: %v", cleanupErr)
		}
		return fmt.Errorf("write pending cert: %w", err)
	}

	// Write pending key (0600)
	// Use atomic write to ensure consistency
	if err := fs.deviceReadWriter.WriteFile(pendingKeyPath, keyPEM, 0o600); err != nil {
		fs.log.Errorf("failed to write pending key to %s: %v", pendingKeyPath, err)
		// Clean up both files on failure
		if cleanupErr := fs.deviceReadWriter.RemoveFile(pendingCertPath); cleanupErr != nil {
			fs.log.Warnf("failed to cleanup pending cert after key write failure: %v", cleanupErr)
		}
		if cleanupErr := fs.deviceReadWriter.RemoveFile(pendingKeyPath); cleanupErr != nil {
			fs.log.Warnf("failed to cleanup pending key after write failure: %v", cleanupErr)
		}
		return fmt.Errorf("write pending key: %w", err)
	}

	fs.log.Debugf("Successfully wrote pending cert and key to %s and %s", pendingCertPath, pendingKeyPath)
	return nil
}

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

// CleanupPending removes pending certificate and key files.
// This is used when validation fails or after successful activation.
func (fs *FileSystemStorage) CleanupPending(ctx context.Context) error {
	pendingCertPath := fs.getPendingCertPath()
	pendingKeyPath := fs.getPendingKeyPath()

	var errors []error

	if err := fs.deviceReadWriter.RemoveFile(pendingCertPath); err != nil {
		if !fileio.IsNotExist(err) {
			errors = append(errors, fmt.Errorf("failed to remove pending cert: %w", err))
			fs.log.Warnf("failed to remove pending cert file %s: %v", pendingCertPath, err)
		}
	}

	if err := fs.deviceReadWriter.RemoveFile(pendingKeyPath); err != nil {
		if !fileio.IsNotExist(err) {
			errors = append(errors, fmt.Errorf("failed to remove pending key: %w", err))
			fs.log.Warnf("failed to remove pending key file %s: %v", pendingKeyPath, err)
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("cleanup pending files failed: %v", errors)
	}

	fs.log.Debugf("Cleaned up pending certificate files at %s and %s", pendingCertPath, pendingKeyPath)
	return nil
}

// AtomicSwap performs an atomic swap of pending certificate to active location.
// It uses POSIX atomic rename operations to ensure the swap is atomic.
//
// Power Loss Resilience:
// - os.Rename is atomic on POSIX systems (single filesystem operation)
// - If power is lost during certificate rename: either old or new certificate is active
// - If power is lost during key rename: certificate may be swapped but key not
// - In this case, the device will fail to authenticate, but certificate is preserved
// - On restart, the device can detect the mismatch and recover
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

	// Get full paths using readWriter
	pendingCertPathFull := fs.deviceReadWriter.PathFor(pendingCertPath)
	activeCertPathFull := fs.deviceReadWriter.PathFor(activeCertPath)
	pendingKeyPathFull := fs.deviceReadWriter.PathFor(pendingKeyPath)
	activeKeyPathFull := fs.deviceReadWriter.PathFor(activeKeyPath)

	// Step 1: Create backup of active certificate (if exists)
	backupCertPath, backupKeyPath, err := fs.backupActiveCertificate(ctx, activeCertPathFull, activeKeyPathFull)
	if err != nil {
		return fmt.Errorf("failed to backup active certificate: %w", err)
	}

	// Step 2: Perform atomic swap using POSIX rename
	// os.Rename is atomic on POSIX systems
	certSwapErr := os.Rename(pendingCertPathFull, activeCertPathFull)
	keySwapErr := os.Rename(pendingKeyPathFull, activeKeyPathFull)

	// Step 3: Handle swap results
	if certSwapErr != nil {
		return fmt.Errorf("failed to atomically swap certificate: %w", certSwapErr)
	}

	if keySwapErr != nil {
		// Certificate swap succeeded but key swap failed
		// Rollback certificate swap using public API
		fs.log.Errorf("Key swap failed, rolling back certificate swap: %v", keySwapErr)
		if rollbackErr := fs.RollbackSwap(ctx, keySwapErr); rollbackErr != nil {
			fs.log.Errorf("Rollback failed: %v", rollbackErr)
		}
		return fmt.Errorf("failed to atomically swap key: %w", keySwapErr)
	}

	// Step 4: Clean up backup after successful swap
	if backupCertPath != "" {
		_ = fs.deviceReadWriter.RemoveFile(backupCertPath)
		_ = fs.deviceReadWriter.RemoveFile(backupKeyPath)
		fs.log.Debugf("Cleaned up backup files after successful swap")
	}

	fs.log.Debugf("Atomic swap completed: %s -> %s, %s -> %s",
		pendingCertPath, activeCertPath, pendingKeyPath, activeKeyPath)
	return nil
}

// backupActiveCertificate creates a backup of the active certificate and key.
func (fs *FileSystemStorage) backupActiveCertificate(ctx context.Context, activeCertPath, activeKeyPath string) (string, string, error) {
	backupCertPath := activeCertPath + ".backup"
	backupKeyPath := activeKeyPath + ".backup"

	// Check if active certificate exists
	exists, err := fs.deviceReadWriter.PathExists(activeCertPath)
	if err != nil {
		return "", "", fmt.Errorf("failed to check active certificate: %w", err)
	}
	if !exists {
		// No active certificate to backup (first certificate)
		return "", "", nil
	}

	// Copy active certificate to backup
	if err := fs.copyFile(activeCertPath, backupCertPath); err != nil {
		return "", "", fmt.Errorf("failed to backup certificate: %w", err)
	}

	// Copy active key to backup
	if err := fs.copyFile(activeKeyPath, backupKeyPath); err != nil {
		// Clean up certificate backup on failure
		_ = fs.deviceReadWriter.RemoveFile(backupCertPath)
		return "", "", fmt.Errorf("failed to backup key: %w", err)
	}

	fs.log.Debugf("Created backup of active certificate at %s and %s", backupCertPath, backupKeyPath)
	return backupCertPath, backupKeyPath, nil
}

// copyFile copies a file from source to destination.
func (fs *FileSystemStorage) copyFile(src, dst string) error {
	data, err := fs.deviceReadWriter.ReadFile(src)
	if err != nil {
		return fmt.Errorf("failed to read source file: %w", err)
	}

	// Get source file permissions - use 0600 for keys, 0644 for certs
	perm := fileio.DefaultFilePermissions
	if strings.HasSuffix(dst, ".key") || (strings.HasSuffix(dst, ".backup") && strings.Contains(dst, ".key")) {
		perm = 0o600
	}

	if err := fs.deviceReadWriter.WriteFile(dst, data, perm); err != nil {
		return fmt.Errorf("failed to write destination file: %w", err)
	}

	return nil
}

// RollbackSwap performs a rollback of a failed certificate swap.
// It restores the old certificate from backup and cleans up pending files.
func (fs *FileSystemStorage) RollbackSwap(ctx context.Context, swapError error) error {
	fs.log.Warnf("Rolling back failed certificate swap: %v", swapError)

	activeCertPath := fs.CertPath
	activeKeyPath := fs.KeyPath
	backupCertPath := activeCertPath + ".backup"
	backupKeyPath := activeKeyPath + ".backup"

	// Step 1: Check if backup exists
	backupExists, err := fs.deviceReadWriter.PathExists(backupCertPath)
	if err != nil {
		fs.log.Errorf("Failed to check backup existence: %v", err)
		// Continue with cleanup even if backup check fails
	}

	// Step 2: Restore from backup if available
	if backupExists {
		fs.log.Infof("Restoring certificate from backup")

		// Get full paths for atomic rename
		backupCertPathFull := fs.deviceReadWriter.PathFor(backupCertPath)
		backupKeyPathFull := fs.deviceReadWriter.PathFor(backupKeyPath)
		activeCertPathFull := fs.deviceReadWriter.PathFor(activeCertPath)
		activeKeyPathFull := fs.deviceReadWriter.PathFor(activeKeyPath)

		// Restore certificate
		if err := os.Rename(backupCertPathFull, activeCertPathFull); err != nil {
			fs.log.Errorf("Failed to restore certificate from backup: %v", err)
			// Continue with cleanup even if restore fails
		} else {
			fs.log.Debugf("Certificate restored from backup")
		}

		// Restore key
		keyBackupExists, _ := fs.deviceReadWriter.PathExists(backupKeyPath)
		if keyBackupExists {
			if err := os.Rename(backupKeyPathFull, activeKeyPathFull); err != nil {
				fs.log.Errorf("Failed to restore key from backup: %v", err)
			} else {
				fs.log.Debugf("Key restored from backup")
			}
		}
	} else {
		fs.log.Warnf("No backup available for rollback - device may need manual intervention")
	}

	// Step 3: Clean up pending files
	if cleanupErr := fs.CleanupPending(ctx); cleanupErr != nil {
		fs.log.Warnf("Failed to cleanup pending files during rollback: %v", cleanupErr)
		// Don't fail rollback if cleanup fails
	}

	// Step 4: Clean up backup files (after restore or if no backup)
	if backupExists {
		_ = fs.deviceReadWriter.RemoveFile(backupCertPath)
		_ = fs.deviceReadWriter.RemoveFile(backupKeyPath)
	}

	fs.log.Infof("Rollback completed")
	return nil
}

// rollbackCertificateSwap attempts to rollback a failed certificate swap.
// This is an internal method that uses full paths. Use RollbackSwap for the public API.
func (fs *FileSystemStorage) rollbackCertificateSwap(ctx context.Context, pendingCertPath, activeCertPath, backupCertPath, backupKeyPath string) error {
	// Check if backup exists
	backupExists, err := fs.deviceReadWriter.PathExists(backupCertPath)
	if err != nil {
		return fmt.Errorf("failed to check backup: %w", err)
	}

	if backupExists {
		// Restore from backup using atomic rename
		fs.log.Warnf("Restoring certificate from backup due to swap failure")
		activeKeyPath := strings.Replace(activeCertPath, ".crt", ".key", 1)
		if err := os.Rename(backupCertPath, activeCertPath); err != nil {
			return fmt.Errorf("failed to restore certificate from backup: %w", err)
		}
		if err := os.Rename(backupKeyPath, activeKeyPath); err != nil {
			return fmt.Errorf("failed to restore key from backup: %w", err)
		}
		fs.log.Infof("Certificate restored from backup")
	} else {
		// No backup available, try to restore from pending if it still exists
		pendingExists, err := fs.deviceReadWriter.PathExists(pendingCertPath)
		if err == nil && pendingExists {
			fs.log.Warnf("No backup available, attempting to restore from pending")
			// This is a best-effort recovery
			// The pending certificate may have been partially swapped
		}
	}

	return nil
}

// Delete removes certificate and private key files from the filesystem.
// It logs warnings if files cannot be deleted but doesn't return errors
// since deletion is a cleanup operation.
func (fs *FileSystemStorage) Delete(_ context.Context) error {
	if err := fs.deviceReadWriter.RemoveFile(fs.CertPath); err != nil {
		fs.log.Warnf("failed to delete cert file %s: %v", fs.CertPath, err)
	}
	if err := fs.deviceReadWriter.RemoveFile(fs.KeyPath); err != nil {
		fs.log.Warnf("failed to delete key file %s: %v", fs.KeyPath, err)
	}
	return nil
}

// FileSystemStorageFactory implements StorageFactory for filesystem-based certificate storage.
// It creates filesystem storage providers that store certificates and keys as files on disk.
type FileSystemStorageFactory struct {
	// File I/O interface for reading and writing files
	rw fileio.ReadWriter
}

// NewFileSystemStorageFactory creates a new filesystem storage factory with the specified file I/O interface.
func NewFileSystemStorageFactory(rw fileio.ReadWriter) *FileSystemStorageFactory {
	return &FileSystemStorageFactory{
		rw: rw,
	}
}

// Type returns the storage type string used as map key in the certificate manager.
func (f *FileSystemStorageFactory) Type() string {
	return string(provider.StorageTypeFilesystem)
}

// New creates a new FileSystemStorage instance from the certificate configuration.
// It decodes the filesystem-specific configuration and sets appropriate default values.
func (f *FileSystemStorageFactory) New(log provider.Logger, cc provider.CertificateConfig) (provider.StorageProvider, error) {
	storage := cc.Storage

	var fsConfig FileSystemStorageConfig
	if err := json.Unmarshal(storage.Config, &fsConfig); err != nil {
		return nil, fmt.Errorf("failed to decode filesystem Storage config for certificate %q: %w", cc.Name, err)
	}

	return NewFileSystemStorage(fsConfig.CertPath, fsConfig.KeyPath, f.rw, log), nil
}

// Validate checks whether the provided configuration is valid for filesystem storage.
// It ensures required fields are present and the configuration is properly formatted.
func (f *FileSystemStorageFactory) Validate(log provider.Logger, cc provider.CertificateConfig) error {
	storage := cc.Storage

	if storage.Type != provider.StorageTypeFilesystem {
		return fmt.Errorf("not a filesystem Storage")
	}

	var fsConfig FileSystemStorageConfig
	if err := json.Unmarshal(storage.Config, &fsConfig); err != nil {
		return fmt.Errorf("failed to decode filesystem Storage config for certificate %q: %w", cc.Name, err)
	}

	if fsConfig.CertPath == "" {
		return fmt.Errorf("cert-path is required for filesystem storage, certificate %s", cc.Name)
	}
	if fsConfig.KeyPath == "" {
		return fmt.Errorf("key-path is required for filesystem storage, certificate %s", cc.Name)
	}

	return nil
}
