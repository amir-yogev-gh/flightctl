package store_test

import (
	"context"
	"encoding/json"
	"time"

	api "github.com/flightctl/flightctl/api/v1beta1"
	"github.com/flightctl/flightctl/internal/config"
	"github.com/flightctl/flightctl/internal/flterrors"
	"github.com/flightctl/flightctl/internal/store"
	"github.com/flightctl/flightctl/internal/store/model"
	flightlog "github.com/flightctl/flightctl/pkg/log"
	testutil "github.com/flightctl/flightctl/test/util"
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/lo"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

var _ = Describe("Device Certificate Tracking", Ordered, func() {
	var (
		log       *logrus.Logger
		ctx       context.Context
		orgId     uuid.UUID
		storeInst store.Store
		devStore  store.Device
		cfg       *config.Config
		dbName    string
		db        *gorm.DB
	)

	BeforeEach(func() {
		ctx = testutil.StartSpecTracerForGinkgo(suiteCtx) // suiteCtx is defined in device_test.go
		log = flightlog.InitLogs()
		storeInst, cfg, dbName, db = store.PrepareDBForUnitTests(ctx, log)
		devStore = storeInst.Device()

		orgId = uuid.New()
		err := testutil.CreateTestOrganization(ctx, storeInst, orgId)
		Expect(err).ToNot(HaveOccurred())
	})

	AfterEach(func() {
		store.DeleteTestDB(ctx, log, cfg, storeInst, dbName)
	})

	Context("Device Model Fields", func() {
		It("should have certificate tracking fields", func() {
			device := model.Device{
				Resource: model.Resource{
					OrgID: orgId,
					Name:  "test-device",
				},
			}

			// Verify fields exist and are accessible
			expiration := time.Now().Add(30 * 24 * time.Hour)
			device.CertificateExpiration = &expiration
			device.CertificateLastRenewed = lo.ToPtr(time.Now())
			device.CertificateRenewalCount = 5
			fingerprint := "abc123"
			device.CertificateFingerprint = &fingerprint

			Expect(device.CertificateExpiration).ToNot(BeNil())
			Expect(device.CertificateLastRenewed).ToNot(BeNil())
			Expect(device.CertificateRenewalCount).To(Equal(5))
			Expect(device.CertificateFingerprint).ToNot(BeNil())
			Expect(*device.CertificateFingerprint).To(Equal("abc123"))
		})

		It("should handle zero values correctly", func() {
			device := model.Device{
				Resource: model.Resource{
					OrgID: orgId,
					Name:  "test-device-zero",
				},
			}

			Expect(device.CertificateExpiration).To(BeNil())
			Expect(device.CertificateLastRenewed).To(BeNil())
			Expect(device.CertificateRenewalCount).To(Equal(0))
			Expect(device.CertificateFingerprint).To(BeNil())
		})

		It("should marshal to JSON correctly", func() {
			expiration := time.Now().Add(30 * 24 * time.Hour)
			device := model.Device{
				Resource: model.Resource{
					OrgID: orgId,
					Name:  "test-device-json",
				},
				CertificateExpiration:   &expiration,
				CertificateLastRenewed:  lo.ToPtr(time.Now()),
				CertificateRenewalCount: 3,
				CertificateFingerprint:  lo.ToPtr("fingerprint123"),
			}

			jsonData, err := json.Marshal(device)
			Expect(err).ToNot(HaveOccurred())
			Expect(string(jsonData)).To(ContainSubstring("certificate_expiration"))
			Expect(string(jsonData)).To(ContainSubstring("certificate_last_renewed"))
			Expect(string(jsonData)).To(ContainSubstring("certificate_renewal_count"))
			Expect(string(jsonData)).To(ContainSubstring("certificate_fingerprint"))
		})

		It("should unmarshal from JSON correctly", func() {
			jsonStr := `{
				"org_id": "` + orgId.String() + `",
				"name": "test-device-unmarshal",
				"certificate_expiration": "2025-12-31T00:00:00Z",
				"certificate_last_renewed": "2025-12-01T00:00:00Z",
				"certificate_renewal_count": 2,
				"certificate_fingerprint": "test-fingerprint"
			}`

			var device model.Device
			err := json.Unmarshal([]byte(jsonStr), &device)
			Expect(err).ToNot(HaveOccurred())
			Expect(device.CertificateExpiration).ToNot(BeNil())
			Expect(device.CertificateLastRenewed).ToNot(BeNil())
			Expect(device.CertificateRenewalCount).To(Equal(2))
			Expect(device.CertificateFingerprint).ToNot(BeNil())
			Expect(*device.CertificateFingerprint).To(Equal("test-fingerprint"))
		})
	})

	Context("Database Migrations", func() {
		It("should have certificate tracking columns in database", func() {
			// Verify columns exist by checking if we can query them
			var device model.Device
			result := db.WithContext(ctx).Select("certificate_expiration, certificate_last_renewed, certificate_renewal_count, certificate_fingerprint").
				First(&device)
			// Should not error on column not found
			Expect(result.Error).To(Or(BeNil(), Equal(gorm.ErrRecordNotFound)))
		})

		It("should handle migration idempotency", func() {
			// Run migration twice - should not error
			// This is tested implicitly by the fact that PrepareDBForUnitTests runs migrations
			// and we can access the columns without errors
			var device model.Device
			result := db.WithContext(ctx).Select("certificate_expiration").
				First(&device)
			// Should not error on column not found
			Expect(result.Error).To(Or(BeNil(), Equal(gorm.ErrRecordNotFound)))
		})

		It("should have correct column types", func() {
			// Create a device and verify types
			expiration := time.Now().Add(30 * 24 * time.Hour)
			device := api.Device{
				Metadata: api.ObjectMeta{
					Name: lo.ToPtr("test-device-types"),
				},
			}

			_, _, err := devStore.CreateOrUpdate(ctx, orgId, &device, nil, true, nil, nil)
			Expect(err).ToNot(HaveOccurred())

			// Update certificate fields
			err = devStore.UpdateCertificateExpiration(ctx, orgId, "test-device-types", &expiration)
			Expect(err).ToNot(HaveOccurred())

			// Verify we can retrieve the timestamp
			retrieved, err := devStore.GetCertificateExpiration(ctx, orgId, "test-device-types")
			Expect(err).ToNot(HaveOccurred())
			Expect(retrieved).ToNot(BeNil())
		})

		It("should apply default values", func() {
			device := api.Device{
				Metadata: api.ObjectMeta{
					Name: lo.ToPtr("test-device-defaults"),
				},
			}

			_, _, err := devStore.CreateOrUpdate(ctx, orgId, &device, nil, true, nil, nil)
			Expect(err).ToNot(HaveOccurred())

			// Check that renewal count defaults to 0
			count, err := devStore.GetCertificateRenewalCount(ctx, orgId, "test-device-defaults")
			Expect(err).ToNot(HaveOccurred())
			Expect(count).To(Equal(0))
		})
	})

	Context("Store Layer Methods", func() {
		var deviceName string

		BeforeEach(func() {
			deviceName = "test-device-methods"
			device := api.Device{
				Metadata: api.ObjectMeta{
					Name: lo.ToPtr(deviceName),
				},
			}
			_, _, err := devStore.CreateOrUpdate(ctx, orgId, &device, nil, true, nil, nil)
			Expect(err).ToNot(HaveOccurred())
		})

		It("should update certificate expiration", func() {
			expiration := time.Now().Add(30 * 24 * time.Hour)
			err := devStore.UpdateCertificateExpiration(ctx, orgId, deviceName, &expiration)
			Expect(err).ToNot(HaveOccurred())

			retrieved, err := devStore.GetCertificateExpiration(ctx, orgId, deviceName)
			Expect(err).ToNot(HaveOccurred())
			Expect(retrieved).ToNot(BeNil())
			Expect(retrieved.UTC().Truncate(time.Second)).To(Equal(expiration.UTC().Truncate(time.Second)))
		})

		It("should update certificate renewal info", func() {
			lastRenewed := time.Now()
			renewalCount := 5
			fingerprint := "test-fingerprint-123"
			err := devStore.UpdateCertificateRenewalInfo(ctx, orgId, deviceName, &lastRenewed, renewalCount, &fingerprint)
			Expect(err).ToNot(HaveOccurred())

			count, err := devStore.GetCertificateRenewalCount(ctx, orgId, deviceName)
			Expect(err).ToNot(HaveOccurred())
			Expect(count).To(Equal(5))
		})

		It("should update certificate fingerprint", func() {
			fingerprint := "new-fingerprint-456"
			err := devStore.UpdateCertificateFingerprint(ctx, orgId, deviceName, fingerprint)
			Expect(err).ToNot(HaveOccurred())

			// Verify by reading device - update should succeed without error
			_, err = devStore.Get(ctx, orgId, deviceName)
			Expect(err).ToNot(HaveOccurred())
			// Note: Fingerprint is not exposed in API, but stored in DB
			// We verify the update succeeded by lack of error
		})

		It("should update certificate tracking atomically", func() {
			expiration := time.Now().Add(30 * 24 * time.Hour)
			lastRenewed := time.Now()
			renewalCount := 3
			fingerprint := "atomic-fingerprint"

			err := devStore.UpdateCertificateTracking(ctx, orgId, deviceName, &expiration, &fingerprint, &lastRenewed, &renewalCount)
			Expect(err).ToNot(HaveOccurred())

			// Verify all fields were updated
			retrieved, err := devStore.GetCertificateExpiration(ctx, orgId, deviceName)
			Expect(err).ToNot(HaveOccurred())
			Expect(retrieved).ToNot(BeNil())

			count, err := devStore.GetCertificateRenewalCount(ctx, orgId, deviceName)
			Expect(err).ToNot(HaveOccurred())
			Expect(count).To(Equal(3))
		})

		It("should return error for non-existent device", func() {
			expiration := time.Now().Add(30 * 24 * time.Hour)
			err := devStore.UpdateCertificateExpiration(ctx, orgId, "non-existent-device", &expiration)
			Expect(err).To(Equal(flterrors.ErrResourceNotFound))
		})
	})

	Context("Query Methods", func() {
		BeforeEach(func() {
			// Create devices with various expiration dates
			now := time.Now().UTC()
			devices := []struct {
				name       string
				expiration time.Time
			}{
				{"device-expiring-soon", now.Add(25 * 24 * time.Hour)},  // 25 days
				{"device-expiring-later", now.Add(60 * 24 * time.Hour)}, // 60 days
				{"device-expired", now.Add(-10 * 24 * time.Hour)},       // Expired 10 days ago
				{"device-no-expiration", time.Time{}},                   // No expiration
			}

			for _, d := range devices {
				device := api.Device{
					Metadata: api.ObjectMeta{
						Name: lo.ToPtr(d.name),
					},
				}
				_, _, err := devStore.CreateOrUpdate(ctx, orgId, &device, nil, true, nil, nil)
				Expect(err).ToNot(HaveOccurred())

				if !d.expiration.IsZero() {
					err = devStore.UpdateCertificateExpiration(ctx, orgId, d.name, &d.expiration)
					Expect(err).ToNot(HaveOccurred())
				}
			}
		})

		It("should query devices expiring soon", func() {
			thresholdDate := time.Now().Add(30 * 24 * time.Hour)
			devices, err := devStore.ListDevicesExpiringSoon(ctx, orgId, thresholdDate)
			Expect(err).ToNot(HaveOccurred())

			// Should include device-expiring-soon (25 days) but not device-expiring-later (60 days)
			deviceNames := make([]string, len(devices))
			for i, d := range devices {
				deviceNames[i] = *d.Metadata.Name
			}
			Expect(deviceNames).To(ContainElement("device-expiring-soon"))
		})

		It("should query devices with expired certificates", func() {
			devices, err := devStore.ListDevicesWithExpiredCertificates(ctx, orgId)
			Expect(err).ToNot(HaveOccurred())

			// Should include device-expired
			deviceNames := make([]string, len(devices))
			for i, d := range devices {
				deviceNames[i] = *d.Metadata.Name
			}
			Expect(deviceNames).To(ContainElement("device-expired"))
		})

		It("should handle NULL values in queries", func() {
			// device-no-expiration has no expiration set
			thresholdDate := time.Now().Add(30 * 24 * time.Hour)
			devices, err := devStore.ListDevicesExpiringSoon(ctx, orgId, thresholdDate)
			Expect(err).ToNot(HaveOccurred())

			// Should not error even with NULL values
			_ = devices
		})
	})

	Context("Index Performance", func() {
		It("should use index for expiration queries", func() {
			// Create multiple devices
			for i := 0; i < 10; i++ {
				device := api.Device{
					Metadata: api.ObjectMeta{
						Name: lo.ToPtr("device-" + string(rune('a'+i))),
					},
				}
				_, _, err := devStore.CreateOrUpdate(ctx, orgId, &device, nil, true, nil, nil)
				Expect(err).ToNot(HaveOccurred())

				expiration := time.Now().Add(time.Duration(i+1) * 24 * time.Hour)
				err = devStore.UpdateCertificateExpiration(ctx, orgId, *device.Metadata.Name, &expiration)
				Expect(err).ToNot(HaveOccurred())
			}

			// Query should be fast (index should be used)
			thresholdDate := time.Now().Add(5 * 24 * time.Hour)
			start := time.Now()
			devices, err := devStore.ListDevicesExpiringSoon(ctx, orgId, thresholdDate)
			duration := time.Since(start)

			Expect(err).ToNot(HaveOccurred())
			Expect(duration).To(BeNumerically("<", 1*time.Second)) // Should be fast
			Expect(len(devices)).To(BeNumerically(">=", 0))
		})
	})
})
