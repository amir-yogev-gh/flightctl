package load

import (
	"fmt"
	"sort"
	"sync"
	"time"
)

// MetricsCollector collects and analyzes performance metrics
type MetricsCollector struct {
	mu                sync.RWMutex
	responseTimes     []time.Duration
	dbQueryTimes      []time.Duration
	queueDepths       []int
	certIssuanceTimes []time.Time
	errors            []error
	startTime         time.Time
	endTime           time.Time
}

// PerformanceMetrics contains aggregated performance metrics
type PerformanceMetrics struct {
	ResponseTimeP50  time.Duration
	ResponseTimeP95  time.Duration
	ResponseTimeP99  time.Duration
	DBQueryTimeP50   time.Duration
	DBQueryTimeP95   time.Duration
	DBQueryTimeP99   time.Duration
	MaxQueueDepth    int
	AvgQueueDepth    float64
	CertIssuanceRate float64 // certificates per second
	ErrorRate        float64 // errors per second
	TotalRequests    int
	TotalErrors      int
	TestDuration     time.Duration
}

// NewMetricsCollector creates a new metrics collector
func NewMetricsCollector() *MetricsCollector {
	return &MetricsCollector{
		responseTimes:     make([]time.Duration, 0),
		dbQueryTimes:      make([]time.Duration, 0),
		queueDepths:       make([]int, 0),
		certIssuanceTimes: make([]time.Time, 0),
		errors:            make([]error, 0),
		startTime:         time.Now(),
	}
}

// Start begins metrics collection
func (mc *MetricsCollector) Start() {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	mc.startTime = time.Now()
}

// Stop ends metrics collection
func (mc *MetricsCollector) Stop() {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	mc.endTime = time.Now()
}

// RecordResponseTime records a response time measurement
func (mc *MetricsCollector) RecordResponseTime(duration time.Duration) {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	mc.responseTimes = append(mc.responseTimes, duration)
}

// RecordDBQueryTime records a database query time measurement
func (mc *MetricsCollector) RecordDBQueryTime(duration time.Duration) {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	mc.dbQueryTimes = append(mc.dbQueryTimes, duration)
}

// RecordQueueDepth records a queue depth measurement
func (mc *MetricsCollector) RecordQueueDepth(depth int) {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	mc.queueDepths = append(mc.queueDepths, depth)
}

// RecordCertIssuance records a certificate issuance event
func (mc *MetricsCollector) RecordCertIssuance() {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	mc.certIssuanceTimes = append(mc.certIssuanceTimes, time.Now())
}

// RecordError records an error
func (mc *MetricsCollector) RecordError(err error) {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	mc.errors = append(mc.errors, err)
}

// GetMetrics calculates and returns aggregated performance metrics
func (mc *MetricsCollector) GetMetrics() *PerformanceMetrics {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	metrics := &PerformanceMetrics{
		TotalRequests: len(mc.responseTimes),
		TotalErrors:   len(mc.errors),
	}

	if !mc.endTime.IsZero() {
		metrics.TestDuration = mc.endTime.Sub(mc.startTime)
	} else {
		metrics.TestDuration = time.Since(mc.startTime)
	}

	// Calculate response time percentiles
	if len(mc.responseTimes) > 0 {
		sorted := make([]time.Duration, len(mc.responseTimes))
		copy(sorted, mc.responseTimes)
		sort.Slice(sorted, func(i, j int) bool { return sorted[i] < sorted[j] })
		metrics.ResponseTimeP50 = percentile(sorted, 50)
		metrics.ResponseTimeP95 = percentile(sorted, 95)
		metrics.ResponseTimeP99 = percentile(sorted, 99)
	}

	// Calculate database query time percentiles
	if len(mc.dbQueryTimes) > 0 {
		sorted := make([]time.Duration, len(mc.dbQueryTimes))
		copy(sorted, mc.dbQueryTimes)
		sort.Slice(sorted, func(i, j int) bool { return sorted[i] < sorted[j] })
		metrics.DBQueryTimeP50 = percentile(sorted, 50)
		metrics.DBQueryTimeP95 = percentile(sorted, 95)
		metrics.DBQueryTimeP99 = percentile(sorted, 99)
	}

	// Calculate queue depth statistics
	if len(mc.queueDepths) > 0 {
		max := 0
		sum := 0
		for _, depth := range mc.queueDepths {
			if depth > max {
				max = depth
			}
			sum += depth
		}
		metrics.MaxQueueDepth = max
		metrics.AvgQueueDepth = float64(sum) / float64(len(mc.queueDepths))
	}

	// Calculate certificate issuance rate
	if len(mc.certIssuanceTimes) > 0 && metrics.TestDuration > 0 {
		metrics.CertIssuanceRate = float64(len(mc.certIssuanceTimes)) / metrics.TestDuration.Seconds()
	}

	// Calculate error rate
	if metrics.TestDuration > 0 {
		metrics.ErrorRate = float64(metrics.TotalErrors) / metrics.TestDuration.Seconds()
	}

	return metrics
}

// percentile calculates the percentile value from a sorted slice
func percentile(sorted []time.Duration, p int) time.Duration {
	if len(sorted) == 0 {
		return 0
	}
	index := (p * len(sorted)) / 100
	if index >= len(sorted) {
		index = len(sorted) - 1
	}
	return sorted[index]
}

// String returns a string representation of the metrics
func (pm *PerformanceMetrics) String() string {
	return fmt.Sprintf(`
Performance Metrics:
  Response Times:
    P50: %v
    P95: %v
    P99: %v
  Database Query Times:
    P50: %v
    P95: %v
    P99: %v
  Queue Depth:
    Max: %d
    Avg: %.2f
  Certificate Issuance Rate: %.2f/sec
  Error Rate: %.2f/sec
  Total Requests: %d
  Total Errors: %d
  Test Duration: %v
`,
		pm.ResponseTimeP50,
		pm.ResponseTimeP95,
		pm.ResponseTimeP99,
		pm.DBQueryTimeP50,
		pm.DBQueryTimeP95,
		pm.DBQueryTimeP99,
		pm.MaxQueueDepth,
		pm.AvgQueueDepth,
		pm.CertIssuanceRate,
		pm.ErrorRate,
		pm.TotalRequests,
		pm.TotalErrors,
		pm.TestDuration,
	)
}
