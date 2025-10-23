package github

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
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

// FetchRecentStars fetches recent star events using GraphQL with pagination
func (c *Client) FetchRecentStars(since time.Time) ([]StarEvent, error) {
	var allStarEvents []StarEvent

	// First, get all repositories
	reposQuery := `
	{
		viewer {
			repositories(first: 100, ownerAffiliations: OWNER) {
				nodes {
					nameWithOwner
				}
			}
		}
	}`

	type ReposResponse struct {
		Viewer struct {
			Repositories struct {
				Nodes []struct {
					NameWithOwner string `json:"nameWithOwner"`
				} `json:"nodes"`
			} `json:"repositories"`
		} `json:"viewer"`
	}

	var reposResp ReposResponse
	if err := c.graphqlClient.Do(reposQuery, nil, &reposResp); err != nil {
		return nil, fmt.Errorf("failed to fetch repositories: %w", err)
	}

	// For each repository, fetch stars with pagination
	for _, repo := range reposResp.Viewer.Repositories.Nodes {
		stars, err := c.fetchStarsForRepo(repo.NameWithOwner, since)
		if err != nil {
			// Log error but continue with other repos
			continue
		}
		allStarEvents = append(allStarEvents, stars...)
	}

	// Sort by starred time (newest first)
	sort.Slice(allStarEvents, func(i, j int) bool {
		return allStarEvents[i].StarredAt.After(allStarEvents[j].StarredAt)
	})

	return allStarEvents, nil
}

// fetchStarsForRepo fetches paginated star events for a single repository
func (c *Client) fetchStarsForRepo(repoName string, since time.Time) ([]StarEvent, error) {
	const maxPages = 10 // Limit to prevent API abuse
	const starsPerPage = 100

	var allStars []StarEvent
	var cursor *string

	for page := 0; page < maxPages; page++ {
		// Build query with optional cursor
		query := fmt.Sprintf(`
		query($cursor: String) {
			repository(owner: "%s", name: "%s") {
				stargazers(first: %d, after: $cursor, orderBy: {field: STARRED_AT, direction: DESC}) {
					edges {
						starredAt
						cursor
						node {
							login
						}
					}
					pageInfo {
						hasNextPage
						endCursor
					}
				}
			}
		}`, getOwner(repoName), getName(repoName), starsPerPage)

		type StarsResponse struct {
			Repository struct {
				Stargazers struct {
					Edges []struct {
						StarredAt time.Time `json:"starredAt"`
						Cursor    string    `json:"cursor"`
						Node      struct {
							Login string `json:"login"`
						} `json:"node"`
					} `json:"edges"`
					PageInfo struct {
						HasNextPage bool   `json:"hasNextPage"`
						EndCursor   string `json:"endCursor"`
					} `json:"pageInfo"`
				} `json:"stargazers"`
			} `json:"repository"`
		}

		variables := make(map[string]interface{})
		if cursor != nil {
			variables["cursor"] = *cursor
		}

		var response StarsResponse
		if err := c.graphqlClient.Do(query, variables, &response); err != nil {
			return allStars, fmt.Errorf("failed to fetch stars for %s: %w", repoName, err)
		}

		// Process stars from this page
		foundOldStar := false
		for _, edge := range response.Repository.Stargazers.Edges {
			if edge.StarredAt.After(since) {
				allStars = append(allStars, StarEvent{
					StarredBy:  edge.Node.Login,
					Repository: repoName,
					StarredAt:  edge.StarredAt,
				})
			} else {
				// Found a star older than our cutoff, no need to fetch more pages
				foundOldStar = true
				break
			}
		}

		// Stop if we found old stars or no more pages
		if foundOldStar || !response.Repository.Stargazers.PageInfo.HasNextPage {
			break
		}

		// Prepare for next page
		cursor = &response.Repository.Stargazers.PageInfo.EndCursor
	}

	return allStars, nil
}

// Helper functions to parse owner/name from "owner/repo" format
func getOwner(repoFullName string) string {
	parts := strings.Split(repoFullName, "/")
	if len(parts) >= 1 {
		return parts[0]
	}
	return ""
}

func getName(repoFullName string) string {
	parts := strings.Split(repoFullName, "/")
	if len(parts) >= 2 {
		return parts[1]
	}
	return ""
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