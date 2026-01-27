package tools

import (
	"context"
	"errors"
	"testing"

	"ccui/backend"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockTool for testing
type mockTool struct {
	name   string
	result ToolResult
	err    error
}

func (m *mockTool) Name() string { return m.name }

func (m *mockTool) Execute(ctx context.Context, input map[string]any) (ToolResult, error) {
	return m.result, m.err
}

func TestRegistry_Register(t *testing.T) {
	a := assert.New(t)

	// given - empty registry
	reg := NewRegistry()

	// when - register a tool
	tool := &mockTool{name: "Read"}
	reg.Register(tool)

	// then - tool is registered
	a.True(reg.Has("Read"))
	a.False(reg.Has("NotRegistered"))
}

func TestRegistry_Execute_Success(t *testing.T) {
	a := assert.New(t)
	r := require.New(t)

	// given - registry with a tool
	reg := NewRegistry()
	tool := &mockTool{
		name:   "Read",
		result: ToolResult{Content: "file contents"},
	}
	reg.Register(tool)

	// when - execute the tool
	result, err := reg.Execute(context.Background(), "Read", map[string]any{"file_path": "/test.txt"})

	// then - returns tool result
	r.NoError(err)
	a.Equal("file contents", result.Content)
	a.False(result.IsError)
}

func TestRegistry_Execute_ToolError(t *testing.T) {
	a := assert.New(t)
	r := require.New(t)

	// given - registry with a tool that returns error result
	reg := NewRegistry()
	tool := &mockTool{
		name:   "Read",
		result: ToolResult{Content: "file not found", IsError: true},
	}
	reg.Register(tool)

	// when - execute the tool
	result, err := reg.Execute(context.Background(), "Read", map[string]any{"file_path": "/missing.txt"})

	// then - returns error result (not execution error)
	r.NoError(err)
	a.Equal("file not found", result.Content)
	a.True(result.IsError)
}

func TestRegistry_Execute_ExecutionError(t *testing.T) {
	a := assert.New(t)

	// given - registry with a tool that fails execution
	reg := NewRegistry()
	tool := &mockTool{
		name: "Bash",
		err:  errors.New("context canceled"),
	}
	reg.Register(tool)

	// when - execute the tool
	_, err := reg.Execute(context.Background(), "Bash", map[string]any{"command": "ls"})

	// then - returns execution error
	a.Error(err)
	a.Contains(err.Error(), "context canceled")
}

func TestRegistry_Execute_UnknownTool(t *testing.T) {
	a := assert.New(t)

	// given - empty registry
	reg := NewRegistry()

	// when - execute unknown tool
	_, err := reg.Execute(context.Background(), "Unknown", nil)

	// then - returns error
	a.Error(err)
	a.ErrorIs(err, ErrToolNotFound)
}

func TestRegistry_Tools(t *testing.T) {
	a := assert.New(t)

	// given - registry with multiple tools
	reg := NewRegistry()
	reg.Register(&mockTool{name: "Read"})
	reg.Register(&mockTool{name: "Write"})
	reg.Register(&mockTool{name: "Bash"})

	// when - get all tools
	tools := reg.Tools()

	// then - returns all registered tools
	a.Len(tools, 3)
	names := make(map[string]bool)
	for _, t := range tools {
		names[t.Name()] = true
	}
	a.True(names["Read"])
	a.True(names["Write"])
	a.True(names["Bash"])
}

func TestToolResult_WithFileDiff(t *testing.T) {
	a := assert.New(t)

	// given - registry with a tool that returns file diff
	reg := NewRegistry()
	tool := &mockTool{
		name: "Edit",
		result: ToolResult{
			Content:    "edited",
			FilePath:   "/test.go",
			OldContent: "old",
			NewContent: "new",
			Hunks: []backend.PatchHunk{{
				OldStart: 1,
				OldLines: 1,
				NewStart: 1,
				NewLines: 1,
				Lines:    []string{"-old", "+new"},
			}},
		},
	}
	reg.Register(tool)

	// when - execute
	result, err := reg.Execute(context.Background(), "Edit", nil)

	// then - result includes diff info
	a.NoError(err)
	a.Equal("/test.go", result.FilePath)
	a.Equal("old", result.OldContent)
	a.Equal("new", result.NewContent)
	a.Len(result.Hunks, 1)
	a.Equal(1, result.Hunks[0].OldStart)
}
