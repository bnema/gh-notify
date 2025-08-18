package cmd

import (
	"fmt"
	"strings"

	"github.com/bnema/gh-notify/internal/cache"
	"github.com/bnema/gh-notify/internal/service"
	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Check the status of the gh-notify systemd service",
	Long: `Display the current status of the gh-notify systemd service including:
- Service installation status
- Timer status (active/inactive)
- Last sync time from cache
- Recent service logs

This command helps you monitor whether the automated sync is working correctly.`,
	RunE: runStatus,
}

func runStatus(cmd *cobra.Command, args []string) error {
	// Check service installation
	systemdMgr, err := service.NewSystemdManager()
	if err != nil {
		return fmt.Errorf("failed to initialize systemd manager: %w", err)
	}

	isInstalled := systemdMgr.IsInstalled()
	
	fmt.Println("=== Service Status ===")
	if isInstalled {
		fmt.Println("✓ Service installed")
		
		// Get detailed status from systemctl
		output, err := systemdMgr.Status()
		if err != nil {
			fmt.Printf("⚠️  Failed to get service status: %v\n", err)
		} else {
			fmt.Println()
			parseAndDisplayStatus(output)
		}
	} else {
		fmt.Println("✗ Service not installed")
		fmt.Println("  Install with: gh-notify install-service")
		return nil
	}

	fmt.Println("\n=== Cache Status ===")
	
	// Check cache status
	cache := cache.New(cacheDir)
	if err := cache.Load(cacheDir); err != nil {
		fmt.Printf("⚠️  Failed to load cache: %v\n", err)
		return nil
	}

	notifications := cache.GetNotifications()
	
	fmt.Printf("Cached notifications: %d\n", len(notifications))
	
	if !cache.LastSync.IsZero() {
		fmt.Printf("Last sync: %s\n", cache.LastSync.Format("2006-01-02 15:04:05"))
	} else {
		fmt.Println("Last sync: Never")
	}

	if len(notifications) > 0 {
		// Show some recent notifications
		fmt.Println("\nRecent notifications:")
		count := len(notifications)
		if count > 5 {
			count = 5
		}
		
		for i := 0; i < count; i++ {
			notif := notifications[i]
			fmt.Printf("  • %s: %s\n", notif.Repository, truncateTitle(notif.Title))
		}
		
		if len(notifications) > 5 {
			fmt.Printf("  ... and %d more\n", len(notifications)-5)
		}
	}

	return nil
}

func parseAndDisplayStatus(output string) {
	lines := strings.Split(output, "\n")
	
	var activeStatus, enabledStatus, lastTrigger string
	var logLines []string
	inLogSection := false
	
	for _, line := range lines {
		line = strings.TrimSpace(line)
		
		if strings.Contains(line, "Active:") {
			activeStatus = line
		} else if strings.Contains(line, "Loaded:") && strings.Contains(line, "enabled") {
			enabledStatus = "enabled"
		} else if strings.Contains(line, "Loaded:") && strings.Contains(line, "disabled") {
			enabledStatus = "disabled"
		} else if strings.Contains(line, "Trigger:") {
			lastTrigger = line
		} else if strings.Contains(line, "Triggered:") {
			lastTrigger = line
		}
		
		// Collect recent log lines
		if inLogSection && line != "" {
			if len(logLines) < 5 {
				logLines = append(logLines, line)
			}
		}
		
		if strings.Contains(line, "Journal begins") || strings.Contains(line, "Logs begin") {
			inLogSection = true
		}
	}
	
	// Display parsed information
	if activeStatus != "" {
		if strings.Contains(activeStatus, "active (waiting)") {
			fmt.Printf("✓ Timer status: %s\n", activeStatus)
		} else if strings.Contains(activeStatus, "inactive") {
			fmt.Printf("⚠️  Timer status: %s\n", activeStatus)
		} else {
			fmt.Printf("Timer status: %s\n", activeStatus)
		}
	}
	
	if enabledStatus != "" {
		if enabledStatus == "enabled" {
			fmt.Printf("✓ Timer enabled: %s\n", enabledStatus)
		} else {
			fmt.Printf("⚠️  Timer enabled: %s\n", enabledStatus)
		}
	}
	
	if lastTrigger != "" {
		fmt.Printf("Next run: %s\n", strings.TrimPrefix(lastTrigger, "Trigger: "))
	}
	
	// Show recent logs if available
	if len(logLines) > 0 {
		fmt.Println("\nRecent logs:")
		for _, logLine := range logLines {
			fmt.Printf("  %s\n", logLine)
		}
	}
}

func truncateTitle(title string) string {
	if len(title) <= 50 {
		return title
	}
	return title[:47] + "..."
}