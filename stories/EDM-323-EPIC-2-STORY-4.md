# Story: Certificate Issuance for Renewals

**Story ID:** EDM-323-EPIC-2-STORY-4  
**Epic:** EDM-323-EPIC-2 (Proactive Certificate Renewal)  
**Feature:** EDM-323 (Agent Certificate Rotation)  
**Story Points:** 3  
**Priority:** High

## Description

**As a** Flight Control service  
**I want** to issue new certificates for approved renewal requests  
**So that** devices receive valid replacement certificates

## Acceptance Criteria

Given an approved renewal CSR  
When the CSR is signed  
Then the service issues a new certificate with standard validity (365 days)

Given a renewal certificate  
When the certificate is issued  
Then it has the same device identity (CN, SAN) as the original certificate

Given a renewal certificate  
When the certificate is issued  
Then it is signed by the same CA as the original certificate

Given a renewal certificate issuance  
When the certificate is issued  
Then the device certificate tracking fields are updated in the database

## Technical Details

### Components to Create/Modify
- `internal/service/certificatesigningrequest.go`
  - `signApprovedCertificateSigningRequest()` (enhance for renewals)
  - Update device certificate tracking after issuance

### Certificate Signing
- Use existing certificate signer
- Issue certificate with 365-day validity
- Preserve device identity from original certificate
- Update device record with new expiration date

### Database Updates
- Update `certificate_expiration` field
- Update `certificate_last_renewed` timestamp
- Increment `certificate_renewal_count`

## Dependencies
- EDM-323-EPIC-2-STORY-3 (Renewal Validation)
- Existing certificate signing infrastructure

## Testing Requirements
- Unit tests for certificate issuance
- Unit tests for database updates
- Integration tests for full renewal flow
- Test certificate identity preservation

## Definition of Done
- [ ] Certificate issuance for renewals implemented
- [ ] Database updates implemented
- [ ] Unit tests written and passing (>80% coverage)
- [ ] Code reviewed and approved
- [ ] Integration tests passing
- [ ] Documentation updated

