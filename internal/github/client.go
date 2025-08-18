package github

import (
	"fmt"
	"regexp"
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

func (c *Client) FetchNotifications() ([]cache.CacheEntry, error) {
	var response []map[string]interface{}
	
	// Always fetch only unread notifications (GitHub API default)
	err := c.restClient.Get("notifications", &response)
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
		webURL := convertAPIURLToWeb(apiURL)

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

// convertAPIURLToWeb converts GitHub API URLs to web URLs
// Example: https://api.github.com/repos/owner/repo/issues/123 -> https://github.com/owner/repo/issues/123
func convertAPIURLToWeb(apiURL string) string {
	if apiURL == "" {
		return ""
	}

	// Regular expression to match different GitHub API URL patterns
	patterns := []struct {
		regex       *regexp.Regexp
		replacement string
	}{
		// Issues: https://api.github.com/repos/owner/repo/issues/123
		{
			regex:       regexp.MustCompile(`^https://api\.github\.com/repos/([^/]+)/([^/]+)/issues/(\d+)$`),
			replacement: "https://github.com/$1/$2/issues/$3",
		},
		// Pull requests: https://api.github.com/repos/owner/repo/pulls/123
		{
			regex:       regexp.MustCompile(`^https://api\.github\.com/repos/([^/]+)/([^/]+)/pulls/(\d+)$`),
			replacement: "https://github.com/$1/$2/pull/$3",
		},
		// Releases: https://api.github.com/repos/owner/repo/releases/123
		{
			regex:       regexp.MustCompile(`^https://api\.github\.com/repos/([^/]+)/([^/]+)/releases/(\d+)$`),
			replacement: "https://github.com/$1/$2/releases/tag/$3", // Note: this might need release tag name
		},
		// Comments: https://api.github.com/repos/owner/repo/issues/comments/123
		{
			regex:       regexp.MustCompile(`^https://api\.github\.com/repos/([^/]+)/([^/]+)/issues/comments/(\d+)$`),
			replacement: "https://github.com/$1/$2/issues", // Will redirect to the issue
		},
	}

	for _, pattern := range patterns {
		if pattern.regex.MatchString(apiURL) {
			return pattern.regex.ReplaceAllString(apiURL, pattern.replacement)
		}
	}

	// Fallback: try to extract owner/repo and create a general repo URL
	repoRegex := regexp.MustCompile(`^https://api\.github\.com/repos/([^/]+)/([^/]+)/`)
	if matches := repoRegex.FindStringSubmatch(apiURL); len(matches) >= 3 {
		return fmt.Sprintf("https://github.com/%s/%s", matches[1], matches[2])
	}

	// If we can't convert, return the original URL
	return apiURL
}