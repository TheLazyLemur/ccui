package tools

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWriteTool_Name(t *testing.T) {
	a := assert.New(t)
	tool := NewWriteTool()
	a.Equal("Write", tool.Name())
}

func TestWriteTool_Execute_BasicWrite(t *testing.T) {
	a := assert.New(t)
	r := require.New(t)

	// given - temp directory
	dir := t.TempDir()
	path := filepath.Join(dir, "test.txt")
	content := "hello world\n"

	tool := NewWriteTool()

	// when - write file
	result, err := tool.Execute(context.Background(), map[string]any{
		"file_path": path,
		"content":   content,
	})

	// then - file created with correct content
	r.NoError(err)
	a.False(result.IsError)
	a.Contains(result.Content, "12 bytes")
	a.Equal(path, result.FilePath)
	a.Equal(content, result.NewContent)

	// verify file on disk
	data, err := os.ReadFile(path)
	r.NoError(err)
	a.Equal(content, string(data))
}

func TestWriteTool_Execute_CreatesParentDirs(t *testing.T) {
	a := assert.New(t)
	r := require.New(t)

	// given - nested path that doesn't exist
	dir := t.TempDir()
	path := filepath.Join(dir, "nested", "deep", "file.txt")
	content := "nested content"

	tool := NewWriteTool()

	// when - write to nested path
	result, err := tool.Execute(context.Background(), map[string]any{
		"file_path": path,
		"content":   content,
	})

	// then - directories created and file written
	r.NoError(err)
	a.False(result.IsError)

	data, err := os.ReadFile(path)
	r.NoError(err)
	a.Equal(content, string(data))
}

func TestWriteTool_Execute_OverwritesExisting(t *testing.T) {
	a := assert.New(t)
	r := require.New(t)

	// given - existing file
	dir := t.TempDir()
	path := filepath.Join(dir, "existing.txt")
	r.NoError(os.WriteFile(path, []byte("old content"), 0644))

	newContent := "new content"
	tool := NewWriteTool()

	// when - write to existing file
	result, err := tool.Execute(context.Background(), map[string]any{
		"file_path": path,
		"content":   newContent,
	})

	// then - file overwritten
	r.NoError(err)
	a.False(result.IsError)
	a.Equal(newContent, result.NewContent)

	data, err := os.ReadFile(path)
	r.NoError(err)
	a.Equal(newContent, string(data))
}

func TestWriteTool_Execute_EmptyContent(t *testing.T) {
	a := assert.New(t)
	r := require.New(t)

	// given - temp directory
	dir := t.TempDir()
	path := filepath.Join(dir, "empty.txt")

	tool := NewWriteTool()

	// when - write empty content
	result, err := tool.Execute(context.Background(), map[string]any{
		"file_path": path,
		"content":   "",
	})

	// then - empty file created
	r.NoError(err)
	a.False(result.IsError)
	a.Contains(result.Content, "0 bytes")

	data, err := os.ReadFile(path)
	r.NoError(err)
	a.Equal("", string(data))
}

func TestWriteTool_Execute_MissingFilePath(t *testing.T) {
	a := assert.New(t)
	r := require.New(t)

	tool := NewWriteTool()

	// when - execute without file_path
	result, err := tool.Execute(context.Background(), map[string]any{
		"content": "hello",
	})

	// then - returns error result
	r.NoError(err)
	a.True(result.IsError)
	a.Contains(result.Content, "file_path")
}

func TestWriteTool_Execute_MissingContent(t *testing.T) {
	a := assert.New(t)
	r := require.New(t)

	tool := NewWriteTool()

	// when - execute without content
	result, err := tool.Execute(context.Background(), map[string]any{
		"file_path": "/tmp/test.txt",
	})

	// then - returns error result
	r.NoError(err)
	a.True(result.IsError)
	a.Contains(result.Content, "content")
}

func TestWriteTool_Execute_InvalidPath(t *testing.T) {
	a := assert.New(t)
	r := require.New(t)

	tool := NewWriteTool()

	// when - write to path we can't create (root of filesystem)
	result, err := tool.Execute(context.Background(), map[string]any{
		"file_path": "/nonexistent_root_dir_test_12345/file.txt",
		"content":   "test",
	})

	// then - returns error result
	r.NoError(err)
	a.True(result.IsError)
	a.Contains(result.Content, "failed")
}
