package github

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/bnema/gh-notify/internal/cache"
	"github.com/bnema/gh-notify/internal/logger"
	"github.com/cli/go-gh/v2/pkg/api"
)

// GitHubClientInterface defines the main client interface for testing
type GitHubClientInterface interface {
	FetchNotifications() ([]cache.CacheEntry, error)
	FetchRecentStars(since time.Time) ([]cache.StarEvent, error)
	GetAuthenticatedUser() (string, error)
	TestAuth() error
}

// GraphQLClient wraps the external GraphQL client for mocking
type GraphQLClient interface {
	Do(query string, variables map[string]interface{}, response interface{}) error
}

// RESTClient wraps the external REST client for mocking
type RESTClient interface {
	Get(path string, response interface{}) error
}

// apiRESTClient wraps api.RESTClient to implement RESTClient interface
type apiRESTClient struct {
	client *api.RESTClient
}

func (c *apiRESTClient) Get(path string, response interface{}) error {
	return c.client.Get(path, response)
}

// apiGraphQLClient wraps api.GraphQLClient to implement GraphQLClient interface with retry logic
type apiGraphQLClient struct {
	client *api.GraphQLClient
}

const (
	maxRetries     = 3
	initialBackoff = 2 * time.Second
	queryTimeout   = 30 * time.Second
)

// Do executes a GraphQL query with exponential backoff retry for rate limit errors
func (c *apiGraphQLClient) Do(query string, variables map[string]interface{}, response interface{}) error {
	ctx, cancel := context.WithTimeout(context.Background(), queryTimeout)
	defer cancel()

	var lastErr error
	backoff := initialBackoff

	for attempt := 0; attempt <= maxRetries; attempt++ {
		// Check if context is cancelled (timeout)
		select {
		case <-ctx.Done():
			return fmt.Errorf("GraphQL query timeout after %v: %w", queryTimeout, ctx.Err())
		default:
		}

		// Attempt the query
		err := c.client.Do(query, variables, response)
		if err == nil {
			// Success!
			if attempt > 0 {
				logger.Info().
					Int("attempts", attempt+1).
					Msg("GraphQL query succeeded after retry")
			}
			return nil
		}

		lastErr = err

		// Check if this is a rate limit error
		if !isRateLimitError(err) {
			// Not a rate limit error, fail immediately
			return err
		}

		// Rate limit error - retry with backoff
		if attempt < maxRetries {
			logger.Warn().
				Err(err).
				Int("attempt", attempt+1).
				Dur("backoff", backoff).
				Msg("Rate limit hit, retrying with backoff")

			// Wait for backoff period (or until context timeout)
			select {
			case <-time.After(backoff):
				// Continue to next attempt
			case <-ctx.Done():
				return fmt.Errorf("timeout while waiting for retry: %w", ctx.Err())
			}

			// Exponential backoff: 2s, 4s, 8s
			backoff *= 2
		}
	}

	return fmt.Errorf("failed after %d retries due to rate limiting: %w", maxRetries, lastErr)
}

// isRateLimitError detects if an error is related to GitHub API rate limiting
func isRateLimitError(err error) bool {
	if err == nil {
		return false
	}

	errStr := strings.ToLower(err.Error())
	rateLimitIndicators := []string{
		"rate limit",
		"api rate limit exceeded",
		"you have exceeded",
		"abuse detection",
		"secondary rate limit",
	}

	for _, indicator := range rateLimitIndicators {
		if strings.Contains(errStr, indicator) {
			return true
		}
	}

	return false
}
