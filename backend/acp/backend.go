package acp

import (
	"context"
	"fmt"
	"os"
	"os/exec"

	"ccui/backend"
)

// ACPBackend implements AgentBackend for claude-code-acp subprocess
type ACPBackend struct {
	ctx    context.Context
	apiKey string
}

// NewACPBackend creates a new ACP backend
func NewACPBackend(ctx context.Context, apiKey string) *ACPBackend {
	return &ACPBackend{ctx: ctx, apiKey: apiKey}
}

// NewSession creates a new ACP session
func (b *ACPBackend) NewSession(ctx context.Context, opts backend.SessionOpts) (backend.Session, error) {
	cmd := exec.CommandContext(ctx, "claude-code-acp")
	cmd.Env = append(os.Environ(), "ANTHROPIC_API_KEY="+b.apiKey)
	cmd.Dir = opts.CWD
	cmd.Stderr = os.Stderr

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("stdin pipe: %w", err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("stdout pipe: %w", err)
	}
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("start: %w", err)
	}

	client := NewClient(ClientConfig{
		Transport:          NewStdioTransport(stdin, stdout),
		EventChan:          opts.EventChan,
		AutoPermission:     opts.AutoPermission,
		SuppressToolEvents: opts.SuppressToolEvents,
		FileChangeStore:    opts.FileChangeStore,
	})

	if err := client.Initialize(); err != nil {
		cmd.Process.Kill()
		return nil, fmt.Errorf("initialize: %w", err)
	}
	if err := client.NewSession(opts.CWD, opts.MCPServers); err != nil {
		cmd.Process.Kill()
		return nil, fmt.Errorf("new session: %w", err)
	}

	return client, nil
}
