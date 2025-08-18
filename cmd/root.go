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
)

var rootCmd = &cobra.Command{
	Use:   "gh-notify",
	Short: "GitHub notification monitor with desktop alerts",
	Long: `A CLI tool that monitors GitHub notifications and sends desktop alerts for new ones.

gh-notify leverages your existing gh CLI authentication to fetch GitHub notifications,
maintains a cache of seen notifications, and sends desktop notifications for new items.

Perfect for running as a systemd service to get real-time notifications.`,
	Version: "1.0.0",
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	// Global flags
	rootCmd.PersistentFlags().StringVar(&cacheDir, "cache-dir", "", "cache directory (default: ~/.cache/gh-notify)")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "verbose output")
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.gh-notify.yaml)")

	// Add subcommands
	rootCmd.AddCommand(syncCmd)
	rootCmd.AddCommand(listCmd)
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

func getVerbose() bool {
	return verbose
}

func getCacheDir() string {
	return cacheDir
}