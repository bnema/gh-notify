package github

import (
	"fmt"
	"regexp"
	"sort"
	"time"

	"github.com/bnema/gh-notify/internal/cache"
	"github.com/cli/go-gh/v2/pkg/api"
)

type Client struct {
	restClient    *api.RESTClient
	graphqlClient *api.GraphQLClient
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

	graphqlClient, err := api.DefaultGraphQLClient()
	if err != nil {
		return nil, fmt.Errorf("failed to create GitHub GraphQL client: %w", err)
	}

	return &Client{
		restClient:    restClient,
		graphqlClient: graphqlClient,
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

// GetAuthenticatedUser returns the username of the authenticated user
func (c *Client) GetAuthenticatedUser() (string, error) {
	var response map[string]interface{}
	err := c.restClient.Get("user", &response)
	if err != nil {
		return "", fmt.Errorf("failed to get authenticated user: %w", err)
	}

	login, ok := response["login"].(string)
	if !ok {
		return "", fmt.Errorf("invalid response from GitHub API: missing login")
	}

	return login, nil
}

// FetchReceivedEvents fetches events received by the user (events on repositories they own)
func (c *Client) FetchReceivedEvents(username string) ([]EventEntry, error) {
	// This method is deprecated - use FetchRecentStars instead
	return []EventEntry{}, nil
}

// FetchRecentStars fetches recent star events using GraphQL
func (c *Client) FetchRecentStars(since time.Time) ([]StarEvent, error) {
	// Use GraphQL to get all repositories with recent stargazers
	query := `
	{
		viewer {
			repositories(first: 100, ownerAffiliations: OWNER) {
				nodes {
					nameWithOwner
					stargazers(last: 20, orderBy: {field: STARRED_AT, direction: DESC}) {
						edges {
							starredAt
							node {
								login
							}
						}
					}
				}
				pageInfo {
					hasNextPage
					endCursor
				}
			}
		}
	}`

	type GraphQLResponse struct {
		Viewer struct {
			Repositories struct {
				Nodes []struct {
					NameWithOwner string `json:"nameWithOwner"`
					Stargazers    struct {
						Edges []struct {
							StarredAt time.Time `json:"starredAt"`
							Node      struct {
								Login string `json:"login"`
							} `json:"node"`
						} `json:"edges"`
					} `json:"stargazers"`
				} `json:"nodes"`
			} `json:"repositories"`
		} `json:"viewer"`
	}

	var response GraphQLResponse
	err := c.graphqlClient.Do(query, nil, &response)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch repositories: %w", err)
	}

	var starEvents []StarEvent
	for _, repo := range response.Viewer.Repositories.Nodes {
		for _, edge := range repo.Stargazers.Edges {
			// Only include stars that happened after the 'since' time
			if edge.StarredAt.After(since) {
				starEvents = append(starEvents, StarEvent{
					StarredBy:   edge.Node.Login,
					Repository:  repo.NameWithOwner,
					StarredAt:   edge.StarredAt,
				})
			}
		}
	}

	// Sort by starred time (newest first)
	sort.Slice(starEvents, func(i, j int) bool {
		return starEvents[i].StarredAt.After(starEvents[j].StarredAt)
	})

	return starEvents, nil
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