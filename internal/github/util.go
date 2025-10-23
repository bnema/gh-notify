package github

import (
	"fmt"
	"regexp"
	"strings"
)

// GetOwner extracts the owner from a full repository name.
// Example: "owner/repo" returns "owner"
func GetOwner(repoFullName string) string {
	parts := strings.Split(repoFullName, "/")
	if len(parts) >= 1 {
		return parts[0]
	}
	return ""
}

// GetName extracts the repository name from a full repository name.
// Example: "owner/repo" returns "repo"
func GetName(repoFullName string) string {
	parts := strings.Split(repoFullName, "/")
	if len(parts) >= 2 {
		return parts[1]
	}
	return ""
}

// ConvertAPIURLToWeb converts GitHub API URLs to web URLs.
// Example: https://api.github.com/repos/owner/repo/issues/123 -> https://github.com/owner/repo/issues/123
func ConvertAPIURLToWeb(apiURL string) string {
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
