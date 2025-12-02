# Flight Control

> [!NOTE]
> This project is currently in beta.

Flight Control is a **declarative edge device management system** designed to manage fleets of thousands of edge devices running container workloads on image-based Linux operating systems.

## 🚀 Key Features

- **Declarative APIs** - GitOps-compatible, Kubernetes-like resource management
- **Fleet Management** - Template-based management of device fleets with smart rollout policies
- **Agent-Based Architecture** - Autonomous agents handle updates and report status
- **Secure by Design** - TPM-backed device identity, mTLS communication, hardware root of trust
- **Container Workloads** - Support for Podman containers and MicroShift/Kubernetes services
- **Image-Based OS** - Transactional updates with automatic rollback using bootc/rpm-ostree
- **Multi-Tenant** - Organization-based isolation and RBAC

## 📚 Documentation

### Getting Started
* **[Project Documentation](docs/PROJECT_DOCUMENTATION.md)** - Complete project overview and documentation index
* **[User Documentation](docs/user/README.md)** - Installation, usage, and configuration guides
* **[Developer Documentation](docs/developer/README.md)** - Building, testing, and development workflow

### Architecture & Design
* **[Architecture Overview](docs/ARCHITECTURE.md)** - System architecture, components, and design principles
* **[Technical Design](docs/TECHNICAL_DESIGN.md)** - Implementation details and technical specifications

### Contributing
* **[Contributing Guidelines](CONTRIBUTING.md)** - How to contribute to the project
* **[Development Workflow](docs/developer/WORKFLOW.md)** - Complete development process and best practices

## 🎯 Use Cases

Flight Control is designed for:
- **Industrial IoT** - Factory automation, warehouse management, retail systems
- **Telecommunications** - 5G edge infrastructure and network functions
- **Transportation** - Connected vehicles and autonomous systems
- **Smart Cities** - Traffic management, surveillance, environmental monitoring
- **Energy** - Smart grid management and renewable energy monitoring
