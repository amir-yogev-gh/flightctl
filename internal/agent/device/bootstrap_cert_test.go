package device

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

	agent_config "github.com/flightctl/flightctl/internal/agent/config"
	"github.com/flightctl/flightctl/internal/agent/device/fileio"
	fccrypto "github.com/flightctl/flightctl/pkg/crypto"
	"github.com/flightctl/flightctl/pkg/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// createTestCertificate creates a test certificate for bootstrap testing.
func createTestCertificate(t *testing.T, cn string, notBefore, notAfter time.Time) (*x509.Certificate, []byte) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			CommonName: cn,
		},
		NotBefore:             notBefore,
		NotAfter:              notAfter,
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
		BasicConstraintsValid: true,
	}

	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &key.PublicKey, key)
	require.NoError(t, err)

	cert, err := x509.ParseCertificate(certDER)
	require.NoError(t, err)

	keyPEM, err := fccrypto.PEMEncodeKey(key)
	require.NoError(t, err)

	return cert, keyPEM
}

func TestGetBootstrapCertificate(t *testing.T) {
	tmpDir := t.TempDir()
	rw := fileio.NewReadWriter(fileio.WithTestRootDir(tmpDir))
	logger := log.NewPrefixLogger("test")
	ctx := context.Background()

	handler := NewBootstrapCertificateHandler(tmpDir, rw, logger)

	t.Run("Loads certificate successfully", func(t *testing.T) {
		// Create test certificate
		now := time.Now()
		cert, keyPEM := createTestCertificate(t, "test-device", now.Add(-time.Hour), now.Add(24*time.Hour))

		// Write certificate and key
		certPEM, err := fccrypto.EncodeCertificatePEM(cert)
		require.NoError(t, err)

		certPath := filepath.Join(tmpDir, agent_config.DefaultCertsDirName, agent_config.EnrollmentCertFile)
		keyPath := filepath.Join(tmpDir, agent_config.DefaultCertsDirName, agent_config.EnrollmentKeyFile)

		err = rw.MkdirAll(filepath.Dir(certPath), 0700)
		require.NoError(t, err)

		err = rw.WriteFile(certPath, certPEM, 0644)
		require.NoError(t, err)

		err = rw.WriteFile(keyPath, keyPEM, 0600)
		require.NoError(t, err)

		// Load certificate
		loadedCert, loadedKey, err := handler.GetBootstrapCertificate(ctx)
		require.NoError(t, err)
		assert.NotNil(t, loadedCert)
		assert.NotNil(t, loadedKey)
		assert.Equal(t, cert.Subject.CommonName, loadedCert.Subject.CommonName)
	})

	t.Run("Returns error when certificate missing", func(t *testing.T) {
		// Don't create certificate files
		cert, key, err := handler.GetBootstrapCertificate(ctx)
		assert.Error(t, err)
		assert.Nil(t, cert)
		assert.Nil(t, key)
		assert.Contains(t, err.Error(), "bootstrap certificate not found")
	})

	t.Run("Returns error when key missing", func(t *testing.T) {
		// Create certificate but not key
		now := time.Now()
		cert, _ := createTestCertificate(t, "test-device", now.Add(-time.Hour), now.Add(24*time.Hour))

		certPEM, err := fccrypto.EncodeCertificatePEM(cert)
		require.NoError(t, err)

		certPath := filepath.Join(tmpDir, agent_config.DefaultCertsDirName, agent_config.EnrollmentCertFile)
		err = rw.MkdirAll(filepath.Dir(certPath), 0700)
		require.NoError(t, err)

		err = rw.WriteFile(certPath, certPEM, 0644)
		require.NoError(t, err)

		// Try to load
		loadedCert, loadedKey, err := handler.GetBootstrapCertificate(ctx)
		assert.Error(t, err)
		assert.Nil(t, loadedCert)
		assert.Nil(t, loadedKey)
		assert.Contains(t, err.Error(), "failed to read bootstrap key")
	})

	t.Run("Returns error when certificate invalid PEM", func(t *testing.T) {
		certPath := filepath.Join(tmpDir, agent_config.DefaultCertsDirName, agent_config.EnrollmentCertFile)
		keyPath := filepath.Join(tmpDir, agent_config.DefaultCertsDirName, agent_config.EnrollmentKeyFile)

		err := rw.MkdirAll(filepath.Dir(certPath), 0700)
		require.NoError(t, err)

		// Write invalid PEM
		err = rw.WriteFile(certPath, []byte("invalid PEM"), 0644)
		require.NoError(t, err)

		// Write valid key
		_, keyPEM := createTestCertificate(t, "test-device", time.Now(), time.Now().Add(24*time.Hour))
		err = rw.WriteFile(keyPath, keyPEM, 0600)
		require.NoError(t, err)

		// Try to load
		cert, key, err := handler.GetBootstrapCertificate(ctx)
		assert.Error(t, err)
		assert.Nil(t, cert)
		assert.Nil(t, key)
		assert.Contains(t, err.Error(), "failed to parse bootstrap certificate")
	})
}

func TestValidateBootstrapCertificate(t *testing.T) {
	tmpDir := t.TempDir()
	rw := fileio.NewReadWriter(fileio.WithTestRootDir(tmpDir))
	logger := log.NewPrefixLogger("test")

	handler := NewBootstrapCertificateHandler(tmpDir, rw, logger)
	now := time.Now()

	t.Run("Validates valid certificate", func(t *testing.T) {
		cert, _ := createTestCertificate(t, "test-device", now.Add(-time.Hour), now.Add(24*time.Hour))
		err := handler.ValidateBootstrapCertificate(cert)
		assert.NoError(t, err)
	})

	t.Run("Rejects expired certificate", func(t *testing.T) {
		expiredCert, _ := createTestCertificate(t, "test-device", now.Add(-48*time.Hour), now.Add(-24*time.Hour))
		err := handler.ValidateBootstrapCertificate(expiredCert)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "bootstrap certificate is expired")
	})

	t.Run("Rejects not-yet-valid certificate", func(t *testing.T) {
		futureCert, _ := createTestCertificate(t, "test-device", now.Add(24*time.Hour), now.Add(48*time.Hour))
		err := handler.ValidateBootstrapCertificate(futureCert)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "bootstrap certificate is not yet valid")
	})
}

func TestGetCertificateForAuth(t *testing.T) {
	tmpDir := t.TempDir()
	rw := fileio.NewReadWriter(fileio.WithTestRootDir(tmpDir))
	logger := log.NewPrefixLogger("test")
	ctx := context.Background()

	handler := NewBootstrapCertificateHandler(tmpDir, rw, logger)
	now := time.Now()

	// Setup bootstrap certificate
	bootstrapCert, bootstrapKey := createTestCertificate(t, "bootstrap-device", now.Add(-time.Hour), now.Add(7*24*time.Hour))
	bootstrapCertPEM, err := fccrypto.EncodeCertificatePEM(bootstrapCert)
	require.NoError(t, err)

	bootstrapCertPath := filepath.Join(tmpDir, agent_config.DefaultCertsDirName, agent_config.EnrollmentCertFile)
	bootstrapKeyPath := filepath.Join(tmpDir, agent_config.DefaultCertsDirName, agent_config.EnrollmentKeyFile)

	err = rw.MkdirAll(filepath.Dir(bootstrapCertPath), 0700)
	require.NoError(t, err)

	err = rw.WriteFile(bootstrapCertPath, bootstrapCertPEM, 0644)
	require.NoError(t, err)

	err = rw.WriteFile(bootstrapKeyPath, bootstrapKey, 0600)
	require.NoError(t, err)

	t.Run("Uses management certificate when valid", func(t *testing.T) {
		// Create valid management certificate
		mgmtCert, mgmtKey := createTestCertificate(t, "test-device", now.Add(-time.Hour), now.Add(24*time.Hour))
		mgmtCertPEM, err := fccrypto.EncodeCertificatePEM(mgmtCert)
		require.NoError(t, err)

		mgmtCertPath := filepath.Join(tmpDir, "management.crt")
		mgmtKeyPath := filepath.Join(tmpDir, "management.key")

		err = rw.WriteFile(mgmtCertPath, mgmtCertPEM, 0644)
		require.NoError(t, err)

		err = rw.WriteFile(mgmtKeyPath, mgmtKey, 0600)
		require.NoError(t, err)

		// Get certificate for auth
		tlsCert, err := handler.GetCertificateForAuth(ctx, mgmtCertPath, mgmtKeyPath)
		require.NoError(t, err)
		assert.NotNil(t, tlsCert)
		assert.Equal(t, 1, len(tlsCert.Certificate))
	})

	t.Run("Falls back to bootstrap when management expired", func(t *testing.T) {
		// Create expired management certificate
		expiredMgmtCert, expiredMgmtKey := createTestCertificate(t, "test-device", now.Add(-48*time.Hour), now.Add(-24*time.Hour))
		expiredMgmtCertPEM, err := fccrypto.EncodeCertificatePEM(expiredMgmtCert)
		require.NoError(t, err)

		mgmtCertPath := filepath.Join(tmpDir, "expired-management.crt")
		mgmtKeyPath := filepath.Join(tmpDir, "expired-management.key")

		err = rw.WriteFile(mgmtCertPath, expiredMgmtCertPEM, 0644)
		require.NoError(t, err)

		err = rw.WriteFile(mgmtKeyPath, expiredMgmtKey, 0600)
		require.NoError(t, err)

		// Get certificate for auth - should fall back to bootstrap
		tlsCert, err := handler.GetCertificateForAuth(ctx, mgmtCertPath, mgmtKeyPath)
		require.NoError(t, err)
		assert.NotNil(t, tlsCert)
		// Should use bootstrap certificate
		assert.Equal(t, 1, len(tlsCert.Certificate))
	})

	t.Run("Falls back to bootstrap when management missing", func(t *testing.T) {
		// Use non-existent management paths
		mgmtCertPath := filepath.Join(tmpDir, "missing-management.crt")
		mgmtKeyPath := filepath.Join(tmpDir, "missing-management.key")

		// Get certificate for auth - should fall back to bootstrap
		tlsCert, err := handler.GetCertificateForAuth(ctx, mgmtCertPath, mgmtKeyPath)
		require.NoError(t, err)
		assert.NotNil(t, tlsCert)
		// Should use bootstrap certificate
		assert.Equal(t, 1, len(tlsCert.Certificate))
	})

	t.Run("Returns error when bootstrap also missing", func(t *testing.T) {
		// Create handler with non-existent bootstrap
		emptyDir := t.TempDir()
		emptyHandler := NewBootstrapCertificateHandler(emptyDir, rw, logger)

		mgmtCertPath := filepath.Join(tmpDir, "missing-management.crt")
		mgmtKeyPath := filepath.Join(tmpDir, "missing-management.key")

		// Get certificate for auth - should fail
		tlsCert, err := emptyHandler.GetCertificateForAuth(ctx, mgmtCertPath, mgmtKeyPath)
		assert.Error(t, err)
		assert.Nil(t, tlsCert)
		assert.Contains(t, err.Error(), "failed to get bootstrap certificate")
	})
}

func TestHasValidBootstrapCertificate(t *testing.T) {
	tmpDir := t.TempDir()
	rw := fileio.NewReadWriter(fileio.WithTestRootDir(tmpDir))
	logger := log.NewPrefixLogger("test")
	ctx := context.Background()

	handler := NewBootstrapCertificateHandler(tmpDir, rw, logger)
	now := time.Now()

	t.Run("Returns true for valid certificate", func(t *testing.T) {
		// Create valid certificate
		cert, key := createTestCertificate(t, "test-device", now.Add(-time.Hour), now.Add(24*time.Hour))
		certPEM, err := fccrypto.EncodeCertificatePEM(cert)
		require.NoError(t, err)

		certPath := filepath.Join(tmpDir, agent_config.DefaultCertsDirName, agent_config.EnrollmentCertFile)
		keyPath := filepath.Join(tmpDir, agent_config.DefaultCertsDirName, agent_config.EnrollmentKeyFile)

		err = rw.MkdirAll(filepath.Dir(certPath), 0700)
		require.NoError(t, err)

		err = rw.WriteFile(certPath, certPEM, 0644)
		require.NoError(t, err)

		err = rw.WriteFile(keyPath, key, 0600)
		require.NoError(t, err)

		hasValid, err := handler.HasValidBootstrapCertificate(ctx)
		require.NoError(t, err)
		assert.True(t, hasValid)
	})

	t.Run("Returns false for expired certificate", func(t *testing.T) {
		// Create expired certificate
		expiredCert, expiredKey := createTestCertificate(t, "test-device", now.Add(-48*time.Hour), now.Add(-24*time.Hour))
		expiredCertPEM, err := fccrypto.EncodeCertificatePEM(expiredCert)
		require.NoError(t, err)

		certPath := filepath.Join(tmpDir, agent_config.DefaultCertsDirName, agent_config.EnrollmentCertFile)
		keyPath := filepath.Join(tmpDir, agent_config.DefaultCertsDirName, agent_config.EnrollmentKeyFile)

		err = rw.WriteFile(certPath, expiredCertPEM, 0644)
		require.NoError(t, err)

		err = rw.WriteFile(keyPath, expiredKey, 0600)
		require.NoError(t, err)

		hasValid, err := handler.HasValidBootstrapCertificate(ctx)
		require.NoError(t, err)
		assert.False(t, hasValid)
	})

	t.Run("Returns false when certificate missing", func(t *testing.T) {
		// Don't create certificate
		hasValid, err := handler.HasValidBootstrapCertificate(ctx)
		require.NoError(t, err)
		assert.False(t, hasValid)
	})
}
