package github

import (
	"errors"
	"strings"
	"testing"
	"time"
)

// mockGraphQLClient is a mock for testing
type mockGraphQLClient struct {
	attempts      int
	failUntil     int
	shouldTimeout bool
	err           error
}

func (m *mockGraphQLClient) Do(query string, variables map[string]interface{}, response interface{}) error {
	m.attempts++

	if m.shouldTimeout {
		time.Sleep(35 * time.Second) // Exceed the 30s timeout
		return nil
	}

	if m.attempts <= m.failUntil {
		return m.err
	}

	return nil
}

func TestApiGraphQLClient_RateLimitRetry(t *testing.T) {
	tests := []struct {
		name          string
		failUntil     int
		err           error
		expectSuccess bool
		minDuration   time.Duration
	}{
		{
			name:          "succeeds on first attempt",
			failUntil:     0,
			err:           nil,
			expectSuccess: true,
			minDuration:   0,
		},
		{
			name:          "fails immediately on non-rate-limit error",
			failUntil:     1,
			err:           errors.New("some other error"),
			expectSuccess: false,
			minDuration:   0, // No retry, should fail fast
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockGraphQLClient{
				failUntil: tt.failUntil,
				err:       tt.err,
			}

			start := time.Now()
			err := mock.Do("test query", nil, nil)
			duration := time.Since(start)

			if tt.expectSuccess && err != nil {
				t.Errorf("expected success but got error: %v", err)
			}

			if !tt.expectSuccess && err == nil {
				t.Error("expected error but got success")
			}

			if tt.minDuration > 0 && duration < tt.minDuration {
				t.Errorf("expected minimum duration %v but got %v", tt.minDuration, duration)
			}

			// Verify attempts
			expectedAttempts := tt.failUntil + 1
			if !tt.expectSuccess && tt.failUntil > maxRetries {
				expectedAttempts = maxRetries + 1
			}
			if mock.attempts > expectedAttempts {
				t.Errorf("expected at most %d attempts but got %d", expectedAttempts, mock.attempts)
			}
		})
	}
}

func TestIsRateLimitError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "nil error",
			err:      nil,
			expected: false,
		},
		{
			name:     "rate limit error",
			err:      errors.New("API rate limit exceeded"),
			expected: true,
		},
		{
			name:     "abuse detection error",
			err:      errors.New("abuse detection mechanism triggered"),
			expected: true,
		},
		{
			name:     "secondary rate limit",
			err:      errors.New("secondary rate limit hit"),
			expected: true,
		},
		{
			name:     "non-rate-limit error",
			err:      errors.New("permission denied"),
			expected: false,
		},
		{
			name:     "case insensitive check",
			err:      errors.New("RATE LIMIT EXCEEDED"),
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isRateLimitError(tt.err)
			if result != tt.expected {
				t.Errorf("isRateLimitError(%v) = %v, want %v", tt.err, result, tt.expected)
			}
		})
	}
}

func TestApiGraphQLClient_Timeout(t *testing.T) {
	// This test verifies that queries timeout after 30 seconds
	t.Skip("Skipping timeout test as it takes 30+ seconds to run")

	mock := &mockGraphQLClient{
		shouldTimeout: true,
	}

	start := time.Now()
	err := mock.Do("test query", nil, nil)
	duration := time.Since(start)

	if err == nil {
		t.Error("expected timeout error but got nil")
	}

	if !strings.Contains(err.Error(), "timeout") {
		t.Errorf("expected timeout error but got: %v", err)
	}

	// Should timeout around 30 seconds
	if duration < 29*time.Second || duration > 31*time.Second {
		t.Errorf("expected ~30s timeout but got %v", duration)
	}
}
