# Story: Certificate Renewal Configuration Schema

**Story ID:** EDM-323-EPIC-1-STORY-3  
**Epic:** EDM-323-EPIC-1 (Certificate Lifecycle Foundation)  
**Feature:** EDM-323 (Agent Certificate Rotation)  
**Story Points:** 3  
**Priority:** High

## Description

**As a** fleet administrator  
**I want** to configure certificate renewal behavior  
**So that** I can customize renewal thresholds and retry policies for my fleet

## Acceptance Criteria

Given agent configuration  
When renewal settings are specified  
Then the agent accepts and validates renewal configuration

Given renewal configuration  
When renewal threshold is set  
Then the threshold is used to determine when to trigger renewal (default: 30 days)

Given renewal configuration  
When check interval is set  
Then the agent checks expiration at the specified interval (default: 24h)

Given renewal configuration  
When retry policy is set  
Then the agent uses the specified retry intervals and max attempts

Given invalid renewal configuration  
When the agent starts  
Then the agent rejects invalid configuration with clear error messages

## Technical Details

### Components to Create/Modify
- `internal/agent/config/config.go`
  - Add `CertificateRenewalConfig` struct
  - Add validation for renewal settings
  - Add default values

### Configuration Schema
```yaml
certificate:
  renewal:
    enabled: true
    threshold_days: 30
    check_interval: 24h
    retry_interval: 1h
    max_retries: 10
    backoff_multiplier: 2.0
    max_backoff: 24h
```

### Validation Rules
- `threshold_days` must be > 0 and < 365
- `check_interval` must be > 0
- `retry_interval` must be > 0
- `max_retries` must be >= 0
- `backoff_multiplier` must be >= 1.0

## Dependencies
- None (can be done in parallel with other stories)

## Testing Requirements
- Unit tests for configuration parsing
- Unit tests for configuration validation
- Unit tests for default values
- Integration test with agent startup

## Definition of Done
- [ ] Configuration schema implemented
- [ ] Configuration validation implemented
- [ ] Unit tests written and passing
- [ ] Code reviewed and approved
- [ ] Configuration documented

