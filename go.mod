module github.com/kubeguardian/kubeguardian

go 1.21

require (
	k8s.io/api v0.28.0
	k8s.io/apimachinery v0.28.0
	k8s.io/client-go v0.28.0
	k8s.io/code-generator v0.28.0
	sigs.k8s.io/controller-runtime v0.16.0
	github.com/go-logr/logr v1.2.4
	github.com/prometheus/client_golang v1.16.0
	github.com/slack-go/slack v0.14.0
	gopkg.in/yaml.v3 v3.0.1
	github.com/spf13/cobra v1.7.0
	github.com/spf13/viper v1.16.0
)
