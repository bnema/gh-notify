package cache

import (
	"testing"
	"time"
)

// TestAddStarEvents_NewStars tests adding stars to an empty cache
func TestAddStarEvents_NewStars(t *testing.T) {
	c := New("")

	stars := []StarEvent{
		{
			ID:         "cursor1",
			Repository: "user/repo1",
			StarredBy:  "stargazer1",
			StarredAt:  time.Now(),
		},
		{
			ID:         "cursor2",
			Repository: "user/repo2",
			StarredBy:  "stargazer2",
			StarredAt:  time.Now().Add(-1 * time.Hour),
		},
	}

	newStars := c.AddStarEvents(stars)

	// All stars should be new
	if len(newStars) != 2 {
		t.Errorf("Expected 2 new stars, got %d", len(newStars))
	}

	// Cache should contain all stars
	if len(c.Stars) != 2 {
		t.Errorf("Expected cache to have 2 stars, got %d", len(c.Stars))
	}

	t.Logf("✓ Add new stars test passed!")
}

// TestAddStarEvents_DuplicatePrevention tests duplicate star detection
func TestAddStarEvents_DuplicatePrevention(t *testing.T) {
	c := New("")

	// Add initial stars
	initialStars := []StarEvent{
		{
			ID:         "cursor1",
			Repository: "user/repo1",
			StarredBy:  "stargazer1",
			StarredAt:  time.Now(),
		},
	}
	c.AddStarEvents(initialStars)

	// Try to add duplicate and one new star
	stars := []StarEvent{
		{
			ID:         "cursor1", // Duplicate
			Repository: "user/repo1",
			StarredBy:  "stargazer1",
			StarredAt:  time.Now(),
		},
		{
			ID:         "cursor2", // New
			Repository: "user/repo2",
			StarredBy:  "stargazer2",
			StarredAt:  time.Now(),
		},
	}

	newStars := c.AddStarEvents(stars)

	// Only 1 star should be new (cursor2)
	if len(newStars) != 1 {
		t.Errorf("Expected 1 new star (excluding duplicate), got %d", len(newStars))
	}

	if newStars[0].ID != "cursor2" {
		t.Errorf("Expected new star to have ID 'cursor2', got '%s'", newStars[0].ID)
	}

	// Cache should have 2 total stars (no duplicates)
	if len(c.Stars) != 2 {
		t.Errorf("Expected cache to have 2 stars total, got %d", len(c.Stars))
	}

	t.Logf("✓ Duplicate prevention test passed!")
}

// TestAddStarEvents_ReturnsNewOnly tests that only new stars are returned
func TestAddStarEvents_ReturnsNewOnly(t *testing.T) {
	c := New("")

	// Add initial stars
	initialStars := []StarEvent{
		{
			ID:         "cursor1",
			Repository: "user/repo1",
			StarredBy:  "stargazer1",
			StarredAt:  time.Now(),
		},
		{
			ID:         "cursor2",
			Repository: "user/repo2",
			StarredBy:  "stargazer2",
			StarredAt:  time.Now(),
		},
	}
	c.AddStarEvents(initialStars)

	// Add all existing stars again
	newStars := c.AddStarEvents(initialStars)

	// Should return no new stars
	if len(newStars) != 0 {
		t.Errorf("Expected 0 new stars, got %d", len(newStars))
	}

	// Cache should still have original 2 stars
	if len(c.Stars) != 2 {
		t.Errorf("Expected cache to still have 2 stars, got %d", len(c.Stars))
	}

	t.Logf("✓ Returns new only test passed!")
}

// TestCleanup_7DayStarRetention tests that stars older than 7 days are removed
func TestCleanup_7DayStarRetention(t *testing.T) {
	c := New("")
	c.MaxEntries = 100 // Set high to test time-based cleanup only

	stars := []StarEvent{
		{
			ID:         "cursor1",
			Repository: "user/repo1",
			StarredBy:  "stargazer1",
			StarredAt:  time.Now().Add(-6 * 24 * time.Hour), // 6 days old - should keep
		},
		{
			ID:         "cursor2",
			Repository: "user/repo2",
			StarredBy:  "stargazer2",
			StarredAt:  time.Now().Add(-8 * 24 * time.Hour), // 8 days old - should remove
		},
		{
			ID:         "cursor3",
			Repository: "user/repo3",
			StarredBy:  "stargazer3",
			StarredAt:  time.Now().Add(-1 * time.Hour), // Recent - should keep
		},
	}
	c.AddStarEvents(stars)

	// Run cleanup
	c.cleanup()

	// Should only have 2 stars (6 days old and recent)
	if len(c.Stars) != 2 {
		t.Errorf("Expected 2 stars after 7-day cleanup, got %d", len(c.Stars))
	}

	// Verify the old star is gone
	for _, star := range c.Stars {
		if star.ID == "cursor2" {
			t.Errorf("Expected old star 'cursor2' to be removed, but it's still there")
		}
	}

	t.Logf("✓ 7-day retention test passed!")
}

// TestCleanup_MaxEntriesForStars tests that MaxEntries limit is respected
func TestCleanup_MaxEntriesForStars(t *testing.T) {
	c := New("")
	c.MaxEntries = 3 // Set low to test size limit

	// Add 5 stars
	stars := []StarEvent{
		{
			ID:         "cursor1",
			Repository: "user/repo1",
			StarredBy:  "stargazer1",
			StarredAt:  time.Now().Add(-5 * time.Hour),
		},
		{
			ID:         "cursor2",
			Repository: "user/repo2",
			StarredBy:  "stargazer2",
			StarredAt:  time.Now().Add(-4 * time.Hour),
		},
		{
			ID:         "cursor3",
			Repository: "user/repo3",
			StarredBy:  "stargazer3",
			StarredAt:  time.Now().Add(-3 * time.Hour),
		},
		{
			ID:         "cursor4",
			Repository: "user/repo4",
			StarredBy:  "stargazer4",
			StarredAt:  time.Now().Add(-2 * time.Hour),
		},
		{
			ID:         "cursor5",
			Repository: "user/repo5",
			StarredBy:  "stargazer5",
			StarredAt:  time.Now().Add(-1 * time.Hour), // Most recent
		},
	}
	c.AddStarEvents(stars)

	// Run cleanup
	c.cleanup()

	// Should only have 3 stars (MaxEntries)
	if len(c.Stars) != 3 {
		t.Errorf("Expected 3 stars after MaxEntries cleanup, got %d", len(c.Stars))
	}

	// Verify the newest 3 are kept (sorted by StarredAt desc)
	if c.Stars[0].ID != "cursor5" {
		t.Errorf("Expected most recent star to be 'cursor5', got '%s'", c.Stars[0].ID)
	}

	// Verify oldest stars are removed
	for _, star := range c.Stars {
		if star.ID == "cursor1" || star.ID == "cursor2" {
			t.Errorf("Expected old star '%s' to be removed, but it's still there", star.ID)
		}
	}

	t.Logf("✓ MaxEntries limit test passed!")
}
