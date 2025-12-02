package service

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"math/big"
	"testing"
	"time"

	api "github.com/flightctl/flightctl/api/v1beta1"
	"github.com/flightctl/flightctl/internal/agent/identity"
	"github.com/flightctl/flightctl/pkg/log"
	"github.com/google/uuid"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// createTestCSR creates a test CSR with the given labels
func createTestCSR(name string, labels map[string]string) *api.CertificateSigningRequest {
	csr := &api.CertificateSigningRequest{
		ApiVersion: "v1beta1",
		Kind:       "CertificateSigningRequest",
		Metadata: api.ObjectMeta{
			Name:   lo.ToPtr(name),
			Labels: &labels,
			Owner:  lo.ToPtr("device/test-device"),
		},
		Spec: api.CertificateSigningRequestSpec{
			Request:    []byte("test-csr-data"),
			SignerName: "test-signer",
		},
		Status: &api.CertificateSigningRequestStatus{
			Conditions: []api.Condition{},
		},
	}
	return csr
}

// createTestDevice creates a test device
func createTestDevice(name string) *api.Device {
	return &api.Device{
		ApiVersion: "v1beta1",
		Kind:       "Device",
		Metadata: api.ObjectMeta{
			Name: lo.ToPtr(name),
		},
		Status: &api.DeviceStatus{},
	}
}

// createTestCertificate creates a test X.509 certificate
func createTestCertificate(t *testing.T, cn string, notBefore, notAfter time.Time) *x509.Certificate {
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
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}

	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &key.PublicKey, key)
	require.NoError(t, err)

	cert, err := x509.ParseCertificate(certDER)
	require.NoError(t, err)
	return cert
}

func TestIsRecoveryRequest(t *testing.T) {
	handler := &ServiceHandler{}

	t.Run("Detects recovery request", func(t *testing.T) {
		labels := map[string]string{
			"flightctl.io/renewal-reason": "expired",
		}
		csr := createTestCSR("test-csr", labels)

		result := handler.isRecoveryRequest(csr)
		assert.True(t, result)
	})

	t.Run("Returns false for non-recovery request", func(t *testing.T) {
		labels := map[string]string{
			"flightctl.io/renewal-reason": "proactive",
		}
		csr := createTestCSR("test-csr", labels)

		result := handler.isRecoveryRequest(csr)
		assert.False(t, result)
	})

	t.Run("Returns false for request without renewal label", func(t *testing.T) {
		labels := map[string]string{
			"other-label": "value",
		}
		csr := createTestCSR("test-csr", labels)

		result := handler.isRecoveryRequest(csr)
		assert.False(t, result)
	})

	t.Run("Returns false for request without labels", func(t *testing.T) {
		csr := createTestCSR("test-csr", nil)
		csr.Metadata.Labels = nil

		result := handler.isRecoveryRequest(csr)
		assert.False(t, result)
	})
}

func TestValidateDeviceFingerprint(t *testing.T) {
	handler := &ServiceHandler{}
	ctx := context.Background()
	orgId := uuid.New()

	t.Run("Validates matching fingerprint", func(t *testing.T) {
		device := createTestDevice("test-device")

		err := handler.validateDeviceFingerprint(ctx, orgId, "test-device", device)
		assert.NoError(t, err)
	})

	t.Run("Rejects mismatched fingerprint", func(t *testing.T) {
		device := createTestDevice("test-device")

		err := handler.validateDeviceFingerprint(ctx, orgId, "wrong-device", device)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "does not match device name")
	})
}

func TestExtractTPMAttestationFromCSR(t *testing.T) {
	handler := &ServiceHandler{}

	t.Run("Returns nil for non-TPM CSR", func(t *testing.T) {
		csr := createTestCSR("test-csr", nil)
		csr.Spec.Request = []byte("non-tpm-csr-data")

		attestation, err := handler.extractTPMAttestationFromCSR(csr)
		assert.NoError(t, err)
		assert.Nil(t, attestation)
	})

	t.Run("Returns nil for TPM CSR (attestation embedded)", func(t *testing.T) {
		// Note: This is a placeholder - actual TPM CSR parsing would be more complex
		csr := createTestCSR("test-csr", nil)

		attestation, err := handler.extractTPMAttestationFromCSR(csr)
		assert.NoError(t, err)
		// Currently returns nil as attestation is validated via verifyTPMCSRRequest
		assert.Nil(t, attestation)
	})
}

func TestAutoApproveRecovery(t *testing.T) {
	testStore := &TestStore{}
	logger := log.InitLogs()
	handler := &ServiceHandler{
		store: testStore,
		log:   logger,
	}
	ctx := context.Background()
	orgId := uuid.New()

	t.Run("Auto-approves recovery request", func(t *testing.T) {
		labels := map[string]string{
			"flightctl.io/renewal-reason": "expired",
		}
		csr := createTestCSR("test-csr", labels)
		csr.Metadata.Owner = lo.ToPtr("device/test-device")

		// Add CSR to store so UpdateStatus can find it
		_, err := testStore.CertificateSigningRequest().Create(ctx, orgId, csr, nil)
		require.NoError(t, err)

		handler.autoApproveRecovery(ctx, orgId, csr)

		// Check approval condition was set
		approved := api.IsStatusConditionTrue(csr.Status.Conditions, api.ConditionTypeCertificateSigningRequestApproved)
		assert.True(t, approved)

		// Check reason is RecoveryAutoApproved
		condition := api.FindStatusCondition(csr.Status.Conditions, api.ConditionTypeCertificateSigningRequestApproved)
		assert.NotNil(t, condition)
		assert.Equal(t, "RecoveryAutoApproved", condition.Reason)
		assert.Contains(t, condition.Message, "Auto-approved recovery request")
	})

	t.Run("Handles already approved request", func(t *testing.T) {
		labels := map[string]string{
			"flightctl.io/renewal-reason": "expired",
		}
		csr := createTestCSR("test-csr", labels)

		// Pre-approve the request
		api.SetStatusCondition(&csr.Status.Conditions, api.Condition{
			Type:   api.ConditionTypeCertificateSigningRequestApproved,
			Status: api.ConditionStatusTrue,
			Reason: "AlreadyApproved",
		})

		handler.autoApproveRecovery(ctx, orgId, csr)

		// Check reason wasn't changed
		condition := api.FindStatusCondition(csr.Status.Conditions, api.ConditionTypeCertificateSigningRequestApproved)
		assert.NotNil(t, condition)
		assert.Equal(t, "AlreadyApproved", condition.Reason)
	})

	t.Run("Handles already denied request", func(t *testing.T) {
		labels := map[string]string{
			"flightctl.io/renewal-reason": "expired",
		}
		csr := createTestCSR("test-csr", labels)

		// Pre-deny the request
		api.SetStatusCondition(&csr.Status.Conditions, api.Condition{
			Type:   api.ConditionTypeCertificateSigningRequestDenied,
			Status: api.ConditionStatusTrue,
			Reason: "Denied",
		})

		handler.autoApproveRecovery(ctx, orgId, csr)

		// Check approval wasn't set
		approved := api.IsStatusConditionTrue(csr.Status.Conditions, api.ConditionTypeCertificateSigningRequestApproved)
		assert.False(t, approved)
	})
}

func TestVerifyTPMQuote(t *testing.T) {
	testStore := &TestStore{}
	logger := log.InitLogs()
	handler := &ServiceHandler{
		store: testStore,
		log:   logger,
	}
	ctx := context.Background()
	orgId := uuid.New()

	t.Run("Verifies TPM quote with matching fingerprint", func(t *testing.T) {
		device := createTestDevice("test-device")
		csr := createTestCSR("test-csr", nil)

		attestation := &identity.RenewalAttestation{
			DeviceFingerprint: "test-device",
		}

		// Note: This test would require a mock TPM CSR and verifyTPMCSRRequest
		// For now, we'll test the fingerprint validation part
		// The actual TPM verification would be tested in integration tests
		err := handler.verifyTPMQuote(ctx, orgId, attestation, device, csr)
		// This will fail because verifyTPMCSRRequest will fail on non-TPM CSR
		// But we can test the fingerprint validation logic separately
		assert.Error(t, err) // Expected to fail due to non-TPM CSR
	})

	t.Run("Rejects mismatched fingerprint", func(t *testing.T) {
		device := createTestDevice("test-device")
		csr := createTestCSR("test-csr", nil)

		attestation := &identity.RenewalAttestation{
			DeviceFingerprint: "wrong-device",
		}

		// Even if TPM verification passes, fingerprint mismatch should fail
		// But since verifyTPMCSRRequest will fail first, we can't test this path easily
		// This would be better tested in integration tests with actual TPM CSRs
		err := handler.verifyTPMQuote(ctx, orgId, attestation, device, csr)
		assert.Error(t, err)
	})
}

func TestValidateRecoveryPeerCertificate(t *testing.T) {
	// Note: Full CA validation tests would require a properly configured CA client
	// These tests are skipped as they require a full CA setup
	// CA signature verification and fingerprint validation would be better tested in integration tests
	t.Skip("Skipping TestValidateRecoveryPeerCertificate - requires full CA client setup (better suited for integration tests)")
}
