# Flight Control - Component Diagram and Interactions

> **Document Type:** Component Architecture  
> **Last Updated:** November 29, 2025

## Table of Contents

- [System Component Overview](#system-component-overview)
- [Component Responsibilities](#component-responsibilities)
- [Component Interactions](#component-interactions)
- [Data Flow Diagrams](#data-flow-diagrams)
- [Deployment Topology](#deployment-topology)
- [Network Architecture](#network-architecture)

---

## System Component Overview

### Service-Side Components

```
┌─────────────────────────────────────────────────────────────────────────┐
│                        Flight Control Service                           │
├─────────────────────────────────────────────────────────────────────────┤
│                                                                         │
│  ┌────────────────────┐     ┌────────────────────┐                    │
│  │   API Server       │     │   Worker Pool      │                    │
│  │                    │     │                    │                    │
│  │  ┌──────────────┐  │     │  ┌──────────────┐ │                    │
│  │  │ User API     │  │     │  │  Rollout     │ │                    │
│  │  │  (HTTPS)     │  │     │  │  Handler     │ │                    │
│  │  └──────────────┘  │     │  └──────────────┘ │                    │
│  │  ┌──────────────┐  │     │  ┌──────────────┐ │                    │
│  │  │ Agent API    │  │     │  │  Template    │ │                    │
│  │  │  (mTLS)      │  │     │  │  Handler     │ │                    │
│  │  └──────────────┘  │     │  └──────────────┘ │                    │
│  │  ┌──────────────┐  │     │  ┌──────────────┐ │                    │
│  │  │ WebSocket    │  │     │  │  Repository  │ │                    │
│  │  │  Handler     │  │     │  │  Handler     │ │                    │
│  │  └──────────────┘  │     │  └──────────────┘ │                    │
│  │  ┌──────────────┐  │     │  ┌──────────────┐ │                    │
│  │  │ Console      │  │     │  │  CSR         │ │                    │
│  │  │  Proxy       │  │     │  │  Handler     │ │                    │
│  │  └──────────────┘  │     │  └──────────────┘ │                    │
│  └────────────────────┘     └────────────────────┘                    │
│                                                                         │
│  ┌────────────────────┐     ┌────────────────────┐                    │
│  │ Periodic Checker   │     │ Alert Exporter     │                    │
│  │                    │     │                    │                    │
│  │  ┌──────────────┐  │     │  ┌──────────────┐ │                    │
│  │  │ Heartbeat    │  │     │  │ Prometheus   │ │                    │
│  │  │ Monitor      │  │     │  │ Exporter     │ │                    │
│  │  └──────────────┘  │     │  └──────────────┘ │                    │
│  │  ┌──────────────┐  │     │  ┌──────────────┐ │                    │
│  │  │ Repository   │  │     │  │ Alert        │ │                    │
│  │  │ Sync         │  │     │  │ Generator    │ │                    │
│  │  └──────────────┘  │     │  └──────────────┘ │                    │
│  │  ┌──────────────┐  │     └────────────────────┘                    │
│  │  │ Fleet        │  │                                                │
│  │  │ Reconciler   │  │     ┌────────────────────┐                    │
│  │  └──────────────┘  │     │ Auxiliary Services │                    │
│  └────────────────────┘     │                    │                    │
│                              │  • Userinfo Proxy  │                    │
│  ┌────────────────────┐     │  • PAM Issuer      │                    │
│  │ Data Layer         │     │  • Telemetry GW    │                    │
│  │                    │     │  • Alertmgr Proxy  │                    │
│  │  ┌──────────────┐  │     └────────────────────┘                    │
│  │  │ PostgreSQL   │  │                                                │
│  │  │ (Primary DB) │  │                                                │
│  │  └──────────────┘  │                                                │
│  │  ┌──────────────┐  │                                                │
│  │  │ Redis        │  │                                                │
│  │  │ (Queue/KV)   │  │                                                │
│  │  └──────────────┘  │                                                │
│  └────────────────────┘                                                │
└─────────────────────────────────────────────────────────────────────────┘
```

### Device-Side Components

```
┌─────────────────────────────────────────────────────────────────────────┐
│                           Edge Device                                   │
├─────────────────────────────────────────────────────────────────────────┤
│                                                                         │
│  ┌────────────────────────────────────────────────────────────────┐   │
│  │                    flightctl-agent                             │   │
│  ├────────────────────────────────────────────────────────────────┤   │
│  │                                                                 │   │
│  │  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐        │   │
│  │  │  Identity    │  │  Spec        │  │  Status      │        │   │
│  │  │  Provider    │  │  Manager     │  │  Manager     │        │   │
│  │  └──────────────┘  └──────────────┘  └──────────────┘        │   │
│  │                                                                 │   │
│  │  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐        │   │
│  │  │  OS          │  │  Config      │  │  Application │        │   │
│  │  │  Controller  │  │  Controller  │  │  Controller  │        │   │
│  │  └──────────────┘  └──────────────┘  └──────────────┘        │   │
│  │                                                                 │   │
│  │  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐        │   │
│  │  │  Resource    │  │  Certificate │  │  Console     │        │   │
│  │  │  Monitor     │  │  Manager     │  │  Manager     │        │   │
│  │  └──────────────┘  └──────────────┘  └──────────────┘        │   │
│  │                                                                 │   │
│  │  ┌──────────────┐  ┌──────────────┐                          │   │
│  │  │  Lifecycle   │  │  Hook        │                          │   │
│  │  │  Manager     │  │  Executor    │                          │   │
│  │  └──────────────┘  └──────────────┘                          │   │
│  └────────────────────────────────────────────────────────────────┘   │
│                                                                         │
│  ┌────────────────────────────────────────────────────────────────┐   │
│  │                   Operating System Layer                       │   │
│  ├────────────────────────────────────────────────────────────────┤   │
│  │                                                                 │   │
│  │  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐        │   │
│  │  │  bootc/      │  │  Podman      │  │  MicroShift  │        │   │
│  │  │  rpm-ostree  │  │              │  │  (optional)  │        │   │
│  │  └──────────────┘  └──────────────┘  └──────────────┘        │   │
│  │                                                                 │   │
│  │  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐        │   │
│  │  │  systemd     │  │  greenboot   │  │  filesystem  │        │   │
│  │  └──────────────┘  └──────────────┘  └──────────────┘        │   │
│  └────────────────────────────────────────────────────────────────┘   │
│                                                                         │
│  ┌────────────────────────────────────────────────────────────────┐   │
│  │                    Hardware Layer                              │   │
│  ├────────────────────────────────────────────────────────────────┤   │
│  │  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐        │   │
│  │  │  TPM 2.0     │  │  Network     │  │  Storage     │        │   │
│  │  └──────────────┘  └──────────────┘  └──────────────┘        │   │
│  └────────────────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────────────┘
```

---

## Component Responsibilities

### API Server

**Primary Functions**:
- Accept and validate HTTP/HTTPS requests
- Authenticate users (JWT) and agents (mTLS)
- Authorize requests based on RBAC/SpiceDB
- CRUD operations for all resources
- Real-time updates via WebSocket
- Device console proxy (SSH over WebSocket)

**Key Interfaces**:
- `Server.Start()` - Start HTTP servers
- `Server.HandleUserRequest()` - Process user API calls
- `Server.HandleAgentRequest()` - Process agent API calls
- `Server.HandleWebSocket()` - Manage WebSocket connections

**Dependencies**:
- PostgreSQL (data persistence)
- Redis (pub/sub, caching)
- External auth providers (Keycloak, OIDC)
- External authz providers (SpiceDB, K8s RBAC)

### Worker Pool

**Primary Functions**:
- Process asynchronous tasks from queue
- Execute rollout operations
- Handle template version changes
- Process repository synchronization
- Manage CSR approvals
- Execute resource sync operations

**Task Types**:
- `RolloutTask` - Fleet rollout execution
- `TemplateVersionTask` - Template changes
- `RepositorySyncTask` - Git repository sync
- `CSRApprovalTask` - Certificate signing
- `ResourceSyncTask` - GitOps sync

**Scaling**:
- Horizontal: Add more worker instances
- Task-specific: Separate workers for different task types
- Priority: High-priority queue for critical tasks

### Periodic Checker

**Primary Functions**:
- Monitor device heartbeats
- Detect stale devices
- Reconcile repository updates
- Trigger periodic fleet reconciliation
- Clean up expired resources
- Generate health alerts

**Schedules**:
- Device heartbeat check: Every 1 minute
- Repository sync: Every 5 minutes (configurable)
- Fleet reconciliation: Every 10 minutes
- Certificate expiry check: Every 1 hour
- Cleanup tasks: Daily

### Device Agent

**Primary Functions**:
- Enroll device with TPM identity
- Poll for configuration updates
- Apply OS updates (bootc/rpm-ostree)
- Manage configuration files
- Deploy and monitor applications
- Report device status
- Manage certificate lifecycle
- Provide console access

**Controllers**:
- `OSController` - OS image updates
- `ConfigController` - File-based configuration
- `ApplicationController` - Container workloads
- `ResourceMonitor` - CPU, memory, disk usage
- `CertificateManager` - Certificate rotation
- `ConsoleManager` - Remote access

### Alert Exporter

**Primary Functions**:
- Convert device conditions to Prometheus alerts
- Monitor fleet health metrics
- Export metrics to Prometheus
- Generate alerts for Alertmanager

**Alert Types**:
- Device offline alerts
- Update failure alerts
- Application failure alerts
- Certificate expiry alerts
- Fleet rollout alerts

---

## Component Interactions

### User Request Flow

```
┌──────────┐
│   User   │
└────┬─────┘
     │ 1. HTTP Request (JWT)
     ▼
┌──────────────────────────────────────┐
│         API Server                    │
├──────────────────────────────────────┤
│  2. Validate JWT                     │
│  3. Check Authorization (RBAC)       │
│  4. Validate Request (OpenAPI)       │
└────┬─────────────────────────────────┘
     │ 5. Store Operation
     ▼
┌──────────────┐
│  PostgreSQL  │
│              │
│  6. Write    │
└────┬─────────┘
     │ 7. Return Result
     ▼
┌──────────────────────────────────────┐
│         API Server                    │
│  8. Enqueue async task (optional)    │
└────┬─────────────────────────────────┘
     │ 9. Publish to queue
     ▼
┌──────────────┐
│    Redis     │
│   (Queue)    │
└──────────────┘
```

### Agent Sync Flow

```
┌──────────────┐
│    Agent     │
│              │
│  1. Poll for │
│     spec     │
└────┬─────────┘
     │ 2. mTLS Request
     ▼
┌──────────────────────────────────────┐
│         API Server                    │
├──────────────────────────────────────┤
│  3. Validate mTLS cert               │
│  4. Extract device identity          │
└────┬─────────────────────────────────┘
     │ 5. Fetch spec
     ▼
┌──────────────┐
│  PostgreSQL  │
│              │
│  6. Render   │
│     spec     │
└────┬─────────┘
     │ 7. Return rendered spec
     ▼
┌──────────────────────────────────────┐
│         API Server                    │
│  8. Cache in Redis (optional)        │
└────┬─────────────────────────────────┘
     │ 9. Return to agent
     ▼
┌──────────────┐
│    Agent     │
│              │
│  10. Apply   │
│      changes │
└────┬─────────┘
     │ 11. Report status
     ▼
┌──────────────────────────────────────┐
│         API Server                    │
│  12. Update status in DB             │
└──────────────────────────────────────┘
```

### Fleet Rollout Flow

```
┌──────────┐
│   User   │
│          │
│  1. Update│
│    fleet  │
└────┬─────┘
     │ 2. HTTP Request
     ▼
┌──────────────────────────────────────┐
│         API Server                    │
│  3. Validate & store                 │
│  4. Increment template version       │
└────┬─────────────────────────────────┘
     │ 5. Enqueue rollout task
     ▼
┌──────────────┐
│    Redis     │
│   (Queue)    │
└────┬─────────┘
     │ 6. Dequeue task
     ▼
┌──────────────────────────────────────┐
│         Worker                        │
├──────────────────────────────────────┤
│  7. Get fleet details                │
│  8. Get matching devices             │
│  9. Calculate batches                │
│ 10. Update devices in batches        │
└────┬─────────────────────────────────┘
     │ 11. Update each device
     ▼
┌──────────────┐
│  PostgreSQL  │
│              │
│ 12. Store    │
│    updates   │
└────┬─────────┘
     │ 13. Publish notifications
     ▼
┌──────────────┐
│    Redis     │
│  (Pub/Sub)   │
└────┬─────────┘
     │ 14. Agents poll
     ▼
┌──────────────┐
│   Agents     │
│              │
│ 15. Apply    │
│    updates   │
└──────────────┘
```

### Device Enrollment Flow

```
┌──────────────┐
│    Agent     │
│              │
│  1. Generate │
│   TPM key    │
└────┬─────────┘
     │ 2. Create CSR with TPM attestation
     ▼
┌──────────────────────────────────────┐
│         API Server                    │
│  3. Receive enrollment request       │
│  4. Store in DB                      │
└────┬─────────────────────────────────┘
     │ 5. Enqueue approval task
     ▼
┌──────────────┐
│    Redis     │
│   (Queue)    │
└────┬─────────┘
     │ 6. Dequeue
     ▼
┌──────────────────────────────────────┐
│         Worker                        │
│  7. Verify TPM attestation           │
│  8. Auto-approve or wait for admin   │
└────┬─────────────────────────────────┘
     │ 9. Approval decision
     ▼
┌──────────────┐
│  PostgreSQL  │
│              │
│ 10. Update   │
│     status   │
└────┬─────────┘
     │ 11. Create CSR
     ▼
┌──────────────────────────────────────┐
│         Worker                        │
│ 12. Sign certificate (CA)            │
└────┬─────────────────────────────────┘
     │ 13. Store certificate
     ▼
┌──────────────┐
│  PostgreSQL  │
└────┬─────────┘
     │ 14. Agent polls for certificate
     ▼
┌──────────────┐
│    Agent     │
│              │
│ 15. Store in │
│     TPM      │
└──────────────┘
```

---

## Data Flow Diagrams

### Configuration Update Data Flow

```
Git Repository
     │
     │ 1. Commit new config
     ▼
┌──────────────────────────────────────┐
│    Periodic Checker                   │
│  2. Poll repository                  │
│  3. Detect changes                   │
└────┬─────────────────────────────────┘
     │ 4. Enqueue sync task
     ▼
┌──────────────┐
│    Redis     │
└────┬─────────┘
     │ 5. Dequeue
     ▼
┌──────────────────────────────────────┐
│         Worker                        │
│  6. Fetch config from Git            │
│  7. Parse and validate               │
│  8. Update fleet/device specs        │
└────┬─────────────────────────────────┘
     │ 9. Store in DB
     ▼
┌──────────────┐
│  PostgreSQL  │
└────┬─────────┘
     │ 10. Notify changes
     ▼
┌──────────────────────────────────────┐
│         Agent (on device)            │
│ 11. Poll for changes                 │
│ 12. Download config                  │
│ 13. Apply configuration              │
│ 14. Report status                    │
└──────────────────────────────────────┘
```

### Status Reporting Data Flow

```
┌──────────────┐
│    Agent     │
│              │
│  1. Collect  │
│    metrics   │
└────┬─────────┘
     │ 2. Build status update
     ▼
┌──────────────────────────────────────┐
│    Agent Status Manager              │
│  3. Compare with last status         │
│  4. Send if changed                  │
└────┬─────────────────────────────────┘
     │ 5. mTLS POST /status
     ▼
┌──────────────────────────────────────┐
│         API Server                    │
│  6. Validate request                 │
│  7. Update device status             │
└────┬─────────────────────────────────┘
     │ 8. Store in DB
     ▼
┌──────────────┐
│  PostgreSQL  │
└────┬─────────┘
     │ 9. Check conditions
     ▼
┌──────────────────────────────────────┐
│      Alert Exporter                  │
│ 10. Convert to Prometheus metrics    │
│ 11. Generate alerts                  │
└────┬─────────────────────────────────┘
     │ 12. Export metrics
     ▼
┌──────────────┐
│  Prometheus  │
└────┬─────────┘
     │ 13. Trigger alerts
     ▼
┌──────────────┐
│ Alertmanager │
└──────────────┘
```

---

## Deployment Topology

### Kubernetes Deployment

```
┌─────────────────────────────────────────────────────────────────┐
│                    Kubernetes Cluster                           │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  ┌────────────────────────────────────────────────────────┐   │
│  │  Namespace: flightctl                                  │   │
│  ├────────────────────────────────────────────────────────┤   │
│  │                                                         │   │
│  │  ┌──────────────────┐        ┌──────────────────┐    │   │
│  │  │  API Server      │        │  Workers         │    │   │
│  │  │  Deployment      │        │  Deployment      │    │   │
│  │  │  (3 replicas)    │        │  (5 replicas)    │    │   │
│  │  └────────┬─────────┘        └─────────┬────────┘    │   │
│  │           │                             │             │   │
│  │           └─────────────┬───────────────┘             │   │
│  │                         │                             │   │
│  │           ┌─────────────┴─────────────┐               │   │
│  │           ▼                           ▼               │   │
│  │  ┌──────────────────┐        ┌──────────────────┐    │   │
│  │  │  PostgreSQL      │        │  Redis           │    │   │
│  │  │  StatefulSet     │        │  StatefulSet     │    │   │
│  │  │  (with PVC)      │        │  (with PVC)      │    │   │
│  │  └──────────────────┘        └──────────────────┘    │   │
│  │                                                       │   │
│  │  ┌──────────────────┐        ┌──────────────────┐    │   │
│  │  │  Periodic        │        │  Alert Exporter  │    │   │
│  │  │  Deployment      │        │  Deployment      │    │   │
│  │  │  (1 replica)     │        │  (1 replica)     │    │   │
│  │  └──────────────────┘        └──────────────────┘    │   │
│  │                                                       │   │
│  │  ┌──────────────────────────────────────────────┐    │   │
│  │  │  Ingress / LoadBalancer                      │    │   │
│  │  │  (External access to API)                    │    │   │
│  │  └──────────────────────────────────────────────┘    │   │
│  └────────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────┘
```

### Linux Systemd Deployment

```
┌─────────────────────────────────────────────────────────────────┐
│                    Linux Host                                   │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  ┌────────────────────────────────────────────────────────┐   │
│  │  systemd Services                                      │   │
│  ├────────────────────────────────────────────────────────┤   │
│  │                                                         │   │
│  │  flightctl-api.service                                 │   │
│  │  flightctl-worker.service (x5)                         │   │
│  │  flightctl-periodic.service                            │   │
│  │  flightctl-alert-exporter.service                      │   │
│  │                                                         │   │
│  └────────────────────────────────────────────────────────┘   │
│                                                                 │
│  ┌────────────────────────────────────────────────────────┐   │
│  │  Container Services (Podman Quadlets)                  │   │
│  ├────────────────────────────────────────────────────────┤   │
│  │                                                         │   │
│  │  postgresql.service                                    │   │
│  │  redis.service                                         │   │
│  │                                                         │   │
│  └────────────────────────────────────────────────────────┘   │
│                                                                 │
│  ┌────────────────────────────────────────────────────────┐   │
│  │  Storage                                               │   │
│  ├────────────────────────────────────────────────────────┤   │
│  │                                                         │   │
│  │  /var/lib/flightctl/                                   │   │
│  │  /etc/flightctl/                                       │   │
│  │  /var/lib/containers/storage/volumes/                 │   │
│  │                                                         │   │
│  └────────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────┘
```

---

## Network Architecture

### Network Communication Patterns

```
┌──────────────────────────────────────────────────────────────────┐
│                    External Networks                             │
├──────────────────────────────────────────────────────────────────┤
│                                                                  │
│  ┌─────────────┐    ┌─────────────┐    ┌─────────────┐        │
│  │  User       │    │  Git Repos  │    │  Container  │        │
│  │  Clients    │    │             │    │  Registries │        │
│  └──────┬──────┘    └──────┬──────┘    └──────┬──────┘        │
│         │ HTTPS            │ HTTPS            │ HTTPS          │
│         │                  │                  │                │
└─────────┼──────────────────┼──────────────────┼────────────────┘
          │                  │                  │
          ▼                  ▼                  ▼
┌──────────────────────────────────────────────────────────────────┐
│                    Service Network                               │
├──────────────────────────────────────────────────────────────────┤
│                                                                  │
│  ┌──────────────────────────────────────────────────────────┐  │
│  │  API Server                                              │  │
│  │  - Port 3333 (User API, HTTPS)                          │  │
│  │  - Port 7443 (Agent API, mTLS)                          │  │
│  │  - Port 8080 (Metrics)                                  │  │
│  └──────────────────────────────────────────────────────────┘  │
│            │              │              │                      │
│            │ PostgreSQL   │ Redis        │ External APIs        │
│            │ :5432        │ :6379        │ (Auth, Cert, etc)    │
│            ▼              ▼              ▼                      │
│  ┌────────────┐    ┌────────────┐    ┌────────────┐          │
│  │ PostgreSQL │    │   Redis    │    │  External  │          │
│  │            │    │            │    │  Services  │          │
│  └────────────┘    └────────────┘    └────────────┘          │
│                                                                  │
└──────────────────────────────────────────────────────────────────┘
          │
          │ mTLS (Port 7443)
          │
┌─────────┼────────────────────────────────────────────────────────┐
│         ▼             Device Network                             │
├──────────────────────────────────────────────────────────────────┤
│                                                                  │
│  ┌────────────┐    ┌────────────┐    ┌────────────┐           │
│  │  Device 1  │    │  Device 2  │    │  Device N  │           │
│  │            │    │            │    │            │           │
│  │  Agent     │    │  Agent     │    │  Agent     │           │
│  │  :7443→    │    │  :7443→    │    │  :7443→    │           │
│  └────────────┘    └────────────┘    └────────────┘           │
│                                                                  │
└──────────────────────────────────────────────────────────────────┘
```

### Port Usage Summary

| Component | Port | Protocol | Purpose |
|-----------|------|----------|---------|
| API Server (User) | 3333 | HTTPS | User API |
| API Server (Agent) | 7443 | mTLS | Agent API |
| API Server (Metrics) | 8080 | HTTP | Prometheus metrics |
| PostgreSQL | 5432 | TCP | Database |
| Redis | 6379 | TCP | Queue/KV store |
| Worker (Metrics) | 8081 | HTTP | Prometheus metrics |
| Periodic (Metrics) | 8082 | HTTP | Prometheus metrics |
| Alert Exporter | 8083 | HTTP | Prometheus metrics |

---

## Appendix

### Component Dependencies

**API Server depends on**:
- PostgreSQL (required)
- Redis (required)
- External auth provider (optional)
- External authz provider (optional)
- Certificate authority (optional, can use built-in)

**Worker depends on**:
- PostgreSQL (required)
- Redis (required)
- API Server (indirectly via DB)

**Periodic Checker depends on**:
- PostgreSQL (required)
- API Server (indirectly via DB)

**Agent depends on**:
- API Server (required)
- TPM 2.0 (optional but recommended)
- bootc/rpm-ostree (required)
- Podman (required)

### Component Scaling Guidelines

| Component | Scaling Type | Trigger | Max Recommended |
|-----------|--------------|---------|-----------------|
| API Server | Horizontal | CPU > 70%, Request rate | 10 replicas |
| Workers | Horizontal | Queue depth > 100 | 20 replicas |
| Periodic | Vertical only | N/A | 1 replica |
| PostgreSQL | Vertical first | Connection count, CPU | 1 primary + replicas |
| Redis | Vertical first | Memory usage | 1 instance or cluster |

### References

- [Architecture Overview](ARCHITECTURE.md)
- [Technical Design](TECHNICAL_DESIGN.md)
- [Deployment Guide](user/installing/installing-service-on-kubernetes.md)

