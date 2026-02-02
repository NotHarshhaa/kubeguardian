<p align="center">
  <img src="https://raw.githubusercontent.com/tandpfun/skill-icons/65dea6c4eaca7da319e552c09f4cf5a9a8dab2c8/icons/Kubernetes.svg" alt="KubeGuardian Logo" width="120">
</p>

<h1 align="center">🚑 KubeGuardian</h1>

<p align="center">
  <strong>Automated Kubernetes Self-Healing & Auto-Remediation Tool</strong>
</p>

<p align="center">
  <a href="https://github.com/NotHarshhaa/kubeguardian/releases/tag/v1.6.0">
    <img src="https://img.shields.io/badge/version-v1.6.0-blue.svg" alt="Version">
  </a>
  <a href="https://goreportcard.com/report/github.com/NotHarshhaa/kubeguardian">
    <img src="https://goreportcard.com/badge/github.com/NotHarshhaa/kubeguardian" alt="Go Report Card">
  </a>
  <a href="LICENSE">
    <img src="https://img.shields.io/badge/license-MIT-green.svg" alt="License">
  </a>
  <img src="https://img.shields.io/badge/go-1.21+-blue.svg" alt="Go Version">
</p>

---

<p align="center">
KubeGuardian is an open-source Kubernetes automation tool that continuously monitors cluster health, detects common failures, and automatically remediates issues — reducing downtime, pager alerts, and manual firefighting for DevOps & SRE teams.
</p>

![banner](images/banner.png)

## 🚀 Why KubeGuardian?

Modern Kubernetes clusters fail often:
- Pods stuck in CrashLoopBackOff
- Bad deployments breaking production
- CPU spikes causing service outages
- On-call engineers waking up at 2 AM 😵

👉 **KubeGuardian fixes these issues automatically before humans need to intervene.**

## 🧠 What KubeGuardian Does

### ✅ Auto-Detects
- CrashLoopBackOff pods
- Failed deployments / rollouts
- High CPU usage
- Memory spikes and OOMKills
- Memory pressure
- Image pull backoffs
- Node issues

### 🔧 Auto-Remediates
- Restarts unhealthy pods
- Rolls back failed deployments
- Scales replicas during CPU spikes
- Restarts pods with memory issues
- Scales replicas for memory pressure
- Handles resource pressure

### 📢 Notifies
- Sends Slack alerts with:
  - What broke
  - What action was taken
  - Final status

## 🏗 Architecture (High Level)

```
[Kubernetes Cluster]
        ↓
[Metrics + Events]
        ↓
[KubeGuardian Controller]
        ↓
[Detection Rules]
        ↓
[Remediation Engine]
        ↓
[Slack / Logs]
```

## 💻 Supported Platforms

KubeGuardian supports multiple architectures for maximum compatibility:

| Platform | Architecture | Use Case |
|----------|-------------|---------|
| 🐧 **Linux** | `amd64` | Standard servers, cloud VMs |
| 🐧 **Linux** | `arm64` | Apple M1/M2, ARM servers, Raspberry Pi 4 |

### 🚀 Multi-Architecture Docker Image

```bash
# Docker automatically pulls the right architecture for your platform
docker pull ghcr.io/NotHarshhaa/kubeguardian/kubeguardian:latest

# Kubernetes handles architecture selection automatically
image: ghcr.io/NotHarshhaa/kubeguardian/kubeguardian:latest
```

### 📦 Platform-Specific Benefits

- **Cloud Native**: Supports all major cloud providers with x86_64 and ARM64
- **Development**: Works natively on macOS (Apple Silicon) and Linux
- **Cost Optimization**: Leverage cheaper ARM64 instances where available
- **Performance**: Optimized for modern ARM64 and x86_64 architectures

## ✨ Features (v1.6.0 - Security & Testing Edition)

- 🚑 **CrashLoopBackOff auto-restart**
- 🔄 **Deployment auto-rollback**
- 📈 **CPU-based auto-scaling**
- 🧠 **Memory-based auto-remediation** - Detect memory spikes and OOMKills
- 🧪 **Dry-run mode** - Test remediation actions safely
- 🏷️ **Namespace-scoped rules** - Different policies per namespace
- ⏱️ **Remediation cooldown window** - Prevent repeated fixes and fix loops
- ⚙️ **YAML-based rule configuration**
- 🔔 **Slack notifications**
- 🔐 **Least-privilege RBAC**
- 📊 **Comprehensive Prometheus metrics** - Detection, remediation, API calls, system metrics
- 🏥 **Advanced health probes** - Liveness, readiness, and comprehensive health checks
- 🐳 **Docker support**
- 📦 **Helm chart**
- 🔌 **Circuit breaker pattern** - Prevent cascading failures from API issues
- ⚡ **Rate limiting** - Token bucket algorithm for remediation actions
- ✅ **Configuration validation** - Comprehensive validation with detailed error reporting
- 🛡️ **Graceful shutdown** - Proper cleanup with timeout handling
- 🚨 **Built-in health checks** - API connectivity, memory, disk usage monitoring
- 🔒 **Security validation** - Input sanitization, DoS prevention, abuse protection
- 🧪 **Comprehensive testing** - Unit, integration, security, chaos engineering tests

## 🗺️ Roadmap

### 🚀 v1.6.0 - Security & Testing Edition (Current)
- ✅ **Security validation** - Comprehensive input validation and abuse prevention
- ✅ **Comprehensive testing** - Unit, integration, security, chaos engineering tests
- ✅ **Low-RAM testing** - Efficient testing suite for resource-constrained environments
- ✅ **Enhanced error reporting** - Detailed validation messages with security context
- ✅ **Production readiness** - Full validation and testing coverage

### 🔮 v1.7.0 - AI/ML Edition (Planned)
- 🤖 **ML-based anomaly detection** - Machine learning for pattern recognition
- 🧠 **Predictive scaling** - AI-powered resource prediction
- 📊 **Intelligent alerting** - Smart notification prioritization
- 🔍 **Advanced diagnostics** - Root cause analysis automation

### 🎯 v2.0.0 - Enterprise Edition (Future)
- 🌐 **Multi-cluster support** - Cross-cluster monitoring and remediation
- 📊 **Advanced analytics** - Comprehensive reporting and insights
- 🔐 **Enterprise security** - Advanced RBAC and compliance features
- 🚀 **Auto-scaling policies** - Intelligent resource management
- 📱 **Mobile app** - On-the-go cluster management

## 🧪 Testing & Quality Assurance

KubeGuardian includes comprehensive testing to ensure reliability and security:

### 🧪 Test Coverage
- **Unit Tests**: Core functionality testing
- **Integration Tests**: Kubernetes cluster integration
- **Security Tests**: Input validation and security checks
- **Chaos Engineering**: Resilience and failure scenarios
- **Performance Benchmarks**: Load and performance testing

### 🚀 Quick Testing
```bash
# Run all tests (low RAM usage)
make test-all

# Or run specific test categories
make test-unit          # Unit tests only
make test-security      # Security validation
make test-benchmark     # Performance tests
make test-chaos        # Chaos engineering
```

### 📊 Test Results
All tests pass with minimal resource usage:
- ✅ Unit Tests: 0.4s, Low RAM
- ✅ Security Tests: 0.2s, Low RAM  
- ✅ Performance Tests: 0.6s, Low RAM
- ✅ Integration Tests: Requires K8s cluster

### 🛡️ Security Validation
KubeGuardian includes comprehensive security validation:
- **DoS Prevention**: Evaluation interval limits
- **Resource Protection**: Retry and cooldown enforcement
- **Input Sanitization**: Namespace and channel validation
- **Abuse Prevention**: Rate limiting and circuit breakers

## 📦 Installation

### 🥇 Recommended: Helm

```bash
# Add the KubeGuardian Helm repository
helm repo add kubeguardian https://NotHarshhaa.github.io/kubeguardian
helm repo update

# Install KubeGuardian
helm install kubeguardian NotHarshhaa/kubeguardian \
  --namespace kubeguardian \
  --create-namespace
```

### 🥈 Simple Install (kubectl)

```bash
# Install with a single command
kubectl apply -f https://raw.githubusercontent.com/NotHarshhaa/kubeguardian/master/deployments/manifests/install.yaml
```

### 🥉 Custom Install

```bash
# Clone the repository
git clone https://github.com/NotHarshhaa/kubeguardian.git
cd kubeguardian

# Customize configuration
cp examples/basic-config.yaml configs/config.yaml

# Apply manifests
kubectl apply -f deployments/manifests/
```

## 🔧 Advanced Configuration

### Configuration Validation

KubeGuardian automatically validates your configuration on startup:

```bash
# Configuration is validated automatically
./kubeguardian --config /path/to/config.yaml

# Validation errors will prevent startup
# Validation warnings will be logged but allow startup
```

#### Validation Examples

✅ **Valid Configuration**:
```yaml
detection:
  evaluationInterval: 30s    # ✅ Valid (>= 1s)
  cpuThresholdPercent: 80.0  # ✅ Valid (0-100)
  memoryThresholdPercent: 85.0 # ✅ Valid (0-100)
```

❌ **Invalid Configuration**:
```yaml
detection:
  evaluationInterval: 100ms  # ❌ Invalid (< 1s)
  cpuThresholdPercent: 150.0 # ❌ Invalid (> 100)
```

### Circuit Breaker Configuration

Protect against cascading failures with circuit breakers:

```yaml
# Circuit breaker is automatically enabled
# Default settings:
# - Max requests: 1
# - Timeout: 60s
# - Interval: 60s
# - Trip after: 5 consecutive failures
```

### Rate Limiting Configuration

Control the rate of remediation actions:

```yaml
# Rate limiting is automatically enabled
# Default settings:
# - Rate limit: 10 actions per second
# - Bucket capacity: 100 tokens
# - Per-action rate limiting
```

### Graceful Shutdown Configuration

KubeGuardian supports graceful shutdown:

```bash
# SIGINT/SIGTERM triggers graceful shutdown
# 30-second shutdown timeout
# Automatic cleanup of resources
```

### Environment Variables

Configure KubeGuardian using environment variables:

```bash
export KUBEGUARDIAN_CONFIG_PATH=/etc/kubeguardian/config.yaml
export KUBEGUARDIAN_DRY_RUN=true
export KUBEGUARDIAN_METRICS_ADDR=:9090
export KUBEGUARDIAN_PROBE_ADDR=:9091
export KUBEGUARDIAN_LEADER_ELECTION=true

./kubeguardian
```

### Basic Configuration

```yaml
controller:
  metricsAddr: ":8080"
  probeAddr: ":8081"
  leaderElection: true

detection:
  evaluationInterval: 30s
  crashLoopThreshold: 3
  failedDeploymentThreshold: 5
  cpuThresholdPercent: 80.0
  memoryThresholdPercent: 85.0  # Memory usage threshold
  oomKillThreshold: 2           # OOMKill threshold

remediation:
  enabled: true
  dryRun: false
  autoRollbackEnabled: true
  autoScaleEnabled: true
  cooldownSeconds: 300  # 5 minutes cooldown between actions

# Namespace-specific rules (optional)
namespaces:
  prod:
    crashloop:
      restartLimit: 2        # Strict - restart after 2 crashes
      checkDuration: 3m
      enabled: true
    memory:
      thresholdPercent: 80.0  # Lower threshold for production
      oomKillThreshold: 1     # Immediate action on OOMKill
      checkDuration: 3m
      enabled: true
    remediation:
      enabled: true
      autoRollbackEnabled: true
      maxRetries: 2
      cooldownSeconds: 600  # 10 minutes for production
  
  dev:
    crashloop:
      restartLimit: 5        # Lenient - restart after 5 crashes
      checkDuration: 10m
      enabled: true
    memory:
      thresholdPercent: 90.0  # Higher threshold for development
      oomKillThreshold: 3     # More tolerant in development
      checkDuration: 10m
      enabled: true
    remediation:
      enabled: true
      autoRollbackEnabled: false  # Don't auto-rollback in dev
      maxRetries: 5
      cooldownSeconds: 120  # 2 minutes for development

notification:
  slack:
    enabled: false
    channel: "#kubeguardian"
    username: "KubeGuardian"
```

### Slack Integration

1. Create a Slack Bot Token:
   ```bash
   # Create a secret with your Slack token
   kubectl create secret generic kubeguardian-secrets \
     --from-literal=slack-token=xoxb-your-slack-token \
     --namespace=kubeguardian
   ```

2. Enable Slack in configuration:
   ```yaml
   notification:
     slack:
       enabled: true
       channel: "#alerts"
       username: "KubeGuardian"
   ```

## 🧪 Dry-Run Mode

Test KubeGuardian safely without making actual changes:

### Command Line Usage
```bash
# Enable dry-run mode via command line
./kubeguardian --dry-run --config /path/to/config.yaml

# Or using the shorthand flag
./kubeguardian -d --config /path/to/config.yaml
```

### Configuration File Usage
```yaml
remediation:
  enabled: true
  dryRun: true  # Enable dry-run mode
```

### What Dry-Run Mode Does
- ✅ **Simulates** remediation actions without executing them
- ✅ **Logs** what would happen with detailed information
- ✅ **Safe testing** in production environments
- ✅ **Builds trust** in the tool's behavior

## 🏷️ Namespace-Scoped Rules

Apply different detection and remediation policies per namespace:

### Configuration Structure
```yaml
detection:
  evaluationInterval: 30s
  # Global defaults
  crashLoopThreshold: 3
  failedDeploymentThreshold: 5
  cpuThresholdPercent: 80.0
  
  # Namespace-specific rules
  namespaces:
    prod:
      crashloop:
        restartLimit: 2        # Strict - restart after 2 crashes
        checkDuration: 3m
        enabled: true
      deployment:
        failureThreshold: 3    # Strict - fail after 3 attempts
        checkDuration: 5m
        enabled: true
      cpu:
        thresholdPercent: 70.0 # Lower threshold for production
        checkDuration: 3m
        enabled: true
      remediation:
        enabled: true
        autoRollbackEnabled: true
        autoScaleEnabled: true
        maxRetries: 2

    dev:
      crashloop:
        restartLimit: 5        # Lenient - restart after 5 crashes
        checkDuration: 10m
        enabled: true
      deployment:
        failureThreshold: 10   # Lenient - fail after 10 attempts
        checkDuration: 15m
        enabled: true
      cpu:
        thresholdPercent: 90.0 # Higher threshold for development
        checkDuration: 10m
        enabled: true
      remediation:
        enabled: true
        autoRollbackEnabled: false  # Don't auto-rollback in dev
        maxRetries: 5
```

### Use Cases
- **Production**: Strict rules with aggressive remediation
- **Development**: Lenient rules with debugging-friendly policies  
- **Staging**: Balanced rules for pre-production testing
- **Test**: Minimal monitoring with manual remediation only

### Benefits
1. **Environment-Specific Policies**: Tailor rules to each environment's needs
2. **Risk Management**: Stricter rules in production, lenient in development
3. **Resource Optimization**: Different monitoring intensities per namespace
4. **Operational Flexibility**: Enable/disable features per environment
5. **Gradual Rollout**: Test new rules in specific namespaces first

## ⏱️ Remediation Cooldown Window

Prevent repeated fixes and avoid fix loops with configurable cooldown periods:

### Configuration
```yaml
remediation:
  enabled: true
  cooldownSeconds: 300  # 5 minutes cooldown between actions

# Namespace-specific cooldown
namespaces:
  prod:
    remediation:
      cooldownSeconds: 600  # 10 minutes for production
  dev:
    remediation:
      cooldownSeconds: 120  # 2 minutes for development
```

### What Cooldown Does
- ✅ **Prevents repeated fixes** - Stops same action on same resource repeatedly
- ✅ **Avoids fix loops** - Prevents endless cycles of restart attempts
- ✅ **Protects stability** - Gives resources time to stabilize
- ✅ **Reduces noise** - Limits unnecessary remediation attempts

### How It Works
1. **Cooldown Key**: `{namespace}:{resourceName}:{action}`
2. **Pre-Action Check**: Verifies if action is in cooldown period
3. **Skip Logic**: Logs and skips if cooldown is active
4. **Post-Action Recording**: Tracks successful actions for future checks

### Cooldown Examples
```yaml
# Conservative (Production)
cooldownSeconds: 600  # 10 minutes

# Moderate (Staging)
cooldownSeconds: 300  # 5 minutes

# Aggressive (Development)
cooldownSeconds: 60   # 1 minute

# Disabled
cooldownSeconds: 0    # No cooldown
```

## 🧠 Memory-Based Auto-Remediation

Detect memory spikes and OOMKills with automatic restart/scaling:

### Configuration
```yaml
detection:
  memoryThresholdPercent: 85.0  # Memory usage threshold
  oomKillThreshold: 2           # OOMKill threshold for remediation

# Namespace-specific memory settings
namespaces:
  prod:
    memory:
      thresholdPercent: 80.0  # Lower threshold for production
      oomKillThreshold: 1     # Immediate action on OOMKill
      checkDuration: 3m
      enabled: true
  dev:
    memory:
      thresholdPercent: 90.0  # Higher threshold for development
      oomKillThreshold: 3     # More tolerant in development
      checkDuration: 10m
      enabled: true
```

### What Memory Detection Does
- ✅ **Memory Spike Detection** - Identifies sustained high memory usage
- ✅ **OOMKill Detection** - Detects pods killed due to memory constraints
- ✅ **Auto-Remediation** - Restarts pods or scales replicas automatically
- ✅ **Namespace-Specific** - Different memory policies per environment

### Memory Detection Rules
```yaml
# High memory usage detection
- name: "high-memory-usage"
  condition: memory.usage > 85%
  duration: 5m
  action: restart-pod
  severity: high

# OOMKill detection
- name: "oom-kill-detected"
  condition: container.state.terminated.reason == "OOMKilled"
  threshold: 2 occurrences
  actions: [restart-pod, scale-replicas]
  severity: critical
```

### Memory Remediation Examples
```yaml
# Conservative (Production)
memoryThresholdPercent: 80.0
oomKillThreshold: 1

# Moderate (Staging)
memoryThresholdPercent: 85.0
oomKillThreshold: 2

# Aggressive (Development)
memoryThresholdPercent: 90.0
oomKillThreshold: 3

# Disabled
memoryThresholdPercent: 0    # Memory monitoring disabled
```

## 🔍 How It Works

1. **Watches** Kubernetes pods, nodes & deployments
2. **Detects** unhealthy states using configurable rules
3. **Decides** the safest remediation action
4. **Executes** fixes via Kubernetes API
5. **Sends** alerts after action is taken

## 📋 Detection Rules

KubeGuardian comes with built-in detection rules:

### CrashLoopBackOff Detection
```yaml
- name: "crash-loop-backoff"
  description: "Detect pods in CrashLoopBackOff state"
  enabled: true
  conditions:
    - resource: "Pod"
      field: "status.containerStatuses[*].state.waiting.reason"
      operator: "equals"
      value: "CrashLoopBackOff"
      duration: "5m"
  actions:
    - "restart-pod"
  severity: "high"
```

### Failed Deployment Detection
```yaml
- name: "failed-deployment"
  description: "Detect failed deployments"
  enabled: true
  conditions:
    - resource: "Deployment"
      field: "status.conditions[*].type"
      operator: "equals"
      value: "Progressing"
    - resource: "Deployment"
      field: "status.conditions[*].status"
      operator: "equals"
      value: "False"
      duration: "10m"
  actions:
    - "rollback-deployment"
  severity: "high"
```

### High CPU Usage Detection
```yaml
- name: "high-cpu-usage"
  description: "Detect high CPU usage"
  enabled: true
  conditions:
    - resource: "Pod"
      field: "metrics.cpu.usage"
      operator: "greater_than"
      value: 80.0
      duration: "5m"
  actions:
    - "scale-replicas"
  severity: "medium"
```

## 🚀 Quick Start

1. **Install KubeGuardian**:
   ```bash
   kubectl apply -f https://raw.githubusercontent.com/NotHarshhaa/kubeguardian/master/deployments/manifests/install.yaml
   ```

2. **Check status**:
   ```bash
   kubectl get pods -n kubeguardian
   kubectl logs -n kubeguardian deployment/kubeguardian
   ```

3. **Configure Slack** (optional):
   ```bash
   kubectl create secret generic kubeguardian-secrets \
     --from-literal=slack-token=YOUR_SLACK_TOKEN \
     --namespace=kubeguardian
   ```

## 📊 Monitoring & Observability

### Prometheus Metrics

KubeGuardian exposes comprehensive Prometheus metrics on port `8080`:

#### Detection Metrics
- `kubeguardian_issues_detected_total` - Total issues detected by rule, severity, and namespace
- `kubeguardian_detection_duration_seconds` - Time spent detecting issues (histogram)
- `kubeguardian_last_detection_timestamp` - Timestamp of last detection cycle

#### Remediation Metrics
- `kubeguardian_remediations_total` - Total remediation actions by action, result, and namespace
- `kubeguardian_remediation_duration_seconds` - Time spent executing remediation (histogram)
- `kubeguardian_cooldown_active` - Number of active cooldown entries by namespace

#### API Metrics
- `kubeguardian_api_calls_total` - Total Kubernetes API calls by method, resource, and status
- `kubeguardian_api_duration_seconds` - Time spent on API calls (histogram)

#### Notification Metrics
- `kubeguardian_notifications_total` - Total notifications sent by type and status

#### System Metrics
- `kubeguardian_uptime_seconds` - Uptime of KubeGuardian in seconds

### Health Checks

KubeGuardian provides comprehensive health endpoints on port `8081`:

#### Liveness Probe
- **Endpoint**: `/healthz`
- **Purpose**: Indicates if the service is running
- **Response**: `200 OK` if service is alive

#### Readiness Probe
- **Endpoint**: `/readyz`
- **Purpose**: Indicates if the service is ready to handle requests
- **Response**: `200 OK` if all health checks pass, `503 Service Unavailable` otherwise

#### Comprehensive Health Check
- **Endpoint**: `/health` (JSON response)
- **Purpose**: Detailed health status of all components
- **Response**: JSON with overall status and individual check results

```json
{
  "status": "healthy",
  "timestamp": "2024-01-20T10:30:00Z",
  "uptime": "2h30m15s",
  "version": "v1.6.0",
  "checks": {
    "kubernetes-api": {
      "status": "healthy",
      "lastChecked": "2024-01-20T10:30:00Z",
      "duration": "150ms",
      "message": "OK"
    },
    "memory": {
      "status": "healthy",
      "lastChecked": "2024-01-20T10:30:00Z",
      "duration": "10ms",
      "message": "Memory usage: 45%"
    },
    "disk": {
      "status": "healthy",
      "lastChecked": "2024-01-20T10:30:00Z",
      "duration": "5ms",
      "message": "Disk usage: 32%"
    }
  }
}
```

### Built-in Health Checks

KubeGuardian includes these built-in health checks:

1. **Kubernetes API Connectivity** - Verifies connection to the Kubernetes API
2. **Memory Usage** - Checks if memory usage is below threshold (default: 80%)
3. **Disk Usage** - Checks if disk usage is below threshold (default: 85%)

### Grafana Dashboard

Import the provided Grafana dashboard to visualize KubeGuardian metrics:

```bash
# Import dashboard (dashboard ID: 12345)
kubectl apply -f deployments/grafana/kubeguardian-dashboard.json
```

## 🛠 Development

### Building

```bash
# Build the binary
make build

# Run tests
make test

# Build Docker image
make docker-build

# Run with race detection (for development)
go run -race cmd/kubeguardian/main.go
```

### Local Development

```bash
# Run locally (requires kubeconfig)
go run cmd/kubeguardian/main.go

# Run with custom config
go run cmd/kubeguardian/main.go --config configs/config.yaml

# Run in dry-run mode
go run cmd/kubeguardian/main.go --dry-run --config configs/config.yaml

# Run with namespace-scoped configuration
go run cmd/kubeguardian/main.go --config examples/namespace-scoped-config.yaml

# Run with cooldown configuration
go run cmd/kubeguardian/main.go --config examples/cooldown-config.yaml

# Run with memory-based remediation configuration
go run cmd/kubeguardian/main.go --config examples/memory-remediation-config.yaml

# Run with verbose logging
go run cmd/kubeguardian/main.go --config configs/config.yaml -v

# Run with leader election disabled (for local development)
go run cmd/kubeguardian/main.go --config configs/config.yaml --leader-elect=false
```

### Testing

```bash
# Run all tests
make test-all

# Run unit tests
make test-unit

# Run integration tests
make test-integration

# Run benchmarks
make test-benchmark

# Run security tests
make test-security

# Run chaos engineering tests
make test-chaos

# Run race condition tests
make test-race

# Generate coverage report
make coverage

# Run performance profiling
make memory-profile
make cpu-profile
```

## 🤝 Contributing

We welcome contributions! Please see our [Contributing Guide](CONTRIBUTING.md) for details.

### 🚀 Quick Start
```bash
# 1. Fork the repository
git clone https://github.com/NotHarshhaa/kubeguardian.git
cd kubeguardian

# 2. Create a feature branch
git checkout -b feature/your-feature

# 3. Make your changes
# 4. Run tests
make test-all

# 5. Submit a Pull Request
```

### 🧪 Development Setup
```bash
# Install dependencies
go mod tidy

# Run tests
make test-all

# Build the binary
go build ./cmd/kubeguardian

# Run with coverage
go test -coverprofile=coverage.out ./pkg/...
go tool cover -html=coverage.out -o coverage.html
```

### 📋 Code Quality
```bash
# Run static analysis
go vet ./...
go fmt ./...
go mod tidy

# Run security scan
govulncheck ./...
```

## 📜 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 🙏‍💼 Acknowledgments

- Kubernetes community for the amazing platform
- [controller-runtime](https://github.com/kubernetes-sigs/controller-runtime) for the controller framework
- [client-go](https://github.com/kubernetes/client-go) for the Kubernetes client library
- [Prometheus](https://prometheus.io/) for metrics collection
- All contributors and users who make KubeGuardian better!

---

<div align="center">
  <sub>Made with ❤️ by the KubeGuardian community</sub>
</div>

## 👨‍💻 Author & Community  

This project is crafted with 💡 by **[Harshhaa](https://github.com/NotHarshhaa)**.  
Your feedback is always welcome! Let's build together. 🚀  

📧 **Connect with me:**  
🔗 **GitHub**: [@NotHarshhaa](https://github.com/NotHarshhaa)  
🔗 **Portfolio**: [Personal Portfolio](https://notharshhaa.site)  
🔗 **Linktree**: [All Links](https://link.notharshhaa.site)  
🔗 **Telegram Community**: [Join Here](https://t.me/prodevopsguy)  
🔗 **LinkedIn**: [Harshhaa Vardhan Reddy](https://www.linkedin.com/in/NotHarshhaa/)  
🔗 **Twitter/X**: [@NotHarshhaa](https://twitter.com/NotHarshhaa)  
🔗 **Instagram**: [@NotHarshhaa](https://instagram.com/NotHarshhaa)  

---

## 🤝 Support the Project  

If this helped you, consider:  
✅ **Starring** ⭐ this repository  
✅ **Sharing** 📢 with your network  
✅ **Supporting** 📢 on [GitHub Sponsors](https://github.com/sponsors/NotHarshhaa)  
✅ **Supporting** ☕ on [BuyMeACoffee](https://www.buymeacoffee.com/NotHarshhaa)  
✅ **Supporting** 📢 on [Patreon](https://www.patreon.com/NotHarshhaa)  
✅ **Supporting** � on [PayPal](https://www.paypal.com/paypalme/NotHarshhaa)  

---

### 🌟 Stay Connected  

<div align="center">
  <a href="https://github.com/NotHarshhaa">
    <img src="https://img.shields.io/badge/GitHub-181717?style=for-the-badge&logo=github&logoColor=white" alt="GitHub"/>
  </a>
  <a href="https://twitter.com/NotHarshhaa">
    <img src="https://img.shields.io/badge/Twitter-1DA1F2?style=for-the-badge&logo=twitter&logoColor=white" alt="Twitter"/>
  </a>
  <a href="https://www.linkedin.com/in/NotHarshhaa/">
    <img src="https://img.shields.io/badge/LinkedIn-0077B5?style=for-the-badge&logo=linkedin&logoColor=white" alt="LinkedIn"/>
  </a>
  <a href="https://notharshhaa.site">
    <img src="https://img.shields.io/badge/Portfolio-FF5722?style=for-the-badge&logo=website&logoColor=white" alt="Portfolio"/>
  </a>
  <a href="https://t.me/prodevopsguy">
    <img src="https://img.shields.io/badge/Telegram-2CA5E0?style=for-the-badge&logo=telegram&logoColor=white" alt="Telegram"/>
  </a>
</div>

---

<div align="center">
  <sub>*Let's make Kubernetes self-healing a reality!* 🚀</sub>
</div>
