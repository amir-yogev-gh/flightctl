# Flight Control - Project Documentation Index

> **Project Status:** Beta  
> **Latest Update:** November 29, 2025

## Quick Start

- **New Users**: Start with [Introduction](user/introduction.md) and [Installation Guide](user/installing/installing-service-on-kubernetes.md)
- **Developers**: Read [Architecture](ARCHITECTURE.md) and [Development Workflow](developer/WORKFLOW.md)
- **Contributors**: See [Contributing Guidelines](../CONTRIBUTING.md)
- **Architects**: Review [Technical Design](TECHNICAL_DESIGN.md) and [Architecture](ARCHITECTURE.md)

---

## Documentation Structure

### 📚 Core Documentation

| Document | Description | Audience |
|----------|-------------|----------|
| [README](../README.md) | Project overview and quick links | Everyone |
| [ARCHITECTURE.md](ARCHITECTURE.md) | System architecture and design | Engineers, Architects |
| [COMPONENTS.md](COMPONENTS.md) | Component diagrams and interactions | Engineers, Architects |
| [TECHNICAL_DESIGN.md](TECHNICAL_DESIGN.md) | Implementation details | Engineers, Contributors |
| [CONTRIBUTING.md](../CONTRIBUTING.md) | Contribution guidelines | Contributors |

### 👥 User Documentation

Located in `docs/user/`:

| Section | Description |
|---------|-------------|
| [Introduction](user/introduction.md) | Concepts and high-level architecture |
| [Installing](user/installing/) | Installation guides for various platforms |
| [Using](user/using/) | Guides for managing devices and fleets |
| [Building](user/building/) | Building OS images and applications |
| [References](user/references/) | API specs, status definitions, metrics |

**Key Topics**:
- Provisioning devices on physical hardware, OpenShift, VMware
- Managing individual devices and device fleets
- Configuring authentication and authorization
- Monitoring and observability
- Troubleshooting

### 🔧 Developer Documentation

Located in `docs/developer/`:

| Document | Description |
|----------|-------------|
| [README](developer/README.md) | Developer quick start |
| [WORKFLOW.md](developer/WORKFLOW.md) | Complete development workflow |
| [devicesimulator.md](developer/devicesimulator.md) | Device simulator guide |

**Key Topics**:
- Building the project
- Running tests (unit, integration, E2E)
- Deploying locally (Kind, Quadlets, Podman)
- Testing with agent VMs and containers
- CI/CD pipeline

---

## Project Overview

### What is Flight Control?

Flight Control is a **declarative edge device management system** designed to manage fleets of thousands of edge devices running container workloads on image-based Linux operating systems.

### Key Features

✅ **Declarative APIs** - GitOps-compatible, Kubernetes-like APIs  
✅ **Fleet Management** - Manage thousands of devices with templates  
✅ **Agent-Based** - Autonomous agents on devices handle updates  
✅ **Secure by Design** - TPM-backed identity, mTLS, hardware root of trust  
✅ **Container Workloads** - Podman containers, MicroShift/K8s services  
✅ **Image-Based OS** - bootc/rpm-ostree for transactional updates  
✅ **Resilient** - Works under adverse network conditions, automatic rollback  
✅ **Multi-Tenant** - Organization-based isolation  
✅ **Pluggable Auth** - Keycloak, OIDC, OpenShift, SpiceDB, K8s RBAC  

### Use Cases

- **Industrial IoT**: Manage edge devices in factories, warehouses, retail
- **Telecommunications**: Edge infrastructure for 5G deployments
- **Transportation**: Connected vehicles, autonomous systems
- **Smart Cities**: Traffic systems, surveillance, environmental monitoring
- **Retail**: Point-of-sale systems, inventory management
- **Energy**: Smart grid, renewable energy monitoring

---

## Architecture at a Glance

```
┌─────────────────────────────────────────────────────────────────┐
│                    Users & Tools                                │
│  CLI │ Web UI │ GitOps Tools │ External APIs                   │
└───────────────────────────┬─────────────────────────────────────┘
                            │ HTTPS + JWT
                            ▼
┌─────────────────────────────────────────────────────────────────┐
│              Flight Control Service                             │
│                                                                 │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐        │
│  │  API Server  │  │   Workers    │  │  Periodic    │        │
│  └──────┬───────┘  └──────┬───────┘  └──────┬───────┘        │
│         │                 │                  │                 │
│         └─────────────────┴──────────────────┘                 │
│                           │                                     │
│         ┌─────────────────┴─────────────────┐                 │
│         ▼                                    ▼                 │
│  ┌─────────────┐                    ┌──────────────┐          │
│  │ PostgreSQL  │                    │ Redis        │          │
│  └─────────────┘                    └──────────────┘          │
└─────────────────────────────────────────────────────────────────┘
                            │ mTLS
                            ▼
┌─────────────────────────────────────────────────────────────────┐
│                      Edge Devices                               │
│                                                                 │
│  ┌──────────────────────────────────────────────────────────┐ │
│  │             flightctl-agent                              │ │
│  └──────────────────────────────────────────────────────────┘ │
│  ┌──────────────────────────────────────────────────────────┐ │
│  │  Image-based OS │ Podman │ MicroShift │ Systemd         │ │
│  └──────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────────┘
```

### Core Components

| Component | Purpose |
|-----------|---------|
| **API Server** | RESTful API, authentication, validation |
| **Workers** | Asynchronous task processing (rollouts, reconciliation) |
| **Periodic Checker** | Time-based monitoring and reconciliation |
| **Agent** | On-device management, updates, status reporting |
| **PostgreSQL** | Primary data store (devices, fleets, configs) |
| **Redis** | Task queues, distributed locks, pub/sub |

---

## Technology Stack

### Languages & Frameworks
- **Go 1.24** with FIPS support
- **gRPC** for internal communication
- **Chi Router** for HTTP routing
- **GORM** for database ORM

### Data Storage
- **PostgreSQL 12/16** - Primary datastore
- **Redis 7.4** - Task queues and caching

### Container & OS
- **Podman** - Container runtime
- **bootc / rpm-ostree** - Image-based OS
- **systemd** - Service management
- **MicroShift** - Lightweight Kubernetes

### Observability
- **Prometheus** - Metrics
- **OpenTelemetry** - Distributed tracing
- **Alertmanager** - Alerting

### Security
- **TPM 2.0** - Hardware root of trust
- **Keycloak / OIDC** - Authentication
- **SpiceDB / K8s RBAC** - Authorization

---

## API Resources

Flight Control exposes the following API resources:

| Resource | Description |
|----------|-------------|
| **Device** | Individual device with OS, config, and applications |
| **Fleet** | Group of devices with common template and policies |
| **Repository** | Git repository for configuration sources |
| **ResourceSync** | GitOps synchronization of resources |
| **EnrollmentRequest** | Device enrollment request |
| **CertificateSigningRequest** | Certificate signing request |
| **Organization** | Multi-tenant organization |
| **AuthProvider** | Authentication provider configuration |

### Example: Device Resource

```yaml
apiVersion: v1beta1
kind: Device
metadata:
  name: edge-device-001
  labels:
    location: factory-berlin
    role: monitoring
spec:
  os:
    image: quay.io/flightctl/edge-os:latest
  config:
    - name: monitoring-config
      configType: GitConfigProviderSpec
      gitRef:
        repository: config-repo
        path: monitoring/config.yaml
  applications:
    - name: prometheus
      type: container
      image: prom/prometheus:latest
status:
  updated: true
  systemInfo:
    architecture: x86_64
    operatingSystem: linux
  resources:
    cpu: 35
    memory: 4096
    disk: 100000
```

---

## Deployment Options

### 1. Kubernetes/OpenShift (Recommended for Production)

Deploy using Helm charts with high availability:

```bash
helm install flightctl ./deploy/helm/flightctl \
  --set api.replicas=3 \
  --set worker.replicas=5
```

**Features**:
- High availability
- Horizontal scaling
- Integration with K8s ecosystem
- Monitoring and observability

### 2. Linux Systemd (Small Deployments)

Deploy using systemd Quadlets:

```bash
make deploy-quadlets
```

**Features**:
- Lightweight deployment
- No Kubernetes required
- Suitable for edge scenarios
- Development and testing

### 3. Podman Compose (Development)

Deploy locally with Podman Compose:

```bash
make deploy
```

**Features**:
- Quick local setup
- Easy debugging
- Integrated with Makefile

---

## Getting Started

### Prerequisites

- Go 1.24+
- Podman or Docker
- PostgreSQL 12+ (or use containerized)
- Redis 7.4+ (or use containerized)
- Git

### Quick Start (Development)

```bash
# Clone repository
git clone https://github.com/flightctl/flightctl.git
cd flightctl

# Build binaries
make build

# Deploy locally
make deploy

# Test with device simulator
bin/devicesimulator --count=10
```

### Quick Start (Production)

```bash
# Install on Kubernetes
helm repo add flightctl https://flightctl.github.io/flightctl
helm install flightctl flightctl/flightctl

# Install CLI
curl -LO https://github.com/flightctl/flightctl/releases/latest/download/flightctl
chmod +x flightctl
sudo mv flightctl /usr/local/bin/

# Configure CLI
flightctl config init --server https://flightctl.example.com

# List devices
flightctl get devices
```

---

## Development

### Building

```bash
make build              # Build all binaries
make build-containers   # Build container images
```

### Testing

```bash
make unit-test         # Run unit tests
make integration-test  # Run integration tests
make e2e-test          # Run E2E tests
```

### Linting

```bash
make lint              # Run all linters
make lint-openapi      # Lint OpenAPI specs
make lint-docs         # Lint documentation
```

### Deployment

```bash
make deploy            # Deploy to Kind cluster
make deploy-quadlets   # Deploy with systemd
make agent-vm          # Create test VM with agent
```

---

## Contributing

We welcome contributions! Please see:

- [Contributing Guidelines](../CONTRIBUTING.md)
- [Development Workflow](developer/WORKFLOW.md)
- [Code of Conduct](../CODE_OF_CONDUCT.md) (if exists)

### Ways to Contribute

- 🐛 Report bugs and issues
- 💡 Propose new features
- 📝 Improve documentation
- 🔧 Submit pull requests
- 🧪 Add tests and improve coverage
- 🌍 Translate documentation

---

## Community

- **GitHub**: https://github.com/flightctl/flightctl
- **Issues**: https://github.com/flightctl/flightctl/issues
- **Discussions**: https://github.com/flightctl/flightctl/discussions
- **Slack**: [Join our Slack](#) (if available)

---

## License

Flight Control is licensed under the [Apache License 2.0](../LICENSE).

---

## Related Projects

- **bootc**: Container-native OS updates
- **Podman**: Container runtime
- **MicroShift**: Lightweight Kubernetes
- **Keycloak**: Identity and access management
- **SpiceDB**: Authorization system

---

## Roadmap

See [GitHub Milestones](https://github.com/flightctl/flightctl/milestones) for upcoming features and releases.

**Planned Features**:
- Advanced rollout strategies (canary, blue-green)
- Enhanced observability and analytics
- Multi-cluster federation
- Improved UI/UX
- Additional authentication providers
- Enhanced TPM attestation

---

## Acknowledgments

Flight Control is maintained by the community and sponsored by Red Hat.

Special thanks to all contributors who have helped make this project possible.

