package github

import (
	"fmt"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"github.com/bnema/gh-notify/internal/cache"
	"github.com/bnema/gh-notify/internal/logger"
)

// ReposResponse represents the GraphQL response for fetching repositories
type ReposResponse struct {
	Viewer struct {
		Repositories struct {
			Nodes []struct {
				NameWithOwner string `json:"nameWithOwner"`
			} `json:"nodes"`
		} `json:"repositories"`
	} `json:"viewer"`
}

// StarsResponse represents the GraphQL response for fetching stargazers
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

const (
	maxWorkers   = 6   // Limit concurrent API calls to avoid rate limiting
	maxPages     = 10  // Limit per repository to prevent API abuse
	starsPerPage = 100 // Number of stars to fetch per page
)

// FetchRecentStars fetches recent star events using GraphQL with pagination and concurrent processing.
// It queries all user-owned repositories and fetches stars that occurred after the 'since' timestamp.
// Uses a worker pool (6 workers) to fetch stars concurrently while respecting rate limits.
// Returns stars sorted by StarredAt time (newest first).
func (c *Client) FetchRecentStars(since time.Time) ([]cache.StarEvent, error) {
	startTotal := time.Now()
	var allStarEvents []cache.StarEvent

	// First, get all repositories
	repos, err := c.fetchUserRepositories()
	if err != nil {
		return nil, err
	}

	logger.Debug().
		Int("repo_count", len(repos)).
		Msg("Fetched repository list")

	// Fetch stars concurrently with worker pool
	allStarEvents = c.fetchStarsWithWorkerPool(repos, since)

	logger.Info().
		Int("total_stars", len(allStarEvents)).
		Int("workers", maxWorkers).
		Dur("total_duration", time.Since(startTotal)).
		Msg("Completed fetching all star events")

	// Sort by starred time (newest first)
	sort.Slice(allStarEvents, func(i, j int) bool {
		return allStarEvents[i].StarredAt.After(allStarEvents[j].StarredAt)
	})

	return allStarEvents, nil
}

// fetchUserRepositories fetches all repositories owned by the authenticated user
func (c *Client) fetchUserRepositories() ([]string, error) {
	startRepos := time.Now()
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

	var reposResp ReposResponse
	if err := c.graphqlClient.Do(reposQuery, nil, &reposResp); err != nil {
		return nil, fmt.Errorf("failed to fetch repositories: %w", err)
	}

	logger.Debug().
		Dur("duration", time.Since(startRepos)).
		Msg("Fetched repository list from GraphQL")

	// Extract repository names
	repos := make([]string, 0, len(reposResp.Viewer.Repositories.Nodes))
	for _, node := range reposResp.Viewer.Repositories.Nodes {
		repos = append(repos, node.NameWithOwner)
	}

	return repos, nil
}

// fetchStarsWithWorkerPool fetches stars for multiple repositories concurrently using a worker pool
func (c *Client) fetchStarsWithWorkerPool(repos []string, since time.Time) []cache.StarEvent {
	totalRepos := len(repos)

	// Create channels for work distribution
	repoChan := make(chan struct {
		name  string
		index int
	}, totalRepos)

	// Result collection with mutex for thread-safety
	var allStarEvents []cache.StarEvent
	var mu sync.Mutex
	var wg sync.WaitGroup
	var completed atomic.Int32

	// Start worker pool
	for i := 0; i < maxWorkers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for job := range repoChan {
				startRepo := time.Now()
				stars, err := c.fetchStarsForRepo(job.name, since)

				if err != nil {
					// Classify error type for better logging
					errorType := ClassifyGitHubError(err)
					logEvent := logger.Warn().
						Str("repo", job.name).
						Int("worker", workerID).
						Str("error_type", errorType).
						Err(err)

					switch errorType {
					case ErrorTypeRateLimit:
						logEvent.Msg("Rate limit exceeded despite retries - skipping repository")
					case ErrorTypePermission:
						logEvent.Msg("Permission denied - repository may be private or deleted")
					case ErrorTypeNotFound:
						logEvent.Msg("Repository not found - may have been deleted or renamed")
					case ErrorTypeTimeout:
						logEvent.Msg("Request timeout - network may be slow or unstable")
					default:
						logEvent.Msg("Failed to fetch stars for repository")
					}

					completed.Add(1)
					continue
				}

				// Thread-safe append
				mu.Lock()
				allStarEvents = append(allStarEvents, stars...)
				mu.Unlock()

				progress := completed.Add(1)
				logger.Debug().
					Str("repo", job.name).
					Int("stars", len(stars)).
					Int("progress", int(progress)).
					Int("total", totalRepos).
					Int("worker", workerID).
					Dur("duration", time.Since(startRepo)).
					Msg("Fetched stars for repository")
			}
		}(i)
	}

	// Send work to workers
	for i, repo := range repos {
		repoChan <- struct {
			name  string
			index int
		}{name: repo, index: i}
	}
	close(repoChan)

	// Wait for all workers to complete
	wg.Wait()

	return allStarEvents
}

// fetchStarsForRepo fetches paginated star events for a single repository
func (c *Client) fetchStarsForRepo(repoName string, since time.Time) ([]cache.StarEvent, error) {
	var allStars []cache.StarEvent
	var cursor *string

	// Define query once with proper variables (not string interpolation)
	query := `
		query($owner: String!, $name: String!, $first: Int!, $cursor: String) {
			repository(owner: $owner, name: $name) {
				stargazers(first: $first, after: $cursor, orderBy: {field: STARRED_AT, direction: DESC}) {
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
		}`

	for page := 0; page < maxPages; page++ {
		// Build variables map with proper typing
		variables := map[string]interface{}{
			"owner": GetOwner(repoName),
			"name":  GetName(repoName),
			"first": starsPerPage,
		}
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
				allStars = append(allStars, cache.StarEvent{
					ID:         edge.Cursor, // Use cursor as unique ID (guaranteed unique by GitHub)
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
