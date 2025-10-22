package github

import (
	"errors"
	"testing"
)

func TestClassifyGitHubError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected string
	}{
		{
			name:     "nil error",
			err:      nil,
			expected: ErrorTypeUnknown,
		},
		{
			name:     "rate limit error",
			err:      errors.New("API rate limit exceeded for user"),
			expected: ErrorTypeRateLimit,
		},
		{
			name:     "abuse detection error",
			err:      errors.New("You have triggered an abuse detection mechanism"),
			expected: ErrorTypeRateLimit,
		},
		{
			name:     "timeout error",
			err:      errors.New("request timeout after 30s"),
			expected: ErrorTypeTimeout,
		},
		{
			name:     "deadline exceeded error",
			err:      errors.New("context deadline exceeded"),
			expected: ErrorTypeTimeout,
		},
		{
			name:     "403 forbidden error",
			err:      errors.New("403 forbidden"),
			expected: ErrorTypePermission,
		},
		{
			name:     "unauthorized error",
			err:      errors.New("unauthorized access"),
			expected: ErrorTypePermission,
		},
		{
			name:     "permission denied error",
			err:      errors.New("permission denied to repository"),
			expected: ErrorTypePermission,
		},
		{
			name:     "404 not found error",
			err:      errors.New("404 Not Found"),
			expected: ErrorTypeNotFound,
		},
		{
			name:     "repository not found error",
			err:      errors.New("repository not found"),
			expected: ErrorTypeNotFound,
		},
		{
			name:     "could not resolve error",
			err:      errors.New("could not resolve to a Repository"),
			expected: ErrorTypeNotFound,
		},
		{
			name:     "connection refused error",
			err:      errors.New("connection refused"),
			expected: ErrorTypeNetwork,
		},
		{
			name:     "no such host error",
			err:      errors.New("no such host"),
			expected: ErrorTypeNetwork,
		},
		{
			name:     "network error",
			err:      errors.New("network unreachable"),
			expected: ErrorTypeNetwork,
		},
		{
			name:     "unknown error",
			err:      errors.New("some random error"),
			expected: ErrorTypeUnknown,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ClassifyGitHubError(tt.err)
			if result != tt.expected {
				t.Errorf("ClassifyGitHubError(%v) = %v, want %v", tt.err, result, tt.expected)
			}
		})
	}
}

// Test case sensitivity
func TestClassifyGitHubError_CaseInsensitive(t *testing.T) {
	tests := []struct {
		name string
		err  error
	}{
		{"uppercase", errors.New("RATE LIMIT EXCEEDED")},
		{"mixed case", errors.New("Rate Limit Exceeded")},
		{"lowercase", errors.New("rate limit exceeded")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ClassifyGitHubError(tt.err)
			if result != ErrorTypeRateLimit {
				t.Errorf("ClassifyGitHubError(%v) = %v, want %v (case insensitive check failed)", tt.err, result, ErrorTypeRateLimit)
			}
		})
	}
}
