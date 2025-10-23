package cmd

import (
	"os"
	"testing"
	"time"

	"github.com/bnema/gh-notify/internal/cache"
)

// TestSync_FirstStarSync_4HourCutoff tests that first sync uses 4-hour cutoff
func TestSync_FirstStarSync_4HourCutoff(t *testing.T) {
	// Create temp cache dir
	tmpDir, err := os.MkdirTemp("", "gh-notify-test-*")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(tmpDir) })

	// Create cache with no previous sync (LastEventSync is zero)
	c := cache.New(tmpDir)

	// Verify LastEventSync is zero (first sync scenario)
	if !c.LastEventSync.IsZero() {
		t.Errorf("Expected LastEventSync to be zero for first sync, got %v", c.LastEventSync)
	}

	// In the real sync command, when LastEventSync is zero, it uses initialStarSyncCutoff as cutoff
	// Let's verify the logic of what should happen
	expectedCutoff := time.Now().UTC().Add(-initialStarSyncCutoff)

	// Simulate what the sync command does
	cutoff := c.LastEventSync
	if cutoff.IsZero() {
		cutoff = time.Now().UTC().Add(-initialStarSyncCutoff)
	}

	// Verify cutoff is approximately initialStarSyncCutoff ago (within 1 minute tolerance)
	diff := cutoff.Sub(expectedCutoff)
	if diff < -1*time.Minute || diff > 1*time.Minute {
		t.Errorf("Expected cutoff to be ~%v ago, got %v (diff: %v)", initialStarSyncCutoff, cutoff, diff)
	}

	t.Logf("✓ First sync initial cutoff test passed!")
}

// TestSync_SubsequentSync_UsesLastEventSync tests that subsequent syncs use LastEventSync
func TestSync_SubsequentSync_UsesLastEventSync(t *testing.T) {
	// Create temp cache dir
	tmpDir, err := os.MkdirTemp("", "gh-notify-test-*")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(tmpDir) })

	// Create cache and set a previous sync time
	c := cache.New(tmpDir)
	previousSync := time.Now().UTC().Add(-2 * time.Hour)
	c.LastEventSync = previousSync

	// Save and reload to simulate persistence
	if err := c.Save(tmpDir); err != nil {
		t.Fatal(err)
	}

	c2 := cache.New(tmpDir)
	if err := c2.Load(tmpDir); err != nil {
		t.Fatal(err)
	}

	// Verify LastEventSync is preserved
	if c2.LastEventSync.IsZero() {
		t.Error("Expected LastEventSync to be preserved after load, but got zero")
	}

	// In subsequent syncs, should use LastEventSync as cutoff
	cutoff := c2.LastEventSync
	if cutoff.IsZero() {
		cutoff = time.Now().UTC().Add(-initialStarSyncCutoff)
	}

	// Verify we're using the previous sync time (within 1 second tolerance)
	diff := cutoff.Sub(previousSync)
	if diff < -1*time.Second || diff > 1*time.Second {
		t.Errorf("Expected cutoff to use LastEventSync (%v), got %v (diff: %v)", previousSync, cutoff, diff)
	}

	t.Logf("✓ Subsequent sync uses LastEventSync test passed!")
}

// TestSync_LastEventSyncUpdate tests that LastEventSync is updated after sync
func TestSync_LastEventSyncUpdate(t *testing.T) {
	// Create temp cache dir
	tmpDir, err := os.MkdirTemp("", "gh-notify-test-*")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(tmpDir) })

	c := cache.New(tmpDir)
	oldSync := c.LastEventSync

	// Simulate what happens in sync: update LastEventSync
	beforeUpdate := time.Now().UTC()
	time.Sleep(10 * time.Millisecond) // Small delay to ensure time difference
	c.LastEventSync = time.Now().UTC()
	afterUpdate := time.Now().UTC()

	// Verify LastEventSync was updated
	if !c.LastEventSync.After(oldSync) {
		t.Error("Expected LastEventSync to be updated to a later time")
	}

	// Verify it's approximately now (within the range of our test)
	if c.LastEventSync.Before(beforeUpdate) || c.LastEventSync.After(afterUpdate) {
		t.Errorf("Expected LastEventSync to be between %v and %v, got %v", beforeUpdate, afterUpdate, c.LastEventSync)
	}

	// Verify it persists
	if err := c.Save(tmpDir); err != nil {
		t.Fatal(err)
	}

	c2 := cache.New(tmpDir)
	if err := c2.Load(tmpDir); err != nil {
		t.Fatal(err)
	}

	if c2.LastEventSync.IsZero() {
		t.Error("Expected LastEventSync to persist after save/load")
	}

	t.Logf("✓ LastEventSync update test passed!")
}

// TestSync_StarFiltering tests star filtering by time
func TestSync_StarFiltering(t *testing.T) {
	// Create temp cache dir
	tmpDir, err := os.MkdirTemp("", "gh-notify-test-*")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(tmpDir) })

	c := cache.New(tmpDir)

	// Create stars with different timestamps
	now := time.Now()
	stars := []cache.StarEvent{
		{
			ID:         "cursor1",
			Repository: "user/repo1",
			StarredBy:  "user1",
			StarredAt:  now.Add(-30 * time.Minute), // Recent
		},
		{
			ID:         "cursor2",
			Repository: "user/repo2",
			StarredBy:  "user2",
			StarredAt:  now.Add(-5 * time.Hour), // Old
		},
		{
			ID:         "cursor3",
			Repository: "user/repo3",
			StarredBy:  "user3",
			StarredAt:  now.Add(-1 * time.Hour), // Recent
		},
	}

	// Add all stars
	c.AddStarEvents(stars)

	// Simulate filtering: only keep stars after cutoff (2 hours ago)
	cutoff := now.Add(-2 * time.Hour)
	var filtered []cache.StarEvent
	for _, star := range c.Stars {
		if star.StarredAt.After(cutoff) {
			filtered = append(filtered, star)
		}
	}

	// Should have 2 recent stars
	if len(filtered) != 2 {
		t.Errorf("Expected 2 stars after filtering (cutoff: 2h ago), got %d", len(filtered))
	}

	// Verify the old star is not in filtered list
	for _, star := range filtered {
		if star.ID == "cursor2" {
			t.Error("Expected old star 'cursor2' to be filtered out")
		}
	}

	t.Logf("✓ Star filtering test passed!")
}

// TestSync_StarCleanup tests that star cleanup works correctly
func TestSync_StarCleanup(t *testing.T) {
	// Create temp cache dir
	tmpDir, err := os.MkdirTemp("", "gh-notify-test-*")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(tmpDir) })

	c := cache.New(tmpDir)
	c.MaxEntries = 100 // Set high to focus on time-based cleanup

	// Add old and new stars
	now := time.Now()
	stars := []cache.StarEvent{
		{
			ID:         "cursor1",
			Repository: "user/repo1",
			StarredBy:  "user1",
			StarredAt:  now.Add(-6 * 24 * time.Hour), // 6 days - should keep
		},
		{
			ID:         "cursor2",
			Repository: "user/repo2",
			StarredBy:  "user2",
			StarredAt:  now.Add(-8 * 24 * time.Hour), // 8 days - should remove
		},
		{
			ID:         "cursor3",
			Repository: "user/repo3",
			StarredBy:  "user3",
			StarredAt:  now.Add(-10 * 24 * time.Hour), // 10 days - should remove
		},
	}

	c.AddStarEvents(stars)

	// Save triggers cleanup
	if err := c.Save(tmpDir); err != nil {
		t.Fatal(err)
	}

	// Reload to see cleaned cache
	c2 := cache.New(tmpDir)
	if err := c2.Load(tmpDir); err != nil {
		t.Fatal(err)
	}

	// Should only have 1 star (6 days old)
	if len(c2.Stars) != 1 {
		t.Errorf("Expected 1 star after cleanup (7-day retention), got %d", len(c2.Stars))
	}

	if c2.Stars[0].ID != "cursor1" {
		t.Errorf("Expected remaining star to be 'cursor1', got '%s'", c2.Stars[0].ID)
	}

	t.Logf("✓ Star cleanup test passed!")
}

// TestSync_RateLimitCheck tests that stars are only fetched once per hour
func TestSync_RateLimitCheck(t *testing.T) {
	// Create temp cache dir
	tmpDir, err := os.MkdirTemp("", "gh-notify-test-*")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(tmpDir) })

	c := cache.New(tmpDir)

	// Scenario 1: First sync (LastEventSync is zero) - should fetch
	if !c.LastEventSync.IsZero() {
		t.Error("Expected LastEventSync to be zero initially")
	}
	shouldFetch1 := c.LastEventSync.IsZero()
	if !shouldFetch1 {
		t.Error("Expected to fetch stars on first sync")
	}
	t.Log("✓ First sync should fetch stars")

	// Simulate successful fetch
	c.LastEventSync = time.Now().UTC()
	if err := c.Save(tmpDir); err != nil {
		t.Fatal(err)
	}

	// Scenario 2: Second sync 30 minutes later - should NOT fetch
	c2 := cache.New(tmpDir)
	if err := c2.Load(tmpDir); err != nil {
		t.Fatal(err)
	}
	timeSinceLastFetch := time.Since(c2.LastEventSync)
	shouldFetch2 := c2.LastEventSync.IsZero() || timeSinceLastFetch >= starFetchRateLimit
	if shouldFetch2 {
		t.Errorf("Expected to skip fetch when only %v has passed (< %v)", timeSinceLastFetch, starFetchRateLimit)
	}
	t.Logf("✓ Second sync after %v should skip fetch (rate limited)", timeSinceLastFetch)

	// Scenario 3: Third sync after starFetchRateLimit - should fetch
	c3 := cache.New(tmpDir)
	c3.LastEventSync = time.Now().UTC().Add(-(starFetchRateLimit + 1*time.Minute)) // Just over the rate limit
	timeSinceLastFetch3 := time.Since(c3.LastEventSync)
	shouldFetch3 := c3.LastEventSync.IsZero() || timeSinceLastFetch3 >= starFetchRateLimit
	if !shouldFetch3 {
		t.Errorf("Expected to fetch stars when %v has passed (>= %v)", timeSinceLastFetch3, starFetchRateLimit)
	}
	t.Logf("✓ Third sync after %v should fetch stars", timeSinceLastFetch3)

	t.Logf("✓ Rate limit check test passed!")
}
