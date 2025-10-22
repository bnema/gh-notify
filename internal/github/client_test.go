package github

import (
	"fmt"
	"testing"
	"time"

	"github.com/bnema/gh-notify/internal/github/mocks"
	"go.uber.org/mock/gomock"
)

// TestFetchRecentStars_Success tests the basic happy path for fetching stars
func TestFetchRecentStars_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Create mock clients
	mockGraphQL := mocks.NewMockGraphQLClient(ctrl)
	mockREST := mocks.NewMockRESTClient(ctrl)

	// Create client with mocks
	client := NewTestClient(mockREST, mockGraphQL)

	// Test data
	since := time.Now().Add(-1 * time.Hour)

	// First call: fetch repositories
	mockGraphQL.EXPECT().
		Do(gomock.Any(), gomock.Nil(), gomock.Any()).
		DoAndReturn(func(query string, variables map[string]interface{}, response interface{}) error {
			// Populate the response with our mock data
			respPtr := response.(*ReposResponse)
			respPtr.Viewer.Repositories.Nodes = []struct {
				NameWithOwner string `json:"nameWithOwner"`
			}{
				{NameWithOwner: "testuser/testrepo"},
			}
			return nil
		}).
		Times(1)

	// Second call: fetch stars for the repository
	mockGraphQL.EXPECT().
		Do(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(query string, variables map[string]interface{}, response interface{}) error {
			// Populate with a single star event
			respPtr := response.(*StarsResponse)

			starTime := time.Now().Add(-30 * time.Minute)
			respPtr.Repository.Stargazers.Edges = []struct {
				StarredAt time.Time `json:"starredAt"`
				Cursor    string    `json:"cursor"`
				Node      struct {
					Login string `json:"login"`
				} `json:"node"`
			}{
				{
					StarredAt: starTime,
					Cursor:    "cursor1",
					Node: struct {
						Login string `json:"login"`
					}{Login: "stargazer1"},
				},
			}
			respPtr.Repository.Stargazers.PageInfo.HasNextPage = false
			return nil
		}).
		Times(1)

	// Call the method
	stars, err := client.FetchRecentStars(since)

	// Assertions
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(stars) != 1 {
		t.Fatalf("Expected 1 star event, got %d", len(stars))
	}

	if stars[0].Repository != "testuser/testrepo" {
		t.Errorf("Expected repository 'testuser/testrepo', got '%s'", stars[0].Repository)
	}

	if stars[0].StarredBy != "stargazer1" {
		t.Errorf("Expected starred by 'stargazer1', got '%s'", stars[0].StarredBy)
	}

	if stars[0].ID != "cursor1" {
		t.Errorf("Expected cursor ID 'cursor1', got '%s'", stars[0].ID)
	}

	t.Logf("✓ Basic mocking test passed!")
}

// TestFetchRecentStars_EmptyRepos tests behavior when user has no repositories
func TestFetchRecentStars_EmptyRepos(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockGraphQL := mocks.NewMockGraphQLClient(ctrl)
	mockREST := mocks.NewMockRESTClient(ctrl)
	client := NewTestClient(mockREST, mockGraphQL)

	since := time.Now().Add(-1 * time.Hour)

	// Mock empty repository list
	mockGraphQL.EXPECT().
		Do(gomock.Any(), gomock.Nil(), gomock.Any()).
		DoAndReturn(func(query string, variables map[string]interface{}, response interface{}) error {
			respPtr := response.(*ReposResponse)
			respPtr.Viewer.Repositories.Nodes = []struct {
				NameWithOwner string `json:"nameWithOwner"`
			}{} // Empty
			return nil
		}).
		Times(1)

	stars, err := client.FetchRecentStars(since)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(stars) != 0 {
		t.Errorf("Expected 0 stars for empty repos, got %d", len(stars))
	}

	t.Logf("✓ Empty repos test passed!")
}

// TestFetchRecentStars_GraphQLError tests error handling from GraphQL
func TestFetchRecentStars_GraphQLError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockGraphQL := mocks.NewMockGraphQLClient(ctrl)
	mockREST := mocks.NewMockRESTClient(ctrl)
	client := NewTestClient(mockREST, mockGraphQL)

	since := time.Now().Add(-1 * time.Hour)

	// Mock error from GraphQL
	mockGraphQL.EXPECT().
		Do(gomock.Any(), gomock.Nil(), gomock.Any()).
		DoAndReturn(func(query string, variables map[string]interface{}, response interface{}) error {
			return fmt.Errorf("GraphQL API error")
		}).
		Times(1)

	stars, err := client.FetchRecentStars(since)

	if err == nil {
		t.Fatal("Expected error from GraphQL, got nil")
	}

	if stars != nil {
		t.Errorf("Expected nil stars on error, got %d stars", len(stars))
	}

	t.Logf("✓ GraphQL error handling test passed!")
}

// TestFetchStarsForRepo_Pagination tests multi-page star fetching
func TestFetchStarsForRepo_Pagination(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockGraphQL := mocks.NewMockGraphQLClient(ctrl)
	mockREST := mocks.NewMockRESTClient(ctrl)
	client := NewTestClient(mockREST, mockGraphQL)

	since := time.Now().Add(-2 * time.Hour)

	// First page with hasNextPage=true
	mockGraphQL.EXPECT().
		Do(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(query string, variables map[string]interface{}, response interface{}) error {
			respPtr := response.(*StarsResponse)
			respPtr.Repository.Stargazers.Edges = []struct {
				StarredAt time.Time `json:"starredAt"`
				Cursor    string    `json:"cursor"`
				Node      struct {
					Login string `json:"login"`
				} `json:"node"`
			}{
				{
					StarredAt: time.Now().Add(-30 * time.Minute),
					Cursor:    "cursor1",
					Node:      struct{ Login string `json:"login"` }{Login: "user1"},
				},
			}
			respPtr.Repository.Stargazers.PageInfo.HasNextPage = true
			respPtr.Repository.Stargazers.PageInfo.EndCursor = "cursor1"
			return nil
		}).
		Times(1)

	// Second page with hasNextPage=false
	mockGraphQL.EXPECT().
		Do(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(query string, variables map[string]interface{}, response interface{}) error {
			respPtr := response.(*StarsResponse)
			respPtr.Repository.Stargazers.Edges = []struct {
				StarredAt time.Time `json:"starredAt"`
				Cursor    string    `json:"cursor"`
				Node      struct {
					Login string `json:"login"`
				} `json:"node"`
			}{
				{
					StarredAt: time.Now().Add(-1 * time.Hour),
					Cursor:    "cursor2",
					Node:      struct{ Login string `json:"login"` }{Login: "user2"},
				},
			}
			respPtr.Repository.Stargazers.PageInfo.HasNextPage = false
			return nil
		}).
		Times(1)

	stars, err := client.fetchStarsForRepo("testuser/testrepo", since)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(stars) != 2 {
		t.Fatalf("Expected 2 stars from pagination, got %d", len(stars))
	}

	t.Logf("✓ Pagination test passed!")
}

// TestFetchStarsForRepo_TimeFiltering tests filtering by "since" time
func TestFetchStarsForRepo_TimeFiltering(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockGraphQL := mocks.NewMockGraphQLClient(ctrl)
	mockREST := mocks.NewMockRESTClient(ctrl)
	client := NewTestClient(mockREST, mockGraphQL)

	since := time.Now().Add(-1 * time.Hour)

	// Return stars, some old, some new
	mockGraphQL.EXPECT().
		Do(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(query string, variables map[string]interface{}, response interface{}) error {
			respPtr := response.(*StarsResponse)
			respPtr.Repository.Stargazers.Edges = []struct {
				StarredAt time.Time `json:"starredAt"`
				Cursor    string    `json:"cursor"`
				Node      struct {
					Login string `json:"login"`
				} `json:"node"`
			}{
				{
					StarredAt: time.Now().Add(-30 * time.Minute), // New (within 1 hour)
					Cursor:    "cursor1",
					Node:      struct{ Login string `json:"login"` }{Login: "user1"},
				},
				{
					StarredAt: time.Now().Add(-2 * time.Hour), // Old (beyond 1 hour)
					Cursor:    "cursor2",
					Node:      struct{ Login string `json:"login"` }{Login: "user2"},
				},
			}
			respPtr.Repository.Stargazers.PageInfo.HasNextPage = false
			return nil
		}).
		Times(1)

	stars, err := client.fetchStarsForRepo("testuser/testrepo", since)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Should only get the star within the time range
	if len(stars) != 1 {
		t.Fatalf("Expected 1 star after time filtering, got %d", len(stars))
	}

	if stars[0].StarredBy != "user1" {
		t.Errorf("Expected user1, got %s", stars[0].StarredBy)
	}

	t.Logf("✓ Time filtering test passed!")
}

// TestFetchStarsForRepo_EarlyTermination tests stopping at old stars
func TestFetchStarsForRepo_EarlyTermination(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockGraphQL := mocks.NewMockGraphQLClient(ctrl)
	mockREST := mocks.NewMockRESTClient(ctrl)
	client := NewTestClient(mockREST, mockGraphQL)

	since := time.Now().Add(-1 * time.Hour)

	// Return stars with an old one - should stop pagination
	mockGraphQL.EXPECT().
		Do(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(query string, variables map[string]interface{}, response interface{}) error {
			respPtr := response.(*StarsResponse)
			respPtr.Repository.Stargazers.Edges = []struct {
				StarredAt time.Time `json:"starredAt"`
				Cursor    string    `json:"cursor"`
				Node      struct {
					Login string `json:"login"`
				} `json:"node"`
			}{
				{
					StarredAt: time.Now().Add(-2 * time.Hour), // Old star
					Cursor:    "cursor1",
					Node:      struct{ Login string `json:"login"` }{Login: "user1"},
				},
			}
			respPtr.Repository.Stargazers.PageInfo.HasNextPage = true // Has more pages
			respPtr.Repository.Stargazers.PageInfo.EndCursor = "cursor1"
			return nil
		}).
		Times(1) // Should only call once, not fetch next page

	stars, err := client.fetchStarsForRepo("testuser/testrepo", since)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Should get no stars and stop early
	if len(stars) != 0 {
		t.Errorf("Expected 0 stars after early termination, got %d", len(stars))
	}

	t.Logf("✓ Early termination test passed!")
}

// TestHelpers_OwnerAndName tests helper functions for parsing repo names
func TestHelpers_OwnerAndName(t *testing.T) {
	testCases := []struct {
		input         string
		expectedOwner string
		expectedName  string
	}{
		{"testuser/testrepo", "testuser", "testrepo"},
		{"github/hub", "github", "hub"},
		{"owner/repo-name", "owner", "repo-name"},
		{"single", "single", ""}, // Edge case: no slash
		{"", "", ""},             // Edge case: empty
	}

	for _, tc := range testCases {
		t.Run(tc.input, func(t *testing.T) {
			owner := getOwner(tc.input)
			name := getName(tc.input)

			if owner != tc.expectedOwner {
				t.Errorf("getOwner(%q) = %q, want %q", tc.input, owner, tc.expectedOwner)
			}

			if name != tc.expectedName {
				t.Errorf("getName(%q) = %q, want %q", tc.input, name, tc.expectedName)
			}
		})
	}

	t.Logf("✓ Helper functions test passed!")
}
