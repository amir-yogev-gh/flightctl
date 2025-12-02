package certmanager

import (
	"crypto/x509"
	"testing"
	"time"

	"github.com/flightctl/flightctl/pkg/log"
)

func TestExpirationMonitor_ParseCertificateExpiration(t *testing.T) {
	logger := log.NewPrefixLogger("test")
	monitor := NewExpirationMonitor(logger)

	tests := []struct {
		name      string
		cert      *x509.Certificate
		wantErr   bool
		checkTime func(*testing.T, time.Time)
	}{
		{
			name: "valid certificate",
			cert: &x509.Certificate{
				NotAfter: time.Now().Add(30 * 24 * time.Hour),
			},
			wantErr: false,
			checkTime: func(t *testing.T, exp time.Time) {
				if exp.IsZero() {
					t.Error("expected non-zero expiration time")
				}
			},
		},
		{
			name:    "nil certificate",
			cert:    nil,
			wantErr: true,
		},
		{
			name: "zero expiration",
			cert: &x509.Certificate{
				NotAfter: time.Time{},
			},
			wantErr: true,
		},
		{
			name: "certificate with different timezone",
			cert: &x509.Certificate{
				NotAfter: time.Now().Add(30 * 24 * time.Hour).In(time.FixedZone("EST", -5*3600)),
			},
			wantErr: false,
			checkTime: func(t *testing.T, exp time.Time) {
				if exp.IsZero() {
					t.Error("expected non-zero expiration time")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			exp, err := monitor.ParseCertificateExpiration(tt.cert)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseCertificateExpiration() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && tt.checkTime != nil {
				tt.checkTime(t, exp)
			}
		})
	}
}

func TestExpirationMonitor_CalculateDaysUntilExpiration(t *testing.T) {
	logger := log.NewPrefixLogger("test")
	monitor := NewExpirationMonitor(logger)
	now := time.Now().UTC()

	tests := []struct {
		name           string
		cert           *x509.Certificate
		wantDays       int
		wantErr        bool
		allowTolerance bool // Allow ±1 day tolerance for timing
	}{
		{
			name: "expires in 30 days",
			cert: &x509.Certificate{
				NotAfter: now.Add(30 * 24 * time.Hour),
			},
			wantDays:       30,
			wantErr:        false,
			allowTolerance: true,
		},
		{
			name: "expires in 1 day",
			cert: &x509.Certificate{
				NotAfter: now.Add(24 * time.Hour),
			},
			wantDays:       1,
			wantErr:        false,
			allowTolerance: true,
		},
		{
			name: "expired yesterday",
			cert: &x509.Certificate{
				NotAfter: now.Add(-24 * time.Hour),
			},
			wantDays:       -1,
			wantErr:        false,
			allowTolerance: true,
		},
		{
			name: "expires today",
			cert: &x509.Certificate{
				NotAfter: now.Add(12 * time.Hour),
			},
			wantDays:       0,
			wantErr:        false,
			allowTolerance: true,
		},
		{
			name:    "nil certificate",
			cert:    nil,
			wantErr: true,
		},
		{
			name: "expires in 1 year",
			cert: &x509.Certificate{
				NotAfter: now.Add(365 * 24 * time.Hour),
			},
			wantDays:       365,
			wantErr:        false,
			allowTolerance: true,
		},
		{
			name: "expires in less than 24 hours",
			cert: &x509.Certificate{
				NotAfter: now.Add(12 * time.Hour),
			},
			wantDays:       0,
			wantErr:        false,
			allowTolerance: false,
		},
		{
			name: "expires in 23 hours 59 minutes",
			cert: &x509.Certificate{
				NotAfter: now.Add(23*time.Hour + 59*time.Minute),
			},
			wantDays:       0,
			wantErr:        false,
			allowTolerance: false,
		},
		{
			name: "expires in 24 hours 1 minute",
			cert: &x509.Certificate{
				NotAfter: now.Add(24*time.Hour + 1*time.Minute),
			},
			wantDays:       1,
			wantErr:        false,
			allowTolerance: false,
		},
		{
			name: "expired 10 days ago",
			cert: &x509.Certificate{
				NotAfter: now.Add(-10 * 24 * time.Hour),
			},
			wantDays:       -10,
			wantErr:        false,
			allowTolerance: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			days, err := monitor.CalculateDaysUntilExpiration(tt.cert)
			if (err != nil) != tt.wantErr {
				t.Errorf("CalculateDaysUntilExpiration() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if tt.allowTolerance {
					// Allow ±1 day tolerance
					if days < tt.wantDays-1 || days > tt.wantDays+1 {
						t.Errorf("CalculateDaysUntilExpiration() = %v, want %v (±1)", days, tt.wantDays)
					}
				} else {
					if days != tt.wantDays {
						t.Errorf("CalculateDaysUntilExpiration() = %v, want %v", days, tt.wantDays)
					}
				}
			}
		})
	}
}

func TestExpirationMonitor_IsExpired(t *testing.T) {
	logger := log.NewPrefixLogger("test")
	monitor := NewExpirationMonitor(logger)
	now := time.Now().UTC()

	tests := []struct {
		name    string
		cert    *x509.Certificate
		want    bool
		wantErr bool
	}{
		{
			name: "expired certificate",
			cert: &x509.Certificate{
				NotAfter: now.Add(-24 * time.Hour),
			},
			want:    true,
			wantErr: false,
		},
		{
			name: "valid certificate",
			cert: &x509.Certificate{
				NotAfter: now.Add(30 * 24 * time.Hour),
			},
			want:    false,
			wantErr: false,
		},
		{
			name: "expires in future",
			cert: &x509.Certificate{
				NotAfter: now.Add(1 * time.Hour),
			},
			want:    false,
			wantErr: false,
		},
		{
			name:    "nil certificate",
			cert:    nil,
			wantErr: true,
		},
		{
			name: "zero expiration",
			cert: &x509.Certificate{
				NotAfter: time.Time{},
			},
			wantErr: true,
		},
		{
			name: "certificate expiring today (not yet expired)",
			cert: &x509.Certificate{
				NotAfter: time.Date(now.Year(), now.Month(), now.Day(), 23, 59, 59, 0, time.UTC),
			},
			want:    false,
			wantErr: false,
		},
		{
			name: "certificate just expired",
			cert: &x509.Certificate{
				NotAfter: now.Add(-1 * time.Second),
			},
			want:    true,
			wantErr: false,
		},
		{
			name: "certificate expiring in 1 second",
			cert: &x509.Certificate{
				NotAfter: now.Add(1 * time.Second),
			},
			want:    false,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expired, err := monitor.IsExpired(tt.cert)
			if (err != nil) != tt.wantErr {
				t.Errorf("IsExpired() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && expired != tt.want {
				t.Errorf("IsExpired() = %v, want %v", expired, tt.want)
			}
		})
	}
}

func TestExpirationMonitor_IsExpiringSoon(t *testing.T) {
	logger := log.NewPrefixLogger("test")
	monitor := NewExpirationMonitor(logger)
	now := time.Now().UTC()

	tests := []struct {
		name          string
		cert          *x509.Certificate
		thresholdDays int
		want          bool
		wantErr       bool
	}{
		{
			name: "expiring within threshold",
			cert: &x509.Certificate{
				NotAfter: now.Add(25 * 24 * time.Hour), // 25 days
			},
			thresholdDays: 30,
			want:          true,
			wantErr:       false,
		},
		{
			name: "expiring exactly at threshold",
			cert: &x509.Certificate{
				NotAfter: now.Add(30 * 24 * time.Hour), // 30 days
			},
			thresholdDays: 30,
			want:          true,
			wantErr:       false,
		},
		{
			name: "expiring beyond threshold",
			cert: &x509.Certificate{
				NotAfter: now.Add(60 * 24 * time.Hour), // 60 days
			},
			thresholdDays: 30,
			want:          false,
			wantErr:       false,
		},
		{
			name: "already expired",
			cert: &x509.Certificate{
				NotAfter: now.Add(-24 * time.Hour), // expired
			},
			thresholdDays: 30,
			want:          true, // expired is considered "expiring soon"
			wantErr:       false,
		},
		{
			name: "negative threshold",
			cert: &x509.Certificate{
				NotAfter: now.Add(30 * 24 * time.Hour),
			},
			thresholdDays: -1,
			wantErr:       true,
		},
		{
			name:          "nil certificate",
			cert:          nil,
			thresholdDays: 30,
			wantErr:       true,
		},
		{
			name: "certificate expiring 1 day after threshold",
			cert: &x509.Certificate{
				NotAfter: now.Add(31*24*time.Hour + 1*time.Hour), // 31 days + 1 hour to ensure it's clearly > 31 days
			},
			thresholdDays: 30,
			want:          false,
			wantErr:       false,
		},
		{
			name: "zero threshold",
			cert: &x509.Certificate{
				NotAfter: now.Add(1 * 24 * time.Hour), // 1 day
			},
			thresholdDays: 0,
			want:          true, // 0 days or less
			wantErr:       false,
		},
		{
			name: "certificate with zero expiration",
			cert: &x509.Certificate{
				NotAfter: time.Time{},
			},
			thresholdDays: 30,
			wantErr:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expiringSoon, err := monitor.IsExpiringSoon(tt.cert, tt.thresholdDays)
			if (err != nil) != tt.wantErr {
				t.Errorf("IsExpiringSoon() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && expiringSoon != tt.want {
				t.Errorf("IsExpiringSoon() = %v, want %v", expiringSoon, tt.want)
			}
		})
	}
}
