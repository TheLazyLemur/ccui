package tools

import (
	"context"
	"os"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBashTool_Name(t *testing.T) {
	a := assert.New(t)
	tool := NewBashTool()
	a.Equal("Bash", tool.Name())
}

func TestBashTool_Execute_Echo(t *testing.T) {
	a := assert.New(t)
	r := require.New(t)

	// given - simple echo command
	tool := NewBashTool()

	// when - execute echo
	result, err := tool.Execute(context.Background(), map[string]any{
		"command": "echo hello",
	})

	// then - returns output
	r.NoError(err)
	a.False(result.IsError)
	a.Equal("hello", result.Content)
}

func TestBashTool_Execute_Ls(t *testing.T) {
	a := assert.New(t)
	r := require.New(t)

	// given - ls command on temp dir
	dir := t.TempDir()
	tool := NewBashTool()

	// when - execute ls
	result, err := tool.Execute(context.Background(), map[string]any{
		"command": "ls " + dir,
	})

	// then - succeeds (empty dir = empty output)
	r.NoError(err)
	a.False(result.IsError)
	a.Equal("", result.Content)
}

func TestBashTool_Execute_LsWithFiles(t *testing.T) {
	a := assert.New(t)
	r := require.New(t)

	// given - dir with file
	dir := t.TempDir()
	r.NoError(os.WriteFile(dir+"/test.txt", []byte("content"), 0644))
	tool := NewBashTool()

	// when - execute ls
	result, err := tool.Execute(context.Background(), map[string]any{
		"command": "ls " + dir,
	})

	// then - shows file
	r.NoError(err)
	a.False(result.IsError)
	a.Contains(result.Content, "test.txt")
}

func TestBashTool_Execute_MissingCommand(t *testing.T) {
	a := assert.New(t)
	r := require.New(t)

	tool := NewBashTool()

	// when - execute without command
	result, err := tool.Execute(context.Background(), map[string]any{})

	// then - returns error result
	r.NoError(err)
	a.True(result.IsError)
	a.Contains(result.Content, "command is required")
}

func TestBashTool_Execute_FailedCommand(t *testing.T) {
	a := assert.New(t)
	r := require.New(t)

	tool := NewBashTool()

	// when - execute command that fails
	result, err := tool.Execute(context.Background(), map[string]any{
		"command": "ls /nonexistent_dir_12345",
	})

	// then - returns error result with stderr
	r.NoError(err)
	a.True(result.IsError)
	a.NotEmpty(result.Content) // contains error message
}

func TestBashTool_Execute_Timeout(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("sleep command differs on Windows")
	}

	a := assert.New(t)
	r := require.New(t)

	tool := NewBashTool()

	// when - execute long-running command with short timeout
	start := time.Now()
	result, err := tool.Execute(context.Background(), map[string]any{
		"command": "sleep 10",
		"timeout": float64(100), // 100ms timeout
	})
	elapsed := time.Since(start)

	// then - times out quickly
	r.NoError(err)
	a.True(result.IsError)
	a.Contains(result.Content, "timeout")
	a.Less(elapsed, 2*time.Second) // should timeout well before 10s
}

func TestBashTool_Execute_DefaultTimeout(t *testing.T) {
	a := assert.New(t)
	r := require.New(t)

	tool := NewBashTool()

	// when - execute quick command (uses default timeout)
	result, err := tool.Execute(context.Background(), map[string]any{
		"command": "echo quick",
	})

	// then - succeeds
	r.NoError(err)
	a.False(result.IsError)
	a.Equal("quick", result.Content)
}

func TestBashTool_Execute_MaxTimeoutCapped(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("sleep command differs on Windows")
	}

	a := assert.New(t)
	r := require.New(t)

	tool := NewBashTool()

	// when - request timeout beyond max (600000ms = 10min)
	// should be capped to max
	result, err := tool.Execute(context.Background(), map[string]any{
		"command": "echo capped",
		"timeout": float64(1000000), // 1000s > 600s max
	})

	// then - still succeeds (just cap timeout, don't error)
	r.NoError(err)
	a.False(result.IsError)
	a.Equal("capped", result.Content)
}

func TestBashTool_Execute_MultilineOutput(t *testing.T) {
	a := assert.New(t)
	r := require.New(t)

	tool := NewBashTool()

	// when - command with multiline output
	result, err := tool.Execute(context.Background(), map[string]any{
		"command": "echo -e 'line1\nline2\nline3'",
	})

	// then - preserves newlines
	r.NoError(err)
	a.False(result.IsError)
	a.Contains(result.Content, "line1")
	a.Contains(result.Content, "line2")
	a.Contains(result.Content, "line3")
}

func TestBashTool_Execute_ContextCancellation(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("sleep command differs on Windows")
	}

	a := assert.New(t)
	r := require.New(t)

	tool := NewBashTool()

	// given - already cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	// when - execute with cancelled context
	result, err := tool.Execute(ctx, map[string]any{
		"command": "sleep 10",
	})

	// then - returns error
	r.NoError(err)
	a.True(result.IsError)
}

