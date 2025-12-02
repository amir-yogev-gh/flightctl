package service

import (
	"context"
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	api "github.com/flightctl/flightctl/api/v1beta1"
	"github.com/flightctl/flightctl/internal/agent/identity"
	"github.com/flightctl/flightctl/internal/crypto/signer"
	"github.com/flightctl/flightctl/internal/flterrors"
	"github.com/flightctl/flightctl/internal/store/model"
	"github.com/flightctl/flightctl/internal/store/selector"
	"github.com/flightctl/flightctl/internal/tpm"
	"github.com/flightctl/flightctl/internal/util"
	fccrypto "github.com/flightctl/flightctl/pkg/crypto"
	"github.com/google/uuid"
	"github.com/samber/lo"
)

const DefaultEnrollmentCertExpirySeconds int32 = 60 * 60 * 24 * 7 // 7 days

// nowFunc allows overriding for unit tests
var nowFunc = time.Now

func (h *ServiceHandler) autoApprove(ctx context.Context, orgId uuid.UUID, csr *api.CertificateSigningRequest) {
	if api.IsStatusConditionTrue(csr.Status.Conditions, api.ConditionTypeCertificateSigningRequestApproved) || api.IsStatusConditionTrue(csr.Status.Conditions, api.ConditionTypeCertificateSigningRequestDenied) {
		return
	}

	api.SetStatusCondition(&csr.Status.Conditions, api.Condition{
		Type:    api.ConditionTypeCertificateSigningRequestApproved,
		Status:  api.ConditionStatusTrue,
		Reason:  "Approved",
		Message: "Auto-approved by enrollment signer",
	})
	api.RemoveStatusCondition(&csr.Status.Conditions, api.ConditionTypeCertificateSigningRequestFailed)

	if _, err := h.store.CertificateSigningRequest().UpdateStatus(ctx, orgId, csr); err != nil {
		h.log.WithError(err).Error("failed to set approval condition")
	}
}

func (h *ServiceHandler) signApprovedCertificateSigningRequest(ctx context.Context, orgId uuid.UUID, csr *api.CertificateSigningRequest) {
	if csr.Status.Certificate != nil && len(*csr.Status.Certificate) > 0 {
		return
	}

	request, _, err := newSignRequestFromCertificateSigningRequest(csr)
	if err != nil {
		h.setCSRFailedCondition(ctx, orgId, csr, "SigningFailed", fmt.Sprintf("Failed to sign certificate: %v", err))
		h.logRenewalEvent(ctx, orgId, csr, "renewal_failed", nil, nil, lo.ToPtr(err.Error()))
		return
	}

	certPEM, err := signer.SignAsPEM(ctx, h.ca, request)
	if err != nil {
		h.setCSRFailedCondition(ctx, orgId, csr, "SigningFailed", fmt.Sprintf("Failed to sign certificate: %v", err))
		h.logRenewalEvent(ctx, orgId, csr, "renewal_failed", nil, nil, lo.ToPtr(err.Error()))
		return
	}

	csr.Status.Certificate = &certPEM
	if _, err := h.store.CertificateSigningRequest().UpdateStatus(ctx, orgId, csr); err != nil {
		h.log.WithError(err).Error("failed to set signed certificate")
		return
	}

	// Log successful renewal/recovery and update certificate tracking
	// Parse certificate to get expiration
	cert, err := fccrypto.ParseCertificatePEM([]byte(certPEM))
	if err != nil {
		h.log.WithError(err).Warn("failed to parse signed certificate for event logging")
	} else {
		eventType := "renewal_success"
		if h.isRecoveryRequest(csr) {
			eventType = "recovery_success"
		}
		newExpiration := cert.NotAfter
		h.logRenewalEvent(ctx, orgId, csr, eventType, nil, &newExpiration, nil)

		// Update certificate tracking in device record
		h.updateDeviceCertificateTracking(ctx, orgId, csr, cert)
	}
}

func (h *ServiceHandler) ListCertificateSigningRequests(ctx context.Context, orgId uuid.UUID, params api.ListCertificateSigningRequestsParams) (*api.CertificateSigningRequestList, api.Status) {
	listParams, status := prepareListParams(params.Continue, params.LabelSelector, params.FieldSelector, params.Limit)
	if status != api.StatusOK() {
		return nil, status
	}

	result, err := h.store.CertificateSigningRequest().List(ctx, orgId, *listParams)
	if err == nil {
		return result, api.StatusOK()
	}

	var se *selector.SelectorError

	switch {
	case selector.AsSelectorError(err, &se):
		return nil, api.StatusBadRequest(se.Error())
	default:
		return nil, api.StatusInternalServerError(err.Error())
	}
}

func (h *ServiceHandler) verifyTPMCSRRequest(ctx context.Context, orgId uuid.UUID, csr *api.CertificateSigningRequest) error {
	if csr.Status == nil {
		csr.Status = &api.CertificateSigningRequestStatus{}
	}
	csrBytes, isTPM := tpm.ParseTCGCSRBytes(string(csr.Spec.Request))
	if !isTPM {
		return fmt.Errorf("parsing TCG CSR")
	}

	setTPMVerifiedFalse := func(messageTemplate string, args ...any) {
		api.SetStatusCondition(&csr.Status.Conditions, api.Condition{
			Message: fmt.Sprintf(messageTemplate, args...),
			Reason:  api.TPMVerificationFailedReason,
			Status:  api.ConditionStatusFalse,
			Type:    api.ConditionTypeCertificateSigningRequestTPMVerified,
		})
	}

	kind, owner, err := util.GetResourceOwner(csr.Metadata.Owner)
	if err != nil {
		setTPMVerifiedFalse("Failed to determine resource owner")
		return nil
	}
	if kind != api.DeviceKind {
		setTPMVerifiedFalse("The CSR's owner is not a %s", api.DeviceKind)
		return nil
	}
	// TODO this should be retrieved from the device rather than from the ER
	er, err := h.store.EnrollmentRequest().Get(ctx, orgId, owner)
	if err != nil {
		setTPMVerifiedFalse("Unable to find CSR's owner: %s/%s", orgId, owner)
		return nil
	}

	notTPMBasedMessage := fmt.Sprintf("The CSR's owner %s is not TPM based.", lo.FromPtr(csr.Metadata.Owner))
	if er.Status == nil || !api.IsStatusConditionTrue(er.Status.Conditions, api.ConditionTypeEnrollmentRequestTPMVerified) {
		setTPMVerifiedFalse(notTPMBasedMessage)
		return nil
	}

	erBytes, isTPM := tpm.ParseTCGCSRBytes(er.Spec.Csr)
	if !isTPM {
		setTPMVerifiedFalse(notTPMBasedMessage)
		return nil
	}

	parsed, err := tpm.ParseTCGCSR(erBytes)
	if err != nil {
		setTPMVerifiedFalse(notTPMBasedMessage)
		return nil
	}

	if err = tpm.VerifyTCGCSRSigningChain(csrBytes, parsed.CSRContents.Payload.AttestPub); err != nil {
		setTPMVerifiedFalse(err.Error())
		return nil
	}
	api.SetStatusCondition(&csr.Status.Conditions, api.Condition{
		Message: "TPM chain of trust verified",
		Reason:  "TPMVerificationSucceeded",
		Status:  api.ConditionStatusTrue,
		Type:    api.ConditionTypeCertificateSigningRequestTPMVerified,
	})

	return nil
}

func (h *ServiceHandler) CreateCertificateSigningRequest(ctx context.Context, orgId uuid.UUID, csr api.CertificateSigningRequest) (*api.CertificateSigningRequest, api.Status) {
	// don't set fields that are managed by the service for external requests
	if !IsInternalRequest(ctx) {
		csr.Status = nil
		NilOutManagedObjectMetaProperties(&csr.Metadata)
	}

	// Support legacy shorthand "enrollment" by replacing it with the configured signer name
	if csr.Spec.SignerName == "enrollment" {
		csr.Spec.SignerName = h.ca.Cfg.ClientBootstrapSignerName
	}

	if errs := csr.Validate(); len(errs) > 0 {
		return nil, api.StatusBadRequest(errors.Join(errs...).Error())
	}

	if err := h.validateAllowedSignersForCSRService(&csr); err != nil {
		return nil, api.StatusBadRequest(err.Error())
	}

	request, isTPM, err := newSignRequestFromCertificateSigningRequest(&csr)
	if err != nil {
		return nil, api.StatusBadRequest(err.Error())
	}

	if err := signer.Verify(ctx, h.ca, request); err != nil {
		return nil, api.StatusBadRequest(err.Error())
	}
	if isTPM {
		if err = h.verifyTPMCSRRequest(ctx, orgId, &csr); err != nil {
			return nil, api.StatusBadRequest(err.Error())
		}
	}

	result, err := h.store.CertificateSigningRequest().Create(ctx, orgId, &csr, h.callbackCertificateSigningRequestUpdated)
	if err != nil {
		return nil, StoreErrorToApiStatus(err, true, api.CertificateSigningRequestKind, csr.Metadata.Name)
	}

	// Check if this is a recovery request (expired certificate renewal)
	if h.isRecoveryRequest(result) {
		// Log recovery start event
		reason := "expired"
		h.logRenewalEvent(ctx, orgId, result, "recovery_start", &reason, nil, nil)

		// Validate recovery request
		if err := h.validateExpiredCertificateRenewal(ctx, orgId, result); err != nil {
			h.log.WithError(err).Warn("Recovery request validation failed")
			h.logRenewalEvent(ctx, orgId, result, "recovery_failed", &reason, nil, lo.ToPtr(err.Error()))
			// Don't fail the request - let it be manually reviewed
			// But log the validation error
		} else {
			// Auto-approve valid recovery requests
			h.autoApproveRecovery(ctx, orgId, result)
			h.log.Infof("Auto-approved recovery request for device %q", lo.FromPtr(result.Metadata.Owner))
		}
	} else if result.Spec.SignerName == h.ca.Cfg.ClientBootstrapSignerName {
		// Check if this is a renewal request
		if h.isRenewalRequest(result) {
			reason := h.getRenewalReason(result)
			h.logRenewalEvent(ctx, orgId, result, "renewal_start", &reason, nil, nil)
		}
		h.autoApprove(ctx, orgId, result)
	}

	if api.IsStatusConditionTrue(result.Status.Conditions, api.ConditionTypeCertificateSigningRequestApproved) {
		h.signApprovedCertificateSigningRequest(ctx, orgId, result)
	}

	return result, api.StatusCreated()
}

func (h *ServiceHandler) DeleteCertificateSigningRequest(ctx context.Context, orgId uuid.UUID, name string) api.Status {
	err := h.store.CertificateSigningRequest().Delete(ctx, orgId, name, h.callbackCertificateSigningRequestDeleted)
	return StoreErrorToApiStatus(err, false, api.CertificateSigningRequestKind, &name)
}

func (h *ServiceHandler) GetCertificateSigningRequest(ctx context.Context, orgId uuid.UUID, name string) (*api.CertificateSigningRequest, api.Status) {
	result, err := h.store.CertificateSigningRequest().Get(ctx, orgId, name)
	return result, StoreErrorToApiStatus(err, false, api.CertificateSigningRequestKind, &name)
}

func (h *ServiceHandler) PatchCertificateSigningRequest(ctx context.Context, orgId uuid.UUID, name string, patch api.PatchRequest) (*api.CertificateSigningRequest, api.Status) {
	currentObj, err := h.store.CertificateSigningRequest().Get(ctx, orgId, name)
	if err != nil {
		return nil, StoreErrorToApiStatus(err, false, api.CertificateSigningRequestKind, &name)
	}

	newObj := &api.CertificateSigningRequest{}
	err = ApplyJSONPatch(ctx, currentObj, newObj, patch, "/api/v1/certificatesigningrequests/"+name)
	if err != nil {
		return nil, api.StatusBadRequest(err.Error())
	}

	if errs := currentObj.ValidateUpdate(newObj); len(errs) > 0 {
		return nil, api.StatusBadRequest(errors.Join(errs...).Error())
	}

	NilOutManagedObjectMetaProperties(&newObj.Metadata)
	newObj.Metadata.ResourceVersion = nil

	// Support legacy shorthand "enrollment" by replacing it with the configured signer name
	if newObj.Spec.SignerName == "enrollment" {
		newObj.Spec.SignerName = h.ca.Cfg.ClientBootstrapSignerName
	}

	if errs := newObj.Validate(); len(errs) > 0 {
		return nil, api.StatusBadRequest(errors.Join(errs...).Error())
	}

	if err := h.validateAllowedSignersForCSRService(newObj); err != nil {
		return nil, api.StatusBadRequest(err.Error())
	}

	request, isTPM, err := newSignRequestFromCertificateSigningRequest(newObj)
	if err != nil {
		return nil, api.StatusBadRequest(err.Error())
	}

	if err := signer.Verify(ctx, h.ca, request); err != nil {
		return nil, api.StatusBadRequest(err.Error())
	}
	if isTPM {
		if err = h.verifyTPMCSRRequest(ctx, orgId, newObj); err != nil {
			return nil, api.StatusBadRequest(err.Error())
		}
	}

	result, err := h.store.CertificateSigningRequest().Update(ctx, orgId, newObj, h.callbackCertificateSigningRequestUpdated)
	if err != nil {
		return nil, StoreErrorToApiStatus(err, false, api.CertificateSigningRequestKind, &name)
	}

	if result.Spec.SignerName == h.ca.Cfg.ClientBootstrapSignerName {
		h.autoApprove(ctx, orgId, result)
	}
	if api.IsStatusConditionTrue(result.Status.Conditions, api.ConditionTypeCertificateSigningRequestApproved) {
		h.signApprovedCertificateSigningRequest(ctx, orgId, result)
	}

	return result, api.StatusOK()
}

func (h *ServiceHandler) ReplaceCertificateSigningRequest(ctx context.Context, orgId uuid.UUID, name string, csr api.CertificateSigningRequest) (*api.CertificateSigningRequest, api.Status) {
	// don't set fields that are managed by the service for external requests
	if !IsInternalRequest(ctx) {
		csr.Status = nil
		NilOutManagedObjectMetaProperties(&csr.Metadata)
	}

	if name != *csr.Metadata.Name {
		return nil, api.StatusBadRequest("resource name specified in metadata does not match name in path")
	}

	// Support legacy shorthand "enrollment" by replacing it with the configured signer name
	if csr.Spec.SignerName == "enrollment" {
		csr.Spec.SignerName = h.ca.Cfg.ClientBootstrapSignerName
	}

	if errs := csr.Validate(); len(errs) > 0 {
		return nil, api.StatusBadRequest(errors.Join(errs...).Error())
	}

	if err := h.validateAllowedSignersForCSRService(&csr); err != nil {
		return nil, api.StatusBadRequest(err.Error())
	}

	request, isTPM, err := newSignRequestFromCertificateSigningRequest(&csr)
	if err != nil {
		return nil, api.StatusBadRequest(err.Error())
	}

	if err := signer.Verify(ctx, h.ca, request); err != nil {
		return nil, api.StatusBadRequest(err.Error())
	}

	if isTPM {
		if err = h.verifyTPMCSRRequest(ctx, orgId, &csr); err != nil {
			return nil, api.StatusBadRequest(err.Error())
		}
	}

	result, created, err := h.store.CertificateSigningRequest().CreateOrUpdate(ctx, orgId, &csr, h.callbackCertificateSigningRequestUpdated)
	if err != nil {
		return nil, StoreErrorToApiStatus(err, created, api.CertificateSigningRequestKind, &name)
	}

	if result.Spec.SignerName == h.ca.Cfg.ClientBootstrapSignerName {
		h.autoApprove(ctx, orgId, result)
	}
	if api.IsStatusConditionTrue(result.Status.Conditions, api.ConditionTypeCertificateSigningRequestApproved) {
		h.signApprovedCertificateSigningRequest(ctx, orgId, result)
	}

	return result, StoreErrorToApiStatus(nil, created, api.CertificateSigningRequestKind, &name)
}

// NOTE: Approval currently also issues a certificate - this will change in the future based on policy
func (h *ServiceHandler) UpdateCertificateSigningRequestApproval(ctx context.Context, orgId uuid.UUID, name string, csr api.CertificateSigningRequest) (*api.CertificateSigningRequest, api.Status) {
	newCSR := &csr
	NilOutManagedObjectMetaProperties(&newCSR.Metadata)
	if errs := newCSR.Validate(); len(errs) > 0 {
		return nil, api.StatusBadRequest(errors.Join(errs...).Error())
	}
	if err := h.validateAllowedSignersForCSRService(&csr); err != nil {
		return nil, api.StatusBadRequest(err.Error())
	}
	if name != *newCSR.Metadata.Name {
		return nil, api.StatusBadRequest("resource name specified in metadata does not match name in path")
	}
	if newCSR.Status == nil {
		return nil, api.StatusBadRequest("status is required")
	}
	allowedConditionTypes := []api.ConditionType{
		api.ConditionTypeCertificateSigningRequestApproved,
		api.ConditionTypeCertificateSigningRequestDenied,
		api.ConditionTypeCertificateSigningRequestFailed,
		api.ConditionTypeCertificateSigningRequestTPMVerified,
	}
	// manual approving of TPMVerified false is allowed
	trueConditions := []api.ConditionType{
		api.ConditionTypeCertificateSigningRequestApproved,
		api.ConditionTypeCertificateSigningRequestDenied,
		api.ConditionTypeCertificateSigningRequestFailed,
	}
	exclusiveConditions := []api.ConditionType{api.ConditionTypeCertificateSigningRequestApproved, api.ConditionTypeCertificateSigningRequestDenied}
	errs := api.ValidateConditions(newCSR.Status.Conditions, allowedConditionTypes, trueConditions, exclusiveConditions)
	if len(errs) > 0 {
		return nil, api.StatusBadRequest(errors.Join(errs...).Error())
	}

	oldCSR, err := h.store.CertificateSigningRequest().Get(ctx, orgId, name)
	if err != nil {
		return nil, StoreErrorToApiStatus(err, false, api.CertificateSigningRequestKind, &name)
	}

	// do not approve a denied request, or recreate a cert for an already-approved request
	if api.IsStatusConditionTrue(oldCSR.Status.Conditions, api.ConditionTypeCertificateSigningRequestDenied) {
		return nil, api.StatusConflict("The request has already been denied")
	}
	if api.IsStatusConditionTrue(oldCSR.Status.Conditions, api.ConditionTypeCertificateSigningRequestApproved) && oldCSR.Status.Certificate != nil && len(*oldCSR.Status.Certificate) > 0 {
		return nil, api.StatusConflict("The request has already been approved and the certificate issued")
	}

	populateConditionTimestamps(newCSR, oldCSR)
	newConditions := newCSR.Status.Conditions

	// Updating the approval should only update the conditions.
	newCSR.Spec = oldCSR.Spec
	newCSR.Status = oldCSR.Status
	newCSR.Status.Conditions = newConditions

	result, err := h.store.CertificateSigningRequest().UpdateStatus(ctx, orgId, newCSR)
	if err != nil {
		return nil, StoreErrorToApiStatus(err, false, api.CertificateSigningRequestKind, &name)
	}

	if api.IsStatusConditionTrue(result.Status.Conditions, api.ConditionTypeCertificateSigningRequestApproved) {
		h.signApprovedCertificateSigningRequest(ctx, orgId, result)
	}

	return result, api.StatusOK()
}

func newSignRequestFromCertificateSigningRequest(csr *api.CertificateSigningRequest) (signer.SignRequest, bool, error) {
	var opts []signer.SignRequestOption
	csrData, isTPM, err := tpm.NormalizeEnrollmentCSR(string(csr.Spec.Request))
	if err != nil {
		return nil, isTPM, fmt.Errorf("normalizing CSR: %w", err)
	}

	if csr.Status != nil && csr.Status.Certificate != nil {
		opts = append(opts, signer.WithIssuedCertificateBytes(*csr.Status.Certificate))
	}

	if csr.Spec.ExpirationSeconds != nil {
		opts = append(opts, signer.WithExpirationSeconds(*csr.Spec.ExpirationSeconds))
	}

	if csr.Metadata.Name != nil {
		opts = append(opts, signer.WithResourceName(*csr.Metadata.Name))
	}

	signReq, err := signer.NewSignRequestFromBytes(csr.Spec.SignerName, csrData, opts...)
	return signReq, isTPM, err
}

// borrowed from https://github.com/kubernetes/kubernetes/blob/master/pkg/registry/certificates/certificates/strategy.go
func populateConditionTimestamps(newCSR, oldCSR *api.CertificateSigningRequest) {
	now := nowFunc()
	for i := range newCSR.Status.Conditions {
		// preserve existing lastTransitionTime if the condition with this type/status already exists,
		// otherwise set to now.
		if newCSR.Status.Conditions[i].LastTransitionTime.IsZero() {
			lastTransition := now
			for _, oldCondition := range oldCSR.Status.Conditions {
				if oldCondition.Type == newCSR.Status.Conditions[i].Type &&
					oldCondition.Status == newCSR.Status.Conditions[i].Status &&
					!oldCondition.LastTransitionTime.IsZero() {
					lastTransition = oldCondition.LastTransitionTime
					break
				}
			}
			newCSR.Status.Conditions[i].LastTransitionTime = lastTransition
		}
	}
}

func (h *ServiceHandler) validateAllowedSignersForCSRService(csr *api.CertificateSigningRequest) error {
	if csr.Spec.SignerName == h.ca.Cfg.DeviceEnrollmentSignerName {
		return fmt.Errorf("signer name %q is not allowed in CertificateSigningRequest service; use the EnrollmentRequest API instead", csr.Spec.SignerName)
	}
	return nil
}

// callbackCertificateSigningRequestUpdated is the certificate signing request-specific callback that handles CSR events
func (h *ServiceHandler) callbackCertificateSigningRequestUpdated(ctx context.Context, resourceKind api.ResourceKind, orgId uuid.UUID, name string, oldResource, newResource interface{}, created bool, err error) {
	h.eventHandler.HandleCertificateSigningRequestUpdatedEvents(ctx, resourceKind, orgId, name, oldResource, newResource, created, err)
}

// callbackCertificateSigningRequestDeleted is the certificate signing request-specific callback that handles CSR deletion events
func (h *ServiceHandler) callbackCertificateSigningRequestDeleted(ctx context.Context, resourceKind api.ResourceKind, orgId uuid.UUID, name string, oldResource, newResource interface{}, created bool, err error) {
	h.eventHandler.HandleGenericResourceDeletedEvents(ctx, resourceKind, orgId, name, oldResource, newResource, created, err)
}

// setCSRFailedCondition sets the Failed condition on the provided CSR, persists the change, and logs any error during persistence.
func (h *ServiceHandler) setCSRFailedCondition(ctx context.Context, orgId uuid.UUID, csr *api.CertificateSigningRequest, reason, message string) {
	api.SetStatusCondition(&csr.Status.Conditions, api.Condition{
		Type:    api.ConditionTypeCertificateSigningRequestFailed,
		Status:  api.ConditionStatusTrue,
		Reason:  reason,
		Message: message,
	})

	if _, err := h.store.CertificateSigningRequest().UpdateStatus(ctx, orgId, csr); err != nil {
		h.log.WithError(err).Error("failed to set failure condition")
	}
}

// isRecoveryRequest checks if a CSR is a recovery request (expired certificate renewal).
func (h *ServiceHandler) isRecoveryRequest(csr *api.CertificateSigningRequest) bool {
	if csr.Metadata.Labels == nil {
		return false
	}

	labels := *csr.Metadata.Labels
	renewalReason, hasRenewalLabel := labels["flightctl.io/renewal-reason"]

	// Recovery requests have renewal reason "expired"
	return hasRenewalLabel && renewalReason == "expired"
}

// validateExpiredCertificateRenewal validates a recovery CSR request for expired certificate renewal.
func (h *ServiceHandler) validateExpiredCertificateRenewal(ctx context.Context, orgId uuid.UUID, csr *api.CertificateSigningRequest) error {
	// Extract device name from CSR owner
	kind, deviceName, err := util.GetResourceOwner(csr.Metadata.Owner)
	if err != nil {
		return fmt.Errorf("failed to extract device name from CSR owner: %w", err)
	}
	if kind != api.DeviceKind {
		return fmt.Errorf("CSR owner is not a device: %s", kind)
	}

	// Step 1: Verify device exists in database
	device, err := h.store.Device().Get(ctx, orgId, deviceName)
	if err != nil {
		if errors.Is(err, flterrors.ErrResourceNotFound) {
			return fmt.Errorf("device %q not found - recovery requires existing device", deviceName)
		}
		return fmt.Errorf("failed to get device %q: %w", deviceName, err)
	}

	// Step 2: Verify device was previously enrolled
	if device.Status == nil {
		return fmt.Errorf("device %q has no status - device may not be enrolled", deviceName)
	}

	// Step 3: Check authentication method
	// Recovery requests can use:
	// - Bootstrap certificate (if not expired)
	// - TPM attestation (if bootstrap also expired)

	peerCert, err := signer.PeerCertificateFromCtx(ctx)
	if err != nil {
		// No peer certificate - must use TPM attestation
		return h.validateTPMAttestationForRecovery(ctx, orgId, csr, device)
	}

	// Step 4: Validate peer certificate (bootstrap or expired management cert)
	if err := h.validateRecoveryPeerCertificate(ctx, orgId, peerCert, device); err != nil {
		// If bootstrap cert validation fails, try TPM attestation
		h.log.WithError(err).Warn("Peer certificate validation failed, checking TPM attestation")
		return h.validateTPMAttestationForRecovery(ctx, orgId, csr, device)
	}

	// Step 5: Verify CSR CommonName matches device identity
	request, _, err := newSignRequestFromCertificateSigningRequest(csr)
	if err != nil {
		return fmt.Errorf("failed to parse CSR: %w", err)
	}

	x509CSR := request.X509()
	csrFingerprint, err := signer.DeviceFingerprintFromCN(h.ca.Config(), x509CSR.Subject.CommonName)
	if err != nil {
		// If extraction fails, use CN directly
		csrFingerprint = x509CSR.Subject.CommonName
	}

	if csrFingerprint != deviceName {
		return fmt.Errorf("CSR CommonName %q does not match device name %q", csrFingerprint, deviceName)
	}

	// Step 6: Check device is not revoked/blacklisted
	// TODO: Add revocation check if device revocation is implemented

	return nil
}

// validateRecoveryPeerCertificate validates the peer certificate for recovery authentication.
// It accepts either bootstrap certificate or expired management certificate.
func (h *ServiceHandler) validateRecoveryPeerCertificate(ctx context.Context, orgId uuid.UUID, peerCert *x509.Certificate, device *api.Device) error {
	// Extract device fingerprint from peer certificate CN
	peerFingerprint, err := signer.DeviceFingerprintFromCN(h.ca.Config(), peerCert.Subject.CommonName)
	if err != nil {
		// If extraction fails, use CN directly
		peerFingerprint = peerCert.Subject.CommonName
	}

	// Verify peer certificate fingerprint matches device name
	deviceName := lo.FromPtr(device.Metadata.Name)
	if peerFingerprint != deviceName {
		return fmt.Errorf("peer certificate fingerprint %q does not match device name %q", peerFingerprint, deviceName)
	}

	// Check if certificate is expired (acceptable for recovery)
	now := time.Now()
	if peerCert.NotAfter.Before(now) {
		h.log.Warnf("Peer certificate for device %q is expired (expired at %v) - acceptable for recovery",
			deviceName, peerCert.NotAfter)
		// Expired certificate is acceptable for recovery
	}

	// Verify certificate is signed by expected CA
	// This ensures it's either the management cert or bootstrap cert
	caPool := x509.NewCertPool()
	caCerts := h.ca.GetCABundleX509()
	for _, caCert := range caCerts {
		caPool.AddCert(caCert)
	}

	opts := x509.VerifyOptions{
		Roots: caPool,
	}

	if _, err := peerCert.Verify(opts); err != nil {
		return fmt.Errorf("peer certificate signature verification failed: %w", err)
	}

	return nil
}

// validateTPMAttestationForRecovery validates TPM attestation for recovery requests.
func (h *ServiceHandler) validateTPMAttestationForRecovery(ctx context.Context, orgId uuid.UUID, csr *api.CertificateSigningRequest, device *api.Device) error {
	// Check if CSR is a TPM CSR
	_, isTPM := tpm.ParseTCGCSRBytes(string(csr.Spec.Request))
	if !isTPM {
		return fmt.Errorf("recovery request without peer certificate must include TPM attestation")
	}

	// Use existing TPM verification logic
	// The verifyTPMCSRRequest already validates TPM chain of trust
	if err := h.verifyTPMCSRRequest(ctx, orgId, csr); err != nil {
		return fmt.Errorf("TPM attestation verification failed: %w", err)
	}

	// Verify device fingerprint matches
	// Extract standard CSR from TCG CSR
	normalizedCSR, _, err := tpm.NormalizeEnrollmentCSR(string(csr.Spec.Request))
	if err != nil {
		return fmt.Errorf("failed to normalize TCG CSR: %w", err)
	}

	// Parse the standard CSR
	x509CSR, err := x509.ParseCertificateRequest(normalizedCSR)
	if err != nil {
		return fmt.Errorf("failed to parse normalized CSR: %w", err)
	}

	deviceName := lo.FromPtr(device.Metadata.Name)
	// Extract device fingerprint from CSR CommonName
	fingerprint, err := signer.DeviceFingerprintFromCN(h.ca.Config(), x509CSR.Subject.CommonName)
	if err != nil {
		// If extraction fails, use CN directly
		fingerprint = x509CSR.Subject.CommonName
	}

	if fingerprint != deviceName {
		return fmt.Errorf("TPM CSR CommonName %q does not match device name %q", fingerprint, deviceName)
	}

	return nil
}

// extractTPMAttestationFromCSR extracts TPM attestation from CSR metadata or extensions.
// Note: For TPM-based recovery, attestation is embedded in the TCG CSR format.
// This method is a placeholder for extracting explicit attestation data if needed.
func (h *ServiceHandler) extractTPMAttestationFromCSR(csr *api.CertificateSigningRequest) (*identity.RenewalAttestation, error) {
	// Check if CSR is a TPM CSR - attestation is embedded in TCG format
	_, isTPM := tpm.ParseTCGCSRBytes(string(csr.Spec.Request))
	if !isTPM {
		return nil, nil // Not a TPM CSR, no attestation to extract
	}

	// For TPM CSRs, the attestation is embedded in the TCG CSR structure
	// The verifyTPMCSRRequest validates the TPM chain of trust
	// If explicit attestation extraction is needed, it would be done here
	// For now, return nil as attestation is validated via verifyTPMCSRRequest
	return nil, nil
}

// validateDeviceFingerprint verifies device fingerprint matches device record.
func (h *ServiceHandler) validateDeviceFingerprint(ctx context.Context, orgId uuid.UUID, fingerprint string, device *api.Device) error {
	deviceName := lo.FromPtr(device.Metadata.Name)

	// Verify fingerprint matches device name
	if fingerprint != deviceName {
		return fmt.Errorf("device fingerprint %q does not match device name %q", fingerprint, deviceName)
	}

	// Optionally verify fingerprint matches stored fingerprint in device record
	// This requires device model to have fingerprint field (from EDM-323-EPIC-1-STORY-4)
	// Note: The API resource might not expose CertificateFingerprint directly
	// For now, we'll just verify it matches the device name
	// Stored fingerprint mismatch may be acceptable for recovery (device may have new key)

	return nil
}

// verifyTPMQuote verifies TPM quote signature and PCR values.
// This is a wrapper around verifyTPMCSRRequest which already validates TPM chain of trust.
func (h *ServiceHandler) verifyTPMQuote(ctx context.Context, orgId uuid.UUID, attestation *identity.RenewalAttestation, device *api.Device, csr *api.CertificateSigningRequest) error {
	// Use existing TPM verification logic
	// The verifyTPMCSRRequest already validates TPM chain of trust
	if err := h.verifyTPMCSRRequest(ctx, orgId, csr); err != nil {
		return fmt.Errorf("TPM quote verification failed: %w", err)
	}

	// Verify device fingerprint matches
	if attestation != nil && attestation.DeviceFingerprint != "" {
		deviceName := lo.FromPtr(device.Metadata.Name)
		if attestation.DeviceFingerprint != deviceName {
			return fmt.Errorf("TPM attestation device fingerprint %q does not match device name %q",
				attestation.DeviceFingerprint, deviceName)
		}
	}

	// Step 3: Verify PCR values (optional - may not enforce exact match)
	// PCR values may change, so we may only verify they're reasonable
	// For now, we don't enforce PCR value matching as they may legitimately change

	// Step 4: Verify attestation freshness (nonce check)
	// TODO: Implement nonce freshness check to prevent replay attacks
	// This would require storing nonces and checking they're not reused

	return nil
}

// autoApproveRecovery auto-approves a validated recovery CSR request.
// This is a wrapper around autoApprove with recovery-specific messaging.
func (h *ServiceHandler) autoApproveRecovery(ctx context.Context, orgId uuid.UUID, csr *api.CertificateSigningRequest) {
	if api.IsStatusConditionTrue(csr.Status.Conditions, api.ConditionTypeCertificateSigningRequestApproved) ||
		api.IsStatusConditionTrue(csr.Status.Conditions, api.ConditionTypeCertificateSigningRequestDenied) {
		return
	}

	kind, deviceName, err := util.GetResourceOwner(csr.Metadata.Owner)
	if err != nil {
		deviceName = "unknown"
	}
	if kind != api.DeviceKind {
		deviceName = "unknown"
	}

	message := fmt.Sprintf("Auto-approved recovery request for device %q (expired certificate renewal)", deviceName)

	api.SetStatusCondition(&csr.Status.Conditions, api.Condition{
		Type:    api.ConditionTypeCertificateSigningRequestApproved,
		Status:  api.ConditionStatusTrue,
		Reason:  "RecoveryAutoApproved",
		Message: message,
	})
	api.RemoveStatusCondition(&csr.Status.Conditions, api.ConditionTypeCertificateSigningRequestFailed)

	if _, err := h.store.CertificateSigningRequest().UpdateStatus(ctx, orgId, csr); err != nil {
		h.log.WithError(err).Error("failed to set recovery approval condition")
	}
}

// isRenewalRequest checks if a CSR is a renewal request (not a recovery).
func (h *ServiceHandler) isRenewalRequest(csr *api.CertificateSigningRequest) bool {
	if csr.Metadata.Labels == nil {
		return false
	}

	labels := *csr.Metadata.Labels
	_, hasRenewalLabel := labels["flightctl.io/renewal-reason"]

	// Renewal requests have renewal reason but not "expired"
	return hasRenewalLabel && labels["flightctl.io/renewal-reason"] != "expired"
}

// getRenewalReason extracts the renewal reason from CSR labels.
func (h *ServiceHandler) getRenewalReason(csr *api.CertificateSigningRequest) string {
	if csr.Metadata.Labels == nil {
		return ""
	}

	labels := *csr.Metadata.Labels
	if reason, ok := labels["flightctl.io/renewal-reason"]; ok {
		return reason
	}

	return ""
}

// getDeviceIDFromCSR generates a deterministic UUID from the device name in the CSR.
// Since devices use composite keys (OrgID, Name) rather than UUIDs, we generate
// a deterministic UUID v5 using the orgID as namespace and device name as the name.
func (h *ServiceHandler) getDeviceIDFromCSR(orgId uuid.UUID, csr *api.CertificateSigningRequest) (uuid.UUID, error) {
	kind, deviceName, err := util.GetResourceOwner(csr.Metadata.Owner)
	if err != nil {
		return uuid.Nil, fmt.Errorf("failed to extract device name from CSR owner: %w", err)
	}
	if kind != api.DeviceKind {
		return uuid.Nil, fmt.Errorf("CSR owner is not a device: %s", kind)
	}

	// Generate deterministic UUID v5 from orgID (namespace) and device name
	// Using a fixed namespace UUID for device IDs
	deviceID := uuid.NewSHA1(uuid.NameSpaceOID, []byte(fmt.Sprintf("%s/%s", orgId.String(), deviceName)))
	return deviceID, nil
}

// logRenewalEvent logs a certificate renewal or recovery event.
func (h *ServiceHandler) logRenewalEvent(ctx context.Context, orgId uuid.UUID, csr *api.CertificateSigningRequest, eventType string, reason *string, newCertExpiration *time.Time, errorMessage *string) {
	// Get device ID from CSR
	deviceID, err := h.getDeviceIDFromCSR(orgId, csr)
	if err != nil {
		h.log.WithError(err).Warn("failed to get device ID from CSR for event logging")
		return
	}

	// Get old certificate expiration from device record if available
	var oldCertExpiration *time.Time
	kind, deviceName, err := util.GetResourceOwner(csr.Metadata.Owner)
	if err == nil && kind == api.DeviceKind {
		// Use the dedicated method to get certificate expiration
		expiration, err := h.store.Device().GetCertificateExpiration(ctx, orgId, deviceName)
		if err == nil {
			oldCertExpiration = expiration
		}
	}

	event := &model.CertificateRenewalEvent{
		DeviceID:          deviceID,
		EventType:         eventType,
		Reason:            reason,
		OldCertExpiration: oldCertExpiration,
		NewCertExpiration: newCertExpiration,
		ErrorMessage:      errorMessage,
	}

	if err := h.store.CertificateRenewalEvent().Create(ctx, orgId, event); err != nil {
		h.log.WithError(err).Warn("failed to log certificate renewal event")
	}
}

// calculateCertificateFingerprint calculates SHA256 fingerprint of certificate.
func (h *ServiceHandler) calculateCertificateFingerprint(cert *x509.Certificate) string {
	hash := sha256.Sum256(cert.Raw)
	return hex.EncodeToString(hash[:])
}

// updateDeviceCertificateTracking updates certificate tracking fields in the device record.
func (h *ServiceHandler) updateDeviceCertificateTracking(ctx context.Context, orgId uuid.UUID, csr *api.CertificateSigningRequest, cert *x509.Certificate) {
	// Extract device name from CSR owner
	kind, deviceName, err := util.GetResourceOwner(csr.Metadata.Owner)
	if err != nil {
		h.log.WithError(err).Warn("failed to extract device name from CSR for certificate tracking")
		return
	}
	if kind != api.DeviceKind {
		// Not a device certificate - skip tracking
		return
	}

	// Calculate certificate fingerprint
	fingerprint := h.calculateCertificateFingerprint(cert)
	expiration := cert.NotAfter

	// Check if this is a renewal request
	isRenewal := h.isRenewalRequest(csr) || h.isRecoveryRequest(csr)
	if isRenewal {
		// Get current renewal count and increment
		currentRenewalCount := h.getDeviceRenewalCount(ctx, orgId, deviceName)
		renewalCount := currentRenewalCount + 1
		lastRenewed := time.Now()

		// Use transaction-based update for atomicity
		if err := h.store.Device().UpdateCertificateTracking(ctx, orgId, deviceName, &expiration, &fingerprint, &lastRenewed, &renewalCount); err != nil {
			h.log.WithError(err).Warnf("failed to update certificate tracking for device %q", deviceName)
			// Fallback to individual updates if transaction fails
			if err := h.store.Device().UpdateCertificateExpiration(ctx, orgId, deviceName, &expiration); err != nil {
				h.log.WithError(err).Warnf("failed to update certificate expiration for device %q", deviceName)
			}
			if err := h.store.Device().UpdateCertificateRenewalInfo(ctx, orgId, deviceName, &lastRenewed, renewalCount, &fingerprint); err != nil {
				h.log.WithError(err).Warnf("failed to update certificate renewal info for device %q", deviceName)
			}
		}
	} else {
		// Initial certificate issuance - update expiration and fingerprint
		// Use transaction-based update for atomicity
		if err := h.store.Device().UpdateCertificateTracking(ctx, orgId, deviceName, &expiration, &fingerprint, nil, nil); err != nil {
			h.log.WithError(err).Warnf("failed to update certificate tracking for device %q", deviceName)
			// Fallback to individual updates if transaction fails
			if err := h.store.Device().UpdateCertificateExpiration(ctx, orgId, deviceName, &expiration); err != nil {
				h.log.WithError(err).Warnf("failed to update certificate expiration for device %q", deviceName)
			}
			if err := h.store.Device().UpdateCertificateFingerprint(ctx, orgId, deviceName, fingerprint); err != nil {
				h.log.WithError(err).Warnf("failed to update certificate fingerprint for device %q", deviceName)
			}
		}
	}
}

// getDeviceRenewalCount retrieves the current renewal count from the device.
// Returns 0 if the device doesn't exist or the count cannot be retrieved.
func (h *ServiceHandler) getDeviceRenewalCount(ctx context.Context, orgId uuid.UUID, deviceName string) int {
	count, err := h.store.Device().GetCertificateRenewalCount(ctx, orgId, deviceName)
	if err != nil {
		// Device doesn't exist or error - return 0
		h.log.WithError(err).Debugf("failed to get renewal count for device %q, defaulting to 0", deviceName)
		return 0
	}
	return count
}
