package cmd

import (
	"fmt"
	"os"

	"github.com/bnema/gh-notify/internal/cache"
	"github.com/spf13/cobra"
)

var (
	cacheDir string
	verbose  bool
	cfgFile  string

	// Version information (injected at build time via ldflags)
	version   = "dev"
	commit    = "none"
	buildDate = "unknown"
)

var rootCmd = &cobra.Command{
	Use:   "gh-notify",
	Short: "GitHub notification monitor with desktop alerts",
	Long: `A CLI tool that monitors GitHub notifications and sends desktop alerts for new ones.

gh-notify leverages your existing gh CLI authentication to fetch unread GitHub notifications,
maintains a smart cache of notifications requiring attention, and sends desktop notifications
for new items.

Perfect for running as a systemd service to get real-time notifications.`,
	Version: version,
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	// Set custom version template to show commit and build date
	rootCmd.SetVersionTemplate(fmt.Sprintf("gh-notify version %s (commit: %s, built: %s)\n", version, commit, buildDate))

	cobra.OnInitialize(initConfig)

	// Global flags
	rootCmd.PersistentFlags().StringVar(&cacheDir, "cache-dir", "", "cache directory (default: ~/.cache/gh-notify)")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "verbose output")
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.gh-notify.yaml)")

	// Add subcommands
	rootCmd.AddCommand(syncCmd)
	rootCmd.AddCommand(listCmd)
	rootCmd.AddCommand(openCmd)
	rootCmd.AddCommand(clearCmd)
	rootCmd.AddCommand(installServiceCmd)
	rootCmd.AddCommand(statusCmd)
}

func initConfig() {
	if cacheDir == "" {
		defaultCacheDir, err := cache.GetDefaultCacheDir()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error getting default cache directory: %v\n", err)
			os.Exit(1)
		}
		cacheDir = defaultCacheDir
	}
}
