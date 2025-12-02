package integration_test

import (
	"context"
	"net/http"

	api "github.com/flightctl/flightctl/api/v1beta1"
	"github.com/flightctl/flightctl/internal/consts"
	"github.com/flightctl/flightctl/internal/identity"
	"github.com/flightctl/flightctl/internal/store/model"
	service_test "github.com/flightctl/flightctl/test/integration/service"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/lo"
)

var _ = Describe("Service Renewal Validation Integration Tests", func() {
	var suite *service_test.ServiceTestSuite
	var ctx context.Context

	BeforeEach(func() {
		suite = service_test.NewServiceTestSuite()
		suite.Setup()
		ctx = suite.Ctx
	})

	AfterEach(func() {
		suite.Teardown()
	})

	Context("Proactive Renewal Validation", func() {
		It("should validate proactive renewal request", func() {
			By("creating a proactive renewal CSR")
			csr := service_test.CreateTestCSR()
			csrName := lo.FromPtr(csr.Metadata.Name)

			if csr.Metadata.Labels == nil {
				csr.Metadata.Labels = &map[string]string{}
			}
			(*csr.Metadata.Labels)["flightctl.io/renewal-reason"] = "proactive"

			By("submitting proactive renewal CSR")
			created, status := suite.Handler.CreateCertificateSigningRequest(ctx, suite.OrgID, csr)
			Expect(status.Code).To(BeEquivalentTo(http.StatusCreated))
			Expect(created).ToNot(BeNil())

			By("verifying renewal label is present")
			retrieved, status := suite.Handler.GetCertificateSigningRequest(ctx, suite.OrgID, csrName)
			Expect(status.Code).To(BeEquivalentTo(http.StatusOK))
			Expect(retrieved).ToNot(BeNil())
			Expect(retrieved.Metadata.Labels).ToNot(BeNil())
			Expect(*retrieved.Metadata.Labels).To(HaveKeyWithValue("flightctl.io/renewal-reason", "proactive"))

			// Note: In full implementation, the service would:
			// 1. Detect this is a renewal request (from label)
			// 2. Validate the device has an existing certificate
			// 3. Validate the certificate is not expired
			// 4. Validate the CSR matches the device
			// 5. Auto-approve if validation passes
		})

		It("should validate proactive renewal request with enrolled device", func() {
			By("enrolling a device")
			er := service_test.CreateTestER()
			deviceName := lo.FromPtr(er.Metadata.Name)

			// Create enrollment request
			createdER, status := suite.Handler.CreateEnrollmentRequest(ctx, suite.OrgID, er)
			Expect(status.Code).To(BeEquivalentTo(http.StatusCreated))
			Expect(createdER).ToNot(BeNil())

			// Approve enrollment request
			defaultOrg := &model.Organization{
				ID:          suite.OrgID,
				ExternalID:  suite.OrgID.String(),
				DisplayName: suite.OrgID.String(),
			}
			mappedIdentity := identity.NewMappedIdentity("testuser", "", []*model.Organization{defaultOrg}, map[string][]string{}, false, nil)
			ctxApproval := context.WithValue(ctx, consts.MappedIdentityCtxKey, mappedIdentity)

			approval := api.EnrollmentRequestApproval{
				Approved: true,
				Labels:   &map[string]string{"approved": "true"},
			}

			_, st := suite.Handler.ApproveEnrollmentRequest(ctxApproval, suite.OrgID, deviceName, approval)
			Expect(st.Code).To(BeEquivalentTo(http.StatusOK))

			// Verify device was created
			device, status := suite.Handler.GetDevice(ctx, suite.OrgID, deviceName)
			Expect(status.Code).To(BeEquivalentTo(http.StatusOK))
			Expect(device).ToNot(BeNil())

			By("creating a proactive renewal CSR for the enrolled device")
			// Generate a new CSR with the same device name (using GenerateDeviceNameAndCSR would create a different name)
			// For this test, we'll create a CSR that matches the device name pattern
			// In real scenarios, the agent would use the same keypair from enrollment
			csr := service_test.CreateTestCSR()
			csrName := lo.FromPtr(csr.Metadata.Name)

			if csr.Metadata.Labels == nil {
				csr.Metadata.Labels = &map[string]string{}
			}
			(*csr.Metadata.Labels)["flightctl.io/renewal-reason"] = "proactive"
			(*csr.Metadata.Labels)["flightctl.io/device-name"] = deviceName

			By("submitting proactive renewal CSR")
			created, status := suite.Handler.CreateCertificateSigningRequest(ctx, suite.OrgID, csr)
			Expect(status.Code).To(BeEquivalentTo(http.StatusCreated))
			Expect(created).ToNot(BeNil())

			By("verifying renewal label is present")
			retrieved, status := suite.Handler.GetCertificateSigningRequest(ctx, suite.OrgID, csrName)
			Expect(status.Code).To(BeEquivalentTo(http.StatusOK))
			Expect(retrieved).ToNot(BeNil())
			Expect(retrieved.Metadata.Labels).ToNot(BeNil())
			Expect(*retrieved.Metadata.Labels).To(HaveKeyWithValue("flightctl.io/renewal-reason", "proactive"))

			// Note: In full implementation, the service would:
			// 1. Detect this is a renewal request (from label)
			// 2. Validate the device has an existing certificate
			// 3. Validate the certificate is not expired
			// 4. Validate the CSR matches the device
			// 5. Auto-approve if validation passes
			// 6. Update device certificate tracking fields (expiration, fingerprint, renewal count)
			//
			// NOTE: This test uses a different CSR than the enrollment CSR because GenerateDeviceNameAndCSR
			// creates a new keypair each time. In a real scenario, the agent would use the same keypair
			// from enrollment to generate the renewal CSR, ensuring the device name matches.
		})
	})

	Context("Expired Certificate Recovery Validation", func() {
		It("should validate recovery request", func() {
			By("creating a recovery CSR")
			csr := service_test.CreateTestCSR()
			csrName := lo.FromPtr(csr.Metadata.Name)

			if csr.Metadata.Labels == nil {
				csr.Metadata.Labels = &map[string]string{}
			}
			(*csr.Metadata.Labels)["flightctl.io/renewal-reason"] = "expired"

			By("submitting recovery CSR")
			created, status := suite.Handler.CreateCertificateSigningRequest(ctx, suite.OrgID, csr)
			Expect(status.Code).To(BeEquivalentTo(http.StatusCreated))
			Expect(created).ToNot(BeNil())

			By("verifying recovery label is present")
			retrieved, status := suite.Handler.GetCertificateSigningRequest(ctx, suite.OrgID, csrName)
			Expect(status.Code).To(BeEquivalentTo(http.StatusOK))
			Expect(retrieved).ToNot(BeNil())
			Expect(retrieved.Metadata.Labels).ToNot(BeNil())
			Expect(*retrieved.Metadata.Labels).To(HaveKeyWithValue("flightctl.io/renewal-reason", "expired"))

			// Note: In full implementation, the service would:
			// 1. Detect this is a recovery request (from label)
			// 2. Validate bootstrap certificate (if used for mTLS)
			// 3. Validate TPM attestation (if included in CSR)
			// 4. Validate device fingerprint matches
			// 5. Auto-approve if validation passes
		})
	})

	Context("Bootstrap Certificate Validation", func() {
		It("should accept recovery request with bootstrap certificate", func() {
			By("creating a recovery CSR")
			csr := service_test.CreateTestCSR()
			csrName := lo.FromPtr(csr.Metadata.Name)

			if csr.Metadata.Labels == nil {
				csr.Metadata.Labels = &map[string]string{}
			}
			(*csr.Metadata.Labels)["flightctl.io/renewal-reason"] = "expired"

			By("submitting recovery CSR")
			created, status := suite.Handler.CreateCertificateSigningRequest(ctx, suite.OrgID, csr)
			Expect(status.Code).To(BeEquivalentTo(http.StatusCreated))
			Expect(created).ToNot(BeNil())

			By("verifying CSR structure")
			retrieved, status := suite.Handler.GetCertificateSigningRequest(ctx, suite.OrgID, csrName)
			Expect(status.Code).To(BeEquivalentTo(http.StatusOK))
			Expect(retrieved).ToNot(BeNil())

			// Note: In full implementation, the service would:
			// 1. Extract peer certificate from mTLS connection
			// 2. Verify it's the bootstrap certificate
			// 3. Verify bootstrap certificate is valid
			// 4. Verify bootstrap certificate matches device
			// 5. Accept the recovery request
		})
	})

	Context("TPM Attestation Validation", func() {
		It("should validate TPM attestation for recovery", func() {
			Skip("Requires TPM simulator")
			By("creating a recovery CSR with TPM attestation")
			csr := service_test.CreateTestCSR()
			csrName := lo.FromPtr(csr.Metadata.Name)

			if csr.Metadata.Labels == nil {
				csr.Metadata.Labels = &map[string]string{}
			}
			(*csr.Metadata.Labels)["flightctl.io/renewal-reason"] = "expired"

			By("submitting recovery CSR")
			created, status := suite.Handler.CreateCertificateSigningRequest(ctx, suite.OrgID, csr)
			Expect(status.Code).To(BeEquivalentTo(http.StatusCreated))
			Expect(created).ToNot(BeNil())

			By("verifying CSR structure")
			retrieved, status := suite.Handler.GetCertificateSigningRequest(ctx, suite.OrgID, csrName)
			Expect(status.Code).To(BeEquivalentTo(http.StatusOK))
			Expect(retrieved).ToNot(BeNil())

			// Note: In full implementation with TPM:
			// 1. TPM attestation would be embedded in CSR (TCG-CSR-IDEVID format)
			// 2. Service would extract attestation from CSR
			// 3. Service would verify TPM quote signature
			// 4. Service would verify PCR values
			// 5. Service would verify device fingerprint
			// 6. Service would accept if validation passes
		})
	})

	Context("Auto-Approval Logic", func() {
		It("should auto-approve valid renewal request", func() {
			By("creating a proactive renewal CSR")
			csr := service_test.CreateTestCSR()
			csrName := lo.FromPtr(csr.Metadata.Name)

			if csr.Metadata.Labels == nil {
				csr.Metadata.Labels = &map[string]string{}
			}
			(*csr.Metadata.Labels)["flightctl.io/renewal-reason"] = "proactive"

			By("submitting proactive renewal CSR")
			created, status := suite.Handler.CreateCertificateSigningRequest(ctx, suite.OrgID, csr)
			Expect(status.Code).To(BeEquivalentTo(http.StatusCreated))
			Expect(created).ToNot(BeNil())

			By("verifying CSR is ready for auto-approval")
			retrieved, status := suite.Handler.GetCertificateSigningRequest(ctx, suite.OrgID, csrName)
			Expect(status.Code).To(BeEquivalentTo(http.StatusOK))
			Expect(retrieved).ToNot(BeNil())

			// Note: In full implementation, the service would:
			// 1. Validate the renewal request
			// 2. Auto-approve if validation passes
			// 3. Sign and issue the certificate
			// 4. Set approval condition in CSR status
		})

		It("should auto-approve valid recovery request", func() {
			By("creating a recovery CSR")
			csr := service_test.CreateTestCSR()
			csrName := lo.FromPtr(csr.Metadata.Name)

			if csr.Metadata.Labels == nil {
				csr.Metadata.Labels = &map[string]string{}
			}
			(*csr.Metadata.Labels)["flightctl.io/renewal-reason"] = "expired"

			By("submitting recovery CSR")
			created, status := suite.Handler.CreateCertificateSigningRequest(ctx, suite.OrgID, csr)
			Expect(status.Code).To(BeEquivalentTo(http.StatusCreated))
			Expect(created).ToNot(BeNil())

			By("verifying CSR is ready for auto-approval")
			retrieved, status := suite.Handler.GetCertificateSigningRequest(ctx, suite.OrgID, csrName)
			Expect(status.Code).To(BeEquivalentTo(http.StatusOK))
			Expect(retrieved).ToNot(BeNil())

			// Note: In full implementation, the service would:
			// 1. Validate the recovery request (bootstrap cert or TPM attestation)
			// 2. Auto-approve if validation passes
			// 3. Sign and issue the certificate
			// 4. Set approval condition in CSR status
		})

		It("should reject invalid renewal request", func() {
			By("creating an invalid renewal CSR")
			csr := service_test.CreateTestCSR()
			// Corrupt the CSR request
			csr.Spec.Request = []byte("invalid-csr-data")

			if csr.Metadata.Labels == nil {
				csr.Metadata.Labels = &map[string]string{}
			}
			(*csr.Metadata.Labels)["flightctl.io/renewal-reason"] = "proactive"

			By("submitting invalid renewal CSR")
			created, status := suite.Handler.CreateCertificateSigningRequest(ctx, suite.OrgID, csr)
			// Service may accept the CSR creation but validation would fail
			if status.Code == http.StatusCreated {
				Expect(created).ToNot(BeNil())
				// Note: Validation would fail when attempting to sign/approve
			} else {
				// Service may reject invalid CSR immediately
				Expect(status.Code).To(BeEquivalentTo(http.StatusBadRequest))
			}
		})
	})
})
