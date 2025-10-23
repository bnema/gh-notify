package notifier

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/bnema/gh-notify/internal/cache"
	"github.com/bnema/gh-notify/internal/nerdfonts"
)

type Notifier struct {
	enabled bool
}

func New(enabled bool) *Notifier {
	return &Notifier{
		enabled: enabled,
	}
}

func (n *Notifier) SendNotification(entry cache.CacheEntry) error {
	if !n.enabled {
		return nil
	}

	title := n.formatTitle(entry)
	message := n.formatMessage(entry)
	urgency := n.getUrgency(entry.Reason)

	return n.sendNotifyNotification(title, message, urgency)
}

func (n *Notifier) SendBulkNotification(entries []cache.CacheEntry) error {
	if !n.enabled || len(entries) == 0 {
		return nil
	}

	if len(entries) == 1 {
		return n.SendNotification(entries[0])
	}

	// For multiple notifications, send a summary
	title := fmt.Sprintf("GitHub - %d new notifications", len(entries))
	message := n.formatBulkMessage(entries)

	return n.sendNotifyNotification(title, message, "normal")
}

// SendStarNotifications sends notifications for new star events
func (n *Notifier) SendStarNotifications(starEvents []cache.StarEvent) error {
	if !n.enabled || len(starEvents) == 0 {
		return nil
	}

	if len(starEvents) == 1 {
		return n.sendStarNotification(starEvents[0])
	}

	// For multiple star events, send a summary
	title := fmt.Sprintf("%s %d new stars!", nerdfonts.StarredRepo, len(starEvents))
	message := n.formatStarBulkMessage(starEvents)

	return n.sendNotifyNotification(title, message, "normal")
}

// sendStarNotification sends a single star event notification
func (n *Notifier) sendStarNotification(star cache.StarEvent) error {
	title := fmt.Sprintf("%s New Star!", nerdfonts.StarredRepo)
	message := fmt.Sprintf("%s starred your repository: %s", star.StarredBy, star.Repository)

	return n.sendNotifyNotification(title, message, "normal")
}

// formatStarBulkMessage formats multiple star events into a summary message
func (n *Notifier) formatStarBulkMessage(starEvents []cache.StarEvent) string {
	var lines []string

	// Group by repository
	repoStars := make(map[string][]string)
	for _, star := range starEvents {
		repoStars[star.Repository] = append(repoStars[star.Repository], star.StarredBy)
	}

	// Show up to 5 repositories
	count := 0
	for repo, users := range repoStars {
		if count >= 5 {
			remaining := len(repoStars) - count
			lines = append(lines, fmt.Sprintf("... and %d more repositories", remaining))
			break
		}

		if len(users) == 1 {
			lines = append(lines, fmt.Sprintf("• %s starred %s", users[0], repo))
		} else if len(users) <= 5 {
			userList := strings.Join(users, ", ")
			lines = append(lines, fmt.Sprintf("• %s starred %s", userList, repo))
		} else {
			// Cap display at 3 users + "and X others" to prevent spam
			first3 := strings.Join(users[:3], ", ")
			remaining := len(users) - 3
			lines = append(lines, fmt.Sprintf("• %s and %d others starred %s", first3, remaining, repo))
		}
		count++
	}

	return strings.Join(lines, "\n")
}

func (n *Notifier) formatTitle(entry cache.CacheEntry) string {
	return fmt.Sprintf("GitHub - %s", entry.Repository)
}

func (n *Notifier) formatMessage(entry cache.CacheEntry) string {
	reasonText := n.formatReason(entry.Reason)

	// Include repository name in the message body as requested
	var message string
	if entry.Type != "" {
		message = fmt.Sprintf("%s [%s] in %s: %s", reasonText, entry.Type, entry.Repository, entry.Title)
	} else {
		message = fmt.Sprintf("%s in %s: %s", reasonText, entry.Repository, entry.Title)
	}

	return message
}

func (n *Notifier) formatBulkMessage(entries []cache.CacheEntry) string {
	var lines []string

	// Group by repository
	repoCount := make(map[string]int)
	for _, entry := range entries {
		repoCount[entry.Repository]++
	}

	// Show up to 5 repositories
	count := 0
	for repo, num := range repoCount {
		if count >= 5 {
			remaining := len(repoCount) - count
			lines = append(lines, fmt.Sprintf("... and %d more repositories", remaining))
			break
		}

		if num == 1 {
			lines = append(lines, fmt.Sprintf("• %s", repo))
		} else {
			lines = append(lines, fmt.Sprintf("• %s (%d)", repo, num))
		}
		count++
	}

	return strings.Join(lines, "\n")
}

func (n *Notifier) formatReason(reason string) string {
	switch reason {
	case "assign":
		return "Assigned"
	case "author":
		return "Author update"
	case "comment":
		return "New comment"
	case "invitation":
		return "Invitation"
	case "manual":
		return "Manual subscription"
	case "mention":
		return "Mentioned"
	case "review_requested":
		return "Review requested"
	case "security_alert":
		return "Security alert"
	case "state_change":
		return "State changed"
	case "subscribed":
		return "Subscribed"
	case "team_mention":
		return "Team mentioned"
	default:
		return "Notification"
	}
}

func (n *Notifier) getUrgency(reason string) string {
	switch reason {
	case "security_alert":
		return "critical"
	case "assign", "review_requested", "mention", "team_mention":
		return "normal"
	default:
		return "normal"
	}
}

// sendNotifyNotification sends a notification using notify-send with clickable default action
func (n *Notifier) sendNotifyNotification(title, message, urgency string) error {
	args := []string{
		"--app-name=GitHub Notify",
		"--urgency=" + urgency,
		"--action", "default=Open GitHub Notifications",
		"--wait",
		title,
		message,
	}

	cmd := exec.Command("notify-send", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("notify-send failed: %w, output: %s", err, string(output))
	}

	// Handle the action response (this runs in background)
	go n.handleNotificationAction(strings.TrimSpace(string(output)))

	return nil
}

// handleNotificationAction processes the action response from notify-send
func (n *Notifier) handleNotificationAction(response string) {
	if response == "default" {
		// Open GitHub notifications page in default browser
		cmd := exec.Command("xdg-open", "https://github.com/notifications")
		if err := cmd.Run(); err != nil {
			// Log error but don't fail - browser opening is not critical
			fmt.Printf("Warning: failed to open browser: %v\n", err)
		}
	}
}

func (n *Notifier) IsEnabled() bool {
	return n.enabled
}

func (n *Notifier) SetEnabled(enabled bool) {
	n.enabled = enabled
}
