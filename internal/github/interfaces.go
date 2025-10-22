package github

import (
	"time"

	"github.com/bnema/gh-notify/internal/cache"
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

// apiGraphQLClient wraps api.GraphQLClient to implement GraphQLClient interface
type apiGraphQLClient struct {
	client *api.GraphQLClient
}

func (c *apiGraphQLClient) Do(query string, variables map[string]interface{}, response interface{}) error {
	return c.client.Do(query, variables, response)
}
