package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/bnema/gh-notify/internal/cache"
	"github.com/bnema/gh-notify/internal/github"
	"github.com/bnema/gh-notify/internal/nerdfonts"
	"github.com/bnema/gh-notify/internal/notifier"
	"github.com/spf13/cobra"
)

var (
	noNotify     bool
	since        time.Duration
	waybarOutput bool
)

type WaybarOutput struct {
	Text    string `json:"text"`
	Tooltip string `json:"tooltip"`
}

var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Sync GitHub notifications and alert on new ones",
	Long: `Fetch unread GitHub notifications, compare with the local cache, and send
desktop notifications for any new notifications found.

The command will:
1. Load the existing notification cache
2. Fetch unread notifications from GitHub using your gh authentication
3. Compare with cached notifications to find new ones
4. Send desktop notifications for new notifications
5. Update the cache with current unread notifications
6. Remove any notifications that are no longer unread (handled on GitHub)
7. Clean up old cache entries

This command is designed to be run periodically (e.g., via systemd timer).`,
	RunE: runSync,
}

func init() {
	syncCmd.Flags().BoolVar(&noNotify, "no-notify", false, "skip desktop notifications, just update cache")
	syncCmd.Flags().DurationVar(&since, "since", 0, "only check notifications updated since duration ago (e.g., 1h, 30m)")
	syncCmd.Flags().BoolVar(&waybarOutput, "waybar-output", false, "output JSON for waybar integration")
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
	notifications, err := ghClient.FetchNotifications()
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

	// Handle waybar output
	if waybarOutput {
		totalNotifications := len(c.GetNotifications())

		var waybar WaybarOutput
		if totalNotifications > 0 {
			waybar = WaybarOutput{
				Text:    fmt.Sprintf("(%d)", totalNotifications),
				Tooltip: buildTooltip(c.GetNotifications()),
			}
		} else {
			waybar = WaybarOutput{
				Text:    "",
				Tooltip: "",
			}
		}

		jsonOutput, err := json.Marshal(waybar)
		if err != nil {
			return fmt.Errorf("failed to marshal waybar output: %w", err)
		}

		fmt.Println(string(jsonOutput))
		return nil // Exit early to only output JSON
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

func buildTooltip(notifications []cache.CacheEntry) string {
	if len(notifications) == 0 {
		return "No pending notifications"
	}

	var tooltip strings.Builder
	tooltip.WriteString("GitHub Notifications:\n")

	// Sort notifications by repository for better organization
	sort.Slice(notifications, func(i, j int) bool {
		if notifications[i].Repository != notifications[j].Repository {
			return notifications[i].Repository < notifications[j].Repository
		}
		return notifications[i].UpdatedAt.After(notifications[j].UpdatedAt)
	})

	currentRepo := ""
	for _, notif := range notifications {
		if notif.Repository != currentRepo {
			if currentRepo != "" {
				tooltip.WriteString("\n")
			}
			tooltip.WriteString(fmt.Sprintf("%s %s:\n", nerdfonts.Repository, notif.Repository))
			currentRepo = notif.Repository
		}

		// Format notification with Nerd Font icon
		icon := getNotificationIcon(notif.Reason, notif.Type)
		tooltip.WriteString(fmt.Sprintf("  %s %s (%s)\n", icon, notif.Title, notif.Reason))
	}

	return tooltip.String()
}

func getNotificationIcon(reason, notifType string) string {
	switch reason {
	case "review_requested":
		return nerdfonts.ReviewRequested
	case "assign":
		return nerdfonts.Assign
	case "mention":
		return nerdfonts.Mention
	case "author":
		return nerdfonts.Author
	case "state_change":
		return nerdfonts.StateChange
	default:
		switch notifType {
		case "PullRequest":
			return nerdfonts.PullRequest
		case "Issue":
			return nerdfonts.Issue
		case "Release":
			return nerdfonts.Release
		default:
			return nerdfonts.DefaultNotif
		}
	}
}
