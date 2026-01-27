package tools

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGrepTool_Name(t *testing.T) {
	a := assert.New(t)
	tool := NewGrepTool()
	a.Equal("Grep", tool.Name())
}

func TestGrepTool_Execute_FilesWithMatches(t *testing.T) {
	a := assert.New(t)
	r := require.New(t)

	// given - temp dir with files containing patterns
	dir := t.TempDir()
	r.NoError(os.WriteFile(filepath.Join(dir, "foo.go"), []byte("func main() {}\nfunc hello() {}"), 0644))
	r.NoError(os.WriteFile(filepath.Join(dir, "bar.go"), []byte("func world() {}"), 0644))
	r.NoError(os.WriteFile(filepath.Join(dir, "baz.txt"), []byte("no match here"), 0644))

	tool := NewGrepTool()

	// when - search for "func" with default output_mode (files_with_matches)
	result, err := tool.Execute(context.Background(), map[string]any{
		"pattern": "func",
		"path":    dir,
	})

	// then - returns list of matching files
	r.NoError(err)
	a.False(result.IsError)
	a.Contains(result.Content, "foo.go")
	a.Contains(result.Content, "bar.go")
	a.NotContains(result.Content, "baz.txt")
}

func TestGrepTool_Execute_ContentMode(t *testing.T) {
	a := assert.New(t)
	r := require.New(t)

	// given - temp dir with files
	dir := t.TempDir()
	r.NoError(os.WriteFile(filepath.Join(dir, "test.go"), []byte("line one\nfunc hello() {}\nline three"), 0644))

	tool := NewGrepTool()

	// when - search with output_mode=content
	result, err := tool.Execute(context.Background(), map[string]any{
		"pattern":     "func",
		"path":        dir,
		"output_mode": "content",
	})

	// then - returns matching lines with line numbers
	r.NoError(err)
	a.False(result.IsError)
	a.Contains(result.Content, "func hello()")
	a.Contains(result.Content, "2") // line number
}

func TestGrepTool_Execute_GlobFilter(t *testing.T) {
	a := assert.New(t)
	r := require.New(t)

	// given - temp dir with mixed file types
	dir := t.TempDir()
	r.NoError(os.WriteFile(filepath.Join(dir, "foo.go"), []byte("func main()"), 0644))
	r.NoError(os.WriteFile(filepath.Join(dir, "bar.txt"), []byte("func test()"), 0644))

	tool := NewGrepTool()

	// when - search with glob filter for .go files only
	result, err := tool.Execute(context.Background(), map[string]any{
		"pattern": "func",
		"path":    dir,
		"glob":    "*.go",
	})

	// then - only matches .go files
	r.NoError(err)
	a.False(result.IsError)
	a.Contains(result.Content, "foo.go")
	a.NotContains(result.Content, "bar.txt")
}

func TestGrepTool_Execute_MissingPattern(t *testing.T) {
	a := assert.New(t)
	r := require.New(t)

	tool := NewGrepTool()

	// when - execute without pattern
	result, err := tool.Execute(context.Background(), map[string]any{
		"path": "/some/path",
	})

	// then - returns error result
	r.NoError(err)
	a.True(result.IsError)
	a.Contains(result.Content, "pattern")
}

func TestGrepTool_Execute_InvalidRegex(t *testing.T) {
	a := assert.New(t)
	r := require.New(t)

	tool := NewGrepTool()

	// when - execute with invalid regex
	result, err := tool.Execute(context.Background(), map[string]any{
		"pattern": "[invalid",
		"path":    "/some/path",
	})

	// then - returns error result
	r.NoError(err)
	a.True(result.IsError)
	a.Contains(result.Content, "invalid")
}

func TestGrepTool_Execute_PathNotFound(t *testing.T) {
	a := assert.New(t)
	r := require.New(t)

	tool := NewGrepTool()

	// when - search in nonexistent path
	result, err := tool.Execute(context.Background(), map[string]any{
		"pattern": "test",
		"path":    "/nonexistent/path/xyz",
	})

	// then - returns error result
	r.NoError(err)
	a.True(result.IsError)
	a.Contains(result.Content, "no such file")
}

func TestGrepTool_Execute_NoMatches(t *testing.T) {
	a := assert.New(t)
	r := require.New(t)

	// given - file with no matching content
	dir := t.TempDir()
	r.NoError(os.WriteFile(filepath.Join(dir, "test.txt"), []byte("hello world"), 0644))

	tool := NewGrepTool()

	// when - search for pattern not in file
	result, err := tool.Execute(context.Background(), map[string]any{
		"pattern": "xyz123",
		"path":    dir,
	})

	// then - succeeds with empty content
	r.NoError(err)
	a.False(result.IsError)
	a.Equal("", result.Content)
}

func TestGrepTool_Execute_Recursive(t *testing.T) {
	a := assert.New(t)
	r := require.New(t)

	// given - nested directory structure
	dir := t.TempDir()
	subDir := filepath.Join(dir, "sub")
	r.NoError(os.MkdirAll(subDir, 0755))
	r.NoError(os.WriteFile(filepath.Join(dir, "top.go"), []byte("func top()"), 0644))
	r.NoError(os.WriteFile(filepath.Join(subDir, "nested.go"), []byte("func nested()"), 0644))

	tool := NewGrepTool()

	// when - search recursively
	result, err := tool.Execute(context.Background(), map[string]any{
		"pattern": "func",
		"path":    dir,
	})

	// then - finds files in nested dirs
	r.NoError(err)
	a.False(result.IsError)
	a.Contains(result.Content, "top.go")
	a.Contains(result.Content, "nested.go")
}

func TestGrepTool_Execute_SingleFile(t *testing.T) {
	a := assert.New(t)
	r := require.New(t)

	// given - single file path
	dir := t.TempDir()
	filePath := filepath.Join(dir, "test.go")
	r.NoError(os.WriteFile(filePath, []byte("func main() {}\nfunc hello()"), 0644))

	tool := NewGrepTool()

	// when - search in single file
	result, err := tool.Execute(context.Background(), map[string]any{
		"pattern":     "func",
		"path":        filePath,
		"output_mode": "content",
	})

	// then - returns matching lines
	r.NoError(err)
	a.False(result.IsError)
	a.Contains(result.Content, "func main()")
	a.Contains(result.Content, "func hello()")
}

func TestGrepTool_Execute_CountMode(t *testing.T) {
	a := assert.New(t)
	r := require.New(t)

	// given - files with multiple matches
	dir := t.TempDir()
	r.NoError(os.WriteFile(filepath.Join(dir, "test.go"), []byte("func a()\nfunc b()\nfunc c()"), 0644))

	tool := NewGrepTool()

	// when - search with output_mode=count
	result, err := tool.Execute(context.Background(), map[string]any{
		"pattern":     "func",
		"path":        dir,
		"output_mode": "count",
	})

	// then - returns count of matches
	r.NoError(err)
	a.False(result.IsError)
	a.Contains(result.Content, "3")
}

func TestGrepTool_Execute_CaseInsensitive(t *testing.T) {
	a := assert.New(t)
	r := require.New(t)

	// given - file with mixed case
	dir := t.TempDir()
	r.NoError(os.WriteFile(filepath.Join(dir, "test.txt"), []byte("Hello\nhello\nHELLO"), 0644))

	tool := NewGrepTool()

	// when - case insensitive search
	result, err := tool.Execute(context.Background(), map[string]any{
		"pattern":     "hello",
		"path":        dir,
		"output_mode": "count",
		"-i":          true,
	})

	// then - matches all cases
	r.NoError(err)
	a.False(result.IsError)
	a.Contains(result.Content, "3")
}

func TestGrepTool_Execute_SkipsBinaryFiles(t *testing.T) {
	a := assert.New(t)
	r := require.New(t)

	// given - binary and text files
	dir := t.TempDir()
	r.NoError(os.WriteFile(filepath.Join(dir, "text.go"), []byte("func main()"), 0644))
	// binary file with null bytes
	binaryContent := []byte{0x00, 0x01, 0x02, 'f', 'u', 'n', 'c', 0x00}
	r.NoError(os.WriteFile(filepath.Join(dir, "binary.bin"), binaryContent, 0644))

	tool := NewGrepTool()

	// when - search
	result, err := tool.Execute(context.Background(), map[string]any{
		"pattern": "func",
		"path":    dir,
	})

	// then - only finds text file
	r.NoError(err)
	a.False(result.IsError)
	a.Contains(result.Content, "text.go")
	a.NotContains(result.Content, "binary.bin")
}

func TestGrepTool_Execute_ContextLines(t *testing.T) {
	a := assert.New(t)
	r := require.New(t)

	// given - file with context around match
	dir := t.TempDir()
	content := "line 1\nline 2\nmatch here\nline 4\nline 5"
	r.NoError(os.WriteFile(filepath.Join(dir, "test.txt"), []byte(content), 0644))

	tool := NewGrepTool()

	// when - search with context lines
	result, err := tool.Execute(context.Background(), map[string]any{
		"pattern":     "match",
		"path":        dir,
		"output_mode": "content",
		"-C":          float64(1),
	})

	// then - includes context lines
	r.NoError(err)
	a.False(result.IsError)
	a.Contains(result.Content, "line 2")
	a.Contains(result.Content, "match here")
	a.Contains(result.Content, "line 4")
}

func TestGrepTool_Execute_GlobWithSubdirs(t *testing.T) {
	a := assert.New(t)
	r := require.New(t)

	// given - nested dirs with glob pattern
	dir := t.TempDir()
	subDir := filepath.Join(dir, "sub")
	r.NoError(os.MkdirAll(subDir, 0755))
	r.NoError(os.WriteFile(filepath.Join(dir, "top.go"), []byte("func top()"), 0644))
	r.NoError(os.WriteFile(filepath.Join(subDir, "nested.go"), []byte("func nested()"), 0644))
	r.NoError(os.WriteFile(filepath.Join(subDir, "nested.txt"), []byte("func text()"), 0644))

	tool := NewGrepTool()

	// when - search with **/*.go glob
	result, err := tool.Execute(context.Background(), map[string]any{
		"pattern": "func",
		"path":    dir,
		"glob":    "**/*.go",
	})

	// then - matches .go in all subdirs
	r.NoError(err)
	a.False(result.IsError)
	a.Contains(result.Content, "top.go")
	a.Contains(result.Content, "nested.go")
	a.NotContains(result.Content, "nested.txt")
}

func TestGrepTool_Execute_HeadLimit(t *testing.T) {
	a := assert.New(t)
	r := require.New(t)

	// given - many matching files
	dir := t.TempDir()
	for i := 0; i < 10; i++ {
		r.NoError(os.WriteFile(filepath.Join(dir, strings.Replace("file_X.go", "X", string(rune('a'+i)), 1)), []byte("func test()"), 0644))
	}

	tool := NewGrepTool()

	// when - search with head_limit
	result, err := tool.Execute(context.Background(), map[string]any{
		"pattern":    "func",
		"path":       dir,
		"head_limit": float64(3),
	})

	// then - returns only 3 files
	r.NoError(err)
	a.False(result.IsError)
	lines := strings.Split(strings.TrimSpace(result.Content), "\n")
	a.Equal(3, len(lines))
}
