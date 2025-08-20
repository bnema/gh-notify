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
	noNotify      bool
	since         time.Duration
	waybarOutput  bool
	includeStars  bool
	starsOnly     bool
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
	syncCmd.Flags().BoolVar(&includeStars, "include-stars", true, "track star events in addition to notifications")
	syncCmd.Flags().BoolVar(&starsOnly, "stars-only", false, "only check for star events, skip regular notifications")
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

	var newNotifications []cache.CacheEntry

	// Fetch notifications (unless stars-only mode)
	if !starsOnly {
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
		newNotifications = c.AddNotifications(notifications)

		if verbose {
			fmt.Printf("Found %d new notifications\n", len(newNotifications))
		}
	}

	// Fetch star events if enabled (but don't cache them)
	var recentStarEvents []github.StarEvent
	if includeStars || starsOnly {
		if verbose {
			fmt.Println("Fetching star events using GraphQL...")
		}

		// Get cutoff time - only check for stars since last sync
		cutoff := c.LastEventSync
		if cutoff.IsZero() {
			// If no previous sync, only show stars from last 4 hours to avoid missing recent events
			cutoff = time.Now().UTC().Add(-4 * time.Hour)
		}

		starEvents, err := ghClient.FetchRecentStars(cutoff)
		if err != nil {
			return fmt.Errorf("failed to fetch star events: %w", err)
		}

		if verbose {
			fmt.Printf("Found %d new star events since %v\n", len(starEvents), cutoff.Format("15:04:05"))
		}

		// Update last event sync time (use UTC to match GitHub API)
		c.LastEventSync = time.Now().UTC()

		recentStarEvents = starEvents
	}

	// Send desktop notifications for new notifications and star events
	if !noNotify {
		notifier := notifier.New(true)
		
		// Send notifications for regular GitHub notifications
		if len(newNotifications) > 0 {
			if err := notifier.SendBulkNotification(newNotifications); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: failed to send desktop notification: %v\n", err)
			} else if verbose {
				fmt.Println("Desktop notification sent for regular notifications")
			}
		}
		
		// Send notifications for new star events
		if len(recentStarEvents) > 0 {
			if err := notifier.SendStarNotifications(recentStarEvents); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: failed to send star notification: %v\n", err)
			} else if verbose {
				fmt.Println("Desktop notification sent for new stars")
			}
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
		
		// Get stars from last hour for tooltip (always fetch for waybar)
		var recentTooltipStars []github.StarEvent
		oneHourAgo := time.Now().UTC().Add(-1 * time.Hour)
		tooltipStars, err := ghClient.FetchRecentStars(oneHourAgo)
		if err == nil {
			recentTooltipStars = tooltipStars
		}

		var waybar WaybarOutput
		totalCount := totalNotifications + len(recentTooltipStars)
		if totalCount > 0 {
			var text string
			if totalNotifications > 0 && len(recentTooltipStars) > 0 {
				text = fmt.Sprintf("%s (%d) %s (%d)", nerdfonts.GitHub, totalNotifications, nerdfonts.StarredRepo, len(recentTooltipStars))
			} else if totalNotifications > 0 {
				text = fmt.Sprintf("%s (%d)", nerdfonts.GitHub, totalNotifications)
			} else {
				text = fmt.Sprintf("%s (%d)", nerdfonts.StarredRepo, len(recentTooltipStars))
			}
			
			waybar = WaybarOutput{
				Text:    text,
				Tooltip: buildTooltip(c.GetNotifications(), recentTooltipStars),
			}
		} else {
			waybar = WaybarOutput{
				Text:    fmt.Sprintf("%s (0)", nerdfonts.GitHub),
				Tooltip: "No notifications or recent stars",
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
	var summaryParts []string
	if len(newNotifications) > 0 {
		summaryParts = append(summaryParts, fmt.Sprintf("%d new notifications", len(newNotifications)))
	}
	if len(recentStarEvents) > 0 {
		summaryParts = append(summaryParts, fmt.Sprintf("%d new stars", len(recentStarEvents)))
	}
	
	if len(summaryParts) > 0 {
		fmt.Printf("✓ %s found\n", strings.Join(summaryParts, " and "))
		if !noNotify {
			fmt.Println("✓ Desktop notifications sent")
		}
	} else {
		fmt.Println("✓ No new notifications or stars")
	}

	return nil
}

func buildTooltip(notifications []cache.CacheEntry, recentStars []github.StarEvent) string {
	var tooltip strings.Builder
	
	// Add notifications section
	if len(notifications) > 0 {
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
	}
	
	// Add recent stars section
	if len(recentStars) > 0 {
		if len(notifications) > 0 {
			tooltip.WriteString("\n")
		}
		tooltip.WriteString(fmt.Sprintf("%s Recent Stars (last hour):\n", nerdfonts.StarredRepo))
		
		// Sort stars by time (newest first)
		sort.Slice(recentStars, func(i, j int) bool {
			return recentStars[i].StarredAt.After(recentStars[j].StarredAt)
		})
		
		for _, star := range recentStars {
			timeAgo := time.Since(star.StarredAt).Round(time.Minute)
			tooltip.WriteString(fmt.Sprintf("  %s %s starred %s (%v ago)\n", 
				nerdfonts.StarredRepo, star.StarredBy, star.Repository, timeAgo))
		}
	}
	
	if len(notifications) == 0 && len(recentStars) == 0 {
		return "No notifications or recent stars"
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
