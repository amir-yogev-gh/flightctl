package config

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/flightctl/flightctl/internal/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultCertificateRenewalConfig(t *testing.T) {
	cfg := DefaultCertificateRenewalConfig()

	assert.True(t, cfg.Enabled)
	assert.Equal(t, DefaultCertificateRenewalThresholdDays, cfg.ThresholdDays)
	assert.Equal(t, DefaultCertificateRenewalCheckInterval, cfg.CheckInterval)
	assert.Equal(t, DefaultCertificateRenewalRetryInterval, cfg.RetryInterval)
	assert.Equal(t, DefaultCertificateRenewalMaxRetries, cfg.MaxRetries)
	assert.Equal(t, DefaultCertificateRenewalBackoffMultiplier, cfg.BackoffMultiplier)
	assert.Equal(t, DefaultCertificateRenewalMaxBackoff, cfg.MaxBackoff)
}

func TestCertificateRenewalConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  CertificateRenewalConfig
		wantErr bool
		errMsg  string
	}{
		{
			name: "Valid configuration",
			config: CertificateRenewalConfig{
				ThresholdDays:     30,
				CheckInterval:     util.Duration(24 * time.Hour),
				RetryInterval:     util.Duration(1 * time.Hour),
				MaxRetries:        10,
				BackoffMultiplier: 2.0,
				MaxBackoff:        util.Duration(24 * time.Hour),
			},
			wantErr: false,
		},
		{
			name: "ThresholdDays too low",
			config: CertificateRenewalConfig{
				ThresholdDays: 0,
			},
			wantErr: true,
			errMsg:  "threshold-days must be at least",
		},
		{
			name: "ThresholdDays too high",
			config: CertificateRenewalConfig{
				ThresholdDays: 400,
			},
			wantErr: true,
			errMsg:  "threshold-days must be at most",
		},
		{
			name: "CheckInterval zero",
			config: CertificateRenewalConfig{
				ThresholdDays: 30,
				CheckInterval: util.Duration(0),
			},
			wantErr: true,
			errMsg:  "check-interval must be positive",
		},
		{
			name: "CheckInterval too low",
			config: CertificateRenewalConfig{
				ThresholdDays: 30,
				CheckInterval: util.Duration(30 * time.Minute),
			},
			wantErr: true,
			errMsg:  "check-interval must be at least",
		},
		{
			name: "RetryInterval zero",
			config: CertificateRenewalConfig{
				ThresholdDays: 30,
				CheckInterval: util.Duration(24 * time.Hour),
				RetryInterval: util.Duration(0),
			},
			wantErr: true,
			errMsg:  "retry-interval must be positive",
		},
		{
			name: "RetryInterval too low",
			config: CertificateRenewalConfig{
				ThresholdDays: 30,
				CheckInterval: util.Duration(24 * time.Hour),
				RetryInterval: util.Duration(30 * time.Second),
			},
			wantErr: true,
			errMsg:  "retry-interval must be at least",
		},
		{
			name: "MaxRetries negative",
			config: CertificateRenewalConfig{
				ThresholdDays: 30,
				CheckInterval: util.Duration(24 * time.Hour),
				RetryInterval: util.Duration(1 * time.Hour),
				MaxRetries:    -1,
			},
			wantErr: true,
			errMsg:  "max-retries must be non-negative",
		},
		{
			name: "BackoffMultiplier too low",
			config: CertificateRenewalConfig{
				ThresholdDays:     30,
				CheckInterval:     util.Duration(24 * time.Hour),
				RetryInterval:     util.Duration(1 * time.Hour),
				MaxRetries:        10,
				BackoffMultiplier: 0.5,
			},
			wantErr: true,
			errMsg:  "backoff-multiplier must be at least",
		},
		{
			name: "MaxBackoff zero",
			config: CertificateRenewalConfig{
				ThresholdDays:     30,
				CheckInterval:     util.Duration(24 * time.Hour),
				RetryInterval:     util.Duration(1 * time.Hour),
				MaxRetries:        10,
				BackoffMultiplier: 2.0,
				MaxBackoff:        util.Duration(0),
			},
			wantErr: true,
			errMsg:  "max-backoff must be positive",
		},
		{
			name: "All valid with minimum values",
			config: CertificateRenewalConfig{
				ThresholdDays:     MinCertificateRenewalThresholdDays,
				CheckInterval:     MinCertificateRenewalCheckInterval,
				RetryInterval:     MinCertificateRenewalRetryInterval,
				MaxRetries:        0,
				BackoffMultiplier: MinCertificateRenewalBackoffMultiplier,
				MaxBackoff:        util.Duration(1 * time.Minute),
			},
			wantErr: false,
		},
		{
			name: "All valid with maximum values",
			config: CertificateRenewalConfig{
				ThresholdDays:     MaxCertificateRenewalThresholdDays,
				CheckInterval:     util.Duration(24 * time.Hour),
				RetryInterval:     util.Duration(1 * time.Hour),
				MaxRetries:        100,
				BackoffMultiplier: 10.0,
				MaxBackoff:        util.Duration(48 * time.Hour),
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantErr {
				require.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestCertificateRenewalConfig_ConfigurationMerging(t *testing.T) {
	// Test that configuration can be merged/overridden
	defaultCfg := DefaultCertificateRenewalConfig()

	// Override some values
	customCfg := CertificateRenewalConfig{
		ThresholdDays: 60,
		MaxRetries:    5,
	}

	// Merge logic (simulated - actual merging would be in config loading)
	mergedCfg := defaultCfg
	if customCfg.ThresholdDays != 0 {
		mergedCfg.ThresholdDays = customCfg.ThresholdDays
	}
	if customCfg.MaxRetries != 0 {
		mergedCfg.MaxRetries = customCfg.MaxRetries
	}

	assert.Equal(t, 60, mergedCfg.ThresholdDays)
	assert.Equal(t, 5, mergedCfg.MaxRetries)
	assert.Equal(t, defaultCfg.CheckInterval, mergedCfg.CheckInterval)
	assert.Equal(t, defaultCfg.RetryInterval, mergedCfg.RetryInterval)
}

func TestCertificateRenewalConfig_EdgeCases(t *testing.T) {
	t.Run("Minimum threshold days", func(t *testing.T) {
		cfg := CertificateRenewalConfig{
			ThresholdDays:     MinCertificateRenewalThresholdDays,
			CheckInterval:     util.Duration(24 * time.Hour),
			RetryInterval:     util.Duration(1 * time.Hour),
			MaxRetries:        10,
			BackoffMultiplier: 2.0,
			MaxBackoff:        util.Duration(24 * time.Hour),
		}
		err := cfg.Validate()
		assert.NoError(t, err)
	})

	t.Run("Maximum threshold days", func(t *testing.T) {
		cfg := CertificateRenewalConfig{
			ThresholdDays:     MaxCertificateRenewalThresholdDays,
			CheckInterval:     util.Duration(24 * time.Hour),
			RetryInterval:     util.Duration(1 * time.Hour),
			MaxRetries:        10,
			BackoffMultiplier: 2.0,
			MaxBackoff:        util.Duration(24 * time.Hour),
		}
		err := cfg.Validate()
		assert.NoError(t, err)
	})

	t.Run("Minimum check interval", func(t *testing.T) {
		cfg := CertificateRenewalConfig{
			ThresholdDays:     30,
			CheckInterval:     MinCertificateRenewalCheckInterval,
			RetryInterval:     util.Duration(1 * time.Hour),
			MaxRetries:        10,
			BackoffMultiplier: 2.0,
			MaxBackoff:        util.Duration(24 * time.Hour),
		}
		err := cfg.Validate()
		assert.NoError(t, err)
	})

	t.Run("Minimum retry interval", func(t *testing.T) {
		cfg := CertificateRenewalConfig{
			ThresholdDays:     30,
			CheckInterval:     util.Duration(24 * time.Hour),
			RetryInterval:     MinCertificateRenewalRetryInterval,
			MaxRetries:        10,
			BackoffMultiplier: 2.0,
			MaxBackoff:        util.Duration(24 * time.Hour),
		}
		err := cfg.Validate()
		assert.NoError(t, err)
	})

	t.Run("Zero max retries", func(t *testing.T) {
		cfg := CertificateRenewalConfig{
			ThresholdDays:     30,
			CheckInterval:     util.Duration(24 * time.Hour),
			RetryInterval:     util.Duration(1 * time.Hour),
			MaxRetries:        0,
			BackoffMultiplier: 2.0,
			MaxBackoff:        util.Duration(24 * time.Hour),
		}
		err := cfg.Validate()
		assert.NoError(t, err)
	})

	t.Run("Minimum backoff multiplier", func(t *testing.T) {
		cfg := CertificateRenewalConfig{
			ThresholdDays:     30,
			CheckInterval:     util.Duration(24 * time.Hour),
			RetryInterval:     util.Duration(1 * time.Hour),
			MaxRetries:        10,
			BackoffMultiplier: MinCertificateRenewalBackoffMultiplier,
			MaxBackoff:        util.Duration(24 * time.Hour),
		}
		err := cfg.Validate()
		assert.NoError(t, err)
	})
}

func TestCertificateRenewalConfig_InvalidConfigurationRejection(t *testing.T) {
	t.Run("All invalid values", func(t *testing.T) {
		cfg := CertificateRenewalConfig{
			ThresholdDays:     -1,
			CheckInterval:     util.Duration(0),
			RetryInterval:     util.Duration(0),
			MaxRetries:        -1,
			BackoffMultiplier: 0.5,
			MaxBackoff:        util.Duration(0),
		}
		err := cfg.Validate()
		require.Error(t, err)
		// Should report multiple errors or the first one
		assert.Contains(t, err.Error(), "threshold-days")
	})
}

func TestCertificateRenewalConfig_Constants(t *testing.T) {
	t.Run("Default values", func(t *testing.T) {
		assert.True(t, DefaultCertificateRenewalEnabled)
		assert.Equal(t, 30, DefaultCertificateRenewalThresholdDays)
		assert.Equal(t, util.Duration(24*time.Hour), DefaultCertificateRenewalCheckInterval)
		assert.Equal(t, util.Duration(1*time.Hour), DefaultCertificateRenewalRetryInterval)
		assert.Equal(t, 10, DefaultCertificateRenewalMaxRetries)
		assert.Equal(t, 2.0, DefaultCertificateRenewalBackoffMultiplier)
		assert.Equal(t, util.Duration(24*time.Hour), DefaultCertificateRenewalMaxBackoff)
	})

	t.Run("Minimum values", func(t *testing.T) {
		assert.Equal(t, 1, MinCertificateRenewalThresholdDays)
		assert.Equal(t, util.Duration(1*time.Hour), MinCertificateRenewalCheckInterval)
		assert.Equal(t, util.Duration(1*time.Minute), MinCertificateRenewalRetryInterval)
		assert.Equal(t, 1.0, MinCertificateRenewalBackoffMultiplier)
	})

	t.Run("Maximum values", func(t *testing.T) {
		assert.Equal(t, 365, MaxCertificateRenewalThresholdDays)
	})
}

func TestCertificateRenewalConfig_JSONMarshaling(t *testing.T) {
	t.Run("Marshal to JSON", func(t *testing.T) {
		cfg := CertificateRenewalConfig{
			Enabled:           true,
			ThresholdDays:     30,
			CheckInterval:     util.Duration(24 * time.Hour),
			RetryInterval:     util.Duration(1 * time.Hour),
			MaxRetries:        10,
			BackoffMultiplier: 2.0,
			MaxBackoff:        util.Duration(24 * time.Hour),
		}

		jsonData, err := json.Marshal(cfg)
		require.NoError(t, err)
		assert.Contains(t, string(jsonData), "threshold-days")
		assert.Contains(t, string(jsonData), "check-interval")
		assert.Contains(t, string(jsonData), "retry-interval")
		assert.Contains(t, string(jsonData), "max-retries")
		assert.Contains(t, string(jsonData), "backoff-multiplier")
		assert.Contains(t, string(jsonData), "max-backoff")
	})

	t.Run("Unmarshal from JSON", func(t *testing.T) {
		jsonStr := `{
			"enabled": true,
			"threshold-days": 60,
			"check-interval": "48h",
			"retry-interval": "2h",
			"max-retries": 5,
			"backoff-multiplier": 3.0,
			"max-backoff": "48h"
		}`

		var cfg CertificateRenewalConfig
		err := json.Unmarshal([]byte(jsonStr), &cfg)
		require.NoError(t, err)
		assert.True(t, cfg.Enabled)
		assert.Equal(t, 60, cfg.ThresholdDays)
		assert.Equal(t, util.Duration(48*time.Hour), cfg.CheckInterval)
		assert.Equal(t, util.Duration(2*time.Hour), cfg.RetryInterval)
		assert.Equal(t, 5, cfg.MaxRetries)
		assert.Equal(t, 3.0, cfg.BackoffMultiplier)
		assert.Equal(t, util.Duration(48*time.Hour), cfg.MaxBackoff)
	})

	t.Run("Zero values", func(t *testing.T) {
		cfg := CertificateRenewalConfig{}
		jsonData, err := json.Marshal(cfg)
		require.NoError(t, err)
		// Zero values should be omitted or included based on omitempty
		var unmarshaled CertificateRenewalConfig
		err = json.Unmarshal(jsonData, &unmarshaled)
		require.NoError(t, err)
	})
}

func TestCertificateRenewalConfig_ConfigurationMergingComprehensive(t *testing.T) {
	t.Run("Merge base and override", func(t *testing.T) {
		base := DefaultCertificateRenewalConfig()
		override := CertificateRenewalConfig{
			ThresholdDays: 60,
			MaxRetries:    5,
		}

		merged := base
		if override.ThresholdDays != 0 {
			merged.ThresholdDays = override.ThresholdDays
		}
		if override.MaxRetries != 0 {
			merged.MaxRetries = override.MaxRetries
		}

		assert.Equal(t, 60, merged.ThresholdDays)
		assert.Equal(t, 5, merged.MaxRetries)
		assert.Equal(t, base.CheckInterval, merged.CheckInterval)
		assert.Equal(t, base.RetryInterval, merged.RetryInterval)
	})

	t.Run("Merge with empty override", func(t *testing.T) {
		base := DefaultCertificateRenewalConfig()
		override := CertificateRenewalConfig{}

		merged := base
		if override.ThresholdDays != 0 {
			merged.ThresholdDays = override.ThresholdDays
		}

		assert.Equal(t, base.ThresholdDays, merged.ThresholdDays)
		assert.Equal(t, base.MaxRetries, merged.MaxRetries)
		assert.Equal(t, base.CheckInterval, merged.CheckInterval)
	})

	t.Run("Merge with partial override", func(t *testing.T) {
		base := DefaultCertificateRenewalConfig()
		override := CertificateRenewalConfig{
			ThresholdDays: 45,
		}

		merged := base
		if override.ThresholdDays != 0 {
			merged.ThresholdDays = override.ThresholdDays
		}

		assert.Equal(t, 45, merged.ThresholdDays)
		assert.Equal(t, base.MaxRetries, merged.MaxRetries)
		assert.Equal(t, base.CheckInterval, merged.CheckInterval)
	})
}

func TestCertificateConfig_IntegrationWithAgentConfig(t *testing.T) {
	t.Run("Config loading with certificate section", func(t *testing.T) {
		jsonStr := `{
			"certificate": {
				"renewal": {
					"enabled": true,
					"threshold-days": 60,
					"check-interval": "12h"
				}
			}
		}`

		var cfg Config
		err := json.Unmarshal([]byte(jsonStr), &cfg)
		require.NoError(t, err)
		assert.True(t, cfg.Certificate.Renewal.Enabled)
		assert.Equal(t, 60, cfg.Certificate.Renewal.ThresholdDays)
		assert.Equal(t, util.Duration(12*time.Hour), cfg.Certificate.Renewal.CheckInterval)
	})

	t.Run("Config loading without certificate section", func(t *testing.T) {
		jsonStr := `{}`

		var cfg Config
		err := json.Unmarshal([]byte(jsonStr), &cfg)
		require.NoError(t, err)
		// After Complete(), defaults should be applied
		err = cfg.Complete()
		require.NoError(t, err)
		assert.True(t, cfg.Certificate.Renewal.Enabled)
		assert.Equal(t, DefaultCertificateRenewalThresholdDays, cfg.Certificate.Renewal.ThresholdDays)
	})

	t.Run("Config validation in agent config", func(t *testing.T) {
		// Test that certificate renewal validation is called when enabled
		// This is tested indirectly through the Validate() method
		// Direct validation test is in TestCertificateRenewalConfig_Validate
		cfg := CertificateRenewalConfig{
			Enabled:       true,
			ThresholdDays: 400, // Invalid: too high
		}
		err := cfg.Validate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "threshold-days")
	})
}
