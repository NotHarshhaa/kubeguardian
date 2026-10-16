package main

import (
	"context"
	"encoding/json"
	"fmt"
	"flag"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/NotHarshhaa/kubeguardian/pkg/config"
	"github.com/NotHarshhaa/kubeguardian/pkg/controller"
	"github.com/NotHarshhaa/kubeguardian/pkg/dashboard"
	"github.com/NotHarshhaa/kubeguardian/pkg/health"
	"github.com/NotHarshhaa/kubeguardian/pkg/metrics"
	"github.com/NotHarshhaa/kubeguardian/pkg/version"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

var (
	configFile     = flag.String("config", "", "Path to configuration file")
	metricsAddr    = flag.String("metrics-bind-address", ":8080", "The address the metric endpoint binds to.")
	probeAddr      = flag.String("health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	leaderElection = flag.Bool("leader-elect", false, "Enable leader election for controller manager. "+
		"Enabling this will ensure there is only one active controller manager.")
	dryRunMode = flag.Bool("dry-run", false, "Enable dry-run mode to simulate remediation actions without making changes")
	enableReload = flag.Bool("enable-config-reload", false, "Enable automatic config file reloading on changes")
	zapOpts    = zap.Options{
		Development: true,
	}
)

func init() {
	flag.StringVar(configFile, "c", "", "Path to configuration file (shorthand)")
	flag.BoolVar(dryRunMode, "d", false, "Enable dry-run mode (shorthand)")
	flag.BoolVar(enableReload, "r", false, "Enable config reload (shorthand)")
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

	// Create controller
	ctrl, err := controller.NewController(cfg, metricsCollector)
	if err != nil {
		logger.Error(err, "Failed to create controller")
		os.Exit(1)
	}

	// Initialize health checks
	healthChecker := health.NewHealthCheck(version.Version, ctrl.GetClient())

	// Setup config reloader if enabled
	var configReloader *config.Reloader
	if *enableReload && *configFile != "" {
		configReloader, err = config.NewReloader(*configFile)
		if err != nil {
			logger.Error(err, "Failed to create config reloader")
			os.Exit(1)
		}
		configReloader.SetConfig(cfg)
		configReloader.RegisterCallback(func(newCfg *config.Config) {
			logger.Info("Configuration reloaded via file watcher")
		})
		if err := configReloader.Start(); err != nil {
			logger.Error(err, "Failed to start config reloader")
			os.Exit(1)
		}
		logger.Info("Config hot-reload enabled", "configFile", *configFile)
	}

	// Setup HTTP servers for health checks and metrics
	setupHTTPServers(cfg, healthChecker, metricsCollector, ctrl)

	// Log configuration
	logger.Info("Configuration loaded",
		"metricsAddr", cfg.Controller.MetricsAddr,
		"probeAddr", cfg.Controller.ProbeAddr,
		"leaderElection", cfg.Controller.LeaderElection,
		"remediationEnabled", cfg.Remediation.Enabled,
		"slackEnabled", cfg.Notification.Slack.Enabled,
		"emailEnabled", cfg.Notification.Email.Enabled,
		"webhookEnabled", cfg.Notification.Webhook.Enabled,
		"nodeDetectionEnabled", cfg.Detection.NodeDetection.Enabled,
		"configReloadEnabled", *enableReload,
		"dryRun", cfg.Remediation.DryRun,
	)

	// Setup signal handling with graceful shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	// Setup graceful shutdown
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Start the controller in a goroutine
	ctrlDone := make(chan error, 1)
	go func() {
		if err := ctrl.Run(ctx); err != nil {
			logger.Error(err, "Controller failed")
		}
		ctrlDone <- nil
	}()

	// Start metrics updater
	go startMetricsUpdater(ctx, metricsCollector)

	// Wait for signals with graceful shutdown
	<-sigCh
	logger.Info("Shutdown signal received, stopping KubeGuardian")

	// Stop config reloader if running
	if configReloader != nil {
		if err := configReloader.Stop(); err != nil {
			logger.Error(err, "Failed to stop config reloader")
		}
	}

	// Graceful shutdown with timeout
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	// Cancel context to stop the controller
	cancel()

	// Wait for graceful shutdown or timeout
	select {
	case <-ctrlDone:
		logger.Info("KubeGuardian stopped gracefully")
	case <-shutdownCtx.Done():
		logger.Info("KubeGuardian stopped due to timeout")
	}
}

// setupHTTPServers sets up HTTP servers for health checks and metrics
func setupHTTPServers(cfg *config.Config, healthChecker *health.HealthCheck, metricsCollector *metrics.Metrics, ctrl *controller.Controller) {
	// Create a mux that combines all health and metrics routes
	mux := http.NewServeMux()

	// Health check endpoint
	mux.HandleFunc("/", healthChecker.HTTPHandler())

	// Readiness probe
	mux.HandleFunc("/readyz", healthChecker.ReadinessHandler())

	// Liveness probe
	mux.HandleFunc("/healthz", healthChecker.LivenessHandler())

	// Metrics endpoint
	mux.Handle("/metrics", promhttp.Handler())

	// Remediation history API
	mux.HandleFunc("/api/v1/remediation/history", makeHistoryHandler(ctrl))
	mux.HandleFunc("/api/v1/remediation/stats", makeStatsHandler(ctrl))

	// Dashboard API
	dash := dashboard.NewDashboard(ctrl)
	dash.RegisterRoutes(mux)

	// Setup health and metrics server
	healthServer := &http.Server{
		Addr:    cfg.Controller.ProbeAddr,
		Handler: mux,
	}

	// Setup dedicated metrics server
	metricsServer := &http.Server{
		Addr:    cfg.Controller.MetricsAddr,
		Handler: promhttp.Handler(),
	}

	go func() {
		if err := healthServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Log.Error(err, "Health server failed")
		}
	}()

	go func() {
		if err := metricsServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Log.Error(err, "Metrics server failed")
		}
	}()
}

// makeHistoryHandler returns an HTTP handler for remediation history
func makeHistoryHandler(ctrl *controller.Controller) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		namespace := r.URL.Query().Get("namespace")
		action := r.URL.Query().Get("action")

		limitStr := r.URL.Query().Get("limit")
		limit := 0
		if limitStr != "" {
			if l, err := parseInt(limitStr); err == nil {
				limit = l
			}
		}

		history := ctrl.GetRemediationHistory(namespace, action, limit)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"history": history,
			"count":   len(history),
		})
	}
}

// makeStatsHandler returns an HTTP handler for remediation stats
func makeStatsHandler(ctrl *controller.Controller) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		stats := ctrl.GetRemediationStats()

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(stats)
	}
}

// parseInt parses a string to int
func parseInt(s string) (int, error) {
	var result int
	for _, c := range s {
		if c < '0' || c > '9' {
			return 0, fmt.Errorf("invalid integer: %s", s)
		}
		result = result*10 + int(c-'0')
	}
	return result, nil
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
