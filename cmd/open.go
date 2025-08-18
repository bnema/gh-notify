package cmd

import (
	"fmt"
	"os/exec"
	"runtime"
	"strconv"

	"github.com/bnema/gh-notify/internal/cache"
	"github.com/spf13/cobra"
)

var openCmd = &cobra.Command{
	Use:   "open [notification-number]",
	Short: "Open a notification URL in the browser",
	Long: `Open a notification URL in the default web browser.

Use 'gh-notify list' to see notification numbers, then use 'gh-notify open N'
where N is the notification number from the list.

Examples:
  gh-notify open 1        # Open the first notification from the list
  gh-notify open 5        # Open the fifth notification from the list`,
	Args: cobra.ExactArgs(1),
	RunE: runOpen,
}

func init() {
	rootCmd.AddCommand(openCmd)
}

func runOpen(cmd *cobra.Command, args []string) error {
	// Parse notification number
	notifNum, err := strconv.Atoi(args[0])
	if err != nil {
		return fmt.Errorf("invalid notification number: %s", args[0])
	}

	if notifNum < 1 {
		return fmt.Errorf("notification number must be greater than 0")
	}

	// Load cache
	cache := cache.New(cacheDir)
	if err := cache.Load(cacheDir); err != nil {
		return fmt.Errorf("failed to load cache: %w", err)
	}

	notifications := cache.GetNotifications()
	if len(notifications) == 0 {
		return fmt.Errorf("no notifications found. Run 'gh-notify sync' first")
	}

	// Convert to 0-based index
	index := notifNum - 1
	if index >= len(notifications) {
		return fmt.Errorf("notification number %d not found. Only %d notifications available", notifNum, len(notifications))
	}

	notification := notifications[index]
	if notification.WebURL == "" {
		return fmt.Errorf("no URL available for notification %d", notifNum)
	}

	if verbose {
		fmt.Printf("Opening: %s\n", notification.WebURL)
		fmt.Printf("Title: %s\n", notification.Title)
	}

	// Open URL in browser
	if err := openURL(notification.WebURL); err != nil {
		return fmt.Errorf("failed to open URL: %w", err)
	}

	fmt.Printf("✓ Opened notification: %s\n", notification.Title)
	return nil
}

// openURL opens a URL in the default browser
func openURL(url string) error {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "linux":
		cmd = exec.Command("xdg-open", url)
	case "darwin": // macOS
		cmd = exec.Command("open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default:
		return fmt.Errorf("unsupported operating system: %s", runtime.GOOS)
	}

	return cmd.Start()
}