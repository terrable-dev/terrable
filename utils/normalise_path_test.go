package utils

import "testing"

func TestNormalisePath(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Empty path",
			input:    "",
			expected: "/",
		},
		{
			name:     "Already normalised path",
			input:    "/api/v1",
			expected: "/api/v1",
		},
		{
			name:     "Path without leading slash",
			input:    "api/v1",
			expected: "/api/v1",
		},
		{
			name:     "Single character path",
			input:    "a",
			expected: "/a",
		},
		{
			name:     "Path with special characters",
			input:    "api-v1/users/{id}",
			expected: "/api-v1/users/{id}",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := NormalisePath(tt.input)
			if result != tt.expected {
				t.Errorf("NormalisePath(%q) = %q, expected %q", tt.input, result, tt.expected)
			}
		})
	}
}
