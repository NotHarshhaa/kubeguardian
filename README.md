<p align="center">
  <img src="https://raw.githubusercontent.com/tandpfun/skill-icons/65dea6c4eaca7da319e552c09f4cf5a9a8dab2c8/icons/Kubernetes.svg" alt="KubeGuardian Logo" width="120">
</p>

<h1 align="center">🚑 KubeGuardian</h1>

<p align="center">
  <strong>Automated Kubernetes Self-Healing & Auto-Remediation Tool</strong>
</p>

<p align="center">
  <img src="https://img.shields.io/badge/version-v1.0.0-blue.svg" alt="Version">
  <img src="https://img.shields.io/badge/license-Apache%202.0-blue.svg" alt="License">
  <img src="https://img.shields.io/badge/kubernetes-1.20+-blue.svg" alt="Kubernetes Version">
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
- Memory pressure
- Image pull backoffs
- Node issues

### 🔧 Auto-Remediates
- Restarts unhealthy pods
- Rolls back failed deployments
- Scales replicas during CPU spikes
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

## ✨ Features (v1 – MVP)

- 🚑 **CrashLoopBackOff auto-restart**
- 🔄 **Deployment auto-rollback**
- 📈 **CPU-based auto-scaling**
- 🧪 **Dry-run mode** - Test remediation actions safely
- 🏷️ **Namespace-scoped rules** - Different policies per namespace
- ⚙️ **YAML-based rule configuration**
- 🔔 **Slack notifications**
- 🔐 **Least-privilege RBAC**
- 📊 **Prometheus metrics**
- 🏥 **Health probes**
- 🐳 **Docker support**
- 📦 **Helm chart**

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

## 🔧 Configuration

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

remediation:
  enabled: true
  dryRun: false
  autoRollbackEnabled: true
  autoScaleEnabled: true

# Namespace-specific rules (optional)
namespaces:
  prod:
    crashloop:
      restartLimit: 2        # Strict - restart after 2 crashes
      checkDuration: 3m
      enabled: true
    remediation:
      enabled: true
      autoRollbackEnabled: true
      maxRetries: 2
  
  dev:
    crashloop:
      restartLimit: 5        # Lenient - restart after 5 crashes
      checkDuration: 10m
      enabled: true
    remediation:
      enabled: true
      autoRollbackEnabled: false  # Don't auto-rollback in dev
      maxRetries: 5

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

## 📊 Monitoring

### Metrics

KubeGuardian exposes Prometheus metrics on port `8080`:

- `kubeguardian_issues_detected_total`
- `kubeguardian_remediations_total`
- `kubeguardian_remediation_success_total`
- `kubeguardian_remediation_failure_total`
- `kubeguardian_detection_duration_seconds`

### Health Checks

- **Liveness**: `/healthz` on port `8081`
- **Readiness**: `/readyz` on port `8081`

## 🛠 Development

### Building

```bash
# Build the binary
make build

# Run tests
make test

# Build Docker image
make docker-build
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
```

### Testing

```bash
# Run unit tests
go test ./...

# Run integration tests
go test -tags=integration ./...

# Generate test coverage
go test -cover ./...
```

## 🔐 Security

KubeGuardian follows security best practices:

- **Least privilege RBAC** - Only required permissions
- **Non-root containers** - Runs as non-root user
- **Read-only filesystem** - Minimal writable paths
- **Resource limits** - CPU and memory constraints
- **Secret management** - Sensitive data in Kubernetes secrets

## 📝 Roadmap

- [ ] **Web UI** - Dashboard for monitoring and configuration
- [ ] **Custom metrics** - Support for application-specific metrics
- [ ] **Multi-cluster** - Support for managing multiple clusters
- [ ] **Policy engine** - Advanced policy-based remediation
- [ ] **Integration hub** - More notification channels (Teams, PagerDuty)
- [ ] **ML detection** - Machine learning for anomaly detection
- [ ] **Audit logging** - Comprehensive audit trail
- [ ] **GitOps support** - Configuration as Code
- [ ] **Rule templates** - Pre-built namespace rule templates
- [ ] **Configuration validation** - Rule validation and testing framework

## 🤝 Contributing

We welcome contributions! Please see our [Contributing Guide](CONTRIBUTING.md) for details.

### Development Setup

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests
5. Submit a pull request

## 📄 License

This project is licensed under the Apache License 2.0 - see the [LICENSE](LICENSE) file for details.

## 🆘 Support

- 📖 [Documentation](https://kubeguardian.io/docs)
- 💬 [Slack Community](https://kubeguardian.slack.com)
- 🐛 [Issue Tracker](https://github.com/NotHarshhaa/kubeguardian/issues)
- 📧 [Email Support](mailto:support@kubeguardian.io)

## 🙏 Acknowledgments

- [Kubernetes](https://kubernetes.io/) - Container orchestration platform
- [Controller Runtime](https://github.com/kubernetes-sigs/controller-runtime) - Kubernetes controller framework
- [Slack Go SDK](https://github.com/slack-go/slack) - Slack integration

---

## Author & Community  

This project is crafted with 💡 by **[Harshhaa](https://github.com/NotHarshhaa)**.  
Your feedback is always welcome! Let's build together. 🚀  

📧 **Connect with me:**  
🔗 **GitHub**: [@NotHarshhaa](https://github.com/NotHarshhaa)  
🔗 **Portfolio**: [Personal Portfolio](https://notharshhaa.site)  
🔗 **Links - Portfolio**: [Links](https://link.notharshhaa.site)  
🔗 **Telegram Community**: [Join Here](https://t.me/prodevopsguy)  
🔗 **LinkedIn**: [Harshhaa Vardhan Reddy](https://www.linkedin.com/in/NotHarshhaa/)  

---

## Support the Project  

If this helped you, consider:  
✅ **Starring** ⭐ this repository  
✅ **Sharing** 📢 with your network  
✅ **Supporting** 📢 on [GitHub Sponsors](https://github.com/sponsors/NotHarshhaa)  
✅ **Supporting** 📢 on [BuyMeACoffee](https://www.buymeacoffee.com/NotHarshhaa)  
✅ **Supporting** 📢 on [Patreon](https://www.patreon.com/NotHarshhaa)  
✅ **Supporting** 📢 on [PayPal](https://www.paypal.com/paypalme/NotHarshhaa)  

---

### Stay Connected  

![Follow Me](https://imgur.com/2j7GSPs.png)

*Let's make Kubernetes self-healing a reality!* 🚀
