package cmd

import (
	"fmt"
	"os"
	"time"

	"github.com/bnema/gh-notify/internal/cache"
	"github.com/bnema/gh-notify/internal/github"
	"github.com/bnema/gh-notify/internal/notifier"
	"github.com/spf13/cobra"
)

var (
	all      bool
	noNotify bool
	since    time.Duration
)

var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Sync GitHub notifications and alert on new ones",
	Long: `Fetch GitHub notifications, compare with the local cache, and send
desktop notifications for any new notifications found.

The command will:
1. Load the existing notification cache
2. Fetch notifications from GitHub using your gh authentication
3. Compare with cached notifications to find new ones
4. Send desktop notifications for new notifications
5. Update the cache with all current notifications
6. Clean up old cache entries

This command is designed to be run periodically (e.g., via systemd timer).`,
	RunE: runSync,
}

func init() {
	syncCmd.Flags().BoolVar(&all, "all", false, "show all notifications, not just unread")
	syncCmd.Flags().BoolVar(&noNotify, "no-notify", false, "skip desktop notifications, just update cache")
	syncCmd.Flags().DurationVar(&since, "since", 0, "only check notifications updated since duration ago (e.g., 1h, 30m)")
}

func runSync(cmd *cobra.Command, args []string) error {
	if verbose {
		fmt.Printf("Starting sync with cache directory: %s\n", cacheDir)
	}

	// Initialize cache
	c := cache.New(cacheDir)
	if err := c.Load(cacheDir); err != nil {
		return fmt.Errorf("failed to load cache: %w", err)
	}

	if verbose {
		fmt.Printf("Loaded cache with %d existing notifications\n", len(c.GetNotifications()))
	}

	// Initialize GitHub client
	ghClient, err := github.NewClient()
	if err != nil {
		return fmt.Errorf("failed to create GitHub client: %w", err)
	}

	// Test authentication
	if err := ghClient.TestAuth(); err != nil {
		return fmt.Errorf("GitHub authentication failed: %w", err)
	}

	if verbose {
		fmt.Println("GitHub authentication successful")
	}

	// Fetch notifications
	notifications, err := ghClient.FetchNotifications(all)
	if err != nil {
		return fmt.Errorf("failed to fetch notifications: %w", err)
	}

	if verbose {
		fmt.Printf("Fetched %d notifications from GitHub\n", len(notifications))
	}

	// Filter by time if since is specified
	if since > 0 {
		cutoff := time.Now().Add(-since)
		var filtered []cache.CacheEntry
		for _, notif := range notifications {
			if notif.UpdatedAt.After(cutoff) {
				filtered = append(filtered, notif)
			}
		}
		notifications = filtered

		if verbose {
			fmt.Printf("Filtered to %d notifications updated since %v ago\n", len(notifications), since)
		}
	}

	// Add notifications to cache and get new ones
	newNotifications := c.AddNotifications(notifications)

	if verbose {
		fmt.Printf("Found %d new notifications\n", len(newNotifications))
	}

	// Send desktop notifications for new notifications
	if !noNotify && len(newNotifications) > 0 {
		notifier := notifier.New(true)

		if err := notifier.SendBulkNotification(newNotifications); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to send desktop notification: %v\n", err)
		} else if verbose {
			fmt.Println("Desktop notification sent")
		}
	}

	// Save updated cache
	if err := c.Save(cacheDir); err != nil {
		return fmt.Errorf("failed to save cache: %w", err)
	}

	if verbose {
		fmt.Println("Cache saved successfully")
	}

	// Output summary
	if len(newNotifications) > 0 {
		fmt.Printf("✓ %d new notifications found\n", len(newNotifications))
		if !noNotify {
			fmt.Println("✓ Desktop notification sent")
		}
	} else {
		fmt.Println("✓ No new notifications")
	}

	return nil
}
