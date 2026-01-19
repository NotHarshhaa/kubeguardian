package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/NotHarshhaa/kubeguardian/pkg/config"
	"github.com/NotHarshhaa/kubeguardian/pkg/controller"
	"github.com/NotHarshhaa/kubeguardian/pkg/version"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

var (
	configFile   = flag.String("config", "", "Path to configuration file")
	metricsAddr  = flag.String("metrics-bind-address", ":8080", "The address the metric endpoint binds to.")
	probeAddr    = flag.String("health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	leaderElection = flag.Bool("leader-elect", false, "Enable leader election for controller manager. "+
		"Enabling this will ensure there is only one active controller manager.")
	zapOpts = zap.Options{
		Development: true,
	}
)

func init() {
	flag.StringVar(&configFile, "c", "", "Path to configuration file (shorthand)")
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

	// Setup signal handling
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	// Start the controller in a goroutine
	go func() {
		if err := ctrl.Run(ctx); err != nil {
			logger.Error(err, "Controller failed")
			os.Exit(1)
		}
	}()

	// Wait for signals
	<-sigCh
	logger.Info("Shutdown signal received, stopping KubeGuardian")

	// Cancel context to stop the controller
	cancel()

	logger.Info("KubeGuardian stopped")
}
