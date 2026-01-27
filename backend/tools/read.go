package tools

import (
	"context"
	"fmt"
	"os"
	"strings"
)

// ReadTool reads files with optional offset and limit
type ReadTool struct{}

// NewReadTool creates a new Read tool
func NewReadTool() *ReadTool {
	return &ReadTool{}
}

// Name returns "Read"
func (r *ReadTool) Name() string {
	return "Read"
}

// Execute reads a file and returns content with line numbers
func (r *ReadTool) Execute(ctx context.Context, input map[string]any) (ToolResult, error) {
	// extract file_path (required)
	filePath, ok := input["file_path"].(string)
	if !ok || filePath == "" {
		return ToolResult{Content: "file_path is required", IsError: true}, nil
	}

	// extract optional offset (1-indexed line number)
	offset := 1
	if v, ok := input["offset"].(float64); ok && v > 0 {
		offset = int(v)
	}

	// extract optional limit
	limit := -1 // -1 means no limit
	if v, ok := input["limit"].(float64); ok && v > 0 {
		limit = int(v)
	}

	// read file
	data, err := os.ReadFile(filePath)
	if err != nil {
		return ToolResult{Content: err.Error(), IsError: true}, nil
	}

	// handle empty file
	if len(data) == 0 {
		return ToolResult{Content: ""}, nil
	}

	// split into lines
	content := string(data)
	lines := strings.Split(content, "\n")

	// handle trailing newline - don't count empty line after final \n
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}

	// apply offset (1-indexed)
	startIdx := offset - 1
	if startIdx >= len(lines) {
		return ToolResult{Content: ""}, nil
	}
	if startIdx < 0 {
		startIdx = 0
	}

	// apply limit
	endIdx := len(lines)
	if limit > 0 && startIdx+limit < endIdx {
		endIdx = startIdx + limit
	}

	// format with line numbers (cat -n style: right-aligned number + tab)
	var sb strings.Builder
	for i := startIdx; i < endIdx; i++ {
		lineNum := i + 1 // 1-indexed
		sb.WriteString(fmt.Sprintf("%d\t%s\n", lineNum, lines[i]))
	}

	// trim final newline for cleaner output
	result := strings.TrimSuffix(sb.String(), "\n")

	return ToolResult{Content: result}, nil
}
