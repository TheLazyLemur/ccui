package anthropic

import (
	"context"

	"ccui/backend"
	"ccui/backend/tools"
	"ccui/permission"
)

const (
	defaultModel   = "claude-sonnet-4-20250514"
	defaultMaxTokens = 8192
	defaultBaseURL = "https://api.anthropic.com"
)

// AnthropicBackend implements AgentBackend for direct Anthropic API calls
type AnthropicBackend struct {
	apiKey    string
	baseURL   string
	model     string
	maxTokens int
	executor  tools.ToolExecutor
	permLayer *permission.Layer
}

// BackendConfig configures the Anthropic backend
type BackendConfig struct {
	APIKey    string
	BaseURL   string
	Model     string
	MaxTokens int
	Executor  tools.ToolExecutor
	PermLayer *permission.Layer
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
	}
}

// NewSession creates a new AnthropicSession
func (b *AnthropicBackend) NewSession(ctx context.Context, opts backend.SessionOpts) (backend.Session, error) {
	return newAnthropicSession(ctx, b, opts), nil
}
