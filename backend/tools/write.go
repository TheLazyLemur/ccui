package tools

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
)

// WriteTool writes content to a file, creating parent directories as needed
type WriteTool struct{}

// NewWriteTool creates a new Write tool
func NewWriteTool() *WriteTool {
	return &WriteTool{}
}

// Name returns "Write"
func (w *WriteTool) Name() string {
	return "Write"
}

// Execute writes content to file_path, creating parent directories if needed
func (w *WriteTool) Execute(ctx context.Context, input map[string]any) (ToolResult, error) {
	// extract file_path (required)
	filePath, ok := input["file_path"].(string)
	if !ok || filePath == "" {
		return ToolResult{Content: "file_path is required", IsError: true}, nil
	}

	// extract content (required)
	content, ok := input["content"].(string)
	if !ok {
		return ToolResult{Content: "content is required", IsError: true}, nil
	}

	// create parent directories
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return ToolResult{Content: fmt.Sprintf("failed to create directory: %s", err), IsError: true}, nil
	}

	// write file
	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		return ToolResult{Content: fmt.Sprintf("failed to write file: %s", err), IsError: true}, nil
	}

	return ToolResult{
		Content:    fmt.Sprintf("wrote %d bytes to %s", len(content), filePath),
		FilePath:   filePath,
		NewContent: content,
	}, nil
}
