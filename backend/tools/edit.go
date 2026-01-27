package tools

import (
	"context"
	"fmt"
	"os"
	"strings"

	"ccui/backend"
)

// EditTool performs string replacement edits on files
type EditTool struct{}

// NewEditTool creates a new Edit tool
func NewEditTool() *EditTool {
	return &EditTool{}
}

// Name returns "Edit"
func (e *EditTool) Name() string {
	return "Edit"
}

// Execute replaces old_string with new_string in file_path
func (e *EditTool) Execute(ctx context.Context, input map[string]any) (ToolResult, error) {
	// extract file_path (required)
	filePath, ok := input["file_path"].(string)
	if !ok || filePath == "" {
		return ToolResult{Content: "file_path is required", IsError: true}, nil
	}

	// extract old_string (required, non-empty)
	oldString, ok := input["old_string"].(string)
	if !ok || oldString == "" {
		return ToolResult{Content: "old_string is required and must be non-empty", IsError: true}, nil
	}

	// extract new_string (required, but can be empty for deletion)
	newString, ok := input["new_string"].(string)
	if !ok {
		return ToolResult{Content: "new_string is required", IsError: true}, nil
	}

	// validate old != new
	if oldString == newString {
		return ToolResult{Content: "old_string and new_string are the same; no change needed", IsError: true}, nil
	}

	// extract replace_all (optional, defaults to false)
	replaceAll := false
	if v, ok := input["replace_all"].(bool); ok {
		replaceAll = v
	}

	// read file
	data, err := os.ReadFile(filePath)
	if err != nil {
		return ToolResult{Content: fmt.Sprintf("failed to read file: %s", err), IsError: true}, nil
	}
	oldContent := string(data)

	// count occurrences
	count := strings.Count(oldContent, oldString)
	if count == 0 {
		return ToolResult{Content: fmt.Sprintf("old_string not found in file: %s", filePath), IsError: true}, nil
	}

	// validate uniqueness when replace_all is false
	if !replaceAll && count > 1 {
		return ToolResult{
			Content: fmt.Sprintf("old_string is not unique: found %d occurrences. Use replace_all=true to replace all, or provide more context to make it unique", count),
			IsError: true,
		}, nil
	}

	// perform replacement
	var newContent string
	if replaceAll {
		newContent = strings.ReplaceAll(oldContent, oldString, newString)
	} else {
		newContent = strings.Replace(oldContent, oldString, newString, 1)
	}

	// write file
	if err := os.WriteFile(filePath, []byte(newContent), 0644); err != nil {
		return ToolResult{Content: fmt.Sprintf("failed to write file: %s", err), IsError: true}, nil
	}

	// generate diff hunks
	hunks := generateHunks(oldContent, newContent)

	return ToolResult{
		Content:    fmt.Sprintf("edited %s", filePath),
		FilePath:   filePath,
		OldContent: oldContent,
		NewContent: newContent,
		Hunks:      hunks,
	}, nil
}

// generateHunks creates unified diff hunks from old and new content
func generateHunks(oldContent, newContent string) []backend.PatchHunk {
	oldLines := splitLinesForDiff(oldContent)
	newLines := splitLinesForDiff(newContent)

	// simple diff: find first difference and create single hunk
	// for more complex diffs, consider using go-diff library
	startOld, startNew := 0, 0
	endOld, endNew := len(oldLines), len(newLines)

	// find first differing line
	for startOld < len(oldLines) && startNew < len(newLines) && oldLines[startOld] == newLines[startNew] {
		startOld++
		startNew++
	}

	// find last differing line (from end)
	for endOld > startOld && endNew > startNew && oldLines[endOld-1] == newLines[endNew-1] {
		endOld--
		endNew--
	}

	// no differences
	if startOld == endOld && startNew == endNew {
		return nil
	}

	// build hunk lines
	var lines []string

	// context before (up to 3 lines)
	contextStart := startOld - 3
	if contextStart < 0 {
		contextStart = 0
	}
	for i := contextStart; i < startOld; i++ {
		lines = append(lines, " "+oldLines[i])
	}

	// removed lines
	for i := startOld; i < endOld; i++ {
		lines = append(lines, "-"+oldLines[i])
	}

	// added lines
	for i := startNew; i < endNew; i++ {
		lines = append(lines, "+"+newLines[i])
	}

	// context after (up to 3 lines)
	contextEnd := endOld + 3
	if contextEnd > len(oldLines) {
		contextEnd = len(oldLines)
	}
	for i := endOld; i < contextEnd; i++ {
		lines = append(lines, " "+oldLines[i])
	}

	hunk := backend.PatchHunk{
		OldStart: contextStart + 1, // 1-indexed
		OldLines: endOld - contextStart + (contextEnd - endOld),
		NewStart: contextStart + 1,
		NewLines: endNew - contextStart + (contextEnd - endOld),
		Lines:    lines,
	}

	return []backend.PatchHunk{hunk}
}

// splitLinesForDiff splits content into lines for diff generation
func splitLinesForDiff(content string) []string {
	if content == "" {
		return []string{}
	}
	lines := strings.Split(content, "\n")
	// remove trailing empty string from final newline
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	return lines
}
