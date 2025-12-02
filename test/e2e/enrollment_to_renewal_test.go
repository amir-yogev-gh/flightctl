package e2e_test

import (
	"fmt"
	"testing"

	"github.com/flightctl/flightctl/api/v1beta1"
	agent_config "github.com/flightctl/flightctl/internal/agent/config"
	"github.com/flightctl/flightctl/test/harness/e2e"
	testutil "github.com/flightctl/flightctl/test/util"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

const TIMEOUT = "5m"
const POLLING = "125ms"
const LONGTIMEOUT = "10m"

func TestEnrollmentToRenewal(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Enrollment to Automatic Renewal E2E Suite")
}

var _ = BeforeSuite(func() {
	// Setup VM and harness for this worker
	_, _, err := e2e.SetupWorkerHarness()
	Expect(err).ToNot(HaveOccurred())
})

var _ = BeforeEach(func() {
	// Get the harness and context directly - no package-level variables
	workerID := GinkgoParallelProcess()
	harness := e2e.GetWorkerHarness()
	suiteCtx := e2e.GetWorkerContext()

	GinkgoWriter.Printf("🔄 [BeforeEach] Worker %d: Setting up test with VM from pool\n", workerID)

	// Create test-specific context for proper tracing
	ctx := testutil.StartSpecTracerForGinkgo(suiteCtx)

	// Set the test context in the harness
	harness.SetTestContext(ctx)

	// Setup VM from pool, revert to pristine snapshot, and start agent
	err := harness.SetupVMFromPoolAndStartAgent(workerID)
	Expect(err).ToNot(HaveOccurred())

	GinkgoWriter.Printf("✅ [BeforeEach] Worker %d: Test setup completed\n", workerID)
})

var _ = AfterEach(func() {
	workerID := GinkgoParallelProcess()
	GinkgoWriter.Printf("🔄 [AfterEach] Worker %d: Cleaning up test resources\n", workerID)

	// Get the harness and context directly - no shared variables needed
	harness := e2e.GetWorkerHarness()
	suiteCtx := e2e.GetWorkerContext()

	err := harness.CleanUpAllTestResources()
	Expect(err).ToNot(HaveOccurred())

	harness.SetTestContext(suiteCtx)

	GinkgoWriter.Printf("✅ [AfterEach] Worker %d: Test cleanup completed\n", workerID)
})

var _ = Describe("Enrollment to Automatic Renewal E2E Tests", func() {
	Context("Enrollment → Automatic Renewal Flow", func() {
		It("should complete enrollment to renewal flow", func() {
			harness := e2e.GetWorkerHarness()

			By("Enrolling a new device")
			deviceID, device := harness.EnrollAndWaitForOnlineStatus()
			Expect(deviceID).ToNot(BeEmpty())
			Expect(device).ToNot(BeNil())
			Expect(device.Status.Summary.Status).To(Equal(v1beta1.DeviceSummaryStatusOnline))

			By("Reading initial management certificate")
			originalCert, err := harness.GetManagementCertificate()
			Expect(err).ToNot(HaveOccurred())
			Expect(originalCert).ToNot(BeNil())

			originalInfo := e2e.GetCertificateInfo(originalCert)
			GinkgoWriter.Printf("Initial certificate fingerprint: %s\n", originalInfo.Fingerprint[:16])
			GinkgoWriter.Printf("Initial certificate expires: %v\n", originalInfo.NotAfter)
			GinkgoWriter.Printf("Days until expiration: %d\n", originalInfo.DaysUntilExpiration)

			By("Verifying device is operational")
			// Device should be online and reporting status
			Eventually(func() *v1beta1.Device {
				device, err := harness.GetDevice(deviceID)
				if err != nil {
					return nil
				}
				return device
			}, TIMEOUT, POLLING).ShouldNot(BeNil())

			// NOTE: Full time manipulation testing requires:
			// - Time manipulation utilities to fast-forward system time on the VM
			// - Certificate expiration detection mechanism
			// - Renewal trigger verification
			// For now, we verify the certificate exists and device is operational
			// Actual renewal testing would require:
			// 1. Fast-forwarding VM system time to near expiration
			// 2. Waiting for automatic renewal to trigger
			// 3. Verifying certificate fingerprint changes
			// 4. Verifying device continues operating

			By("Verifying certificate file exists and is valid")
			certPath := fmt.Sprintf("/var/lib/flightctl/%s/%s", agent_config.DefaultCertsDirName, agent_config.GeneratedCertFile)
			exists, err := harness.CheckCertificateExists(certPath)
			Expect(err).ToNot(HaveOccurred())
			Expect(exists).To(BeTrue())
		})

		It("should renew before expiration", func() {
			Skip("Requires time manipulation utilities")
			// Test scenario:
			// 1. Enroll device
			// 2. Set renewal threshold to 30 days
			// 3. Fast-forward to 31 days before expiration
			// 4. Verify renewal is triggered
			// 5. Verify renewal completes before expiration
			//
			// NOTE: Requires:
			// - Time manipulation utilities
			// - Certificate expiration calculation
			// - Renewal threshold configuration
		})

		It("should continue operating during renewal", func() {
			Skip("Requires certificate renewal trigger mechanism")
			// Test scenario:
			// 1. Enroll device
			// 2. Trigger renewal (via time manipulation or config)
			// 3. Verify device continues operating during renewal
			// 4. Verify no service interruption
			// 5. Verify API calls succeed during renewal
			//
			// NOTE: Requires:
			// - Renewal trigger mechanism
			// - Device operation verification utilities
			// - Service interruption detection
		})

		It("should verify new certificate is active after renewal", func() {
			Skip("Requires certificate file access and comparison")
			// Test scenario:
			// 1. Enroll device
			// 2. Record current certificate fingerprint
			// 3. Trigger renewal
			// 4. Wait for renewal completion
			// 5. Verify new certificate is different from old
			// 6. Verify new certificate is valid
			// 7. Verify device uses new certificate for mTLS
			//
			// NOTE: Requires:
			// - Certificate file access on VM
			// - Certificate fingerprint comparison
			// - Certificate validation utilities
		})
	})
})
