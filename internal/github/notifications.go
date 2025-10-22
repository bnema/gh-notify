package github

import (
	"fmt"
	"time"

	"github.com/bnema/gh-notify/internal/cache"
)

// FetchNotifications fetches unread GitHub notifications from the REST API.
// It returns a list of cache entries containing notification details.
// Only unread notifications are fetched (GitHub API default behavior).
func (c *Client) FetchNotifications() ([]cache.CacheEntry, error) {
	var response []map[string]interface{}

	// Always fetch only unread notifications (GitHub API default)
	err := c.restClient.Get("notifications", &response)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch notifications: %w", err)
	}

	return c.parseNotifications(response), nil
}

// parseNotifications converts raw GitHub API notification responses to cache entries
func (c *Client) parseNotifications(response []map[string]interface{}) []cache.CacheEntry {
	var entries []cache.CacheEntry

	now := time.Now().UTC()

	for _, notification := range response {
		id, ok := notification["id"].(string)
		if !ok || id == "" {
			continue
		}

		var repository, title, reason, notifType, apiURL string
		var updatedAt time.Time

		if repo, ok := notification["repository"].(map[string]interface{}); ok {
			if fullName, ok := repo["full_name"].(string); ok {
				repository = fullName
			}
		}

		if subject, ok := notification["subject"].(map[string]interface{}); ok {
			if subjectTitle, ok := subject["title"].(string); ok {
				title = subjectTitle
			}
			if subjectType, ok := subject["type"].(string); ok {
				notifType = subjectType
			}
			if subjectURL, ok := subject["url"].(string); ok {
				apiURL = subjectURL
			}
		}

		if reasonStr, ok := notification["reason"].(string); ok {
			reason = reasonStr
		}

		if updatedAtStr, ok := notification["updated_at"].(string); ok {
			updatedAt = parseTime(updatedAtStr)
		}

		// Convert API URL to web URL
		webURL := ConvertAPIURLToWeb(apiURL)

		entry := cache.CacheEntry{
			ID:         id,
			Repository: repository,
			Title:      title,
			Reason:     reason,
			Type:       notifType,
			URL:        apiURL,
			WebURL:     webURL,
			Timestamp:  now,
			UpdatedAt:  updatedAt,
		}

		entries = append(entries, entry)
	}

	return entries
}

// parseTime parses GitHub API time strings (RFC3339 format)
func parseTime(timeStr string) time.Time {
	if timeStr == "" {
		return time.Time{}
	}

	// GitHub API returns times in RFC3339 format
	parsedTime, err := time.Parse(time.RFC3339, timeStr)
	if err != nil {
		return time.Time{}
	}

	return parsedTime
}
