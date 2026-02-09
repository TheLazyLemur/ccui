package mcpclient

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExtractServerName(t *testing.T) {
	tests := []struct {
		name     string
		baseURL  string
		expected string
	}{
		{
			name:     "localhost URL returns default ccui",
			baseURL:  "http://127.0.0.1:8080/sse",
			expected: "ccui",
		},
		{
			name:     "localhost with port returns default ccui",
			baseURL:  "http://localhost:9000",
			expected: "ccui",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractServerName(tt.baseURL)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestExtractToolName(t *testing.T) {
	tests := []struct {
		name         string
		prefixedName string
		serverName   string
		expected     string
	}{
		{
			name:         "prefixed tool name",
			prefixedName: "mcp__ccui__ask_user_question",
			serverName:   "ccui",
			expected:     "ask_user_question",
		},
		{
			name:         "non-prefixed name returns as-is",
			prefixedName: "Read",
			serverName:   "ccui",
			expected:     "Read",
		},
		{
			name:         "different server prefix",
			prefixedName: "mcp__other__some_tool",
			serverName:   "other",
			expected:     "some_tool",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractToolName(tt.prefixedName, tt.serverName)
			assert.Equal(t, tt.expected, result)
		})
	}
}
