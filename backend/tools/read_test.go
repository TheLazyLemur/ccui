package tools

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReadTool_Name(t *testing.T) {
	a := assert.New(t)
	tool := NewReadTool()
	a.Equal("Read", tool.Name())
}

func TestReadTool_Execute_FullFile(t *testing.T) {
	a := assert.New(t)
	r := require.New(t)

	// given - temp file with known content
	dir := t.TempDir()
	path := filepath.Join(dir, "test.txt")
	content := "line one\nline two\nline three\n"
	r.NoError(os.WriteFile(path, []byte(content), 0644))

	tool := NewReadTool()

	// when - read entire file
	result, err := tool.Execute(context.Background(), map[string]any{
		"file_path": path,
	})

	// then - returns content with line numbers (cat -n format)
	r.NoError(err)
	a.False(result.IsError)
	a.Contains(result.Content, "1\tline one")
	a.Contains(result.Content, "2\tline two")
	a.Contains(result.Content, "3\tline three")
}

func TestReadTool_Execute_WithOffset(t *testing.T) {
	a := assert.New(t)
	r := require.New(t)

	// given - file with multiple lines
	dir := t.TempDir()
	path := filepath.Join(dir, "test.txt")
	content := "line one\nline two\nline three\nline four\nline five\n"
	r.NoError(os.WriteFile(path, []byte(content), 0644))

	tool := NewReadTool()

	// when - read with offset 3 (start at line 3)
	result, err := tool.Execute(context.Background(), map[string]any{
		"file_path": path,
		"offset":    float64(3), // JSON numbers come as float64
	})

	// then - returns from line 3 onward with original line numbers
	r.NoError(err)
	a.False(result.IsError)
	a.NotContains(result.Content, "line one")
	a.NotContains(result.Content, "line two")
	a.Contains(result.Content, "3\tline three")
	a.Contains(result.Content, "4\tline four")
	a.Contains(result.Content, "5\tline five")
}

func TestReadTool_Execute_WithLimit(t *testing.T) {
	a := assert.New(t)
	r := require.New(t)

	// given - file with multiple lines
	dir := t.TempDir()
	path := filepath.Join(dir, "test.txt")
	content := "line one\nline two\nline three\nline four\nline five\n"
	r.NoError(os.WriteFile(path, []byte(content), 0644))

	tool := NewReadTool()

	// when - read with limit 2
	result, err := tool.Execute(context.Background(), map[string]any{
		"file_path": path,
		"limit":     float64(2),
	})

	// then - returns only first 2 lines
	r.NoError(err)
	a.False(result.IsError)
	a.Contains(result.Content, "1\tline one")
	a.Contains(result.Content, "2\tline two")
	a.NotContains(result.Content, "line three")
}

func TestReadTool_Execute_WithOffsetAndLimit(t *testing.T) {
	a := assert.New(t)
	r := require.New(t)

	// given - file with multiple lines
	dir := t.TempDir()
	path := filepath.Join(dir, "test.txt")
	content := "line one\nline two\nline three\nline four\nline five\n"
	r.NoError(os.WriteFile(path, []byte(content), 0644))

	tool := NewReadTool()

	// when - read lines 2-3 (offset 2, limit 2)
	result, err := tool.Execute(context.Background(), map[string]any{
		"file_path": path,
		"offset":    float64(2),
		"limit":     float64(2),
	})

	// then - returns lines 2 and 3 only
	r.NoError(err)
	a.False(result.IsError)
	a.NotContains(result.Content, "line one")
	a.Contains(result.Content, "2\tline two")
	a.Contains(result.Content, "3\tline three")
	a.NotContains(result.Content, "line four")
}

func TestReadTool_Execute_MissingFilePath(t *testing.T) {
	a := assert.New(t)
	r := require.New(t)

	tool := NewReadTool()

	// when - execute without file_path
	result, err := tool.Execute(context.Background(), map[string]any{})

	// then - returns error result
	r.NoError(err) // execution succeeds, but result indicates error
	a.True(result.IsError)
	a.Contains(result.Content, "file_path")
}

func TestReadTool_Execute_FileNotFound(t *testing.T) {
	a := assert.New(t)
	r := require.New(t)

	tool := NewReadTool()

	// when - read nonexistent file
	result, err := tool.Execute(context.Background(), map[string]any{
		"file_path": "/nonexistent/path/to/file.txt",
	})

	// then - returns error result
	r.NoError(err)
	a.True(result.IsError)
	a.Contains(result.Content, "no such file")
}

func TestReadTool_Execute_EmptyFile(t *testing.T) {
	a := assert.New(t)
	r := require.New(t)

	// given - empty file
	dir := t.TempDir()
	path := filepath.Join(dir, "empty.txt")
	r.NoError(os.WriteFile(path, []byte(""), 0644))

	tool := NewReadTool()

	// when - read empty file
	result, err := tool.Execute(context.Background(), map[string]any{
		"file_path": path,
	})

	// then - succeeds with empty content
	r.NoError(err)
	a.False(result.IsError)
	a.Equal("", result.Content)
}

func TestReadTool_Execute_OffsetBeyondFile(t *testing.T) {
	a := assert.New(t)
	r := require.New(t)

	// given - small file
	dir := t.TempDir()
	path := filepath.Join(dir, "small.txt")
	content := "line one\nline two\n"
	r.NoError(os.WriteFile(path, []byte(content), 0644))

	tool := NewReadTool()

	// when - offset beyond file length
	result, err := tool.Execute(context.Background(), map[string]any{
		"file_path": path,
		"offset":    float64(100),
	})

	// then - returns empty content
	r.NoError(err)
	a.False(result.IsError)
	a.Equal("", result.Content)
}
