package tools

import (
	"context"
	"errors"
	"sync"

	"ccui/backend"
)

// ErrToolNotFound returned when executing unregistered tool
var ErrToolNotFound = errors.New("tool not found")

// ToolResult returned by tool execution
type ToolResult struct {
	Content    string             // output text
	IsError    bool               // true if tool reports an error
	FilePath   string             // for file-modifying tools
	OldContent string             // original content before edit
	NewContent string             // content after edit
	Hunks      []backend.PatchHunk // diff hunks for file changes
}

// Tool interface for individual tool implementations
type Tool interface {
	Name() string
	Execute(ctx context.Context, input map[string]any) (ToolResult, error)
}

// ToolExecutor executes tools by name
type ToolExecutor interface {
	Execute(ctx context.Context, name string, input map[string]any) (ToolResult, error)
}

// Registry stores tools and dispatches execution
type Registry struct {
	tools map[string]Tool
	mu    sync.RWMutex
}

// NewRegistry creates an empty tool registry
func NewRegistry() *Registry {
	return &Registry{tools: make(map[string]Tool)}
}

// Register adds a tool to the registry
func (r *Registry) Register(tool Tool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.tools[tool.Name()] = tool
}

// Has checks if a tool is registered
func (r *Registry) Has(name string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	_, ok := r.tools[name]
	return ok
}

// Execute runs the named tool with the given input
func (r *Registry) Execute(ctx context.Context, name string, input map[string]any) (ToolResult, error) {
	r.mu.RLock()
	tool, ok := r.tools[name]
	r.mu.RUnlock()

	if !ok {
		return ToolResult{}, ErrToolNotFound
	}
	return tool.Execute(ctx, input)
}

// Tools returns all registered tools
func (r *Registry) Tools() []Tool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]Tool, 0, len(r.tools))
	for _, t := range r.tools {
		result = append(result, t)
	}
	return result
}
