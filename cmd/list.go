package cmd

import (
	"fmt"
	"os"
	"sort"
	"text/tabwriter"
	"time"

	"github.com/bnema/gh-notify/internal/cache"
	"github.com/spf13/cobra"
)

var (
	limit      int
	repository string
	reason     string
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List cached unread notifications",
	Long: `Display unread notifications from the local cache in a formatted table.

This command shows all cached notifications with their repository,
title, reason, and timestamps. All displayed notifications are unread
and require your attention.`,
	RunE: runList,
}

func init() {
	listCmd.Flags().IntVarP(&limit, "limit", "l", 20, "maximum number of notifications to show")
	listCmd.Flags().StringVarP(&repository, "repository", "r", "", "filter by repository name (supports partial matching)")
	listCmd.Flags().StringVar(&reason, "reason", "", "filter by notification reason")
}

func runList(cmd *cobra.Command, args []string) error {
	// Load cache
	c := cache.New(cacheDir)
	if err := c.Load(cacheDir); err != nil {
		return fmt.Errorf("failed to load cache: %w", err)
	}

	notifications := c.GetNotifications()

	// Apply filters
	if repository != "" {
		var filtered []cache.CacheEntry
		for _, notif := range notifications {
			if containsIgnoreCase(notif.Repository, repository) {
				filtered = append(filtered, notif)
			}
		}
		notifications = filtered
	}

	if reason != "" {
		var filtered []cache.CacheEntry
		for _, notif := range notifications {
			if notif.Reason == reason {
				filtered = append(filtered, notif)
			}
		}
		notifications = filtered
	}

	// Sort by UpdatedAt (newest first)
	sort.Slice(notifications, func(i, j int) bool {
		return notifications[i].UpdatedAt.After(notifications[j].UpdatedAt)
	})

	// Apply limit
	if limit > 0 && len(notifications) > limit {
		notifications = notifications[:limit]
	}

	// Display results
	if len(notifications) == 0 {
		fmt.Println("No notifications found.")
		return nil
	}

	// Create table writer
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	defer func() {
		if err := w.Flush(); err != nil {
			fmt.Fprintf(os.Stderr, "Error flushing table: %v\n", err)
		}
	}()

	// Header
	if _, err := fmt.Fprintln(w, "#\tREPOSITORY\tTYPE\tREASON\tAGE\tTITLE\tURL"); err != nil {
		return fmt.Errorf("failed to write header: %w", err)
	}
	if _, err := fmt.Fprintln(w, "-\t----------\t----\t------\t---\t-----\t---"); err != nil {
		return fmt.Errorf("failed to write header separator: %w", err)
	}

	// Rows
	now := time.Now().UTC()
	for i, notif := range notifications {
		age := formatAge(now.Sub(notif.UpdatedAt))
		title := truncateString(notif.Title, 40)
		url := truncateString(notif.WebURL, 50)
		notifType := notif.Type
		if notifType == "" {
			notifType = "Unknown"
		}

		if _, err := fmt.Fprintf(w, "%d\t%s\t%s\t%s\t%s\t%s\t%s\n",
			i+1,
			notif.Repository,
			notifType,
			notif.Reason,
			age,
			title,
			url); err != nil {
			return fmt.Errorf("failed to write notification row: %w", err)
		}
	}

	// Summary
	fmt.Printf("\nShowing %d notifications", len(notifications))
	if limit > 0 && len(c.GetNotifications()) > limit {
		fmt.Printf(" (limited from %d total)", len(c.GetNotifications()))
	}
	fmt.Println()

	return nil
}

func formatAge(duration time.Duration) string {
	if duration < time.Minute {
		return fmt.Sprintf("%ds", int(duration.Seconds()))
	}
	if duration < time.Hour {
		return fmt.Sprintf("%dm", int(duration.Minutes()))
	}
	if duration < 24*time.Hour {
		return fmt.Sprintf("%dh", int(duration.Hours()))
	}
	days := int(duration.Hours() / 24)
	return fmt.Sprintf("%dd", days)
}

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

func containsIgnoreCase(s, substr string) bool {
	if len(substr) == 0 {
		return true
	}
	return findSubstring(s, substr)
}

func findSubstring(s, substr string) bool {
	sLower := make([]rune, 0, len(s))
	substrLower := make([]rune, 0, len(substr))

	for _, r := range s {
		if r >= 'A' && r <= 'Z' {
			sLower = append(sLower, r+32)
		} else {
			sLower = append(sLower, r)
		}
	}

	for _, r := range substr {
		if r >= 'A' && r <= 'Z' {
			substrLower = append(substrLower, r+32)
		} else {
			substrLower = append(substrLower, r)
		}
	}

	sStr := string(sLower)
	substrStr := string(substrLower)

	return len(sStr) >= len(substrStr) &&
		findInString(sStr, substrStr)
}

func findInString(s, substr string) bool {
	if len(substr) == 0 {
		return true
	}
	if len(s) < len(substr) {
		return false
	}

	for i := 0; i <= len(s)-len(substr); i++ {
		match := true
		for j := 0; j < len(substr); j++ {
			if s[i+j] != substr[j] {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}
