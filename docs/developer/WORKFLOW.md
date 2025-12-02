# Development Workflow Guide

This document describes the standard development workflow for contributing to the flightctl project.

## Table of Contents
- [Prerequisites](#prerequisites)
- [Setting Up Your Development Environment](#setting-up-your-development-environment)
- [Development Workflow](#development-workflow)
- [Testing Your Changes](#testing-your-changes)
- [Code Quality Checks](#code-quality-checks)
- [Submitting Changes](#submitting-changes)
- [CI/CD Pipeline](#cicd-pipeline)

## Prerequisites

Before you begin development, ensure you have the following installed:

* `git`, `make`, and `go` (>= 1.23)
* `openssl`, `openssl-devel`
* `buildah`, `podman`, `podman-compose`
* `container-selinux` (>= 2.241)
* `pam-devel`
* `go-rpm-macros` (for building RPMs)
* `python3`, `python3-pyyaml` (or install PyYAML via pip)
* `gotestsum` for running tests:
  ```bash
  go install gotest.tools/gotestsum@latest
  ```
* `mockgen` for generating mocks:
  ```bash
  go install go.uber.org/mock/mockgen@v0.4.0
  ```

## Setting Up Your Development Environment

### 1. Fork and Clone the Repository

```bash
# Fork the repository on GitHub first, then clone your fork
git clone https://github.com/YOUR_USERNAME/flightctl.git
cd flightctl

# Add the upstream remote
git remote add upstream https://github.com/flightctl/flightctl.git
```

### 2. Enable Podman Socket

The flightctl agent reports the status of running rootless containers:

```bash
systemctl --user enable --now podman.socket
```

### 3. Build the Project

```bash
make build
```

## Development Workflow

### 1. Create a Feature Branch

Always work on a feature branch, never directly on `main`:

```bash
# Fetch the latest changes from upstream
git fetch upstream
git checkout main
git merge upstream/main

# Create your feature branch
git checkout -b feature/your-feature-name
```

### 2. Make Your Changes

- Write clean, idiomatic Go code
- Follow the project's coding conventions
- Add unit tests for new functionality
- Update documentation as needed

### 3. Generate Code and Mocks

If you've modified API definitions or interfaces:

```bash
make generate
```

This generates:
- API code
- Mock implementations
- Any other auto-generated code

### 4. Build and Test Locally

```bash
# Build the project
make build

# Run unit tests
make unit-test

# Run specific tests
go test -v ./path/to/package/...
```

## Testing Your Changes

### Unit Tests

```bash
make unit-test
```

### Integration Tests

```bash
# Deploy the service locally using kind
make deploy

# Or deploy using systemd Quadlets
make deploy-quadlets
```

### Testing with Agent VMs

Create test VMs to verify agent functionality:

```bash
# Create a single VM
make agent-vm

# Create multiple VMs with custom configurations
make agent-vm VMNAME=flightctl-device-1 VMCPUS=2 VMRAM=1024
make agent-vm VMNAME=flightctl-device-2
make agent-vm VMNAME=flightctl-device-3

# Connect to a VM console (exit with Ctrl + ])
make agent-vm-console VMNAME=flightctl-device-1

# Clean up VMs
make clean-agent-vm VMNAME=flightctl-device-1
```

### Testing with Containerized Agents

For lightweight testing without VMs:

```bash
make agent-container

# Clean up
make clean-agent-container
```

### Load Testing with Device Simulator

```bash
bin/devicesimulator --count=100
```

### Testing Different Database Configurations

```bash
# Small configuration (up to 1000 devices)
make deploy DB_VERSION=small-1k

# Medium configuration (up to 10k devices)
make deploy DB_VERSION=medium-10k
```

## Code Quality Checks

### Linting

```bash
# Run all linters
make lint

# Run specific linters
make lint-docs
make lint-helm
make lint-openapi
```

### Format Check

Ensure your code is properly formatted:

```bash
go fmt ./...
```

### Commit Message Format

Follow the conventional commit format:
- `feat:` New feature
- `fix:` Bug fix
- `docs:` Documentation changes
- `test:` Test changes
- `refactor:` Code refactoring
- `chore:` Maintenance tasks

Example:
```
feat: add support for custom device configurations

This commit adds the ability to specify custom configurations
for devices through the fleet API.

Fixes #123
```

## Submitting Changes

### 1. Commit Your Changes

```bash
git add .
git commit -m "feat: your feature description"
```

### 2. Keep Your Branch Up to Date

```bash
git fetch upstream
git rebase upstream/main
```

### 3. Push to Your Fork

```bash
git push origin feature/your-feature-name
```

### 4. Create a Pull Request

1. Go to your fork on GitHub
2. Click "New Pull Request"
3. Select your feature branch
4. Fill out the PR template with:
   - Description of changes
   - Related issues
   - Testing performed
   - Screenshots/logs if applicable

### 5. Address Review Feedback

- Make requested changes
- Commit and push updates
- Respond to review comments
- Request re-review when ready

## CI/CD Pipeline

When you create a pull request, the following checks run automatically:

### Required Checks

1. **Unit Tests** (`unit-tests.yaml`)
   - Runs all unit tests
   - Must pass before merging

2. **Code Quality** (`lint.yaml`)
   - Runs Go linters
   - Checks code style and quality

3. **Documentation Linting** (`lint-docs.yaml`)
   - Validates Markdown formatting
   - Checks documentation quality

4. **OpenAPI Validation** (`lint-openapi.yaml`)
   - Validates OpenAPI specifications
   - Checks for breaking changes

5. **PR Smoke Testing** (`pr-smoke-testing.yaml`)
   - Basic functionality tests
   - Quick validation of core features

6. **PR E2E Testing** (`pr-e2e-testing.yaml`)
   - Full end-to-end testing
   - Comprehensive integration tests

### Optional Workflows

- **Integration Tests** (`integration-tests.yaml`)
  - Can be triggered manually with `workflow_dispatch`

- **Check Doc Links** (`check-doc-links.yml`)
  - Validates all documentation links

### Claude PR Assistant

You can invoke the Claude AI assistant on pull requests by mentioning `@claude` in a review comment. This can help with:
- Code review suggestions
- Bug detection
- Optimization recommendations

## Troubleshooting

### Firewall Issues with Agent VMs

If the agent cannot connect to the API:

```bash
sudo firewall-cmd --zone=libvirt --add-rich-rule='rule family="ipv4" source address="<virbr0s subnet>" accept' --permanent
sudo firewall-cmd --reload
```

### ARM64 (M1/M2 Mac) PostgreSQL Issues

```bash
export PGSQL_IMAGE=registry.redhat.io/rhel9/postgresql-16
podman login registry.redhat.io
```

### Console Size Issues

When connected to a VM console:

```bash
stty rows 80
stty columns 140
```

## Additional Resources

- [Developer Documentation](README.md)
- [Device Simulator Guide](devicesimulator.md)
- [API Documentation](../../api/)
- [User Documentation](../user/)

## Getting Help

- Check existing [GitHub Issues](https://github.com/flightctl/flightctl/issues)
- Join the community Slack channel
- Review the [Contributing Guidelines](../../CONTRIBUTING.md)

