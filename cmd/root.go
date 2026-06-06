// SPDX-License-Identifier: Apache-2.0
// Copyright Authors of z9s

package cmd

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/spf13/cobra"
	"github.com/yourusername/z9s/internal/app"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

const (
	appName      = "z9s"
	shortAppDesc = "Unified Kubernetes CLI combining k9s and ktop"
	longAppDesc  = `z9s is a unified CLI tool that merges the power of k9s (cluster management)
and ktop (metrics visualization) into a single seamless experience.

Press Ctrl+F10 to toggle between k9s and ktop modes.`
)

var (
	// Version information (set at build time)
	version, commit, date = "dev", "dev", "dev"

	// CLI flags
	k9sFlags *k9sCliFlags
	ktopFlags *ktopCliFlags
	k8sFlags *genericclioptions.ConfigFlags

	rootCmd = &cobra.Command{
		Use:   appName,
		Short: shortAppDesc,
		Long:  longAppDesc,
		RunE:  run,
	}
)

// k9sCliFlags holds k9s-specific flags
type k9sCliFlags struct {
	RefreshRate *float32
	LogFile     *string
	LogLevel    *string
}

// ktopCliFlags holds ktop-specific flags
type ktopCliFlags struct {
	Namespace     *string
	AllNamespaces *bool
	Context       *string
	MetricsSource *string
}

func init() {
	// Initialize K8s flags
	k8sFlags = genericclioptions.NewConfigFlags(false)

	// Initialize k9s flags
	k9sFlags = &k9sCliFlags{
		RefreshRate: new(float32),
		LogFile:     new(string),
		LogLevel:    new(string),
	}
	*k9sFlags.RefreshRate = 2.0
	*k9sFlags.LogLevel = "info"

	// Initialize ktop flags
	ktopFlags = &ktopCliFlags{
		Namespace:     new(string),
		AllNamespaces: new(bool),
		Context:       new(string),
		MetricsSource: new(string),
	}
	*ktopFlags.MetricsSource = "prometheus"

	// Add flags to root command
	initK9sFlags()
	initKtopFlags()
	k8sFlags.AddFlags(rootCmd.Flags())

	// Add version command
	rootCmd.AddCommand(versionCmd())
}

// Execute runs the root command
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

// run is the main entry point
func run(*cobra.Command, []string) error {
	logger := initializeLogger(*k9sFlags.LogLevel)

	logger.Info("🚀 z9s starting up...",
		"version", version,
		"commit", commit,
		"mode", app.GetModeName(app.ModeK9s),
	)

	// Print welcome message
	fmt.Println(`
╔════════════════════════════════════════════════════════════╗
║                     KT9S - v0.0.1                         ║
║         Unified Kubernetes CLI (k9s + ktop)              ║
╠════════════════════════════════════════════════════════════╣
║                                                            ║
║  Keybindings:                                              ║
║    Ctrl+F10  - Toggle between k9s and ktop modes         ║
║    q         - Quit application                           ║
║    Ctrl+C    - Force quit                                 ║
║                                                            ║
║  Status: 🟡 Skeleton complete, integration pending        ║
║  Next: Wire up k9s and ktop actual apps                   ║
║                                                            ║
╚════════════════════════════════════════════════════════════╝
`)

	logger.Info("Skeleton ready - awaiting app integration")

	return nil
}

// initializeLogger sets up the logger
func initializeLogger(level string) *slog.Logger {
	var logLevel slog.Level
	switch level {
	case "debug":
		logLevel = slog.LevelDebug
	case "warn":
		logLevel = slog.LevelWarn
	case "error":
		logLevel = slog.LevelError
	default:
		logLevel = slog.LevelInfo
	}

	return slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: logLevel,
	}))
}

// initK9sFlags registers k9s-specific flags
func initK9sFlags() {
	rootCmd.Flags().Float32VarP(
		k9sFlags.RefreshRate,
		"refresh", "r",
		2.0,
		"Specify the default refresh rate as a float (sec)",
	)
	rootCmd.Flags().StringVar(
		k9sFlags.LogFile,
		"logFile",
		"",
		"Log file location (leave empty to disable file logging)",
	)
	rootCmd.Flags().StringVar(
		k9sFlags.LogLevel,
		"log-level",
		"info",
		"Log level: debug, info, warn, error",
	)
}

// initKtopFlags registers ktop-specific flags
func initKtopFlags() {
	rootCmd.Flags().StringVarP(
		ktopFlags.Namespace,
		"namespace", "n",
		"default",
		"Kubernetes namespace",
	)
	rootCmd.Flags().BoolVarP(
		ktopFlags.AllNamespaces,
		"all-namespaces", "A",
		false,
		"Show metrics for all namespaces",
	)
	rootCmd.Flags().StringVar(
		ktopFlags.Context,
		"context",
		"",
		"Kubernetes context to use",
	)
	rootCmd.Flags().StringVar(
		ktopFlags.MetricsSource,
		"metrics-source",
		"prometheus",
		"Metrics source: 'prometheus' or 'metrics-server'",
	)
}

// versionCmd returns the version command
func versionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print version information",
		Run: func(_ *cobra.Command, _ []string) {
			fmt.Printf("%s version %s (commit: %s, date: %s)\n", appName, version, commit, date)
		},
	}
}
