# Contributing to KubeGuardian

Thank you for your interest in contributing to KubeGuardian! This document provides guidelines and information for contributors.

## 🚀 Getting Started

### Prerequisites

- Go 1.21 or later
- Docker
- kubectl
- Access to a Kubernetes cluster (for testing)
- Git

### Development Setup

1. **Fork the repository**
   ```bash
   # Fork https://github.com/NotHarshhaa/kubeguardian
   # Clone your fork
   git clone https://github.com/NotHarshhaa/kubeguardian.git
   cd kubeguardian
   ```

2. **Set up development environment**
   ```bash
   # Install dependencies
   go mod download
   
   # Run tests to verify setup
   make test
   ```

3. **Create a feature branch**
   ```bash
   git checkout -b feature/your-feature-name
   ```

## 🏗 Project Structure

```
kubeguardian/
├── cmd/kubeguardian/          # Main application entry point
├── pkg/                       # Core packages
│   ├── controller/           # Main controller logic
│   ├── detection/            # Detection engine
│   ├── remediation/          # Remediation engine
│   ├── notification/         # Notification system
│   ├── config/              # Configuration management
│   └── version/             # Version information
├── deployments/              # Deployment manifests
│   ├── helm/                # Helm chart
│   └── manifests/           # Kubernetes manifests
├── configs/                 # Configuration examples
├── examples/                # Usage examples
├── docs/                   # Documentation
└── scripts/                # Helper scripts
```

## 📝 Development Guidelines

### Code Style

- Follow Go conventions and best practices
- Use `gofmt` to format code
- Use `golint` and `go vet` to check code quality
- Write meaningful commit messages
- Add comments for public functions and complex logic

### Testing

- Write unit tests for new features
- Add integration tests where applicable
- Ensure all tests pass before submitting PR
- Maintain test coverage above 80%

```bash
# Run all tests
make test

# Run with coverage
go test -cover ./...

# Run integration tests
go test -tags=integration ./...
```

### Documentation

- Update README.md for user-facing changes
- Add inline comments for complex logic
- Update API documentation for new features
- Add examples for new configuration options

## 🐛 Bug Reports

When reporting bugs, please include:

1. **Environment information**
   - Kubernetes version
   - KubeGuardian version
   - Operating system
   - Go version

2. **Steps to reproduce**
   - Clear, reproducible steps
   - Expected vs actual behavior
   - Relevant logs and configuration

3. **Additional context**
   - Screenshots if applicable
   - Related issues or PRs

## ✨ Feature Requests

1. **Check existing issues** - Search for similar requests
2. **Open a new issue** with:
   - Clear description of the feature
   - Use case and motivation
   - Proposed implementation (if any)
   - Acceptance criteria

## 🔄 Pull Request Process

1. **Create a feature branch** from `main`
2. **Make your changes** following the guidelines
3. **Add tests** for your changes
4. **Update documentation** as needed
5. **Run tests** to ensure everything passes
6. **Submit a pull request** with:
   - Clear title and description
   - Link to related issues
   - Testing instructions
   - Screenshots if applicable

### PR Template

```markdown
## Description
Brief description of changes

## Type of Change
- [ ] Bug fix
- [ ] New feature
- [ ] Breaking change
- [ ] Documentation update

## Testing
- [ ] Unit tests pass
- [ ] Integration tests pass
- [ ] Manual testing completed

## Checklist
- [ ] Code follows style guidelines
- [ ] Self-review completed
- [ ] Documentation updated
- [ ] Tests added/updated
```

## 🏷 Release Process

1. **Update version** in `pkg/version/version.go`
2. **Update CHANGELOG.md**
3. **Create git tag**
4. **Build and push Docker image**
5. **Update Helm chart**
6. **Create GitHub release**

## 🤝 Community

### Code of Conduct

- Be respectful and inclusive
- Welcome newcomers and help them learn
- Focus on constructive feedback
- Maintain professional communication

### Getting Help

- Create an issue for bugs or questions
- Join our Slack community
- Check existing documentation
- Review similar issues and PRs

## 📚 Resources

- [Kubernetes Documentation](https://kubernetes.io/docs/)
- [Go Documentation](https://golang.org/doc/)
- [Controller Runtime](https://github.com/kubernetes-sigs/controller-runtime)
- [Helm Documentation](https://helm.sh/docs/)

## 🎯 Contribution Areas

We're looking for contributions in:

- **Core Features**: New detection rules, remediation actions
- **Integrations**: New notification channels, monitoring systems
- **UI/UX**: Web dashboard, CLI improvements
- **Documentation**: Tutorials, guides, examples
- **Testing**: Unit tests, integration tests, e2e tests
- **Performance**: Optimization, scaling improvements

## 🏆 Recognition

Contributors will be recognized in:

- README.md contributors section
- Release notes
- Community highlights
- Annual contributor awards

---

Thank you for contributing to KubeGuardian! 🚀
