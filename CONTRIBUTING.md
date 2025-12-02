# Contributing to flightctl

Thank you for your interest in contributing to flightctl! This document provides guidelines and information for contributors.

## Getting Started

1. **Read the Documentation**
   - [Developer Documentation](docs/developer/README.md)
   - [Development Workflow Guide](docs/developer/WORKFLOW.md)
   - [Device Simulator Guide](docs/developer/devicesimulator.md)

2. **Set Up Your Development Environment**
   - Follow the [prerequisites](docs/developer/WORKFLOW.md#prerequisites) in the workflow guide
   - Fork and clone the repository
   - Build the project with `make build`

3. **Find Something to Work On**
   - Check [existing issues](https://github.com/flightctl/flightctl/issues)
   - Look for issues labeled `good first issue` or `help wanted`
   - Propose new features by creating an issue first

## Development Process

Please follow our [Development Workflow Guide](docs/developer/WORKFLOW.md) for detailed instructions on:

- Setting up your development environment
- Creating feature branches
- Making and testing changes
- Code quality requirements
- Submitting pull requests

## Code Standards

### Go Code

- Follow standard Go conventions and idioms
- Run `go fmt ./...` before committing
- Ensure code passes `make lint`
- Add tests for new functionality
- Maintain or improve code coverage

### Commit Messages

Use conventional commit format:

```
<type>: <description>

[optional body]

[optional footer]
```

Types:
- `feat`: New feature
- `fix`: Bug fix
- `docs`: Documentation changes
- `test`: Adding or updating tests
- `refactor`: Code refactoring
- `chore`: Maintenance tasks
- `ci`: CI/CD changes

Example:
```
feat: add support for custom device labels

This commit adds the ability to specify custom labels
for devices through the fleet API.

Fixes #123
```

### Documentation

- Update documentation when adding features or changing behavior
- Keep README files up to date
- Add inline comments for complex logic
- Follow Markdown best practices

## Testing Requirements

### Before Submitting a PR

1. **Run Unit Tests**
   ```bash
   make unit-test
   ```

2. **Run Linters**
   ```bash
   make lint
   ```

3. **Test Locally**
   ```bash
   make deploy
   # Test your changes
   ```

4. **Verify Integration**
   - Test with agent VMs or containers
   - Use the device simulator for load testing
   - Verify all affected workflows

## Pull Request Process

1. **Create a Pull Request**
   - Use a clear, descriptive title
   - Fill out the PR template completely
   - Reference related issues with `Fixes #123` or `Relates to #456`
   - Include screenshots or logs for UI/behavior changes

2. **CI/CD Checks**
   - All automated checks must pass
   - Fix any failing tests or linting issues
   - Ensure documentation builds correctly

3. **Code Review**
   - Address reviewer feedback promptly
   - Make requested changes in new commits
   - Request re-review when ready
   - Be respectful and professional

4. **Merging**
   - Maintainers will merge once approved
   - PRs are typically squashed when merged
   - Delete your branch after merging

## Getting Help

- **Documentation Issues**: Check the [docs](docs/) directory
- **Code Issues**: Review existing [issues](https://github.com/flightctl/flightctl/issues) and [discussions](https://github.com/flightctl/flightctl/discussions)
- **Questions**: Ask in GitHub Discussions or community Slack
- **Bugs**: Open a new issue with a clear description and reproduction steps

## Community Guidelines

- Be respectful and inclusive
- Assume good intentions
- Provide constructive feedback
- Help others when you can
- Follow the project's Code of Conduct

## License

By contributing to flightctl, you agree that your contributions will be licensed under the same license as the project. See [LICENSE](LICENSE) for details.

## Recognition

Contributors are recognized in:
- Git history and GitHub profiles
- Release notes for significant contributions
- Project documentation where appropriate

Thank you for contributing to flightctl! 🚀

