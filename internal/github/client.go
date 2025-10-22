package github

import (
	"fmt"

	"github.com/cli/go-gh/v2/pkg/api"
)

// Client is the main GitHub API client that wraps REST and GraphQL clients
type Client struct {
	restClient    RESTClient
	graphqlClient GraphQLClient
}

// Ensure Client implements GitHubClientInterface
var _ GitHubClientInterface = (*Client)(nil)

// NewClient creates a new GitHub API client using default gh CLI authentication
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
		restClient:    &apiRESTClient{client: restClient},
		graphqlClient: &apiGraphQLClient{client: graphqlClient},
	}, nil
}

// NewTestClient creates a client with injected dependencies for testing
func NewTestClient(restClient RESTClient, graphqlClient GraphQLClient) *Client {
	return &Client{
		restClient:    restClient,
		graphqlClient: graphqlClient,
	}
}

// TestAuth verifies that the GitHub authentication is working
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
