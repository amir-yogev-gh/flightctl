package identity

import (
	"context"
	"crypto/rand"
	"fmt"

	"github.com/flightctl/flightctl/internal/tpm"
	"github.com/flightctl/flightctl/pkg/log"
)

// RenewalAttestation contains TPM attestation data for certificate renewal.
type RenewalAttestation struct {
	// TPMQuote is the TPM quote (attestation signature)
	// Note: This is a placeholder - actual quote generation requires TPM2_Quote command
	TPMQuote []byte `json:"tpm_quote"`

	// PCRValues contains PCR register values (map of PCR index to value)
	PCRValues map[int][]byte `json:"pcr_values"`

	// DeviceFingerprint is the device fingerprint derived from TPM public key
	DeviceFingerprint string `json:"device_fingerprint"`

	// Nonce is a random value to prevent replay attacks
	Nonce []byte `json:"nonce"`

	// LAKPublicKey is the Local Attestation Key public key blob
	// This is used for quote verification
	LAKPublicKey []byte `json:"lak_public_key,omitempty"`

	// LDevIDPublicKey is the Local Device Identity Key public key blob
	// This is used for device identification
	LDevIDPublicKey []byte `json:"ldevid_public_key,omitempty"`
}

// TPMRenewalProvider provides TPM attestation for certificate renewal.
type TPMRenewalProvider struct {
	tpmClient tpm.Client
	log       *log.PrefixLogger
}

// NewTPMRenewalProvider creates a new TPM renewal provider.
func NewTPMRenewalProvider(tpmClient tpm.Client, log *log.PrefixLogger) *TPMRenewalProvider {
	return &TPMRenewalProvider{
		tpmClient: tpmClient,
		log:       log,
	}
}

// GenerateTPMQuote generates a TPM quote for attestation.
// The quote includes PCR values and is signed by the TPM's attestation key.
// Note: This is a simplified implementation. Full TPM quote generation requires
// TPM2_Quote command which is not directly exposed in the current TPM client.
// For now, we use the CertifyKey operation as a proxy for attestation.
func (trp *TPMRenewalProvider) GenerateTPMQuote(ctx context.Context, nonce []byte) ([]byte, error) {
	if trp.tpmClient == nil {
		return nil, fmt.Errorf("TPM client not available")
	}

	trp.log.Debug("Generating TPM quote for renewal attestation")

	// For now, we use CertifyKey as a form of attestation
	// The actual TPM quote would require TPM2_Quote command
	// This is a placeholder that can be enhanced with actual quote generation
	// when the TPM client exposes quote functionality

	// Generate a CSR which includes attestation data
	// The CSR generation includes LAK certification which serves as attestation
	csr, err := trp.tpmClient.MakeCSR("renewal-attestation", nonce)
	if err != nil {
		return nil, fmt.Errorf("failed to generate attestation CSR: %w", err)
	}

	// Extract attestation data from CSR
	// For now, return the CSR as it contains attestation information
	// In a full implementation, we would extract just the quote portion
	return csr, nil
}

// ReadPCRValues reads PCR register values from the TPM.
// Returns a map of PCR index to PCR value.
// Note: This requires TPM2_PCR_Read command which may not be directly exposed.
// This is a placeholder implementation.
func (trp *TPMRenewalProvider) ReadPCRValues(ctx context.Context, pcrIndices []int) (map[int][]byte, error) {
	if trp.tpmClient == nil {
		return nil, fmt.Errorf("TPM client not available")
	}

	pcrValues := make(map[int][]byte)

	trp.log.Debugf("Reading PCR values for indices: %v", pcrIndices)

	// TODO: Implement actual PCR reading using TPM2_PCR_Read
	// This requires access to the TPM session/connection which is not directly exposed
	// For now, return empty map - this will be implemented when TPM client exposes PCR reading
	// The PCR values are typically included in the TPM quote, so they may not need separate reading

	return pcrValues, nil
}

// ReadAllPCRValues reads all PCR register values (0-23).
func (trp *TPMRenewalProvider) ReadAllPCRValues(ctx context.Context) (map[int][]byte, error) {
	// Read all PCRs (0-23)
	pcrIndices := make([]int, 24)
	for i := 0; i < 24; i++ {
		pcrIndices[i] = i
	}
	return trp.ReadPCRValues(ctx, pcrIndices)
}

// GetDeviceFingerprint generates device fingerprint from TPM public key.
func (trp *TPMRenewalProvider) GetDeviceFingerprint(ctx context.Context) (string, error) {
	if trp.tpmClient == nil {
		return "", fmt.Errorf("TPM client not available")
	}

	// Get public key from TPM
	publicKey := trp.tpmClient.Public()
	if publicKey == nil {
		return "", fmt.Errorf("failed to get TPM public key")
	}

	// Generate device fingerprint from public key
	// This should match the fingerprint used during enrollment
	fingerprint, err := generateDeviceName(publicKey)
	if err != nil {
		return "", fmt.Errorf("failed to generate device fingerprint: %w", err)
	}

	return fingerprint, nil
}

// GenerateRenewalAttestation generates complete TPM attestation for certificate renewal.
func (trp *TPMRenewalProvider) GenerateRenewalAttestation(ctx context.Context) (*RenewalAttestation, error) {
	trp.log.Debug("Generating TPM renewal attestation")

	// Step 1: Generate nonce for replay protection
	nonce := make([]byte, 32)
	if _, err := rand.Read(nonce); err != nil {
		return nil, fmt.Errorf("failed to generate nonce: %w", err)
	}

	// Step 2: Generate TPM quote (using CSR as proxy for now)
	tpmQuote, err := trp.GenerateTPMQuote(ctx, nonce)
	if err != nil {
		return nil, fmt.Errorf("failed to generate TPM quote: %w", err)
	}

	// Step 3: Read PCR values
	// Read key PCRs (typically 0-7 for boot measurements)
	keyPCRs := []int{0, 1, 2, 3, 4, 5, 6, 7}
	pcrValues, err := trp.ReadPCRValues(ctx, keyPCRs)
	if err != nil {
		// PCR reading may not be available yet - log warning but continue
		trp.log.Warnf("Failed to read PCR values (may not be implemented): %v", err)
		pcrValues = make(map[int][]byte)
	}

	// Step 4: Get device fingerprint
	fingerprint, err := trp.GetDeviceFingerprint(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get device fingerprint: %w", err)
	}

	// Step 5: Get LAK and LDevID public keys for verification
	// These are included in the CSR, but we extract them separately for clarity
	// Note: This requires access to session which may not be directly available
	// For now, we'll leave these empty - they can be extracted from the CSR if needed

	attestation := &RenewalAttestation{
		TPMQuote:          tpmQuote,
		PCRValues:         pcrValues,
		DeviceFingerprint: fingerprint,
		Nonce:             nonce,
	}

	trp.log.Infof("Generated TPM renewal attestation for device %q", fingerprint)
	return attestation, nil
}
