package github

import "strings"

// Error type constants for GitHub API error classification
const (
	ErrorTypeRateLimit  = "rate_limit"
	ErrorTypeTimeout    = "timeout"
	ErrorTypePermission = "permission"
	ErrorTypeNotFound   = "not_found"
	ErrorTypeNetwork    = "network"
	ErrorTypeUnknown    = "unknown"
)

// ClassifyGitHubError classifies GitHub API errors for better logging and handling.
// This works for any GitHub API error (REST, GraphQL, notifications, stars, etc.)
func ClassifyGitHubError(err error) string {
	if err == nil {
		return ErrorTypeUnknown
	}

	errStr := strings.ToLower(err.Error())

	// Check for rate limiting
	if strings.Contains(errStr, "rate limit") || strings.Contains(errStr, "abuse detection") {
		return ErrorTypeRateLimit
	}

	// Check for timeout errors
	if strings.Contains(errStr, "timeout") || strings.Contains(errStr, "deadline exceeded") {
		return ErrorTypeTimeout
	}

	// Check for permission/access errors
	if strings.Contains(errStr, "forbidden") ||
		strings.Contains(errStr, "unauthorized") ||
		strings.Contains(errStr, "permission denied") ||
		strings.Contains(errStr, "403") {
		return ErrorTypePermission
	}

	// Check for not found errors
	if strings.Contains(errStr, "not found") ||
		strings.Contains(errStr, "404") ||
		strings.Contains(errStr, "could not resolve") {
		return ErrorTypeNotFound
	}

	// Check for network errors
	if strings.Contains(errStr, "connection refused") ||
		strings.Contains(errStr, "no such host") ||
		strings.Contains(errStr, "network") {
		return ErrorTypeNetwork
	}

	return ErrorTypeUnknown
}
