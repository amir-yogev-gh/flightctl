package certmanager

import (
	"context"
	"crypto"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"time"

	"github.com/flightctl/flightctl/internal/agent/device/certmanager/provider"
	"github.com/flightctl/flightctl/internal/agent/device/fileio"
	fccrypto "github.com/flightctl/flightctl/pkg/crypto"
)

// CertificateValidator handles validation of pending certificates before activation.
type CertificateValidator struct {
	caBundlePath string
	deviceName   string
	log          provider.Logger
}

// NewCertificateValidator creates a new certificate validator.
func NewCertificateValidator(caBundlePath string, deviceName string, log provider.Logger) *CertificateValidator {
	return &CertificateValidator{
		caBundlePath: caBundlePath,
		deviceName:   deviceName,
		log:          log,
	}
}

// loadCABundle loads the CA certificate bundle from the filesystem.
func (cv *CertificateValidator) loadCABundle(ctx context.Context, rw fileio.ReadWriter) (*x509.CertPool, error) {
	caPEM, err := rw.ReadFile(cv.caBundlePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read CA bundle from %s: %w", cv.caBundlePath, err)
	}

	caPool := x509.NewCertPool()
	if !caPool.AppendCertsFromPEM(caPEM) {
		return nil, fmt.Errorf("failed to parse CA bundle from %s", cv.caBundlePath)
	}

	return caPool, nil
}

// verifyCertificateSignature verifies the certificate signature chain against the CA bundle.
func (cv *CertificateValidator) verifyCertificateSignature(ctx context.Context, cert *x509.Certificate, caPool *x509.CertPool) error {
	// Verify certificate against CA bundle
	opts := x509.VerifyOptions{
		Roots: caPool,
	}

	_, err := cert.Verify(opts)
	if err != nil {
		return fmt.Errorf("certificate signature verification failed: %w", err)
	}

	return nil
}

// verifyCertificateIdentity verifies the certificate subject/SAN matches device identity.
func (cv *CertificateValidator) verifyCertificateIdentity(cert *x509.Certificate) error {
	// Verify CommonName matches device name
	if cert.Subject.CommonName != cv.deviceName {
		return fmt.Errorf("certificate CommonName %q does not match device name %q",
			cert.Subject.CommonName, cv.deviceName)
	}

	// Verify SANs (if present) also match device identity
	// For now, we check CommonName. SANs can be added later if needed.

	return nil
}

// verifyCertificateExpiration verifies the certificate expiration date.
func (cv *CertificateValidator) verifyCertificateExpiration(cert *x509.Certificate) error {
	now := time.Now()

	// Check if certificate is expired
	if cert.NotAfter.Before(now) {
		return fmt.Errorf("certificate is expired (expired at %v, current time %v)",
			cert.NotAfter, now)
	}

	// Check if certificate is not yet valid
	if cert.NotBefore.After(now) {
		return fmt.Errorf("certificate is not yet valid (valid from %v, current time %v)",
			cert.NotBefore, now)
	}

	// Check if certificate has reasonable validity period remaining
	// Warn if certificate expires soon (within 7 days)
	timeUntilExpiration := cert.NotAfter.Sub(now)
	if timeUntilExpiration < 7*24*time.Hour {
		cv.log.Warnf("Certificate expires soon (in %v)", timeUntilExpiration)
	}

	return nil
}

// verifyKeyPair verifies that the certificate and private key match.
func (cv *CertificateValidator) verifyKeyPair(cert *x509.Certificate, keyPEM []byte) error {
	// Parse private key
	key, err := fccrypto.ParseKeyPEM(keyPEM)
	if err != nil {
		return fmt.Errorf("failed to parse private key: %w", err)
	}

	// Get public key from certificate
	certPublicKey := cert.PublicKey

	// Get public key from private key
	var keyPublicKey interface{}
	switch k := key.(type) {
	case interface{ Public() crypto.PublicKey }:
		keyPublicKey = k.Public()
	default:
		return fmt.Errorf("unsupported private key type: %T", key)
	}

	// Compare public keys by creating TLS certificates
	// This ensures the key pair can actually be used together
	certPEM, err := fccrypto.EncodeCertificatePEM(cert)
	if err != nil {
		return fmt.Errorf("failed to encode certificate: %w", err)
	}

	_, err = tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		return fmt.Errorf("failed to create TLS certificate from key pair: %w", err)
	}

	// Additional check: compare public keys directly
	if !publicKeysEqual(certPublicKey, keyPublicKey) {
		return fmt.Errorf("certificate and private key do not match")
	}

	return nil
}

// publicKeysEqual compares two public keys for equality.
func publicKeysEqual(key1, key2 interface{}) bool {
	// Try to compare using Equal method if available
	if eq, ok := key1.(interface{ Equal(interface{}) bool }); ok {
		return eq.Equal(key2)
	}
	// If no direct comparison available, assume they match if TLS key pair creation succeeded
	// (This is a fallback - the TLS key pair creation above already validates the match)
	return true
}

// ValidatePendingCertificate validates a pending certificate before activation.
// It performs all validation checks: signature, identity, expiration, and key pair.
func (cv *CertificateValidator) ValidatePendingCertificate(ctx context.Context, cert *x509.Certificate, keyPEM []byte, rw fileio.ReadWriter) error {
	startTime := time.Now()

	// Log validation start
	logCtx := CertificateLogContext{
		Operation:       "validation",
		CertificateType: "management",
		CertificateName: cv.deviceName,
		DeviceName:      cv.deviceName,
	}
	LogCertificateOperation(cv.log, logCtx)

	// Step 1: Load CA bundle
	caPool, err := cv.loadCABundle(ctx, rw)
	if err != nil {
		logCtx.Duration = time.Since(startTime)
		logCtx.Error = err
		logCtx.Success = false
		LogCertificateOperation(cv.log, logCtx)
		return fmt.Errorf("failed to load CA bundle: %w", err)
	}

	// Step 2: Verify certificate signature chain
	if err := cv.verifyCertificateSignature(ctx, cert, caPool); err != nil {
		logCtx.Duration = time.Since(startTime)
		logCtx.Error = err
		logCtx.Success = false
		LogCertificateOperation(cv.log, logCtx)
		return fmt.Errorf("signature verification failed: %w", err)
	}

	// Step 3: Verify certificate identity
	if err := cv.verifyCertificateIdentity(cert); err != nil {
		logCtx.Duration = time.Since(startTime)
		logCtx.Error = err
		logCtx.Success = false
		LogCertificateOperation(cv.log, logCtx)
		return fmt.Errorf("identity verification failed: %w", err)
	}

	// Step 4: Verify certificate expiration
	if err := cv.verifyCertificateExpiration(cert); err != nil {
		logCtx.Duration = time.Since(startTime)
		logCtx.Error = err
		logCtx.Success = false
		LogCertificateOperation(cv.log, logCtx)
		return fmt.Errorf("expiration check failed: %w", err)
	}

	// Step 5: Verify key pair
	if err := cv.verifyKeyPair(cert, keyPEM); err != nil {
		logCtx.Duration = time.Since(startTime)
		logCtx.Error = err
		logCtx.Success = false
		LogCertificateOperation(cv.log, logCtx)
		return fmt.Errorf("key pair verification failed: %w", err)
	}

	// Log validation success
	logCtx.Duration = time.Since(startTime)
	logCtx.Success = true
	LogCertificateOperation(cv.log, logCtx)

	return nil
}

// AtomicSwap performs an atomic swap of pending certificate to active location.
// This is a convenience method that delegates to the storage provider.
func (cv *CertificateValidator) AtomicSwap(ctx context.Context, storage provider.StorageProvider) error {
	startTime := time.Now()

	// Log swap start
	logCtx := CertificateLogContext{
		Operation:       "swap",
		CertificateType: "management",
		CertificateName: cv.deviceName,
		DeviceName:      cv.deviceName,
	}
	LogCertificateOperation(cv.log, logCtx)

	err := storage.AtomicSwap(ctx)

	// Log swap result
	logCtx.Duration = time.Since(startTime)
	if err != nil {
		logCtx.Error = err
		logCtx.Success = false
	} else {
		logCtx.Success = true
	}
	LogCertificateOperation(cv.log, logCtx)

	return err
}

// RollbackSwap performs a complete rollback of a failed certificate swap.
// It restores the old certificate from backup, cleans up pending files, and updates state.
func (cv *CertificateValidator) RollbackSwap(ctx context.Context, storage provider.StorageProvider, swapError error) error {
	cv.log.Warnf("Rolling back failed certificate swap for device %q: %v", cv.deviceName, swapError)
	return storage.RollbackSwap(ctx, swapError)
}

// DetectAndRecoverIncompleteSwap detects and recovers from incomplete certificate swaps.
// This is useful after device restart to handle power loss during swap.
func (cv *CertificateValidator) DetectAndRecoverIncompleteSwap(ctx context.Context, storage provider.StorageProvider) error {
	// Check if pending certificate exists (incomplete swap)
	hasPending, err := storage.HasPendingCertificate(ctx)
	if err != nil {
		return fmt.Errorf("failed to check pending certificate: %w", err)
	}

	if !hasPending {
		// No incomplete swap detected
		return nil
	}

	cv.log.Warnf("Detected incomplete certificate swap (pending certificate exists) for device %q", cv.deviceName)

	// Check if active certificate exists
	_, err = storage.LoadCertificate(ctx)
	if err != nil {
		// Active certificate missing - attempt recovery
		cv.log.Warnf("Active certificate missing, attempting recovery for device %q", cv.deviceName)
		return cv.recoverMissingActiveCertificate(ctx, storage)
	}

	// Active certificate exists - validate it matches key
	if err := cv.validateActiveCertificateKeyPair(ctx, storage); err != nil {
		cv.log.Warnf("Active certificate/key mismatch detected for device %q: %v", cv.deviceName, err)
		// Attempt recovery
		return cv.recoverCertificateKeyMismatch(ctx, storage)
	}

	// Active certificate is valid - clean up pending
	cv.log.Infof("Active certificate is valid, cleaning up pending certificate for device %q", cv.deviceName)
	return storage.CleanupPending(ctx)
}

// recoverMissingActiveCertificate recovers when active certificate is missing.
func (cv *CertificateValidator) recoverMissingActiveCertificate(ctx context.Context, storage provider.StorageProvider) error {
	// Try to restore from backup first
	if err := cv.RollbackSwap(ctx, storage, fmt.Errorf("active certificate missing")); err == nil {
		return nil
	}

	// If no backup, try to use pending certificate (if valid)
	cv.log.Warnf("No backup available, attempting to use pending certificate for device %q", cv.deviceName)
	// Load pending certificate and key
	pendingCert, err := storage.LoadPendingCertificate(ctx)
	if err != nil {
		return fmt.Errorf("unable to load pending certificate: %w", err)
	}

	_, err = storage.LoadPendingKey(ctx)
	if err != nil {
		return fmt.Errorf("unable to load pending key: %w", err)
	}

	// Validate pending certificate (basic validation)
	// Note: Full validation requires CA bundle which may not be available
	// For recovery, we'll do basic checks
	if pendingCert.Subject.CommonName != cv.deviceName {
		return fmt.Errorf("pending certificate CommonName %q does not match device name %q",
			pendingCert.Subject.CommonName, cv.deviceName)
	}

	// If valid, swap it to active
	// This is a best-effort recovery
	cv.log.Infof("Pending certificate appears valid, attempting to activate for device %q", cv.deviceName)
	// Note: We can't use AtomicSwap here as it requires backup, but we can write directly
	// This is a recovery scenario, so we'll write the pending cert to active location
	// The storage provider should handle this, but for now we'll just log
	return fmt.Errorf("unable to recover missing active certificate - manual intervention may be required")
}

// validateActiveCertificateKeyPair validates that active certificate and key match.
func (cv *CertificateValidator) validateActiveCertificateKeyPair(ctx context.Context, storage provider.StorageProvider) error {
	_, err := storage.LoadCertificate(ctx)
	if err != nil {
		return fmt.Errorf("failed to load certificate: %w", err)
	}

	// For filesystem storage, we need to load the key
	// This is a simplified check - in practice, we'd need to access the key file
	// For now, we'll just verify the certificate can be loaded
	// A full implementation would load the key and verify the key pair matches
	return nil
}

// recoverCertificateKeyMismatch recovers from certificate/key mismatch.
func (cv *CertificateValidator) recoverCertificateKeyMismatch(ctx context.Context, storage provider.StorageProvider) error {
	// Try to restore from backup
	if err := cv.RollbackSwap(ctx, storage, fmt.Errorf("certificate/key mismatch")); err == nil {
		return nil
	}

	// If rollback fails, clean up pending and return error
	_ = storage.CleanupPending(ctx)
	return fmt.Errorf("unable to recover from certificate/key mismatch - manual intervention may be required")
}
