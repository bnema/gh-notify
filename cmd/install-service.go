package cmd

import (
	"fmt"
	"time"

	"github.com/bnema/gh-notify/internal/service"
	"github.com/spf13/cobra"
)

var (
	interval  time.Duration
	uninstall bool
	dryRun    bool
)

var installServiceCmd = &cobra.Command{
	Use:   "install-service",
	Short: "Install and enable systemd user service for automatic syncing",
	Long: `Installs a systemd user service and timer that runs gh-notify sync 
at the specified interval. The service will start automatically on login.

The service creates:
- ~/.config/systemd/user/gh-notify.service
- ~/.config/systemd/user/gh-notify.timer

Use --uninstall to remove the service completely.
Use --dry-run to see what would be installed without making changes.`,
	RunE: runInstallService,
}

func init() {
	installServiceCmd.Flags().DurationVar(&interval, "interval", 10*time.Second, "sync interval (e.g., 10s, 1m, 5m)")
	installServiceCmd.Flags().BoolVar(&uninstall, "uninstall", false, "remove the service instead of installing")
	installServiceCmd.Flags().BoolVar(&dryRun, "dry-run", false, "show what would be installed without doing it")
}

func runInstallService(cmd *cobra.Command, args []string) error {
	systemdMgr, err := service.NewSystemdManager()
	if err != nil {
		return fmt.Errorf("failed to initialize systemd manager: %w", err)
	}

	if uninstall {
		return runUninstall(systemdMgr)
	}

	return runInstall(systemdMgr)
}

func runInstall(systemdMgr *service.SystemdManager) error {
	// Check if already installed
	if systemdMgr.IsInstalled() && !dryRun {
		fmt.Println("⚠️  Service is already installed")
		fmt.Println("Use --uninstall first, or check status with: gh-notify status")
		return nil
	}

	if dryRun {
		fmt.Printf("Installing gh-notify systemd service (interval: %v)\n\n", interval)
		return systemdMgr.Install(interval, true)
	}

	if verbose {
		fmt.Printf("Installing systemd service with %v interval...\n", interval)
	}

	if err := systemdMgr.Install(interval, false); err != nil {
		return fmt.Errorf("failed to install service: %w", err)
	}

	fmt.Println("✓ Service installed successfully!")
	fmt.Printf("✓ Timer configured to run every %v\n", interval)
	fmt.Println("✓ Service enabled and started")
	fmt.Println()
	fmt.Println("The service will now run automatically on login.")
	fmt.Println("Check status with: gh-notify status")
	fmt.Println("View logs with: journalctl --user -u gh-notify.service -f")

	return nil
}

func runUninstall(systemdMgr *service.SystemdManager) error {
	if !systemdMgr.IsInstalled() {
		fmt.Println("ℹ️  Service is not installed")
		return nil
	}

	if verbose {
		fmt.Println("Uninstalling systemd service...")
	}

	if err := systemdMgr.Uninstall(); err != nil {
		return fmt.Errorf("failed to uninstall service: %w", err)
	}

	fmt.Println("✓ Service uninstalled successfully!")
	fmt.Println("✓ Timer stopped and disabled")
	fmt.Println("✓ Service files removed")

	return nil
}