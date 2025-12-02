# Developer Story: TPM Attestation Generation for Recovery

**Story ID:** EDM-323-EPIC-4-STORY-3  
**Epic:** EDM-323-EPIC-4 (Expired Certificate Recovery)  
**Feature:** EDM-323 (Agent Certificate Rotation)  
**Story Points:** 5  
**Priority:** High

## Overview

Implement TPM attestation generation for expired certificate recovery. When both management and bootstrap certificates are expired, the agent should generate TPM attestation (quote and PCR values) to prove device identity for certificate renewal.

## Implementation Tasks

### Task 1: Define Renewal Attestation Structure

**File:** `internal/agent/identity/tpm_renewal.go` (new)

**Objective:** Create new file for TPM renewal attestation.

**Implementation Steps:**

1. **Create tpm_renewal.go file:**
```go
package identity

import (
    "context"
    "crypto"
    "fmt"

    "github.com/flightctl/flightctl/internal/tpm"
    "github.com/flightctl/flightctl/pkg/log"
)

// RenewalAttestation contains TPM attestation data for certificate renewal.
type RenewalAttestation struct {
    // TPMQuote is the TPM quote (attestation signature)
    TPMQuote []byte `json:"tpm_quote"`
    
    // PCRValues contains PCR register values (map of PCR index to value)
    PCRValues map[int][]byte `json:"pcr_values"`
    
    // DeviceFingerprint is the device fingerprint derived from TPM public key
    DeviceFingerprint string `json:"device_fingerprint"`
    
    // Nonce is a random value to prevent replay attacks
    Nonce []byte `json:"nonce"`
}

// TPMRenewalProvider provides TPM attestation for certificate renewal.
type TPMRenewalProvider struct {
    tpmClient tpm.Client
    log       *log.PrefixLogger
}
```

**Testing:**
- Test RenewalAttestation struct can be marshaled
- Test struct fields are correct

---

### Task 2: Implement TPM Quote Generation

**File:** `internal/agent/identity/tpm_renewal.go` (modify)

**Objective:** Generate TPM quote for attestation.

**Implementation Steps:**

1. **Add GenerateTPMQuote method:**
```go
// GenerateTPMQuote generates a TPM quote for attestation.
// The quote includes PCR values and is signed by the TPM's attestation key.
func (trp *TPMRenewalProvider) GenerateTPMQuote(ctx context.Context, nonce []byte) ([]byte, error) {
    if trp.tpmClient == nil {
        return nil, fmt.Errorf("TPM client not available")
    }

    // Get TPM session (if available through client)
    // Note: This depends on TPM client implementation
    // We may need to access the session through the client
    
    // Generate quote using TPM
    // This typically involves:
    // 1. Selecting PCRs to quote
    // 2. Generating nonce
    // 3. Requesting quote from TPM
    // 4. Receiving signed quote
    
    // For now, placeholder implementation
    // Actual implementation depends on TPM client API
    trp.log.Debug("Generating TPM quote for renewal attestation")
    
    // TODO: Implement actual TPM quote generation
    // This will use the TPM client's quote generation capabilities
    
    return nil, fmt.Errorf("TPM quote generation not yet implemented")
}
```

**Note:** This requires understanding the TPM client's quote generation API. The implementation will depend on the actual TPM client interface.

**Testing:**
- Test GenerateTPMQuote generates quote
- Test GenerateTPMQuote handles TPM errors
- Test GenerateTPMQuote uses nonce correctly

---

### Task 3: Implement PCR Value Reading

**File:** `internal/agent/identity/tpm_renewal.go` (modify)

**Objective:** Read PCR values from TPM.

**Implementation Steps:**

1. **Add ReadPCRValues method:**
```go
// ReadPCRValues reads PCR register values from the TPM.
// Returns a map of PCR index to PCR value.
func (trp *TPMRenewalProvider) ReadPCRValues(ctx context.Context, pcrIndices []int) (map[int][]byte, error) {
    if trp.tpmClient == nil {
        return nil, fmt.Errorf("TPM client not available")
    }

    pcrValues := make(map[int][]byte)

    // Read PCR values for specified indices
    // This typically involves:
    // 1. Opening TPM connection
    // 2. Reading PCR values for each index
    // 3. Returning map of index to value
    
    trp.log.Debugf("Reading PCR values for indices: %v", pcrIndices)
    
    // TODO: Implement actual PCR reading
    // This will use the TPM client's PCR reading capabilities
    
    // For now, return empty map
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
```

**Testing:**
- Test ReadPCRValues reads correct PCRs
- Test ReadAllPCRValues reads all PCRs
- Test ReadPCRValues handles TPM errors

---

### Task 4: Implement Device Fingerprint Generation

**File:** `internal/agent/identity/tpm_renewal.go` (modify)

**Objective:** Generate device fingerprint from TPM public key.

**Implementation Steps:**

1. **Add GetDeviceFingerprint method:**
```go
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
```

**Testing:**
- Test GetDeviceFingerprint generates correct fingerprint
- Test GetDeviceFingerprint matches enrollment fingerprint
- Test GetDeviceFingerprint handles errors

---

### Task 5: Implement Complete Attestation Generation

**File:** `internal/agent/identity/tpm_renewal.go` (modify)

**Objective:** Combine all components into complete attestation generation.

**Implementation Steps:**

1. **Add GenerateRenewalAttestation method:**
```go
// GenerateRenewalAttestation generates complete TPM attestation for certificate renewal.
func (trp *TPMRenewalProvider) GenerateRenewalAttestation(ctx context.Context) (*RenewalAttestation, error) {
    trp.log.Debug("Generating TPM renewal attestation")

    // Step 1: Generate nonce for replay protection
    nonce := make([]byte, 32)
    if _, err := crypto.Rand.Read(nonce); err != nil {
        return nil, fmt.Errorf("failed to generate nonce: %w", err)
    }

    // Step 2: Generate TPM quote
    tpmQuote, err := trp.GenerateTPMQuote(ctx, nonce)
    if err != nil {
        return nil, fmt.Errorf("failed to generate TPM quote: %w", err)
    }

    // Step 3: Read PCR values
    // Read key PCRs (typically 0-7 for boot measurements)
    keyPCRs := []int{0, 1, 2, 3, 4, 5, 6, 7}
    pcrValues, err := trp.ReadPCRValues(ctx, keyPCRs)
    if err != nil {
        return nil, fmt.Errorf("failed to read PCR values: %w", err)
    }

    // Step 4: Get device fingerprint
    fingerprint, err := trp.GetDeviceFingerprint(ctx)
    if err != nil {
        return nil, fmt.Errorf("failed to get device fingerprint: %w", err)
    }

    attestation := &RenewalAttestation{
        TPMQuote:          tpmQuote,
        PCRValues:         pcrValues,
        DeviceFingerprint: fingerprint,
        Nonce:             nonce,
    }

    trp.log.Infof("Generated TPM renewal attestation for device %q", fingerprint)
    return attestation, nil
}
```

2. **Add NewTPMRenewalProvider constructor:**
```go
// NewTPMRenewalProvider creates a new TPM renewal provider.
func NewTPMRenewalProvider(tpmClient tpm.Client, log *log.PrefixLogger) *TPMRenewalProvider {
    return &TPMRenewalProvider{
        tpmClient: tpmClient,
        log:       log,
    }
}
```

**Testing:**
- Test GenerateRenewalAttestation generates complete attestation
- Test all components are included
- Test error handling

---

### Task 6: Integrate with TPM Provider

**File:** `internal/agent/identity/tpm.go` (modify)

**Objective:** Add renewal attestation methods to TPM provider.

**Implementation Steps:**

1. **Add renewal provider to tpmProvider:**
```go
// In tpmProvider struct, add:
type tpmProvider struct {
    // ... existing fields ...
    renewalProvider *TPMRenewalProvider
}

// In newTPMProvider, initialize renewal provider:
func newTPMProvider(...) *tpmProvider {
    // ... existing code ...
    
    provider := &tpmProvider{
        // ... existing fields ...
    }
    
    // Initialize renewal provider if TPM client is available
    if client != nil {
        provider.renewalProvider = NewTPMRenewalProvider(client, log)
    }
    
    return provider
}
```

2. **Add GenerateRenewalAttestation method:**
```go
// GenerateRenewalAttestation generates TPM attestation for certificate renewal.
func (t *tpmProvider) GenerateRenewalAttestation(ctx context.Context) (*RenewalAttestation, error) {
    if t.renewalProvider == nil {
        return nil, fmt.Errorf("TPM renewal provider not available")
    }
    return t.renewalProvider.GenerateRenewalAttestation(ctx)
}
```

**Testing:**
- Test TPM provider generates attestation
- Test TPM provider handles missing TPM client

---

### Task 7: Add Attestation to CSR Metadata

**File:** `internal/agent/device/certmanager/provider/provisioner/csr.go` (modify)

**Objective:** Include TPM attestation in renewal CSR requests.

**Implementation Steps:**

1. **Add attestation to CSR metadata:**
```go
// In Provision, when renewal context exists and TPM attestation is available:
if p.renewalContext != nil && p.renewalContext.Reason == "expired" {
    // Check if TPM attestation is needed (bootstrap cert also expired)
    // Get TPM attestation if available
    // Add attestation to CSR metadata
    
    // This will be fully implemented in EDM-323-EPIC-4-STORY-5
    // For now, just add placeholder
}
```

**Note:** Full integration will be in EDM-323-EPIC-4-STORY-5.

**Testing:**
- Test attestation is included in CSR
- Test CSR metadata is correct

---

## Unit Tests

### Test File: `internal/agent/identity/tpm_renewal_test.go` (new)

**Test Cases:**

1. **TestGenerateTPMQuote:**
   - Generates TPM quote correctly
   - Uses nonce correctly
   - Handles TPM errors

2. **TestReadPCRValues:**
   - Reads PCR values correctly
   - Reads all PCRs correctly
   - Handles TPM errors

3. **TestGetDeviceFingerprint:**
   - Generates correct fingerprint
   - Matches enrollment fingerprint
   - Handles errors

4. **TestGenerateRenewalAttestation:**
   - Generates complete attestation
   - Includes all components
   - Handles errors

---

## Integration Tests

### Test File: `test/integration/tpm_renewal_attestation_test.go` (new)

**Test Cases:**

1. **TestTPMAttestationGeneration:**
   - TPM attestation is generated
   - Attestation includes quote and PCRs
   - Attestation can be included in CSR

2. **TestTPMAttestationWithTPMSimulator:**
   - Works with TPM simulator
   - Quote is valid
   - PCR values are correct

---

## Code Review Checklist

- [ ] RenewalAttestation struct is well-defined
- [ ] TPM quote generation works
- [ ] PCR value reading works
- [ ] Device fingerprint generation works
- [ ] Complete attestation generation works
- [ ] Integration with TPM provider works
- [ ] Error handling is comprehensive
- [ ] Logging is appropriate
- [ ] Unit tests achieve >80% coverage
- [ ] Integration tests cover main flows
- [ ] Code follows existing patterns

---

## Definition of Done

- [ ] RenewalAttestation struct created
- [ ] TPM quote generation implemented
- [ ] PCR value reading implemented
- [ ] Device fingerprint generation implemented
- [ ] Complete attestation generation implemented
- [ ] Integration with TPM provider added
- [ ] Unit tests written and passing (>80% coverage)
- [ ] Integration tests written and passing
- [ ] TPM hardware/simulator tests passing
- [ ] Code reviewed and approved
- [ ] Documentation updated

---

## Related Files

- `internal/agent/identity/tpm_renewal.go` - TPM renewal attestation
- `internal/agent/identity/tpm.go` - TPM provider
- `internal/tpm/client.go` - TPM client
- `internal/agent/device/certmanager/provider/provisioner/csr.go` - CSR provisioner

---

## Dependencies

- **EDM-323-EPIC-4-STORY-1**: Expired Certificate Detection (must be completed)
  - Requires expiration detection to trigger recovery

- **Existing TPM Infrastructure**: Uses existing TPM client and session management

---

## Notes

- **TPM Quote**: A TPM quote is a signed statement from the TPM about PCR values. It proves the device's state and identity.

- **PCR Values**: Platform Configuration Registers (PCRs) store measurements of system state. They're used to prove device integrity.

- **Device Fingerprint**: The device fingerprint is derived from the TPM's public key and matches the fingerprint used during enrollment.

- **Nonce**: A random nonce is included to prevent replay attacks. The service can verify the nonce is fresh.

- **TPM Client API**: The actual implementation depends on the TPM client's API for quote generation and PCR reading. This may require additional methods in the TPM client interface.

- **Attestation Format**: The attestation format should match what the service expects for validation. This may need to align with existing TCG CSR format.

- **Error Handling**: TPM operations can fail for various reasons (TPM unavailable, permission errors, etc.). All errors should be handled gracefully.

---

**Document End**

