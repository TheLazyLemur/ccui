package main

import (
	"bufio"
	"encoding/json"
	"strconv"
	"strings"
)

type ToolEventAdapter interface {
	Name() string
	CanHandle(update UpdateContent) bool
	ToolName(update UpdateContent) string
	DiffBlocks(update UpdateContent) []DiffBlock
	ToolResponse(update UpdateContent) *ToolResponse
}

func defaultToolAdapters() []ToolEventAdapter {
	return []ToolEventAdapter{
		ClaudeCodeAdapter{},
		OpenCodeAdapter{},
	}
}

func (c *ACPClient) adapterFor(update UpdateContent) ToolEventAdapter {
	for _, adapter := range c.toolAdapters {
		if adapter.CanHandle(update) {
			return adapter
		}
	}
	return nil
}

func resolveToolName(adapter ToolEventAdapter, update UpdateContent) string {
	if adapter != nil {
		name := adapter.ToolName(update)
		if name != "" {
			return name
		}
	}
	return normalizeToolName(update.Title, update.ToolKind)
}

type ClaudeCodeAdapter struct{}

func (ClaudeCodeAdapter) Name() string {
	return "claude-code"
}

func (ClaudeCodeAdapter) CanHandle(update UpdateContent) bool {
	return update.Meta != nil && update.Meta.ClaudeCode != nil
}

func (ClaudeCodeAdapter) ToolName(update UpdateContent) string {
	if update.Meta != nil && update.Meta.ClaudeCode != nil {
		return normalizeToolName(update.Meta.ClaudeCode.ToolName, "")
	}
	return ""
}

func (ClaudeCodeAdapter) DiffBlocks(update UpdateContent) []DiffBlock {
	// Claude Code provides diff data via ToolResponse, not Content field
	return nil
}

func (ClaudeCodeAdapter) ToolResponse(update UpdateContent) *ToolResponse {
	if update.Meta == nil || update.Meta.ClaudeCode == nil {
		return nil
	}
	return update.Meta.ClaudeCode.ToolResponse
}

type OpenCodeAdapter struct{}

func (OpenCodeAdapter) Name() string {
	return "opencode"
}

func (OpenCodeAdapter) CanHandle(update UpdateContent) bool {
	return true
}

func (OpenCodeAdapter) ToolName(update UpdateContent) string {
	return normalizeToolName(update.Title, update.ToolKind)
}

func (OpenCodeAdapter) DiffBlocks(update UpdateContent) []DiffBlock {
	return parseDiffBlocks(update.Content)
}

func (OpenCodeAdapter) ToolResponse(update UpdateContent) *ToolResponse {
	diffs := parseDiffBlocks(update.Content)
	meta := extractOpenCodeMeta(update.RawOutput)
	primary := firstDiffBlock(diffs)
	toolName := normalizeToolName(update.Title, update.ToolKind)
	filePath := firstNonEmpty(primary.Path, meta.filePath)
	oldText := primary.OldText
	newText := primary.NewText
	if filePath == "" && meta.filePath == "" && oldText == "" && newText == "" && meta.original == "" && meta.current == "" {
		return nil
	}

	tr := &ToolResponse{
		FilePath:     filePath,
		OldString:    oldText,
		NewString:    newText,
		OriginalFile: oldText,
		Type:         primary.Type,
	}
	if tr.FilePath == "" {
		tr.FilePath = meta.filePath
	}
	if meta.original != "" {
		tr.OriginalFile = meta.original
	}
	if meta.current != "" {
		tr.Content = meta.current
	}
	if tr.Content == "" && newText != "" {
		tr.Content = newText
	}
	if tr.OriginalFile == "" {
		tr.OriginalFile = oldText
	}
	tr.StructuredPatch = meta.hunks
	if len(tr.StructuredPatch) == 0 {
		tr.StructuredPatch = buildHunksFromTexts(tr.OriginalFile, tr.Content)
	}
	if tr.Content == "" && toolName == "Write" {
		tr.Content = newText
	}
	return tr
}

type openCodeMeta struct {
	filePath string
	original string
	current  string
	hunks    []PatchHunk
}

func extractOpenCodeMeta(rawOutput *ToolRawOutput) openCodeMeta {
	if rawOutput == nil || rawOutput.Metadata == nil {
		return openCodeMeta{}
	}
	meta := openCodeMeta{}
	if rawOutput.Metadata.Filediff != nil {
		meta.filePath = rawOutput.Metadata.Filediff.File
		meta.original = rawOutput.Metadata.Filediff.Before
		meta.current = rawOutput.Metadata.Filediff.After
	}
	if meta.filePath == "" {
		meta.filePath = rawOutput.Metadata.Filepath
	}
	meta.hunks = parseUnifiedDiff(rawOutput.Metadata.Diff)
	return meta
}

func firstDiffBlock(diffs []DiffBlock) DiffBlock {
	for _, diff := range diffs {
		if diff.Type == "diff" {
			return diff
		}
	}
	return DiffBlock{}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func normalizeToolName(title, kind string) string {
	name := title
	if name == "" {
		name = kind
	}
	if strings.EqualFold(name, "edit") {
		return "Edit"
	}
	if strings.EqualFold(name, "write") {
		return "Write"
	}
	return name
}

func parseDiffBlocks(content json.RawMessage) []DiffBlock {
	if len(content) == 0 || content[0] != '[' {
		return nil
	}
	var diffs []DiffBlock
	if err := json.Unmarshal(content, &diffs); err != nil {
		return nil
	}
	return diffs
}

func buildHunksFromTexts(oldText, newText string) []PatchHunk {
	oldLines := splitLines(oldText)
	newLines := splitLines(newText)
	if len(oldLines) == 0 && len(newLines) == 0 {
		return nil
	}
	lines := make([]string, 0, len(oldLines)+len(newLines))
	for _, line := range oldLines {
		lines = append(lines, "-"+line)
	}
	for _, line := range newLines {
		lines = append(lines, "+"+line)
	}
	return []PatchHunk{{
		OldStart: 1,
		OldLines: len(oldLines),
		NewStart: 1,
		NewLines: len(newLines),
		Lines:    lines,
	}}
}

func parseUnifiedDiff(diffText string) []PatchHunk {
	if diffText == "" {
		return nil
	}
	scanner := bufio.NewScanner(strings.NewReader(diffText))
	var hunks []PatchHunk
	var current *PatchHunk
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "@@") {
			oldStart, oldLines, newStart, newLines, ok := parseHunkHeader(line)
			if !ok {
				current = nil
				continue
			}
			hunk := PatchHunk{
				OldStart: oldStart,
				OldLines: oldLines,
				NewStart: newStart,
				NewLines: newLines,
			}
			hunks = append(hunks, hunk)
			current = &hunks[len(hunks)-1]
			continue
		}
		if current == nil {
			continue
		}
		if strings.HasPrefix(line, "\\") {
			continue
		}
		current.Lines = append(current.Lines, line)
	}
	return hunks
}

func parseHunkHeader(line string) (int, int, int, int, bool) {
	trimmed := strings.TrimSpace(strings.TrimPrefix(line, "@@"))
	trimmed = strings.TrimSuffix(trimmed, "@@")
	trimmed = strings.TrimSpace(trimmed)
	parts := strings.Split(trimmed, " ")
	if len(parts) < 2 {
		return 0, 0, 0, 0, false
	}
	oldStart, oldLines, ok := parseRange(strings.TrimPrefix(parts[0], "-"))
	if !ok {
		return 0, 0, 0, 0, false
	}
	newStart, newLines, ok := parseRange(strings.TrimPrefix(parts[1], "+"))
	if !ok {
		return 0, 0, 0, 0, false
	}
	return oldStart, oldLines, newStart, newLines, true
}

func parseRange(part string) (int, int, bool) {
	if part == "" {
		return 0, 0, false
	}
	pieces := strings.Split(part, ",")
	start, err := strconv.Atoi(pieces[0])
	if err != nil {
		return 0, 0, false
	}
	lines := 1
	if len(pieces) > 1 {
		lines, err = strconv.Atoi(pieces[1])
		if err != nil {
			return 0, 0, false
		}
	}
	return start, lines, true
}

func splitLines(text string) []string {
	if text == "" {
		return nil
	}
	normalized := strings.ReplaceAll(text, "\r\n", "\n")
	lines := strings.Split(normalized, "\n")
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		return lines[:len(lines)-1]
	}
	return lines
}
