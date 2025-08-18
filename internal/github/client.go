package github

import (
	"fmt"
	"time"

	"github.com/bnema/gh-notify/internal/cache"
	"github.com/cli/go-gh/v2/pkg/api"
)

type Client struct {
	restClient *api.RESTClient
}

type Notification struct {
	ID         string
	Repository string
	Title      string
	Reason     string
	Type       string
	UpdatedAt  time.Time
	Unread     bool
	URL        string
}

func NewClient() (*Client, error) {
	restClient, err := api.DefaultRESTClient()
	if err != nil {
		return nil, fmt.Errorf("failed to create GitHub REST client: %w", err)
	}

	return &Client{
		restClient: restClient,
	}, nil
}

func (c *Client) FetchNotifications(all bool) ([]cache.CacheEntry, error) {
	var response []map[string]interface{}
	var err error

	if all {
		err = c.restClient.Get("notifications?all=true", &response)
	} else {
		err = c.restClient.Get("notifications", &response)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to fetch notifications: %w", err)
	}

	return c.parseNotifications(response), nil
}

func (c *Client) parseNotifications(response []map[string]interface{}) []cache.CacheEntry {
	var entries []cache.CacheEntry

	now := time.Now()

	for _, notification := range response {
		id, ok := notification["id"].(string)
		if !ok || id == "" {
			continue
		}

		var repository, title, reason string
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
		}

		if reasonStr, ok := notification["reason"].(string); ok {
			reason = reasonStr
		}

		if updatedAtStr, ok := notification["updated_at"].(string); ok {
			updatedAt = parseTime(updatedAtStr)
		}

		entry := cache.CacheEntry{
			ID:         id,
			Repository: repository,
			Title:      title,
			Reason:     reason,
			Timestamp:  now,
			UpdatedAt:  updatedAt,
		}

		entries = append(entries, entry)
	}

	return entries
}

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

func (c *Client) TestAuth() error {
	var response map[string]interface{}
	err := c.restClient.Get("user", &response)
	if err != nil {
		return fmt.Errorf("failed to authenticate with GitHub: %w", err)
	}

	if _, ok := response["login"]; !ok {
		return fmt.Errorf("invalid response from GitHub API")
	}

	return nil
}