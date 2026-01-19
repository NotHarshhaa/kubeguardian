# Changelog

All notable changes to KubeGuardian will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- 🧪 **Dry-Run Mode** - Simulate remediation actions without making changes
- 🏷️ **Namespace-Scoped Rules** - Apply different detection and remediation policies per namespace
- ⏱️ **Remediation Cooldown Window** - Prevent repeated fixes and avoid fix loops
- 🧠 **Memory-Based Auto-Remediation** - Detect memory spikes and OOMKills with automatic restart/scaling
- Command-line flags for dry-run mode (`--dry-run` and `-d`)
- Namespace-specific configuration fallback to global defaults
- Per-namespace crash loop restart limits
- Per-namespace deployment failure thresholds
- Per-namespace CPU monitoring thresholds
- Per-namespace memory monitoring thresholds and OOMKill detection
- Per-namespace remediation policies (enable/disable, rollback, scaling)
- Cooldown tracking with configurable cooldown periods
- Memory spike detection with configurable thresholds
- OOMKill detection with automatic remediation actions
- Enhanced logging with namespace-specific context

### Features
- CrashLoopBackOff pod detection and restart
- Failed deployment detection and rollback
- High CPU usage detection and auto-scaling
- Memory spike detection and auto-remediation
- OOMKill detection with automatic restart/scaling
- Memory pressure detection
- Image pull backoff detection
- Node health monitoring
- Configurable YAML-based rules
- Leader election support
- Dry-run mode for testing
- Namespace-specific rule configuration
- Remediation cooldown window to prevent fix loops

### Security
- Least-privilege RBAC configuration
- Non-root container execution
- Read-only filesystem where possible
- Resource limits and constraints

## [1.0.0] - 2024-01-19

### Added
- 🚑 **CrashLoopBackOff auto-restart** - Automatically restarts pods stuck in CrashLoopBackOff
- 🔄 **Deployment auto-rollback** - Automatically rolls back failed deployments
- 📈 **CPU-based auto-scaling** - Scales replicas based on CPU usage
- ⚙️ **YAML-based rule configuration** - Flexible rule definition system
- 🔔 **Slack notifications** - Real-time alerts for detected issues
- 🔐 **Least-privilege RBAC** - Secure permissions model
- 📊 **Prometheus metrics** - Comprehensive monitoring metrics
- 🏥 **Health probes** - Liveness and readiness checks
- 🐳 **Docker support** - Containerized deployment
- 📦 **Helm chart** - Easy installation and configuration

### Detection Rules
- Pod CrashLoopBackOff detection
- Failed deployment detection
- High CPU usage detection
- Memory pressure detection
- Image pull backoff detection
- Node not ready detection
- Pending pod detection
- High restart rate detection

### Remediation Actions
- Pod restart
- Deployment rollback
- Replica scaling
- Notification-only mode

### Installation Methods
- Helm chart installation
- kubectl manifest installation
- Custom configuration support

### Configuration
- YAML-based configuration
- Environment variable support
- Secret management for sensitive data
- Multiple environment configurations (dev, staging, prod)

### Monitoring & Observability
- Prometheus metrics on port 8080
- Health probes on port 8081
- Structured logging
- Event creation for Kubernetes

### Documentation
- Comprehensive README
- Installation guides
- Configuration examples
- Contributing guidelines
- API documentation

### Security Features
- Non-root container execution
- Read-only filesystem
- Resource limits
- Network policies (optional)
- Pod security contexts

## [Future Releases]

### Planned for v1.1.0
- [ ] Web UI dashboard
- [ ] Custom metrics support
- [ ] Additional notification channels (Teams, PagerDuty)
- [ ] Advanced rule conditions
- [ ] Audit logging
- [ ] Multi-namespace rule templates
- [ ] Rule validation and testing framework

### Planned for v1.2.0
- [ ] Multi-cluster support
- [ ] Policy engine
- [ ] GitOps integration
- [ ] Performance optimizations
- [ ] Extended integrations

### Planned for v2.0.0
- [ ] Machine learning detection
- [ ] Advanced analytics
- [ ] Custom remediation scripts
- [ ] Plugin system
- [ ] Enterprise features

---

**Note**: This project follows [Semantic Versioning](https://semver.org/).
