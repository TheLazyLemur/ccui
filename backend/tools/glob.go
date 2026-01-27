package tools

import (
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/bmatcuk/doublestar/v4"
)

// GlobTool finds files matching glob patterns
type GlobTool struct{}

// NewGlobTool creates a new Glob tool
func NewGlobTool() *GlobTool {
	return &GlobTool{}
}

// Name returns "Glob"
func (g *GlobTool) Name() string {
	return "Glob"
}

// Execute finds files matching the pattern, sorted by modification time (newest first)
func (g *GlobTool) Execute(ctx context.Context, input map[string]any) (ToolResult, error) {
	// extract pattern (required)
	pattern, ok := input["pattern"].(string)
	if !ok || pattern == "" {
		return ToolResult{Content: "pattern is required", IsError: true}, nil
	}

	// extract optional base path
	basePath := "."
	if v, ok := input["path"].(string); ok && v != "" {
		basePath = v
	}

	// resolve to absolute path
	absPath, err := filepath.Abs(basePath)
	if err != nil {
		return ToolResult{Content: err.Error(), IsError: true}, nil
	}

	// verify path exists
	if _, err := os.Stat(absPath); err != nil {
		return ToolResult{Content: err.Error(), IsError: true}, nil
	}

	// find matching files
	type fileEntry struct {
		path    string
		modTime int64
	}
	var matches []fileEntry

	err = filepath.WalkDir(absPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil // skip errors, continue walking
		}
		if d.IsDir() {
			return nil
		}

		// get relative path for matching
		relPath, err := filepath.Rel(absPath, path)
		if err != nil {
			return nil
		}

		// check if pattern matches
		matched, err := doublestar.Match(pattern, relPath)
		if err != nil {
			return nil // invalid pattern, skip
		}
		if !matched {
			// also try matching just the filename for simple patterns
			matched, _ = doublestar.Match(pattern, filepath.Base(path))
		}

		if matched {
			info, err := d.Info()
			if err != nil {
				return nil
			}
			matches = append(matches, fileEntry{
				path:    path,
				modTime: info.ModTime().UnixNano(),
			})
		}
		return nil
	})
	if err != nil {
		return ToolResult{Content: err.Error(), IsError: true}, nil
	}

	// sort by modification time (newest first)
	sort.Slice(matches, func(i, j int) bool {
		return matches[i].modTime > matches[j].modTime
	})

	// build result
	if len(matches) == 0 {
		return ToolResult{Content: ""}, nil
	}

	var sb strings.Builder
	for i, m := range matches {
		sb.WriteString(m.path)
		if i < len(matches)-1 {
			sb.WriteByte('\n')
		}
	}

	return ToolResult{Content: sb.String()}, nil
}
