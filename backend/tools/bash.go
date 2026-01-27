package tools

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

const (
	defaultTimeoutMs = 120000  // 2 minutes
	maxTimeoutMs     = 600000  // 10 minutes
)

// BashTool executes bash commands
type BashTool struct{}

// NewBashTool creates a new Bash tool
func NewBashTool() *BashTool {
	return &BashTool{}
}

// Name returns "Bash"
func (b *BashTool) Name() string {
	return "Bash"
}

// Execute runs a bash command with optional timeout
func (b *BashTool) Execute(ctx context.Context, input map[string]any) (ToolResult, error) {
	// extract command (required)
	command, ok := input["command"].(string)
	if !ok || command == "" {
		return ToolResult{Content: "command is required", IsError: true}, nil
	}

	// extract timeout (optional, defaults to 120000ms, max 600000ms)
	timeoutMs := defaultTimeoutMs
	if v, ok := input["timeout"].(float64); ok && v > 0 {
		timeoutMs = int(v)
		if timeoutMs > maxTimeoutMs {
			timeoutMs = maxTimeoutMs
		}
	}

	// create context with timeout
	timeout := time.Duration(timeoutMs) * time.Millisecond
	cmdCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// run command via bash -c
	cmd := exec.CommandContext(cmdCtx, "bash", "-c", command)

	// capture combined stdout+stderr
	var output bytes.Buffer
	cmd.Stdout = &output
	cmd.Stderr = &output

	// execute
	err := cmd.Run()

	// trim trailing whitespace from output
	result := strings.TrimRight(output.String(), "\n\r\t ")

	// check for timeout
	if cmdCtx.Err() == context.DeadlineExceeded {
		return ToolResult{
			Content: fmt.Sprintf("command timeout after %dms: %s", timeoutMs, result),
			IsError: true,
		}, nil
	}

	// check for context cancellation
	if ctx.Err() == context.Canceled {
		return ToolResult{
			Content: "command cancelled",
			IsError: true,
		}, nil
	}

	// check for execution error
	if err != nil {
		// include output with error (often contains useful stderr)
		if result != "" {
			return ToolResult{Content: result, IsError: true}, nil
		}
		return ToolResult{Content: err.Error(), IsError: true}, nil
	}

	return ToolResult{Content: result}, nil
}
