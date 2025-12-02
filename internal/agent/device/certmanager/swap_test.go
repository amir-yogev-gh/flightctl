package certmanager

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"math/big"
	"path/filepath"
	"testing"
	"time"

	"github.com/flightctl/flightctl/internal/agent/device/fileio"
	fccrypto "github.com/flightctl/flightctl/pkg/crypto"
	"github.com/flightctl/flightctl/pkg/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// createTestCA creates a test CA certificate and key for testing.
func createTestCA(t *testing.T) (*x509.Certificate, *rsa.PrivateKey) {
	caKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	caTemplate := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			CommonName: "Test CA",
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(365 * 24 * time.Hour),
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		BasicConstraintsValid: true,
		IsCA:                  true,
	}

	caDER, err := x509.CreateCertificate(rand.Reader, &caTemplate, &caTemplate, &caKey.PublicKey, caKey)
	require.NoError(t, err)

	caCert, err := x509.ParseCertificate(caDER)
	require.NoError(t, err)

	return caCert, caKey
}

// createTestCertificate creates a test certificate signed by the given CA.
func createTestCertificate(t *testing.T, caCert *x509.Certificate, caKey *rsa.PrivateKey, cn string, notBefore, notAfter time.Time) *x509.Certificate {
	certKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	template := x509.Certificate{
		SerialNumber: big.NewInt(2),
		Subject: pkix.Name{
			CommonName: cn,
		},
		NotBefore:             notBefore,
		NotAfter:              notAfter,
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}

	certDER, err := x509.CreateCertificate(rand.Reader, &template, caCert, &certKey.PublicKey, caKey)
	require.NoError(t, err)

	cert, err := x509.ParseCertificate(certDER)
	require.NoError(t, err)

	return cert
}

// createTestKeyPEM creates a test private key in PEM format.
func createTestKeyPEM(t *testing.T) []byte {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	keyPEM, err := fccrypto.PEMEncodeKey(key)
	require.NoError(t, err)

	return keyPEM
}

// createMismatchedKeyPEM creates a different private key (for testing mismatched key pairs).
func createMismatchedKeyPEM(t *testing.T) []byte {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	keyPEM, err := fccrypto.PEMEncodeKey(key)
	require.NoError(t, err)

	return keyPEM
}

func TestLoadCABundle(t *testing.T) {
	tmpDir := t.TempDir()
	rw := fileio.NewReadWriter(fileio.WithTestRootDir(tmpDir))
	logger := log.NewPrefixLogger("test")

	caCert, _ := createTestCA(t)
	caPEM, err := fccrypto.EncodeCertificatePEM(caCert)
	require.NoError(t, err)

	caBundlePath := filepath.Join(tmpDir, "ca.crt")
	err = rw.WriteFile(caBundlePath, caPEM, 0644)
	require.NoError(t, err)

	validator := NewCertificateValidator(caBundlePath, "test-device", logger)
	ctx := context.Background()

	t.Run("Loads CA bundle correctly", func(t *testing.T) {
		caPool, err := validator.loadCABundle(ctx, rw)
		require.NoError(t, err)
		assert.NotNil(t, caPool)
	})

	t.Run("Handles missing file", func(t *testing.T) {
		missingPath := filepath.Join(tmpDir, "missing.crt")
		validator2 := NewCertificateValidator(missingPath, "test-device", logger)
		caPool, err := validator2.loadCABundle(ctx, rw)
		require.Error(t, err)
		assert.Nil(t, caPool)
		assert.Contains(t, err.Error(), "failed to read CA bundle")
	})

	t.Run("Handles invalid PEM", func(t *testing.T) {
		invalidPath := filepath.Join(tmpDir, "invalid.crt")
		err := rw.WriteFile(invalidPath, []byte("not a valid PEM"), 0644)
		require.NoError(t, err)

		validator2 := NewCertificateValidator(invalidPath, "test-device", logger)
		caPool, err := validator2.loadCABundle(ctx, rw)
		require.Error(t, err)
		assert.Nil(t, caPool)
		assert.Contains(t, err.Error(), "failed to parse CA bundle")
	})
}

func TestVerifyCertificateSignature(t *testing.T) {
	logger := log.NewPrefixLogger("test")
	ctx := context.Background()

	caCert, caKey := createTestCA(t)
	caPool := x509.NewCertPool()
	caPool.AddCert(caCert)

	// Create certificate signed by CA
	validCert := createTestCertificate(t, caCert, caKey, "test-device", time.Now(), time.Now().Add(24*time.Hour))

	// Create certificate signed by different CA
	otherCA, otherCAKey := createTestCA(t)
	otherCAPool := x509.NewCertPool()
	otherCAPool.AddCert(otherCA)
	invalidCert := createTestCertificate(t, otherCA, otherCAKey, "test-device", time.Now(), time.Now().Add(24*time.Hour))

	validator := NewCertificateValidator("", "test-device", logger)

	t.Run("Verifies valid signature", func(t *testing.T) {
		err := validator.verifyCertificateSignature(ctx, validCert, caPool)
		assert.NoError(t, err)
	})

	t.Run("Rejects invalid signature", func(t *testing.T) {
		// Create a certificate with corrupted signature (self-signed but not matching)
		selfSignedCert := createTestCertificate(t, caCert, caKey, "test-device", time.Now(), time.Now().Add(24*time.Hour))
		// Modify the certificate to break signature (this is tricky, so we'll use wrong CA pool)
		err := validator.verifyCertificateSignature(ctx, selfSignedCert, otherCAPool)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "certificate signature verification failed")
	})

	t.Run("Rejects wrong CA", func(t *testing.T) {
		err := validator.verifyCertificateSignature(ctx, invalidCert, caPool)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "certificate signature verification failed")
	})
}

func TestVerifyCertificateIdentity(t *testing.T) {
	logger := log.NewPrefixLogger("test")

	caCert, caKey := createTestCA(t)
	validCert := createTestCertificate(t, caCert, caKey, "test-device", time.Now(), time.Now().Add(24*time.Hour))
	mismatchedCert := createTestCertificate(t, caCert, caKey, "wrong-device", time.Now(), time.Now().Add(24*time.Hour))
	emptyCNCert := createTestCertificate(t, caCert, caKey, "", time.Now(), time.Now().Add(24*time.Hour))

	validator := NewCertificateValidator("", "test-device", logger)

	t.Run("Verifies matching identity", func(t *testing.T) {
		err := validator.verifyCertificateIdentity(validCert)
		assert.NoError(t, err)
	})

	t.Run("Rejects mismatched identity", func(t *testing.T) {
		err := validator.verifyCertificateIdentity(mismatchedCert)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "does not match device name")
	})

	t.Run("Handles empty CommonName", func(t *testing.T) {
		err := validator.verifyCertificateIdentity(emptyCNCert)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "does not match device name")
	})
}

func TestVerifyCertificateExpiration(t *testing.T) {
	logger := log.NewPrefixLogger("test")

	caCert, caKey := createTestCA(t)
	now := time.Now()

	validCert := createTestCertificate(t, caCert, caKey, "test-device", now.Add(-time.Hour), now.Add(24*time.Hour))
	expiredCert := createTestCertificate(t, caCert, caKey, "test-device", now.Add(-48*time.Hour), now.Add(-24*time.Hour))
	notYetValidCert := createTestCertificate(t, caCert, caKey, "test-device", now.Add(24*time.Hour), now.Add(48*time.Hour))
	soonToExpireCert := createTestCertificate(t, caCert, caKey, "test-device", now.Add(-time.Hour), now.Add(3*24*time.Hour)) // 3 days

	validator := NewCertificateValidator("", "test-device", logger)

	t.Run("Verifies valid certificate", func(t *testing.T) {
		err := validator.verifyCertificateExpiration(validCert)
		assert.NoError(t, err)
	})

	t.Run("Rejects expired certificate", func(t *testing.T) {
		err := validator.verifyCertificateExpiration(expiredCert)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "certificate is expired")
	})

	t.Run("Rejects not-yet-valid certificate", func(t *testing.T) {
		err := validator.verifyCertificateExpiration(notYetValidCert)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "certificate is not yet valid")
	})

	t.Run("Warns on soon-to-expire certificate", func(t *testing.T) {
		// This test verifies the warning is logged, but validation still succeeds
		err := validator.verifyCertificateExpiration(soonToExpireCert)
		// Should not error, but warning should be logged
		// We can't easily test the warning without a mock logger, so we just verify no error
		assert.NoError(t, err)
	})
}

func TestVerifyKeyPair(t *testing.T) {
	logger := log.NewPrefixLogger("test")

	caCert, caKey := createTestCA(t)

	// Create matching key pair
	certKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	// Create certificate with this key
	certDER, err := x509.CreateCertificate(rand.Reader, &x509.Certificate{
		SerialNumber: big.NewInt(3),
		Subject: pkix.Name{
			CommonName: "test-device",
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(24 * time.Hour),
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}, caCert, &certKey.PublicKey, caKey)
	require.NoError(t, err)
	matchingCert, err := x509.ParseCertificate(certDER)
	require.NoError(t, err)

	matchingKeyPEM, err := fccrypto.PEMEncodeKey(certKey)
	require.NoError(t, err)

	// Create mismatched key
	mismatchedKeyPEM := createMismatchedKeyPEM(t)

	// Create invalid key PEM
	invalidKeyPEM := []byte("-----BEGIN INVALID KEY-----\ninvalid\n-----END INVALID KEY-----\n")

	validator := NewCertificateValidator("", "test-device", logger)

	t.Run("Verifies matching key pair", func(t *testing.T) {
		err := validator.verifyKeyPair(matchingCert, matchingKeyPEM)
		assert.NoError(t, err)
	})

	t.Run("Rejects mismatched key pair", func(t *testing.T) {
		err := validator.verifyKeyPair(matchingCert, mismatchedKeyPEM)
		require.Error(t, err)
		// Error can be from TLS key pair creation or direct comparison
		errMsg := err.Error()
		assert.True(t,
			containsString(errMsg, "certificate and private key do not match") ||
				containsString(errMsg, "failed to create TLS certificate from key pair") ||
				containsString(errMsg, "private key does not match"),
			"Error should mention key pair mismatch: %s", errMsg)
	})

	t.Run("Rejects invalid key", func(t *testing.T) {
		err := validator.verifyKeyPair(matchingCert, invalidKeyPEM)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to parse private key")
	})

	t.Run("Creates valid TLS certificate", func(t *testing.T) {
		// This is implicitly tested by verifyKeyPair, which uses tls.X509KeyPair
		// If the key pair doesn't match, tls.X509KeyPair will fail
		err := validator.verifyKeyPair(matchingCert, matchingKeyPEM)
		assert.NoError(t, err)
	})
}

func TestValidatePendingCertificate(t *testing.T) {
	tmpDir := t.TempDir()
	rw := fileio.NewReadWriter(fileio.WithTestRootDir(tmpDir))
	logger := log.NewPrefixLogger("test")
	ctx := context.Background()

	// Create CA and write to file
	caCert, caKey := createTestCA(t)
	caPEM, err := fccrypto.EncodeCertificatePEM(caCert)
	require.NoError(t, err)

	caBundlePath := filepath.Join(tmpDir, "ca.crt")
	err = rw.WriteFile(caBundlePath, caPEM, 0644)
	require.NoError(t, err)

	// Create valid certificate and key
	certKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	certDER, err := x509.CreateCertificate(rand.Reader, &x509.Certificate{
		SerialNumber: big.NewInt(4),
		Subject: pkix.Name{
			CommonName: "test-device",
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(24 * time.Hour),
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}, caCert, &certKey.PublicKey, caKey)
	require.NoError(t, err)
	validCert, err := x509.ParseCertificate(certDER)
	require.NoError(t, err)

	validKeyPEM, err := fccrypto.PEMEncodeKey(certKey)
	require.NoError(t, err)

	validator := NewCertificateValidator(caBundlePath, "test-device", logger)

	t.Run("Validates valid certificate", func(t *testing.T) {
		err := validator.ValidatePendingCertificate(ctx, validCert, validKeyPEM, rw)
		assert.NoError(t, err)
	})

	t.Run("Rejects invalid signature", func(t *testing.T) {
		// Create certificate signed by different CA
		otherCA, otherCAKey := createTestCA(t)
		invalidCert := createTestCertificate(t, otherCA, otherCAKey, "test-device", time.Now(), time.Now().Add(24*time.Hour))
		invalidKeyPEM := createTestKeyPEM(t)

		err := validator.ValidatePendingCertificate(ctx, invalidCert, invalidKeyPEM, rw)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "signature verification failed")
	})

	t.Run("Rejects wrong identity", func(t *testing.T) {
		wrongCert := createTestCertificate(t, caCert, caKey, "wrong-device", time.Now(), time.Now().Add(24*time.Hour))
		wrongKeyPEM := createTestKeyPEM(t)

		err := validator.ValidatePendingCertificate(ctx, wrongCert, wrongKeyPEM, rw)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "identity verification failed")
	})

	t.Run("Rejects expired certificate", func(t *testing.T) {
		expiredCert := createTestCertificate(t, caCert, caKey, "test-device", time.Now().Add(-48*time.Hour), time.Now().Add(-24*time.Hour))
		expiredKeyPEM := createTestKeyPEM(t)

		err := validator.ValidatePendingCertificate(ctx, expiredCert, expiredKeyPEM, rw)
		require.Error(t, err)
		// Expired certificates fail at signature verification (which checks expiration) or expiration check
		// x509.Verify includes expiration check, so it may fail there first
		errMsg := err.Error()
		assert.True(t,
			containsString(errMsg, "expiration check failed") ||
				containsString(errMsg, "signature verification failed") ||
				containsString(errMsg, "certificate has expired") ||
				containsString(errMsg, "expired"),
			"Error should mention expiration: %s", errMsg)
	})

	t.Run("Rejects mismatched key pair", func(t *testing.T) {
		mismatchedKeyPEM := createMismatchedKeyPEM(t)
		err := validator.ValidatePendingCertificate(ctx, validCert, mismatchedKeyPEM, rw)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "key pair verification failed")
	})

	t.Run("Rejects when CA bundle is missing", func(t *testing.T) {
		missingCAValidator := NewCertificateValidator(filepath.Join(tmpDir, "missing.crt"), "test-device", logger)
		err := missingCAValidator.ValidatePendingCertificate(ctx, validCert, validKeyPEM, rw)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to load CA bundle")
	})
}

// containsString is a helper to check if a string contains a substring
func containsString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
