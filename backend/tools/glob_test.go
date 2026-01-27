package tools

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGlobTool_Name(t *testing.T) {
	a := assert.New(t)
	tool := NewGlobTool()
	a.Equal("Glob", tool.Name())
}

func TestGlobTool_Execute_MissingPattern(t *testing.T) {
	a := assert.New(t)
	r := require.New(t)

	tool := NewGlobTool()

	// when - execute without pattern
	result, err := tool.Execute(context.Background(), map[string]any{})

	// then - returns error result
	r.NoError(err)
	a.True(result.IsError)
	a.Contains(result.Content, "pattern")
}

func TestGlobTool_Execute_SimplePattern(t *testing.T) {
	a := assert.New(t)
	r := require.New(t)

	// given - temp dir with files
	dir := t.TempDir()
	r.NoError(os.WriteFile(filepath.Join(dir, "file1.txt"), []byte("a"), 0644))
	r.NoError(os.WriteFile(filepath.Join(dir, "file2.txt"), []byte("b"), 0644))
	r.NoError(os.WriteFile(filepath.Join(dir, "file3.go"), []byte("c"), 0644))

	tool := NewGlobTool()

	// when - glob for *.txt
	result, err := tool.Execute(context.Background(), map[string]any{
		"pattern": "*.txt",
		"path":    dir,
	})

	// then - returns matching files
	r.NoError(err)
	a.False(result.IsError)
	a.Contains(result.Content, "file1.txt")
	a.Contains(result.Content, "file2.txt")
	a.NotContains(result.Content, "file3.go")
}

func TestGlobTool_Execute_DoubleStarPattern(t *testing.T) {
	a := assert.New(t)
	r := require.New(t)

	// given - nested dir structure
	dir := t.TempDir()
	subdir := filepath.Join(dir, "subdir")
	r.NoError(os.MkdirAll(subdir, 0755))
	r.NoError(os.WriteFile(filepath.Join(dir, "root.go"), []byte("a"), 0644))
	r.NoError(os.WriteFile(filepath.Join(subdir, "nested.go"), []byte("b"), 0644))
	r.NoError(os.WriteFile(filepath.Join(subdir, "other.txt"), []byte("c"), 0644))

	tool := NewGlobTool()

	// when - glob for **/*.go
	result, err := tool.Execute(context.Background(), map[string]any{
		"pattern": "**/*.go",
		"path":    dir,
	})

	// then - returns all .go files recursively
	r.NoError(err)
	a.False(result.IsError)
	a.Contains(result.Content, "root.go")
	a.Contains(result.Content, "nested.go")
	a.NotContains(result.Content, "other.txt")
}

func TestGlobTool_Execute_SortedByModTime(t *testing.T) {
	a := assert.New(t)
	r := require.New(t)

	// given - files with different mod times
	dir := t.TempDir()
	file1 := filepath.Join(dir, "oldest.txt")
	file2 := filepath.Join(dir, "middle.txt")
	file3 := filepath.Join(dir, "newest.txt")

	r.NoError(os.WriteFile(file1, []byte("a"), 0644))
	r.NoError(os.WriteFile(file2, []byte("b"), 0644))
	r.NoError(os.WriteFile(file3, []byte("c"), 0644))

	// set mod times: oldest < middle < newest
	now := time.Now()
	r.NoError(os.Chtimes(file1, now.Add(-2*time.Hour), now.Add(-2*time.Hour)))
	r.NoError(os.Chtimes(file2, now.Add(-1*time.Hour), now.Add(-1*time.Hour)))
	r.NoError(os.Chtimes(file3, now, now))

	tool := NewGlobTool()

	// when - glob all txt files
	result, err := tool.Execute(context.Background(), map[string]any{
		"pattern": "*.txt",
		"path":    dir,
	})

	// then - newest first (most recently modified)
	r.NoError(err)
	a.False(result.IsError)
	lines := splitLines(result.Content)
	r.Len(lines, 3)
	a.Contains(lines[0], "newest.txt")
	a.Contains(lines[1], "middle.txt")
	a.Contains(lines[2], "oldest.txt")
}

func TestGlobTool_Execute_NoMatches(t *testing.T) {
	a := assert.New(t)
	r := require.New(t)

	// given - dir with no matching files
	dir := t.TempDir()
	r.NoError(os.WriteFile(filepath.Join(dir, "file.txt"), []byte("a"), 0644))

	tool := NewGlobTool()

	// when - glob for non-matching pattern
	result, err := tool.Execute(context.Background(), map[string]any{
		"pattern": "*.go",
		"path":    dir,
	})

	// then - empty result, no error
	r.NoError(err)
	a.False(result.IsError)
	a.Equal("", result.Content)
}

func TestGlobTool_Execute_InvalidPath(t *testing.T) {
	a := assert.New(t)
	r := require.New(t)

	tool := NewGlobTool()

	// when - glob in nonexistent directory
	result, err := tool.Execute(context.Background(), map[string]any{
		"pattern": "*.txt",
		"path":    "/nonexistent/path",
	})

	// then - returns error result
	r.NoError(err)
	a.True(result.IsError)
	a.Contains(result.Content, "no such file")
}

func TestGlobTool_Execute_DefaultPath(t *testing.T) {
	a := assert.New(t)
	r := require.New(t)

	// given - we're in some directory with go files
	tool := NewGlobTool()

	// when - glob without path (uses cwd)
	result, err := tool.Execute(context.Background(), map[string]any{
		"pattern": "*.go",
	})

	// then - should succeed (we're in tools pkg dir)
	r.NoError(err)
	a.False(result.IsError)
	// should find this test file or glob.go
	a.Contains(result.Content, ".go")
}

func splitLines(s string) []string {
	if s == "" {
		return nil
	}
	var lines []string
	for _, line := range splitNonEmpty(s, '\n') {
		lines = append(lines, line)
	}
	return lines
}

func splitNonEmpty(s string, sep byte) []string {
	var result []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == sep {
			if i > start {
				result = append(result, s[start:i])
			}
			start = i + 1
		}
	}
	if start < len(s) {
		result = append(result, s[start:])
	}
	return result
}
