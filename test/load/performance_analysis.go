package load

import (
	"fmt"
	"time"
)

// PerformanceTargets defines performance targets for load tests
type PerformanceTargets struct {
	ResponseTimeP95Max     time.Duration
	DBQueryTimeMax         time.Duration
	MaxQueueDepth          int
	MinCertIssuanceRate    float64 // certificates per second
	MaxErrorRate           float64 // errors per second
	MaxCPUUtilization      float64 // percentage
	MaxMemoryUtilization   float64 // percentage
	MinQueueProcessingRate float64 // items per second
}

// DefaultPerformanceTargets returns default performance targets
func DefaultPerformanceTargets() *PerformanceTargets {
	return &PerformanceTargets{
		ResponseTimeP95Max:     5 * time.Second,
		DBQueryTimeMax:         100 * time.Millisecond,
		MaxQueueDepth:          1000,
		MinCertIssuanceRate:    100.0,
		MaxErrorRate:           0.01, // 1% error rate
		MaxCPUUtilization:      80.0,
		MaxMemoryUtilization:   80.0,
		MinQueueProcessingRate: 50.0,
	}
}

// RecoveryPerformanceTargets returns performance targets for recovery load tests
func RecoveryPerformanceTargets() *PerformanceTargets {
	return &PerformanceTargets{
		ResponseTimeP95Max:     10 * time.Second,
		DBQueryTimeMax:         200 * time.Millisecond,
		MaxQueueDepth:          1000,
		MinCertIssuanceRate:    50.0,
		MaxErrorRate:           0.01,
		MaxCPUUtilization:      80.0,
		MaxMemoryUtilization:   80.0,
		MinQueueProcessingRate: 25.0,
	}
}

// PerformanceAnalysis analyzes performance metrics against targets
type PerformanceAnalysis struct {
	Metrics *PerformanceMetrics
	Targets *PerformanceTargets
	Results []TargetResult
}

// TargetResult represents the result of comparing a metric against a target
type TargetResult struct {
	Metric  string
	Target  string
	Actual  string
	Passed  bool
	Message string
}

// Analyze compares performance metrics against targets
func Analyze(metrics *PerformanceMetrics, targets *PerformanceTargets) *PerformanceAnalysis {
	analysis := &PerformanceAnalysis{
		Metrics: metrics,
		Targets: targets,
		Results: make([]TargetResult, 0),
	}

	// Check response time P95
	analysis.Results = append(analysis.Results, TargetResult{
		Metric:  "Response Time P95",
		Target:  targets.ResponseTimeP95Max.String(),
		Actual:  metrics.ResponseTimeP95.String(),
		Passed:  metrics.ResponseTimeP95 <= targets.ResponseTimeP95Max,
		Message: fmt.Sprintf("Response time P95: %v (target: <= %v)", metrics.ResponseTimeP95, targets.ResponseTimeP95Max),
	})

	// Check database query time P95
	analysis.Results = append(analysis.Results, TargetResult{
		Metric:  "Database Query Time P95",
		Target:  targets.DBQueryTimeMax.String(),
		Actual:  metrics.DBQueryTimeP95.String(),
		Passed:  metrics.DBQueryTimeP95 <= targets.DBQueryTimeMax,
		Message: fmt.Sprintf("Database query time P95: %v (target: <= %v)", metrics.DBQueryTimeP95, targets.DBQueryTimeMax),
	})

	// Check queue depth
	analysis.Results = append(analysis.Results, TargetResult{
		Metric:  "Max Queue Depth",
		Target:  fmt.Sprintf("%d", targets.MaxQueueDepth),
		Actual:  fmt.Sprintf("%d", metrics.MaxQueueDepth),
		Passed:  metrics.MaxQueueDepth <= targets.MaxQueueDepth,
		Message: fmt.Sprintf("Max queue depth: %d (target: <= %d)", metrics.MaxQueueDepth, targets.MaxQueueDepth),
	})

	// Check certificate issuance rate
	analysis.Results = append(analysis.Results, TargetResult{
		Metric:  "Certificate Issuance Rate",
		Target:  fmt.Sprintf("%.2f/sec", targets.MinCertIssuanceRate),
		Actual:  fmt.Sprintf("%.2f/sec", metrics.CertIssuanceRate),
		Passed:  metrics.CertIssuanceRate >= targets.MinCertIssuanceRate,
		Message: fmt.Sprintf("Certificate issuance rate: %.2f/sec (target: >= %.2f/sec)", metrics.CertIssuanceRate, targets.MinCertIssuanceRate),
	})

	// Check error rate
	analysis.Results = append(analysis.Results, TargetResult{
		Metric:  "Error Rate",
		Target:  fmt.Sprintf("%.4f/sec", targets.MaxErrorRate),
		Actual:  fmt.Sprintf("%.4f/sec", metrics.ErrorRate),
		Passed:  metrics.ErrorRate <= targets.MaxErrorRate,
		Message: fmt.Sprintf("Error rate: %.4f/sec (target: <= %.4f/sec)", metrics.ErrorRate, targets.MaxErrorRate),
	})

	return analysis
}

// GenerateReport generates a performance report
func (pa *PerformanceAnalysis) GenerateReport() string {
	report := fmt.Sprintf(`
Performance Analysis Report
===========================

Performance Metrics:
%s

Performance Targets:
  Response Time P95: <= %v
  Database Query Time: <= %v
  Max Queue Depth: <= %d
  Min Certificate Issuance Rate: >= %.2f/sec
  Max Error Rate: <= %.4f/sec

Target Results:
`, pa.Metrics.String(), pa.Targets.ResponseTimeP95Max, pa.Targets.DBQueryTimeMax,
		pa.Targets.MaxQueueDepth, pa.Targets.MinCertIssuanceRate, pa.Targets.MaxErrorRate)

	passedCount := 0
	for _, result := range pa.Results {
		status := "❌ FAIL"
		if result.Passed {
			status = "✅ PASS"
			passedCount++
		}
		report += fmt.Sprintf("  %s: %s\n    %s\n", result.Metric, status, result.Message)
	}

	report += fmt.Sprintf("\nSummary: %d/%d targets passed\n", passedCount, len(pa.Results))

	if passedCount < len(pa.Results) {
		report += "\n⚠️  Some performance targets were not met. Consider investigating bottlenecks.\n"
	} else {
		report += "\n✅ All performance targets met!\n"
	}

	return report
}

// IdentifyBottlenecks identifies potential performance bottlenecks
func (pa *PerformanceAnalysis) IdentifyBottlenecks() []string {
	bottlenecks := make([]string, 0)

	if !pa.Results[0].Passed {
		bottlenecks = append(bottlenecks, "Response time exceeds target - service may be overloaded or database queries are slow")
	}

	if !pa.Results[1].Passed {
		bottlenecks = append(bottlenecks, "Database query time exceeds target - database may need optimization or connection pooling")
	}

	if !pa.Results[2].Passed {
		bottlenecks = append(bottlenecks, "Queue depth exceeds target - queue processing may be too slow or load too high")
	}

	if !pa.Results[3].Passed {
		bottlenecks = append(bottlenecks, "Certificate issuance rate below target - certificate signing may be a bottleneck")
	}

	if !pa.Results[4].Passed {
		bottlenecks = append(bottlenecks, "Error rate exceeds target - system may be experiencing failures under load")
	}

	return bottlenecks
}
