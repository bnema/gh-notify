package cache

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"
)

type CacheEntry struct {
	ID         string    `json:"id"`
	Repository string    `json:"repository"`
	Title      string    `json:"title"`
	Reason     string    `json:"reason"`
	Type       string    `json:"type"`
	URL        string    `json:"url"`
	WebURL     string    `json:"web_url"`
	Timestamp  time.Time `json:"timestamp"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// StarEvent represents a star event for caching
type StarEvent struct {
	ID         string    `json:"id"`
	Repository string    `json:"repository"` // Full name: "owner/repo"
	StarredBy  string    `json:"starred_by"`
	StarredAt  time.Time `json:"starred_at"`
	Notified   bool      `json:"notified"`
}

type Cache struct {
	Version       string       `json:"version"`
	LastSync      time.Time    `json:"last_sync"`
	Notifications []CacheEntry `json:"notifications"`
	Stars         []StarEvent  `json:"stars"`
	LastEventSync time.Time    `json:"last_event_sync"` // Track last sync for rate limiting
	MaxEntries    int          `json:"max_entries"`
}

const (
	DefaultMaxEntries = 500
	MaxAge            = 30 * 24 * time.Hour // 30 days
	CacheVersion      = "1.0"
)

func New(cacheDir string) *Cache {
	return &Cache{
		Version:       CacheVersion,
		LastSync:      time.Time{},
		Notifications: []CacheEntry{},
		Stars:         []StarEvent{},
		LastEventSync: time.Time{},
		MaxEntries:    DefaultMaxEntries,
	}
}

func (c *Cache) getCacheFile(cacheDir string) string {
	return filepath.Join(cacheDir, "notifications.json")
}

func (c *Cache) ensureCacheDir(cacheDir string) error {
	return os.MkdirAll(cacheDir, 0755)
}

func (c *Cache) Load(cacheDir string) error {
	if err := c.ensureCacheDir(cacheDir); err != nil {
		return fmt.Errorf("failed to create cache directory: %w", err)
	}

	cacheFile := c.getCacheFile(cacheDir)

	data, err := os.ReadFile(cacheFile)
	if err != nil {
		if os.IsNotExist(err) {
			// Cache file doesn't exist, use empty cache
			return nil
		}
		return fmt.Errorf("failed to read cache file: %w", err)
	}

	if err := json.Unmarshal(data, c); err != nil {
		return fmt.Errorf("failed to unmarshal cache: %w", err)
	}

	// Ensure MaxEntries is set
	if c.MaxEntries == 0 {
		c.MaxEntries = DefaultMaxEntries
	}

	return nil
}

func (c *Cache) Save(cacheDir string) error {
	if err := c.ensureCacheDir(cacheDir); err != nil {
		return fmt.Errorf("failed to create cache directory: %w", err)
	}

	// Cleanup before saving
	c.cleanup()

	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal cache: %w", err)
	}

	cacheFile := c.getCacheFile(cacheDir)
	if err := os.WriteFile(cacheFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write cache file: %w", err)
	}

	return nil
}

func (c *Cache) AddNotifications(notifications []CacheEntry) []CacheEntry {
	c.LastSync = time.Now()

	// Create map of existing notification IDs
	existing := make(map[string]bool)
	for _, entry := range c.Notifications {
		existing[entry.ID] = true
	}

	// Find genuinely new notifications
	var newNotifications []CacheEntry
	for _, notification := range notifications {
		if !existing[notification.ID] {
			newNotifications = append(newNotifications, notification)
		}
	}

	// Replace entire cache with current unread notifications from GitHub
	// This automatically removes notifications that were read (not in incoming list)
	c.Notifications = notifications

	return newNotifications
}

// AddStarEvents adds star events to the cache and returns only new ones
func (c *Cache) AddStarEvents(starEvents []StarEvent) []StarEvent {
	// Create map of existing star IDs
	existing := make(map[string]bool)
	for _, star := range c.Stars {
		existing[star.ID] = true
	}

	// Find genuinely new star events
	var newStarEvents []StarEvent
	for _, star := range starEvents {
		if !existing[star.ID] {
			newStarEvents = append(newStarEvents, star)
			c.Stars = append(c.Stars, star)
		}
	}

	return newStarEvents
}

func (c *Cache) cleanup() {
	now := time.Now()

	// Cleanup notifications
	// All cached entries are unread by definition
	// Only need age and size limits as safety net
	var validEntries []CacheEntry
	for _, entry := range c.Notifications {
		if now.Sub(entry.Timestamp) <= MaxAge {
			validEntries = append(validEntries, entry)
		}
	}

	// Sort by UpdatedAt (newest first)
	sort.Slice(validEntries, func(i, j int) bool {
		return validEntries[i].UpdatedAt.After(validEntries[j].UpdatedAt)
	})

	// Apply max entries limit
	if len(validEntries) > c.MaxEntries {
		validEntries = validEntries[:c.MaxEntries]
	}

	c.Notifications = validEntries

	// Cleanup stars - keep stars for last 7 days
	var validStars []StarEvent
	for _, star := range c.Stars {
		if now.Sub(star.StarredAt) <= 7*24*time.Hour {
			validStars = append(validStars, star)
		}
	}

	// Sort by StarredAt (newest first)
	sort.Slice(validStars, func(i, j int) bool {
		return validStars[i].StarredAt.After(validStars[j].StarredAt)
	})

	// Apply max entries limit for stars (reuse MaxEntries)
	if len(validStars) > c.MaxEntries {
		validStars = validStars[:c.MaxEntries]
	}

	c.Stars = validStars
}

func (c *Cache) GetNotifications() []CacheEntry {
	// Return a copy to prevent external modification
	result := make([]CacheEntry, len(c.Notifications))
	copy(result, c.Notifications)
	return result
}

// GetStars returns a copy of cached star events
func (c *Cache) GetStars() []StarEvent {
	result := make([]StarEvent, len(c.Stars))
	copy(result, c.Stars)
	return result
}

func (c *Cache) Clear() {
	c.Notifications = []CacheEntry{}
	c.Stars = []StarEvent{}
	c.LastSync = time.Time{}
	c.LastEventSync = time.Time{}
}

func GetDefaultCacheDir() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get user home directory: %w", err)
	}

	return filepath.Join(homeDir, ".cache", "gh-notify"), nil
}
