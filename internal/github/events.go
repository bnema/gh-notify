package github

import (
	"strings"
	"time"
)

// EventEntry represents a GitHub event from the activity API
type EventEntry struct {
	ID         string                 `json:"id"`
	Type       string                 `json:"type"`
	Actor      Actor                  `json:"actor"`
	Repository Repository             `json:"repo"`
	Payload    map[string]interface{} `json:"payload"`
	CreatedAt  time.Time              `json:"created_at"`
	Public     bool                   `json:"public"`
}

// Actor represents the user who triggered the event
type Actor struct {
	ID          int    `json:"id"`
	Login       string `json:"login"`
	DisplayLogin string `json:"display_login"`
	GravatarID  string `json:"gravatar_id"`
	URL         string `json:"url"`
	AvatarURL   string `json:"avatar_url"`
}

// Repository represents the repository where the event occurred
type Repository struct {
	ID   int    `json:"id"`
	Name string `json:"name"` // Format: "owner/repo"
	URL  string `json:"url"`
}

// StarEvent represents a star event for caching
type StarEvent struct {
	ID         string    `json:"id"`
	Repository string    `json:"repository"` // Full name: "owner/repo"
	StarredBy  string    `json:"starred_by"`
	StarredAt  time.Time `json:"starred_at"`
	Notified   bool      `json:"notified"`
}

// IsStarEvent checks if an EventEntry is a star event
func (e EventEntry) IsStarEvent() bool {
	if e.Type != "WatchEvent" {
		return false
	}
	
	// Check if the payload contains action: "started"
	if payload, ok := e.Payload["action"]; ok {
		if action, ok := payload.(string); ok {
			return action == "started"
		}
	}
	
	return false
}

// ToStarEvent converts an EventEntry to a StarEvent
func (e EventEntry) ToStarEvent() StarEvent {
	return StarEvent{
		ID:         e.ID,
		Repository: e.Repository.Name,
		StarredBy:  e.Actor.Login,
		StarredAt:  e.CreatedAt,
		Notified:   false,
	}
}

// GetRepositoryOwner extracts the owner from the repository name (format: "owner/repo")
func (e EventEntry) GetRepositoryOwner() string {
	// Repository.Name is in format "owner/repo"
	repoName := e.Repository.Name
	if repoName == "" {
		return ""
	}
	
	// Split by "/" and get the first part (owner)
	parts := strings.Split(repoName, "/")
	if len(parts) >= 1 {
		return parts[0]
	}
	
	return ""
}