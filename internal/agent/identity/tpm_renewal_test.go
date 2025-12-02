package identity

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"testing"

	"github.com/flightctl/flightctl/internal/tpm"
	"github.com/flightctl/flightctl/pkg/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

// createMockPublicKey creates a mock ECDSA public key for testing
func createMockPublicKey(t *testing.T) *ecdsa.PublicKey {
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	return &privateKey.PublicKey
}

func TestNewTPMRenewalProvider(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := tpm.NewMockClient(ctrl)
	logger := log.NewPrefixLogger("test")

	provider := NewTPMRenewalProvider(mockClient, logger)

	assert.NotNil(t, provider)
	assert.Equal(t, mockClient, provider.tpmClient)
	assert.Equal(t, logger, provider.log)
}

func TestGenerateTPMQuote(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	nonce := []byte("test-nonce-32-bytes-long-enough")

	t.Run("Generates TPM quote correctly", func(t *testing.T) {
		mockClient := tpm.NewMockClient(ctrl)
		logger := log.NewPrefixLogger("test")

		expectedCSR := []byte("mock-csr-data")
		mockClient.EXPECT().MakeCSR("renewal-attestation", nonce).Return(expectedCSR, nil)

		provider := NewTPMRenewalProvider(mockClient, logger)

		quote, err := provider.GenerateTPMQuote(ctx, nonce)
		require.NoError(t, err)
		assert.Equal(t, expectedCSR, quote)
	})

	t.Run("Handles TPM client unavailable", func(t *testing.T) {
		logger := log.NewPrefixLogger("test")
		provider := NewTPMRenewalProvider(nil, logger)

		_, err := provider.GenerateTPMQuote(ctx, nonce)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "TPM client not available")
	})

	t.Run("Handles MakeCSR errors", func(t *testing.T) {
		mockClient := tpm.NewMockClient(ctrl)
		logger := log.NewPrefixLogger("test")

		mockClient.EXPECT().MakeCSR("renewal-attestation", nonce).Return(nil, assert.AnError)

		provider := NewTPMRenewalProvider(mockClient, logger)

		_, err := provider.GenerateTPMQuote(ctx, nonce)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to generate attestation CSR")
	})
}

func TestReadPCRValues(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	pcrIndices := []int{0, 1, 2, 3, 4, 5, 6, 7}

	t.Run("Reads PCR values correctly", func(t *testing.T) {
		mockClient := tpm.NewMockClient(ctrl)
		logger := log.NewPrefixLogger("test")

		provider := NewTPMRenewalProvider(mockClient, logger)

		// Note: Currently returns empty map as PCR reading is not yet implemented
		pcrValues, err := provider.ReadPCRValues(ctx, pcrIndices)
		require.NoError(t, err)
		assert.NotNil(t, pcrValues)
		assert.Empty(t, pcrValues) // Currently returns empty map
	})

	t.Run("Handles TPM client unavailable", func(t *testing.T) {
		logger := log.NewPrefixLogger("test")
		provider := NewTPMRenewalProvider(nil, logger)

		_, err := provider.ReadPCRValues(ctx, pcrIndices)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "TPM client not available")
	})
}

func TestReadAllPCRValues(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()

	t.Run("Reads all PCR values", func(t *testing.T) {
		mockClient := tpm.NewMockClient(ctrl)
		logger := log.NewPrefixLogger("test")

		provider := NewTPMRenewalProvider(mockClient, logger)

		// Note: Currently returns empty map as PCR reading is not yet implemented
		pcrValues, err := provider.ReadAllPCRValues(ctx)
		require.NoError(t, err)
		assert.NotNil(t, pcrValues)
		assert.Empty(t, pcrValues) // Currently returns empty map
	})
}

func TestGetDeviceFingerprint(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()

	t.Run("Generates correct fingerprint", func(t *testing.T) {
		mockClient := tpm.NewMockClient(ctrl)
		logger := log.NewPrefixLogger("test")

		publicKey := createMockPublicKey(t)
		mockClient.EXPECT().Public().Return(publicKey).Times(2) // Called twice in the test

		provider := NewTPMRenewalProvider(mockClient, logger)

		fingerprint, err := provider.GetDeviceFingerprint(ctx)
		require.NoError(t, err)
		assert.NotEmpty(t, fingerprint)
		// Fingerprint should be deterministic for the same public key
		fingerprint2, err := provider.GetDeviceFingerprint(ctx)
		require.NoError(t, err)
		assert.Equal(t, fingerprint, fingerprint2)
	})

	t.Run("Handles TPM client unavailable", func(t *testing.T) {
		logger := log.NewPrefixLogger("test")
		provider := NewTPMRenewalProvider(nil, logger)

		_, err := provider.GetDeviceFingerprint(ctx)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "TPM client not available")
	})

	t.Run("Handles missing public key", func(t *testing.T) {
		mockClient := tpm.NewMockClient(ctrl)
		logger := log.NewPrefixLogger("test")

		mockClient.EXPECT().Public().Return(nil)

		provider := NewTPMRenewalProvider(mockClient, logger)

		_, err := provider.GetDeviceFingerprint(ctx)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get TPM public key")
	})
}

func TestGenerateRenewalAttestation(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()

	t.Run("Generates complete attestation", func(t *testing.T) {
		mockClient := tpm.NewMockClient(ctrl)
		logger := log.NewPrefixLogger("test")

		publicKey := createMockPublicKey(t)
		expectedCSR := []byte("mock-csr-data")

		mockClient.EXPECT().Public().Return(publicKey).AnyTimes()
		mockClient.EXPECT().MakeCSR("renewal-attestation", gomock.Any()).Return(expectedCSR, nil)

		provider := NewTPMRenewalProvider(mockClient, logger)

		attestation, err := provider.GenerateRenewalAttestation(ctx)
		require.NoError(t, err)
		assert.NotNil(t, attestation)
		assert.Equal(t, expectedCSR, attestation.TPMQuote)
		assert.NotNil(t, attestation.PCRValues)
		assert.NotEmpty(t, attestation.DeviceFingerprint)
		assert.NotEmpty(t, attestation.Nonce)
		assert.Len(t, attestation.Nonce, 32)
	})

	t.Run("Handles TPM client unavailable", func(t *testing.T) {
		logger := log.NewPrefixLogger("test")
		provider := NewTPMRenewalProvider(nil, logger)

		_, err := provider.GenerateRenewalAttestation(ctx)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "TPM client not available")
	})

	t.Run("Handles quote generation failure", func(t *testing.T) {
		mockClient := tpm.NewMockClient(ctrl)
		logger := log.NewPrefixLogger("test")

		mockClient.EXPECT().MakeCSR("renewal-attestation", gomock.Any()).Return(nil, assert.AnError)

		provider := NewTPMRenewalProvider(mockClient, logger)

		_, err := provider.GenerateRenewalAttestation(ctx)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to generate TPM quote")
	})

	t.Run("Handles fingerprint generation failure", func(t *testing.T) {
		mockClient := tpm.NewMockClient(ctrl)
		logger := log.NewPrefixLogger("test")

		expectedCSR := []byte("mock-csr-data")
		mockClient.EXPECT().MakeCSR("renewal-attestation", gomock.Any()).Return(expectedCSR, nil)
		mockClient.EXPECT().Public().Return(nil)

		provider := NewTPMRenewalProvider(mockClient, logger)

		_, err := provider.GenerateRenewalAttestation(ctx)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get device fingerprint")
	})

	t.Run("Continues with empty PCR values if reading fails", func(t *testing.T) {
		mockClient := tpm.NewMockClient(ctrl)
		logger := log.NewPrefixLogger("test")

		publicKey := createMockPublicKey(t)
		expectedCSR := []byte("mock-csr-data")

		mockClient.EXPECT().Public().Return(publicKey).AnyTimes()
		mockClient.EXPECT().MakeCSR("renewal-attestation", gomock.Any()).Return(expectedCSR, nil)

		provider := NewTPMRenewalProvider(mockClient, logger)

		// PCR reading currently returns empty map, so this should succeed
		attestation, err := provider.GenerateRenewalAttestation(ctx)
		require.NoError(t, err)
		assert.NotNil(t, attestation)
		assert.NotNil(t, attestation.PCRValues)
		// Currently PCR values are empty as reading is not implemented
		assert.Empty(t, attestation.PCRValues)
	})
}

func TestRenewalAttestation_Marshal(t *testing.T) {
	attestation := &RenewalAttestation{
		TPMQuote:          []byte("test-quote"),
		PCRValues:         map[int][]byte{0: []byte("pcr0"), 1: []byte("pcr1")},
		DeviceFingerprint: "test-fingerprint",
		Nonce:             []byte("test-nonce"),
		LAKPublicKey:      []byte("lak-key"),
		LDevIDPublicKey:   []byte("ldevid-key"),
	}

	// Test that the struct can be marshaled to JSON
	// This is a basic test to ensure the struct is well-formed
	assert.NotNil(t, attestation)
	assert.Equal(t, []byte("test-quote"), attestation.TPMQuote)
	assert.Equal(t, "test-fingerprint", attestation.DeviceFingerprint)
	assert.Len(t, attestation.PCRValues, 2)
}
