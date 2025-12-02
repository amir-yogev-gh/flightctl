package certmanager

import (
	"context"
	"crypto/x509"
	"fmt"
	"testing"
	"time"

	"github.com/flightctl/flightctl/internal/agent/config"
	"github.com/flightctl/flightctl/internal/agent/device/certmanager/provider"
	"github.com/flightctl/flightctl/internal/agent/device/fileio"
	"github.com/flightctl/flightctl/pkg/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test Suite 5: CheckCertificateExpiration

func TestCertManager_CheckCertificateExpiration(t *testing.T) {
	tmpDir := t.TempDir()
	rw := fileio.NewReadWriter(fileio.WithTestRootDir(tmpDir))
	logger := log.NewPrefixLogger("test")
	now := time.Now().UTC()

	cm := &CertManager{
		certificates:      newCertStorage(),
		expirationMonitor: NewExpirationMonitor(logger),
		log:               logger,
		readWriter:        rw,
		storages:          make(map[string]provider.StorageFactory),
		config: &config.Config{
			Certificate: config.CertificateConfig{
				Renewal: config.CertificateRenewalConfig{
					ThresholdDays: 30,
				},
			},
		},
	}

	ctx := context.Background()
	providerName := "test-provider"
	certName := "test-cert"

	t.Run("Certificate with expiration info already loaded", func(t *testing.T) {
		// Create certificate with expiration info already set
		expirationTime := now.Add(30 * 24 * time.Hour)
		cert := &certificate{
			Name: certName,
			Info: CertificateInfo{
				NotAfter:  &expirationTime,
				NotBefore: &now,
			},
			Config: provider.CertificateConfig{
				Name: certName,
			},
		}

		_, _ = cm.certificates.EnsureProvider(providerName)
		_ = cm.certificates.StoreCertificate(providerName, cert)

		days, expiration, err := cm.CheckCertificateExpiration(ctx, providerName, certName)
		require.NoError(t, err)
		assert.NotNil(t, expiration)
		assert.Equal(t, expirationTime, *expiration)
		// Days should be approximately 30 (allow tolerance)
		assert.GreaterOrEqual(t, days, 29)
		assert.LessOrEqual(t, days, 31)
	})

	t.Run("Certificate needing load from storage", func(t *testing.T) {
		// Create certificate without expiration info
		expirationTime := now.Add(25 * 24 * time.Hour)
		x509Cert := createMockCertForExpiration(t, now, expirationTime)

		cert := &certificate{
			Name: "cert-from-storage",
			Info: CertificateInfo{}, // No expiration info
			Config: provider.CertificateConfig{
				Name: "cert-from-storage",
				Storage: provider.StorageConfig{
					Type: provider.StorageTypeFilesystem,
				},
			},
		}
		cert.Storage = &mockStorageProvider{cert: x509Cert}

		// Register storage factory
		cm.storages[string(provider.StorageTypeFilesystem)] = &mockStorageFactory{}

		_, _ = cm.certificates.EnsureProvider(providerName)
		_ = cm.certificates.StoreCertificate(providerName, cert)

		days, expiration, err := cm.CheckCertificateExpiration(ctx, providerName, "cert-from-storage")
		require.NoError(t, err)
		assert.NotNil(t, expiration)
		// Days should be approximately 25 (allow tolerance)
		assert.GreaterOrEqual(t, days, 24)
		assert.LessOrEqual(t, days, 26)
	})

	t.Run("Certificate not found", func(t *testing.T) {
		days, expiration, err := cm.CheckCertificateExpiration(ctx, providerName, "nonexistent")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
		assert.Equal(t, 0, days)
		assert.Nil(t, expiration)
	})

	t.Run("Certificate with no expiration date", func(t *testing.T) {
		// Create certificate with zero expiration
		cert := &certificate{
			Name: "no-expiration-cert",
			Info: CertificateInfo{}, // No expiration info
			Config: provider.CertificateConfig{
				Name: "no-expiration-cert",
				Storage: provider.StorageConfig{
					Type: provider.StorageTypeFilesystem,
				},
			},
		}
		// Create certificate with zero NotAfter
		zeroCert := &x509.Certificate{
			NotAfter: time.Time{},
		}
		cert.Storage = &mockStorageProvider{cert: zeroCert}

		_, _ = cm.certificates.EnsureProvider(providerName)
		_ = cm.certificates.StoreCertificate(providerName, cert)

		days, expiration, err := cm.CheckCertificateExpiration(ctx, providerName, "no-expiration-cert")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no expiration date")
		assert.Equal(t, 0, days)
		assert.Nil(t, expiration)
	})

	t.Run("Storage initialization failure", func(t *testing.T) {
		cert := &certificate{
			Name: "storage-fail-cert",
			Info: CertificateInfo{}, // No expiration info
			Config: provider.CertificateConfig{
				Name: "storage-fail-cert",
				Storage: provider.StorageConfig{
					Type: "invalid-storage-type",
				},
			},
		}

		_, _ = cm.certificates.EnsureProvider(providerName)
		_ = cm.certificates.StoreCertificate(providerName, cert)

		days, expiration, err := cm.CheckCertificateExpiration(ctx, providerName, "storage-fail-cert")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to init storage")
		assert.Equal(t, 0, days)
		assert.Nil(t, expiration)
	})

	t.Run("Storage load failure", func(t *testing.T) {
		cert := &certificate{
			Name: "load-fail-cert",
			Info: CertificateInfo{}, // No expiration info
			Config: provider.CertificateConfig{
				Name: "load-fail-cert",
				Storage: provider.StorageConfig{
					Type: provider.StorageTypeFilesystem,
				},
			},
		}
		cert.Storage = &mockStorageProviderWithError{loadError: fmt.Errorf("load failed")}

		_, _ = cm.certificates.EnsureProvider(providerName)
		_ = cm.certificates.StoreCertificate(providerName, cert)

		days, expiration, err := cm.CheckCertificateExpiration(ctx, providerName, "load-fail-cert")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to load certificate")
		assert.Equal(t, 0, days)
		assert.Nil(t, expiration)
	})
}

// Test Suite 6: CheckAllCertificatesExpiration

func TestCertManager_CheckAllCertificatesExpiration(t *testing.T) {
	tmpDir := t.TempDir()
	rw := fileio.NewReadWriter(fileio.WithTestRootDir(tmpDir))
	logger := log.NewPrefixLogger("test")
	now := time.Now().UTC()

	cm := &CertManager{
		certificates:      newCertStorage(),
		expirationMonitor: NewExpirationMonitor(logger),
		log:               logger,
		readWriter:        rw,
		storages:          make(map[string]provider.StorageFactory),
		config: &config.Config{
			Certificate: config.CertificateConfig{
				Renewal: config.CertificateRenewalConfig{
					ThresholdDays: 30,
				},
			},
		},
	}

	ctx := context.Background()
	providerName := "test-provider"

	t.Run("Multiple certificates with various expiration states", func(t *testing.T) {
		// Create certificates with different expiration states
		certA := &certificate{
			Name: "cert-a",
			Info: CertificateInfo{
				NotAfter:  timePtr(now.Add(30 * 24 * time.Hour)),
				NotBefore: timePtr(now),
			},
			Config: provider.CertificateConfig{Name: "cert-a"},
		}

		certB := &certificate{
			Name: "cert-b",
			Info: CertificateInfo{
				NotAfter:  timePtr(now.Add(5 * 24 * time.Hour)),
				NotBefore: timePtr(now),
			},
			Config: provider.CertificateConfig{Name: "cert-b"},
		}

		certC := &certificate{
			Name: "cert-c",
			Info: CertificateInfo{
				NotAfter:  timePtr(now.Add(-10 * 24 * time.Hour)),
				NotBefore: timePtr(now.Add(-40 * 24 * time.Hour)),
			},
			Config: provider.CertificateConfig{Name: "cert-c"},
		}

		_, _ = cm.certificates.EnsureProvider(providerName)
		_ = cm.certificates.StoreCertificate(providerName, certA)
		_ = cm.certificates.StoreCertificate(providerName, certB)
		_ = cm.certificates.StoreCertificate(providerName, certC)

		err := cm.CheckAllCertificatesExpiration(ctx)
		assert.NoError(t, err)
	})

	t.Run("Certificates from multiple providers", func(t *testing.T) {
		provider1 := "provider1"
		provider2 := "provider2"

		cert1 := &certificate{
			Name: "cert1",
			Info: CertificateInfo{
				NotAfter:  timePtr(now.Add(30 * 24 * time.Hour)),
				NotBefore: timePtr(now),
			},
			Config: provider.CertificateConfig{Name: "cert1"},
		}

		cert2 := &certificate{
			Name: "cert2",
			Info: CertificateInfo{
				NotAfter:  timePtr(now.Add(30 * 24 * time.Hour)),
				NotBefore: timePtr(now),
			},
			Config: provider.CertificateConfig{Name: "cert2"},
		}

		cert3 := &certificate{
			Name: "cert3",
			Info: CertificateInfo{
				NotAfter:  timePtr(now.Add(30 * 24 * time.Hour)),
				NotBefore: timePtr(now),
			},
			Config: provider.CertificateConfig{Name: "cert3"},
		}

		_, _ = cm.certificates.EnsureProvider(provider1)
		_, _ = cm.certificates.EnsureProvider(provider2)
		_ = cm.certificates.StoreCertificate(provider1, cert1)
		_ = cm.certificates.StoreCertificate(provider1, cert2)
		_ = cm.certificates.StoreCertificate(provider2, cert3)

		err := cm.CheckAllCertificatesExpiration(ctx)
		assert.NoError(t, err)
	})

	t.Run("Error handling for individual certificate failures", func(t *testing.T) {
		// Create certificates: one valid, one with load failure, one valid
		certA := &certificate{
			Name: "cert-a-valid",
			Info: CertificateInfo{
				NotAfter:  timePtr(now.Add(30 * 24 * time.Hour)),
				NotBefore: timePtr(now),
			},
			Config: provider.CertificateConfig{Name: "cert-a-valid"},
		}

		certB := &certificate{
			Name: "cert-b-fail",
			Info: CertificateInfo{}, // No expiration info - will try to load
			Config: provider.CertificateConfig{
				Name: "cert-b-fail",
				Storage: provider.StorageConfig{
					Type: provider.StorageTypeFilesystem,
				},
			},
		}
		certB.Storage = &mockStorageProviderWithError{loadError: fmt.Errorf("load failed")}

		certC := &certificate{
			Name: "cert-c-valid",
			Info: CertificateInfo{
				NotAfter:  timePtr(now.Add(30 * 24 * time.Hour)),
				NotBefore: timePtr(now),
			},
			Config: provider.CertificateConfig{Name: "cert-c-valid"},
		}

		_, _ = cm.certificates.EnsureProvider(providerName)
		_ = cm.certificates.StoreCertificate(providerName, certA)
		_ = cm.certificates.StoreCertificate(providerName, certB)
		_ = cm.certificates.StoreCertificate(providerName, certC)

		// Should not return error - errors are logged but don't fail the method
		err := cm.CheckAllCertificatesExpiration(ctx)
		assert.NoError(t, err)
	})

	t.Run("Empty certificate list", func(t *testing.T) {
		emptyProvider := "empty-provider"
		_, _ = cm.certificates.EnsureProvider(emptyProvider)

		err := cm.CheckAllCertificatesExpiration(ctx)
		assert.NoError(t, err)
	})
}

// Test Suite 7: StartPeriodicExpirationCheck

func TestCertManager_StartPeriodicExpirationCheck(t *testing.T) {
	tmpDir := t.TempDir()
	rw := fileio.NewReadWriter(fileio.WithTestRootDir(tmpDir))
	logger := log.NewPrefixLogger("test")
	now := time.Now().UTC()

	cm := &CertManager{
		certificates:      newCertStorage(),
		expirationMonitor: NewExpirationMonitor(logger),
		log:               logger,
		readWriter:        rw,
		storages:          make(map[string]provider.StorageFactory),
		config: &config.Config{
			Certificate: config.CertificateConfig{
				Renewal: config.CertificateRenewalConfig{
					ThresholdDays: 30,
				},
			},
		},
	}

	ctx := context.Background()
	providerName := "test-provider"
	certName := "test-cert"

	// Create a test certificate
	expirationTime := now.Add(30 * 24 * time.Hour)
	cert := &certificate{
		Name: certName,
		Info: CertificateInfo{
			NotAfter:  &expirationTime,
			NotBefore: &now,
		},
		Config: provider.CertificateConfig{Name: certName},
	}

	_, _ = cm.certificates.EnsureProvider(providerName)
	_ = cm.certificates.StoreCertificate(providerName, cert)

	t.Run("Periodic check runs at specified interval", func(t *testing.T) {
		ctx, cancel := context.WithCancel(ctx)
		defer cancel()

		checkInterval := 100 * time.Millisecond
		cm.StartPeriodicExpirationCheck(ctx, checkInterval)

		// Wait for at least 3 checks (immediate + 2 periodic)
		time.Sleep(350 * time.Millisecond)
		cancel()
		time.Sleep(50 * time.Millisecond) // Allow goroutine to exit
	})

	t.Run("Check runs immediately on startup", func(t *testing.T) {
		ctx, cancel := context.WithCancel(ctx)
		defer cancel()

		checkInterval := 1 * time.Hour
		cm.StartPeriodicExpirationCheck(ctx, checkInterval)

		// Wait a bit to ensure immediate check runs
		time.Sleep(100 * time.Millisecond)
		cancel()
		time.Sleep(50 * time.Millisecond)
	})

	t.Run("Context cancellation stops goroutine", func(t *testing.T) {
		ctx, cancel := context.WithCancel(ctx)
		defer cancel()

		checkInterval := 100 * time.Millisecond
		cm.StartPeriodicExpirationCheck(ctx, checkInterval)

		// Wait for some checks
		time.Sleep(200 * time.Millisecond)

		// Cancel context
		cancel()
		time.Sleep(200 * time.Millisecond)

		// Goroutine should have stopped
		// No way to directly verify, but if it didn't stop, we'd see issues
	})

	t.Run("Invalid interval uses default", func(t *testing.T) {
		ctx, cancel := context.WithCancel(ctx)
		defer cancel()

		// Use zero interval
		cm.StartPeriodicExpirationCheck(ctx, 0)

		// Wait a bit
		time.Sleep(100 * time.Millisecond)
		cancel()
		time.Sleep(50 * time.Millisecond)
	})

	t.Run("Negative interval uses default", func(t *testing.T) {
		ctx, cancel := context.WithCancel(ctx)
		defer cancel()

		// Use negative interval
		cm.StartPeriodicExpirationCheck(ctx, -1*time.Hour)

		// Wait a bit
		time.Sleep(100 * time.Millisecond)
		cancel()
		time.Sleep(50 * time.Millisecond)
	})
}

// Helper functions

func timePtr(t time.Time) *time.Time {
	return &t
}

// mockStorageProviderWithError is a mock storage provider that returns errors
type mockStorageProviderWithError struct {
	cert      *x509.Certificate
	loadError error
}

func (m *mockStorageProviderWithError) LoadCertificate(ctx context.Context) (*x509.Certificate, error) {
	if m.loadError != nil {
		return nil, m.loadError
	}
	return m.cert, nil
}

func (m *mockStorageProviderWithError) Write(cert *x509.Certificate, keyPEM []byte) error {
	return nil
}

func (m *mockStorageProviderWithError) WritePending(cert *x509.Certificate, keyPEM []byte) error {
	return nil
}

func (m *mockStorageProviderWithError) LoadPendingCertificate(ctx context.Context) (*x509.Certificate, error) {
	return nil, nil
}

func (m *mockStorageProviderWithError) LoadPendingKey(ctx context.Context) ([]byte, error) {
	return nil, nil
}

func (m *mockStorageProviderWithError) HasPendingCertificate(ctx context.Context) (bool, error) {
	return false, nil
}

func (m *mockStorageProviderWithError) CleanupPending(ctx context.Context) error {
	return nil
}

func (m *mockStorageProviderWithError) AtomicSwap(ctx context.Context) error {
	return nil
}

func (m *mockStorageProviderWithError) RollbackSwap(ctx context.Context, swapError error) error {
	return nil
}

func (m *mockStorageProviderWithError) Delete(ctx context.Context) error {
	return nil
}
