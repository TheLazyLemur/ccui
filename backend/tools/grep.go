package tools

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// GrepTool searches files for patterns using regex
type GrepTool struct{}

// NewGrepTool creates a new Grep tool
func NewGrepTool() *GrepTool {
	return &GrepTool{}
}

// Name returns "Grep"
func (g *GrepTool) Name() string {
	return "Grep"
}

// Execute searches for pattern in files
func (g *GrepTool) Execute(ctx context.Context, input map[string]any) (ToolResult, error) {
	// extract pattern (required)
	pattern, ok := input["pattern"].(string)
	if !ok || pattern == "" {
		return ToolResult{Content: "pattern is required", IsError: true}, nil
	}

	// case insensitive flag
	caseInsensitive := false
	if v, ok := input["-i"].(bool); ok {
		caseInsensitive = v
	}

	// compile regex
	if caseInsensitive {
		pattern = "(?i)" + pattern
	}
	re, err := regexp.Compile(pattern)
	if err != nil {
		return ToolResult{Content: fmt.Sprintf("invalid regex: %v", err), IsError: true}, nil
	}

	// extract path (optional, defaults to cwd)
	searchPath := "."
	if v, ok := input["path"].(string); ok && v != "" {
		searchPath = v
	}

	// verify path exists
	info, err := os.Stat(searchPath)
	if err != nil {
		return ToolResult{Content: err.Error(), IsError: true}, nil
	}

	// extract glob filter
	globPattern := ""
	if v, ok := input["glob"].(string); ok {
		globPattern = v
	}

	// extract output_mode (default: files_with_matches)
	outputMode := "files_with_matches"
	if v, ok := input["output_mode"].(string); ok {
		outputMode = v
	}

	// extract context lines (-C, -A, -B)
	contextBefore := 0
	contextAfter := 0
	if v, ok := input["-C"].(float64); ok && v > 0 {
		contextBefore = int(v)
		contextAfter = int(v)
	}
	if v, ok := input["-A"].(float64); ok && v > 0 {
		contextAfter = int(v)
	}
	if v, ok := input["-B"].(float64); ok && v > 0 {
		contextBefore = int(v)
	}

	// extract head_limit
	headLimit := 0
	if v, ok := input["head_limit"].(float64); ok && v > 0 {
		headLimit = int(v)
	}

	var results []string
	var totalCount int

	// search function for a single file
	searchFile := func(filePath string) error {
		// check head_limit early
		if headLimit > 0 && len(results) >= headLimit {
			return filepath.SkipAll
		}

		data, err := os.ReadFile(filePath)
		if err != nil {
			return nil // skip unreadable files
		}

		// skip binary files (files with null bytes in first 8000 chars)
		checkLen := len(data)
		if checkLen > 8000 {
			checkLen = 8000
		}
		if bytes.Contains(data[:checkLen], []byte{0}) {
			return nil
		}

		lines := strings.Split(string(data), "\n")
		var matches []int

		for i, line := range lines {
			if re.MatchString(line) {
				matches = append(matches, i)
			}
		}

		if len(matches) == 0 {
			return nil
		}

		switch outputMode {
		case "files_with_matches":
			results = append(results, filePath)

		case "count":
			totalCount += len(matches)

		case "content":
			// collect lines with context
			includedLines := make(map[int]bool)
			for _, matchIdx := range matches {
				start := matchIdx - contextBefore
				if start < 0 {
					start = 0
				}
				end := matchIdx + contextAfter + 1
				if end > len(lines) {
					end = len(lines)
				}
				for i := start; i < end; i++ {
					includedLines[i] = true
				}
			}

			// format output
			var sb strings.Builder
			sb.WriteString(filePath)
			sb.WriteString(":\n")
			for i := 0; i < len(lines); i++ {
				if includedLines[i] {
					sb.WriteString(fmt.Sprintf("%d\t%s\n", i+1, lines[i]))
				}
			}
			results = append(results, strings.TrimSuffix(sb.String(), "\n"))
		}

		return nil
	}

	if info.IsDir() {
		err = filepath.WalkDir(searchPath, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return nil // skip errors
			}
			if d.IsDir() {
				return nil
			}

			// apply glob filter
			if globPattern != "" {
				matched, err := matchGlob(globPattern, searchPath, path)
				if err != nil || !matched {
					return nil
				}
			}

			return searchFile(path)
		})
	} else {
		err = searchFile(searchPath)
	}

	if err != nil && err != filepath.SkipAll {
		return ToolResult{Content: err.Error(), IsError: true}, nil
	}

	// format final output
	var output string
	switch outputMode {
	case "count":
		output = fmt.Sprintf("%d", totalCount)
	default:
		if headLimit > 0 && len(results) > headLimit {
			results = results[:headLimit]
		}
		output = strings.Join(results, "\n")
	}

	return ToolResult{Content: output}, nil
}

// matchGlob checks if path matches glob pattern relative to base
func matchGlob(pattern, base, path string) (bool, error) {
	relPath, err := filepath.Rel(base, path)
	if err != nil {
		return false, err
	}

	// handle ** pattern for recursive matching
	if strings.HasPrefix(pattern, "**/") {
		// match against filename and full relative path
		filename := filepath.Base(path)
		simplePattern := strings.TrimPrefix(pattern, "**/")

		matched, err := filepath.Match(simplePattern, filename)
		if err != nil {
			return false, err
		}
		if matched {
			return true, nil
		}

		// try matching each component
		parts := strings.Split(relPath, string(filepath.Separator))
		for i := range parts {
			subPath := filepath.Join(parts[i:]...)
			matched, err := filepath.Match(simplePattern, subPath)
			if err == nil && matched {
				return true, nil
			}
		}
		return false, nil
	}

	// simple glob match
	return filepath.Match(pattern, filepath.Base(path))
}
