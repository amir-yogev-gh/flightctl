# Story: Comprehensive Certificate Metrics

**Story ID:** EDM-323-EPIC-5-STORY-1  
**Epic:** EDM-323-EPIC-5 (Configuration and Observability)  
**Feature:** EDM-323 (Agent Certificate Rotation)  
**Story Points:** 5  
**Priority:** Medium

## Description

**As a** fleet administrator  
**I want** comprehensive metrics for certificate lifecycle  
**So that** I can monitor certificate health across my fleet

## Acceptance Criteria

Given a device with certificates  
When metrics are exposed  
Then certificate expiration timestamps are available as metrics

Given a device with certificates  
When metrics are exposed  
Then days until expiration are available as metrics

Given certificate renewal operations  
When metrics are exposed  
Then renewal attempts, successes, and failures are tracked

Given certificate recovery operations  
When metrics are exposed  
Then recovery attempts, successes, and failures are tracked

Given certificate operations  
When metrics are exposed  
Then operation duration is tracked

Given metrics  
When exposed via Prometheus  
Then all metrics follow Prometheus naming conventions

## Technical Details

### Components to Create/Modify
- `internal/agent/instrumentation/metrics/` (new or extend)
  - Certificate expiration metrics
  - Renewal operation metrics
  - Recovery operation metrics
  - Duration metrics

### Metrics to Expose
**Agent Metrics:**
- `flightctl_agent_certificate_expiration_timestamp{type="management|bootstrap"}`
- `flightctl_agent_certificate_days_until_expiration{type="management|bootstrap"}`
- `flightctl_agent_certificate_renewal_attempts_total{reason="proactive|expired"}`
- `flightctl_agent_certificate_renewal_success_total{reason="proactive|expired"}`
- `flightctl_agent_certificate_renewal_failures_total{reason="proactive|expired"}`
- `flightctl_agent_certificate_recovery_attempts_total`
- `flightctl_agent_certificate_recovery_success_total`
- `flightctl_agent_certificate_recovery_failures_total`
- `flightctl_agent_certificate_renewal_duration_seconds`

**Service Metrics:**
- `flightctl_service_certificate_renewal_requests_total{reason="proactive|expired"}`
- `flightctl_service_certificate_renewal_issued_total{reason="proactive|expired"}`
- `flightctl_service_certificate_renewal_rejected_total{reason="proactive|expired"}`
- `flightctl_service_certificate_validation_failures_total{reason="attestation|fingerprint|revoked"}`
- `flightctl_service_certificate_renewal_duration_seconds`

## Dependencies
- EDM-323-EPIC-2 (Proactive Renewal)
- EDM-323-EPIC-4 (Expired Recovery)
- Existing metrics infrastructure

## Testing Requirements
- Unit tests for metrics collection
- Integration tests for metrics exposure
- Test metrics are correctly labeled
- Test metrics are updated correctly

## Definition of Done
- [ ] All metrics implemented
- [ ] Unit tests written and passing
- [ ] Code reviewed and approved
- [ ] Integration tests passing
- [ ] Metrics documented
- [ ] Documentation updated

