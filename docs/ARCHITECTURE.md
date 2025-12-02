# Flight Control - Architecture Documentation

> **Document Version:** 1.0  
> **Last Updated:** November 29, 2025  
> **Status:** Beta

## Table of Contents

- [Executive Overview](#executive-overview)
- [System Architecture](#system-architecture)
- [Core Components](#core-components)
- [Data Model](#data-model)
- [API Architecture](#api-architecture)
- [Security Architecture](#security-architecture)
- [Deployment Architecture](#deployment-architecture)
- [Technology Stack](#technology-stack)
- [Design Patterns](#design-patterns)
- [Development Workflow](#development-workflow)

---

## Executive Overview

Flight Control is a **declarative edge device management system** designed for large-scale fleet management. It provides a GitOps-compatible control plane for managing thousands of edge devices running container workloads on image-based Linux operating systems.

### Key Design Principles

1. **Declarative by Default** - All device configuration is declarative and version-controlled
2. **Agent-Based** - Autonomous agents on devices handle updates and report status
3. **Secure by Design** - TPM-backed device identity, mTLS communication, hardware root of trust
4. **Scale-First** - Designed to manage fleets of 10,000+ devices efficiently
5. **GitOps-Native** - APIs follow Kubernetes patterns for seamless GitOps integration
6. **Resilient** - Designed for unreliable networks, transactional updates with rollback

### Core Value Proposition

- **For Operators**: Manage thousands of edge devices with GitOps workflows
- **For Developers**: Kubernetes-like APIs without Kubernetes complexity
- **For Security Teams**: Hardware-backed device identity and attestation
- **For Edge Deployments**: Works under adverse network conditions

---

## System Architecture

### High-Level Architecture

```
┌─────────────────────────────────────────────────────────────────────┐
│                         User/Admin Layer                            │
├─────────────────────────────────────────────────────────────────────┤
│  CLI (flightctl) │ Web UI (Console) │ GitOps Tools │ External APIs │
└────────────┬────────────────────────────────────────────────────────┘
             │
             ▼ HTTPS + JWT (User API)
┌─────────────────────────────────────────────────────────────────────┐
│                     Flight Control Service                          │
├─────────────────────────────────────────────────────────────────────┤
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐            │
│  │  API Server  │  │   Workers    │  │  Periodic    │            │
│  │              │  │              │  │   Checker    │            │
│  └──────┬───────┘  └──────┬───────┘  └──────┬───────┘            │
│         │                 │                  │                     │
│         └─────────────────┴──────────────────┘                     │
│                           │                                         │
│         ┌─────────────────┴─────────────────┐                     │
│         ▼                                    ▼                     │
│  ┌─────────────┐                    ┌──────────────┐              │
│  │ PostgreSQL  │                    │ Redis (KV)   │              │
│  │  Database   │                    │  + Queues    │              │
│  └─────────────┘                    └──────────────┘              │
└─────────────────────────────────────────────────────────────────────┘
             │
             ▼ mTLS (Agent API)
┌─────────────────────────────────────────────────────────────────────┐
│                         Edge Devices                                │
├─────────────────────────────────────────────────────────────────────┤
│  ┌────────────────────────────────────────────────────────────┐   │
│  │                   flightctl-agent                          │   │
│  ├────────────────────────────────────────────────────────────┤   │
│  │ TPM Module │ Identity Mgmt │ Spec Mgmt │ Status Reporter  │   │
│  └────────────────────────────────────────────────────────────┘   │
│  ┌────────────────────────────────────────────────────────────┐   │
│  │          Image-based OS (bootc/rpm-ostree)                 │   │
│  │  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐    │   │
│  │  │   Podman     │  │  MicroShift  │  │  Systemd     │    │   │
│  │  └──────────────┘  └──────────────┘  └──────────────┘    │   │
│  └────────────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────────┘
```

### Component Interaction Flow

```
1. User declares desired state → API Server
2. API Server validates → Store in PostgreSQL
3. Workers process changes → Enqueue tasks (Redis)
4. Agent polls for changes → API Server (mTLS)
5. Agent downloads artifacts → Container Registries / Git Repos
6. Agent applies changes → OS, Config, Applications
7. Agent reports status → API Server
8. Periodic checker monitors → Trigger reconciliation
```

---

## Core Components

### 1. API Server (`flightctl-api`)

**Purpose**: Primary interface for all user and agent interactions

**Responsibilities**:
- Handle user authentication (JWT) and agent authentication (mTLS)
- Expose RESTful APIs for resource management
- Validate incoming requests against OpenAPI spec
- Store resources in PostgreSQL
- Enqueue work items for asynchronous processing
- Serve WebSocket connections for real-time updates
- Provide device console access via SSH proxy

**Key Interfaces**:
```go
// Main server structure
type Server struct {
    config      *config.Config
    store       store.Store
    ca          *crypto.CA
    authProvider authn.Provider
    queueProvider queues.Provider
}
```

**Endpoints**:
- `/api/v1/*` - RESTful API (user-facing)
- `/api/v1/agent/*` - Agent API (mTLS-protected)
- `/auth/*` - Authentication endpoints
- `/console/*` - Web console
- `/metrics` - Prometheus metrics

**Configuration**:
- Listen address and ports (separate for user/agent APIs)
- TLS certificates
- Database connection
- Authentication provider settings
- Rate limiting configuration

### 2. Worker (`flightctl-worker`)

**Purpose**: Asynchronous task processing and background operations

**Responsibilities**:
- Process rollout tasks (fleet updates)
- Handle template version changes
- Manage device enrollment approvals
- Process CSR (Certificate Signing Request) approvals
- Handle repository synchronization
- Execute resource sync operations
- Manage device lifecycle operations

**Architecture**:
```
Worker Pool (Configurable Size)
    ↓
Task Queue (Redis Streams)
    ↓
Task Handlers
    ├─ Rollout Handler
    ├─ Template Handler
    ├─ Repository Handler
    ├─ CSR Handler
    └─ Resource Sync Handler
```

**Key Features**:
- Horizontal scaling support
- Task priority management
- Retry logic with exponential backoff
- Dead letter queue for failed tasks
- Distributed locking (Redis)

### 3. Periodic Checker (`flightctl-periodic`)

**Purpose**: Time-based reconciliation and monitoring

**Responsibilities**:
- Monitor device heartbeats
- Check for stale device status
- Reconcile repository updates
- Trigger periodic rollouts
- Clean up expired certificates
- Monitor fleet rollout progress
- Generate alerts for unhealthy devices

**Scheduling**:
- Configurable cron-like schedules
- Independent goroutines per task
- Graceful shutdown handling

### 4. Agent (`flightctl-agent`)

**Purpose**: On-device management and reconciliation

**Responsibilities**:
- **Enrollment**: Bootstrap device identity using TPM
- **Spec Management**: Fetch and cache device configuration
- **OS Updates**: Coordinate with bootc/rpm-ostree
- **Config Management**: Write configuration files
- **Application Management**: Deploy containers (Podman/MicroShift)
- **Status Reporting**: Report device and application health
- **Certificate Management**: Rotate certificates before expiry
- **Console Access**: Provide SSH-based remote access

**Agent Architecture**:
```go
type Agent struct {
    // Core services
    identityProvider  *identity.Provider    // TPM-backed identity
    specManager       *spec.Manager         // Desired state
    statusManager     *status.Manager       // Current state
    
    // Controllers (run in parallel)
    osController      *os.Controller        // OS updates
    configController  *config.Controller    // File configuration
    appController     *applications.Controller // Containers
    resourceMonitor   *resource.Monitor     // System resources
    consoleManager    *console.Manager      // Remote access
    
    // Lifecycle
    lifecycleManager  *lifecycle.Manager    // Hooks and events
}
```

**State Machine**:
```
┌─────────────┐
│ Uninitialized│
└──────┬──────┘
       │
       ▼
┌─────────────┐      ┌──────────────┐
│  Enrolling  │─────▶│  Enrollment  │
│             │      │   Failed     │
└──────┬──────┘      └──────────────┘
       │
       ▼
┌─────────────┐
│   Enrolled  │◀──────┐
└──────┬──────┘       │
       │              │
       ▼              │
┌─────────────┐       │
│   Syncing   │       │
└──────┬──────┘       │
       │              │
       ▼              │
┌─────────────┐       │
│   Updating  │───────┘
└──────┬──────┘
       │
       ▼
┌─────────────┐
│    Idle     │
└─────────────┘
```

### 5. Additional Components

#### Alert Exporter (`flightctl-alert-exporter`)
- Converts device conditions to Prometheus alerts
- Integrates with Alertmanager
- Monitors fleet health metrics

#### Alertmanager Proxy (`flightctl-alertmanager-proxy`)
- Forwards alerts to external Alertmanager
- Handles alert routing and deduplication

#### Userinfo Proxy (`flightctl-userinfo-proxy`)
- Provides user information from external identity providers
- Caches user data for performance

#### PAM Issuer (`flightctl-pam-issuer`)
- Issues tokens for privileged access management
- Integrates with Red Hat Ansible Automation Platform

#### Telemetry Gateway (`flightctl-telemetry-gateway`)
- Collects device telemetry
- Forwards to observability platforms

#### DB Migrate (`flightctl-db-migrate`)
- Database schema migrations
- Version management
- Safe upgrade paths

#### Device Simulator (`devicesimulator`)
- Simulate thousands of devices for testing
- Load testing and performance validation
- Development aid

---

## Data Model

### Core Resources

#### Device
```yaml
apiVersion: v1beta1
kind: Device
metadata:
  name: device-001
  labels:
    location: factory-berlin
    role: forklift
spec:
  os:
    image: quay.io/flightctl/edge-os:latest
  config:
    - name: system-config
      configType: GitConfigProviderSpec
      gitRef:
        repository: config-repo
        path: configs/system.yaml
  applications:
    - name: workload
      type: container
      image: quay.io/app/workload:v1.2
  hooks:
    before:
      - actions:
          - type: command
            command: ["/usr/bin/backup.sh"]
status:
  updated: true
  os:
    image: quay.io/flightctl/edge-os:latest
    bootedImage: quay.io/flightctl/edge-os:latest
  applications:
    - name: workload
      status: Running
  systemInfo:
    architecture: x86_64
    bootID: "abc-123"
  resources:
    cpu: 45
    memory: 2048
    disk: 50000
```

#### Fleet
```yaml
apiVersion: v1beta1
kind: Fleet
metadata:
  name: factory-forklifts
spec:
  selector:
    matchLabels:
      location: factory-berlin
      role: forklift
  template:
    metadata:
      labels:
        managed-by: fleet
    spec:
      os:
        image: quay.io/flightctl/edge-os:stable
      applications:
        - name: workload
          type: container
          image: quay.io/app/workload:stable
  policy:
    rolloutPolicy:
      maxUnavailable: 10%
      maxSurge: 0
```

#### Repository
```yaml
apiVersion: v1beta1
kind: Repository
metadata:
  name: config-repo
spec:
  type: git
  url: https://github.com/org/configs.git
  ref:
    branch: main
  credentialsSecretRef:
    name: git-credentials
```

#### ResourceSync
```yaml
apiVersion: v1beta1
kind: ResourceSync
metadata:
  name: fleet-definitions
spec:
  repository: config-repo
  path: fleets/
```

### Database Schema

**Key Tables**:
- `devices` - Device inventory and specifications
- `fleets` - Fleet definitions and templates
- `template_versions` - Versioned fleet templates
- `repositories` - Git repository configurations
- `resource_syncs` - GitOps synchronization
- `enrollment_requests` - Device enrollment requests
- `certificate_signing_requests` - CSR approvals
- `events` - Audit trail and event log
- `organizations` - Multi-tenancy support
- `auth_providers` - Authentication configuration

**Indexes**:
- Device labels (JSONB GIN index)
- Device owner (B-tree)
- Fleet selectors
- Timestamps for efficient queries

---

## API Architecture

### API Design Principles

1. **Kubernetes-Compatible**: Follow K8s API conventions
2. **RESTful**: Standard HTTP methods (GET, POST, PUT, PATCH, DELETE)
3. **OpenAPI-First**: API-first design with OpenAPI 3.0 spec
4. **Versioned**: `v1beta1` with clear upgrade paths
5. **Field/Label Selectors**: Powerful filtering capabilities

### API Structure

```
/api/v1beta1/
├── devices
│   ├── {name}
│   ├── {name}/status
│   └── {name}/rendered
├── fleets
│   └── {name}
├── repositories
│   └── {name}
├── resourcesyncs
│   └── {name}
├── enrollmentrequests
│   └── {name}
├── certificatesigningrequests
│   └── {name}
└── agent/
    ├── devices/{name}/spec
    ├── devices/{name}/status
    └── devices/{name}/rendered
```

### API Generation

APIs are generated using **oapi-codegen** from OpenAPI specs:

```bash
# Generate server stubs
oapi-codegen -config api/v1beta1/spec.gen.cfg api/v1beta1/openapi.yaml

# Generate types
oapi-codegen -config api/v1beta1/types.gen.cfg api/v1beta1/openapi.yaml
```

### Authentication & Authorization

**User API** (HTTPS + JWT):
```
1. User authenticates with IdP → JWT token
2. User includes token in Authorization header
3. API validates token → Extract user identity
4. Check permissions (SpiceDB/K8s RBAC)
5. Process request
```

**Agent API** (mTLS):
```
1. Agent presents client certificate
2. API validates certificate against CA
3. Extract device identity from cert CN
4. Process request (no additional authz needed)
```

---

## Security Architecture

### Hardware Root of Trust

**TPM Integration**:
- Device identity stored in TPM
- Private keys never leave TPM
- Attestation support (EK, AK, Quote)
- Measurement validation

**Enrollment Flow**:
```
1. Device generates identity key in TPM
2. Device creates CSR with TPM-backed key
3. Device submits enrollment request with TPM attestation
4. Service verifies TPM attestation
5. Service approves and signs certificate
6. Device receives certificate, stores in TPM
```

### Certificate Management

**Certificate Hierarchy**:
```
Root CA
  └─ Intermediate CA (Flight Control)
      ├─ Device Certificates (mTLS)
      └─ Service Certificates (TLS)
```

**Rotation**:
- Automatic rotation before expiry
- Zero-downtime rotation
- Graceful fallback on rotation failure

### Secure Communication

**User → Service**: HTTPS with JWT
**Agent → Service**: mTLS with client certificates
**Service → Database**: TLS optional
**Service → Redis**: TLS optional

### Multi-Tenancy

**Organization Isolation**:
- Logical separation of devices and fleets
- RBAC per organization
- Data isolation in database (org_id foreign key)

---

## Deployment Architecture

### Deployment Options

#### 1. Kubernetes/OpenShift (Helm Chart)

**Components**:
```yaml
- API Server (Deployment, 3 replicas)
- Workers (Deployment, 5 replicas)
- Periodic (Deployment, 1 replica)
- PostgreSQL (StatefulSet or external)
- Redis (StatefulSet or external)
- Additional proxies and exporters
```

**High Availability**:
- Multi-replica API servers behind LoadBalancer
- Horizontal scaling of workers
- Database replication (PostgreSQL streaming)
- Redis Sentinel for KV store HA

#### 2. Linux Systemd (Quadlets)

**For development and small deployments**:
```
Services (systemd):
├── flightctl-api.service
├── flightctl-worker.service
├── flightctl-periodic.service
├── postgresql.service
└── redis.service
```

#### 3. Podman Compose

**For local development**:
```yaml
services:
  - db (PostgreSQL)
  - redis
  - api
  - worker
  - periodic
```

### Scaling Considerations

**API Server**:
- Stateless, scale horizontally
- Add replicas for higher throughput
- Use LoadBalancer for distribution

**Workers**:
- Scale based on queue depth
- Independent workers for different task types
- Monitor Redis queue metrics

**Database**:
- Vertical scaling for small-medium deployments (< 5K devices)
- Read replicas for read-heavy workloads
- Connection pooling (PgBouncer)

**Redis**:
- Scale vertically for queue throughput
- Cluster mode for large deployments (> 10K devices)

### Resource Requirements

**Small Deployment (up to 1,000 devices)**:
- API: 2 vCPU, 2GB RAM
- Worker: 2 vCPU, 2GB RAM
- Periodic: 1 vCPU, 1GB RAM
- PostgreSQL: 2 vCPU, 4GB RAM, 50GB disk
- Redis: 1 vCPU, 2GB RAM

**Medium Deployment (up to 10,000 devices)**:
- API: 4 vCPU, 4GB RAM (2 replicas)
- Worker: 4 vCPU, 4GB RAM (3 replicas)
- Periodic: 2 vCPU, 2GB RAM
- PostgreSQL: 8 vCPU, 16GB RAM, 200GB disk
- Redis: 4 vCPU, 8GB RAM

---

## Technology Stack

### Core Languages & Frameworks

- **Go 1.24** - Primary language (with FIPS support)
- **gRPC** - Internal service communication
- **Chi Router** - HTTP routing
- **GORM** - ORM for PostgreSQL

### Data Storage

- **PostgreSQL 12/16** - Primary datastore
  - JSONB for flexible schemas
  - Full-text search capabilities
  - Strong consistency guarantees
  
- **Redis 7.4** - Key-value store and task queues
  - Redis Streams for task queues
  - Pub/sub for real-time updates
  - Distributed locks

### External Integrations

- **Keycloak / OIDC** - User authentication
- **SpiceDB / K8s RBAC** - Authorization
- **Prometheus** - Metrics collection
- **OpenTelemetry** - Distributed tracing
- **Alertmanager** - Alert routing

### Container & OS Technologies

- **Podman** - Container runtime
- **bootc / rpm-ostree** - Image-based OS
- **systemd** - Service management
- **Ignition** - Initial provisioning
- **MicroShift** - Lightweight Kubernetes

### Build & Test Tools

- **Make** - Build automation
- **Buildah/Podman** - Container builds
- **Ginkgo/Gomega** - Testing framework
- **golangci-lint** - Code linting
- **oapi-codegen** - OpenAPI code generation
- **mockgen** - Mock generation

### CI/CD

- **GitHub Actions** - CI/CD pipelines
- **Kind** - Kubernetes testing
- **Helm** - Kubernetes packaging

---

## Design Patterns

### 1. Agent-Based Architecture

**Pattern**: Devices "call home" to service
**Benefits**:
- Works with NAT and firewalls
- Devices can be on private networks
- Service doesn't need device inventory for connectivity

### 2. Declarative API

**Pattern**: Users declare desired state, system reconciles
**Benefits**:
- GitOps-compatible
- Idempotent operations
- Clear separation of intent and implementation

### 3. Controller Pattern

**Pattern**: Continuous reconciliation loops
```go
for {
    desired := getDesiredState()
    current := getCurrentState()
    if desired != current {
        reconcile(desired, current)
    }
    wait()
}
```

### 4. Template-Based Fleet Management

**Pattern**: Fleet as a template for many devices
**Benefits**:
- DRY principle for device configuration
- Consistent configuration across fleet
- Simplified updates

### 5. Event Sourcing

**Pattern**: Record all changes as events
**Benefits**:
- Complete audit trail
- Time-travel debugging
- Reconstruct state at any point

### 6. Asynchronous Task Processing

**Pattern**: Enqueue work, process asynchronously
**Benefits**:
- API remains responsive
- Retry failed operations
- Scale workers independently

### 7. TPM-Backed Identity

**Pattern**: Hardware root of trust for device identity
**Benefits**:
- Strong device authentication
- Attestation support
- Resistant to credential theft

---

## Development Workflow

### Project Structure

```
flightctl/
├── api/                    # API definitions (OpenAPI)
│   ├── v1beta1/           # v1beta1 API specs and generated code
│   └── grpc/              # gRPC definitions
├── cmd/                    # Entry points for binaries
│   ├── flightctl/         # CLI
│   ├── flightctl-api/     # API server
│   ├── flightctl-agent/   # Device agent
│   ├── flightctl-worker/  # Worker process
│   └── ...
├── internal/              # Internal packages
│   ├── agent/            # Agent implementation
│   ├── api_server/       # API server implementation
│   ├── store/            # Database layer
│   ├── service/          # Business logic
│   └── ...
├── pkg/                   # Public libraries
│   ├── log/              # Logging
│   ├── queues/           # Queue abstraction
│   └── ...
├── deploy/                # Deployment configs
│   ├── helm/             # Helm charts
│   ├── podman/           # Podman compose
│   └── scripts/          # Deployment scripts
├── test/                  # Tests
│   ├── e2e/              # End-to-end tests
│   ├── integration/      # Integration tests
│   └── ...
├── docs/                  # Documentation
└── examples/             # Example configurations
```

### Build System

**Makefile targets**:
```bash
make build              # Build all binaries
make build-containers   # Build container images
make unit-test         # Run unit tests
make integration-test  # Run integration tests
make e2e-test          # Run E2E tests
make lint              # Run linters
make deploy            # Deploy to Kind cluster
make deploy-quadlets   # Deploy with systemd
```

### Code Generation

```bash
make generate          # Generate all code
  ├─ API code (oapi-codegen)
  ├─ Mocks (mockgen)
  ├─ Protobuf (protoc)
  └─ Embedded files (go:embed)
```

### Testing Strategy

1. **Unit Tests** - Test individual functions/methods
2. **Integration Tests** - Test component interactions
3. **E2E Tests** - Full system tests
4. **Load Tests** - Device simulator with 1000+ devices
5. **Smoke Tests** - Quick validation on PRs

---

## Appendix

### Glossary

- **bootc**: Boot-and-switch containers for OS updates
- **CSR**: Certificate Signing Request
- **EK**: TPM Endorsement Key
- **Fleet**: Group of devices with common configuration
- **GitOps**: Git as source of truth for configuration
- **mTLS**: Mutual TLS (both client and server authenticate)
- **Quadlet**: Systemd generator for containers
- **TPM**: Trusted Platform Module

### References

- [User Documentation](user/README.md)
- [Developer Documentation](developer/README.md)
- [API Documentation](user/references/api-resources.md)
- [Contributing Guide](../CONTRIBUTING.md)

### License

See [LICENSE](../LICENSE) file for details.

