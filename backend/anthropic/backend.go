package anthropic

import (
	"context"

	"ccui/backend"
	"ccui/backend/tools"
	"ccui/permission"
)

const (
	defaultModel     = "claude-sonnet-4-20250514"
	defaultMaxTokens = 8192
	defaultBaseURL   = "https://api.anthropic.com"
)

// ToolProvider provides tools for the Anthropic backend
type ToolProvider interface {
	Name() string
	GetTools() []Tool
	Execute(ctx context.Context, name string, input map[string]any) (tools.ToolResult, error)
}

// AnthropicBackend implements AgentBackend for direct Anthropic API calls
type AnthropicBackend struct {
	apiKey     string
	baseURL    string
	model      string
	maxTokens  int
	executor   tools.ToolExecutor
	permLayer  *permission.Layer
	providers  []ToolProvider // MCP tool providers
}

// BackendConfig configures the Anthropic backend
type BackendConfig struct {
	APIKey    string
	BaseURL   string
	Model     string
	MaxTokens int
	Executor  tools.ToolExecutor
	PermLayer *permission.Layer
	Providers []ToolProvider // MCP tool providers
}

// NewAnthropicBackend creates a new backend with config
func NewAnthropicBackend(cfg BackendConfig) *AnthropicBackend {
	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = defaultBaseURL
	}
	model := cfg.Model
	if model == "" {
		model = defaultModel
	}
	maxTokens := cfg.MaxTokens
	if maxTokens == 0 {
		maxTokens = defaultMaxTokens
	}
	return &AnthropicBackend{
		apiKey:    cfg.APIKey,
		baseURL:   baseURL,
		model:     model,
		maxTokens: maxTokens,
		executor:  cfg.Executor,
		permLayer: cfg.PermLayer,
		providers: cfg.Providers,
	}
}

// getAllTools returns combined tools from default tools and all providers
func (b *AnthropicBackend) getAllTools() []Tool {
	allTools := DefaultTools()

	for _, provider := range b.providers {
		providerTools := provider.GetTools()
		allTools = append(allTools, providerTools...)
	}

	return allTools
}

// findProvider finds a provider that can execute the given tool name
func (b *AnthropicBackend) findProvider(toolName string) ToolProvider {
	// Check if it's an MCP tool (prefixed with "mcp__")
	if len(toolName) > 5 && toolName[:5] == "mcp__" {
		// Extract server name from "mcp__{server}__{tool}"
		parts := splitToolName(toolName)
		if len(parts) >= 2 {
			serverName := parts[1]
			for _, provider := range b.providers {
				if provider.Name() == serverName {
					return provider
				}
			}
		}
	}
	return nil
}

// splitToolName splits a tool name like "mcp__server__tool" into parts
func splitToolName(name string) []string {
	var parts []string
	current := ""
	for i := 0; i < len(name); i++ {
		if i+1 < len(name) && name[i] == '_' && name[i+1] == '_' {
			parts = append(parts, current)
			current = ""
			i++ // Skip second underscore
		} else {
			current += string(name[i])
		}
	}
	parts = append(parts, current)
	return parts
}

// NewSession creates a new AnthropicSession
func (b *AnthropicBackend) NewSession(ctx context.Context, opts backend.SessionOpts) (backend.Session, error) {
	return newAnthropicSession(ctx, b, opts), nil
}
