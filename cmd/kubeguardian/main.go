package main

import (
	"context"
	"flag"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/NotHarshhaa/kubeguardian/pkg/config"
	"github.com/NotHarshhaa/kubeguardian/pkg/controller"
	"github.com/NotHarshhaa/kubeguardian/pkg/health"
	"github.com/NotHarshhaa/kubeguardian/pkg/metrics"
	"github.com/NotHarshhaa/kubeguardian/pkg/version"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

var (
	configFile     = flag.String("config", "", "Path to configuration file")
	metricsAddr    = flag.String("metrics-bind-address", ":8080", "The address the metric endpoint binds to.")
	probeAddr      = flag.String("health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	leaderElection = flag.Bool("leader-elect", false, "Enable leader election for controller manager. "+
		"Enabling this will ensure there is only one active controller manager.")
	dryRunMode     = flag.Bool("dry-run", false, "Enable dry-run mode to simulate remediation actions without making changes")
	zapOpts = zap.Options{
		Development: true,
	}
)

func init() {
	flag.StringVar(configFile, "c", "", "Path to configuration file (shorthand)")
	flag.BoolVar(dryRunMode, "d", false, "Enable dry-run mode (shorthand)")
	zapOpts.BindFlags(flag.CommandLine)
}

func main() {
	flag.Parse()

	// Setup logging
	logger := zap.New(zap.UseFlagOptions(&zapOpts))
	log.SetLogger(logger)

	ctx := log.IntoContext(context.Background(), logger)
	logger.Info("Starting KubeGuardian", "version", version.Version)

	// Load configuration
	cfg, err := config.LoadConfig(*configFile)
	if err != nil {
		logger.Error(err, "Failed to load configuration")
		os.Exit(1)
	}

	// Override config with command line flags
	if *metricsAddr != ":8080" {
		cfg.Controller.MetricsAddr = *metricsAddr
	}
	if *probeAddr != ":8081" {
		cfg.Controller.ProbeAddr = *probeAddr
	}
	cfg.Controller.LeaderElection = *leaderElection
	
	// Override dry-run mode if specified via command line
	if *dryRunMode {
		cfg.Remediation.DryRun = true
	}

	// Initialize metrics
	metricsCollector := metrics.NewMetrics()

	// Initialize health checks
	healthChecker := health.NewHealthCheck(version.Version)
	healthChecker.RegisterCheck(health.NewKubernetesAPICheck())
	healthChecker.RegisterCheck(health.NewMemoryCheck(80.0)) // 80% memory threshold
	healthChecker.RegisterCheck(health.NewDiskCheck("/", 85.0))   // 85% disk threshold

	// Setup HTTP servers for health checks and metrics
	setupHTTPServers(cfg, healthChecker, metricsCollector)

	// Log configuration
	logger.Info("Configuration loaded",
		"metricsAddr", cfg.Controller.MetricsAddr,
		"probeAddr", cfg.Controller.ProbeAddr,
		"leaderElection", cfg.Controller.LeaderElection,
		"remediationEnabled", cfg.Remediation.Enabled,
		"slackEnabled", cfg.Notification.Slack.Enabled,
		"dryRun", cfg.Remediation.DryRun,
	)

	// Create controller
	ctrl, err := controller.NewController(cfg)
	if err != nil {
		logger.Error(err, "Failed to create controller")
		os.Exit(1)
	}

	// Setup graceful shutdown
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Setup signal handling with graceful shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	// Start the controller in a goroutine
	go func() {
		if err := ctrl.Run(ctx); err != nil {
			logger.Error(err, "Controller failed")
			os.Exit(1)
		}
	}()

	// Start metrics updater
	go startMetricsUpdater(ctx, metricsCollector)

	// Wait for signals with graceful shutdown
	<-sigCh
	logger.Info("Shutdown signal received, stopping KubeGuardian")

	// Graceful shutdown with timeout
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	// Cancel context to stop the controller
	cancel()

	// Wait for graceful shutdown or timeout
	done := make(chan struct{})
	go func() {
		// Give some time for cleanup
		time.Sleep(5 * time.Second)
		close(done)
	}()

	select {
	case <-done:
		logger.Info("KubeGuardian stopped gracefully")
	case <-shutdownCtx.Done():
		logger.Info("KubeGuardian stopped due to timeout")
	}
}

// setupHTTPServers sets up HTTP servers for health checks and metrics
func setupHTTPServers(cfg *config.Config, healthChecker *health.HealthCheck, metricsCollector *metrics.Metrics) {
	// Setup health check server
	healthServer := &http.Server{
		Addr:    cfg.Controller.ProbeAddr,
		Handler: healthChecker.HTTPHandler(),
	}

	// Setup readiness probe
	http.HandleFunc("/readyz", healthChecker.ReadinessHandler())
	http.HandleFunc("/healthz", healthChecker.LivenessHandler())

	// Setup metrics server (handled by controller-runtime)
	go func() {
		if err := healthServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Log.Error(err, "Health server failed")
		}
	}()
}

// startMetricsUpdater starts a goroutine to update metrics periodically
func startMetricsUpdater(ctx context.Context, metricsCollector *metrics.Metrics) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			metricsCollector.UpdateUptime()
		}
	}
}
