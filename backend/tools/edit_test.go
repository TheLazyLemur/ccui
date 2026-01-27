package tools

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEditTool_Name(t *testing.T) {
	a := assert.New(t)
	tool := NewEditTool()
	a.Equal("Edit", tool.Name())
}

func TestEditTool_Execute_BasicEdit(t *testing.T) {
	a := assert.New(t)
	r := require.New(t)

	// given - file with content
	dir := t.TempDir()
	path := filepath.Join(dir, "test.txt")
	original := "hello world\n"
	r.NoError(os.WriteFile(path, []byte(original), 0644))

	tool := NewEditTool()

	// when - replace "world" with "gopher"
	result, err := tool.Execute(context.Background(), map[string]any{
		"file_path":  path,
		"old_string": "world",
		"new_string": "gopher",
	})

	// then - replacement made
	r.NoError(err)
	a.False(result.IsError)
	a.Equal(path, result.FilePath)
	a.Equal(original, result.OldContent)
	a.Equal("hello gopher\n", result.NewContent)

	// verify file on disk
	data, err := os.ReadFile(path)
	r.NoError(err)
	a.Equal("hello gopher\n", string(data))
}

func TestEditTool_Execute_MultilineEdit(t *testing.T) {
	a := assert.New(t)
	r := require.New(t)

	// given - multiline file
	dir := t.TempDir()
	path := filepath.Join(dir, "test.txt")
	original := "line1\nline2\nline3\n"
	r.NoError(os.WriteFile(path, []byte(original), 0644))

	tool := NewEditTool()

	// when - replace multiple lines
	result, err := tool.Execute(context.Background(), map[string]any{
		"file_path":  path,
		"old_string": "line2\nline3",
		"new_string": "replaced",
	})

	// then - multiline replacement works
	r.NoError(err)
	a.False(result.IsError)
	a.Equal("line1\nreplaced\n", result.NewContent)
}

func TestEditTool_Execute_NonUniqueString_Fails(t *testing.T) {
	a := assert.New(t)
	r := require.New(t)

	// given - file with duplicate occurrences
	dir := t.TempDir()
	path := filepath.Join(dir, "test.txt")
	content := "foo bar foo baz foo\n"
	r.NoError(os.WriteFile(path, []byte(content), 0644))

	tool := NewEditTool()

	// when - try to replace non-unique string without replace_all
	result, err := tool.Execute(context.Background(), map[string]any{
		"file_path":  path,
		"old_string": "foo",
		"new_string": "qux",
	})

	// then - returns error about uniqueness
	r.NoError(err)
	a.True(result.IsError)
	a.Contains(result.Content, "3")  // should mention count
	a.Contains(result.Content, "unique") // should mention uniqueness
}

func TestEditTool_Execute_ReplaceAll(t *testing.T) {
	a := assert.New(t)
	r := require.New(t)

	// given - file with multiple occurrences
	dir := t.TempDir()
	path := filepath.Join(dir, "test.txt")
	content := "foo bar foo baz foo\n"
	r.NoError(os.WriteFile(path, []byte(content), 0644))

	tool := NewEditTool()

	// when - replace all with replace_all=true
	result, err := tool.Execute(context.Background(), map[string]any{
		"file_path":   path,
		"old_string":  "foo",
		"new_string":  "qux",
		"replace_all": true,
	})

	// then - all occurrences replaced
	r.NoError(err)
	a.False(result.IsError)
	a.Equal("qux bar qux baz qux\n", result.NewContent)

	data, err := os.ReadFile(path)
	r.NoError(err)
	a.Equal("qux bar qux baz qux\n", string(data))
}

func TestEditTool_Execute_StringNotFound(t *testing.T) {
	a := assert.New(t)
	r := require.New(t)

	// given - file without target string
	dir := t.TempDir()
	path := filepath.Join(dir, "test.txt")
	content := "hello world\n"
	r.NoError(os.WriteFile(path, []byte(content), 0644))

	tool := NewEditTool()

	// when - try to replace non-existent string
	result, err := tool.Execute(context.Background(), map[string]any{
		"file_path":  path,
		"old_string": "missing",
		"new_string": "replacement",
	})

	// then - returns error
	r.NoError(err)
	a.True(result.IsError)
	a.Contains(result.Content, "not found")
}

func TestEditTool_Execute_FileNotFound(t *testing.T) {
	a := assert.New(t)
	r := require.New(t)

	tool := NewEditTool()

	// when - edit non-existent file
	result, err := tool.Execute(context.Background(), map[string]any{
		"file_path":  "/nonexistent/path/file.txt",
		"old_string": "old",
		"new_string": "new",
	})

	// then - returns error
	r.NoError(err)
	a.True(result.IsError)
}

func TestEditTool_Execute_MissingFilePath(t *testing.T) {
	a := assert.New(t)
	r := require.New(t)

	tool := NewEditTool()

	// when - execute without file_path
	result, err := tool.Execute(context.Background(), map[string]any{
		"old_string": "old",
		"new_string": "new",
	})

	// then - returns error
	r.NoError(err)
	a.True(result.IsError)
	a.Contains(result.Content, "file_path")
}

func TestEditTool_Execute_MissingOldString(t *testing.T) {
	a := assert.New(t)
	r := require.New(t)

	tool := NewEditTool()

	// when - execute without old_string
	result, err := tool.Execute(context.Background(), map[string]any{
		"file_path":  "/tmp/test.txt",
		"new_string": "new",
	})

	// then - returns error
	r.NoError(err)
	a.True(result.IsError)
	a.Contains(result.Content, "old_string")
}

func TestEditTool_Execute_MissingNewString(t *testing.T) {
	a := assert.New(t)
	r := require.New(t)

	tool := NewEditTool()

	// when - execute without new_string
	result, err := tool.Execute(context.Background(), map[string]any{
		"file_path":  "/tmp/test.txt",
		"old_string": "old",
	})

	// then - returns error
	r.NoError(err)
	a.True(result.IsError)
	a.Contains(result.Content, "new_string")
}

func TestEditTool_Execute_SameOldAndNew(t *testing.T) {
	a := assert.New(t)
	r := require.New(t)

	// given - file with content
	dir := t.TempDir()
	path := filepath.Join(dir, "test.txt")
	content := "hello world\n"
	r.NoError(os.WriteFile(path, []byte(content), 0644))

	tool := NewEditTool()

	// when - old_string equals new_string
	result, err := tool.Execute(context.Background(), map[string]any{
		"file_path":  path,
		"old_string": "world",
		"new_string": "world",
	})

	// then - returns error (no-op edit)
	r.NoError(err)
	a.True(result.IsError)
	a.Contains(result.Content, "same")
}

func TestEditTool_Execute_EmptyOldString(t *testing.T) {
	a := assert.New(t)
	r := require.New(t)

	tool := NewEditTool()

	// when - empty old_string
	result, err := tool.Execute(context.Background(), map[string]any{
		"file_path":  "/tmp/test.txt",
		"old_string": "",
		"new_string": "new",
	})

	// then - returns error
	r.NoError(err)
	a.True(result.IsError)
	a.Contains(result.Content, "old_string")
}

func TestEditTool_Execute_ReturnsHunks(t *testing.T) {
	a := assert.New(t)
	r := require.New(t)

	// given - multiline file
	dir := t.TempDir()
	path := filepath.Join(dir, "test.txt")
	original := "line1\nline2\nline3\nline4\nline5\n"
	r.NoError(os.WriteFile(path, []byte(original), 0644))

	tool := NewEditTool()

	// when - edit middle line
	result, err := tool.Execute(context.Background(), map[string]any{
		"file_path":  path,
		"old_string": "line3",
		"new_string": "modified",
	})

	// then - hunks populated
	r.NoError(err)
	a.False(result.IsError)
	a.NotEmpty(result.Hunks, "should return diff hunks")
}

func TestEditTool_Execute_DeleteContent(t *testing.T) {
	a := assert.New(t)
	r := require.New(t)

	// given - file with content to delete
	dir := t.TempDir()
	path := filepath.Join(dir, "test.txt")
	content := "keep this DELETE_ME keep this too\n"
	r.NoError(os.WriteFile(path, []byte(content), 0644))

	tool := NewEditTool()

	// when - replace with empty string (delete)
	result, err := tool.Execute(context.Background(), map[string]any{
		"file_path":  path,
		"old_string": "DELETE_ME ",
		"new_string": "",
	})

	// then - content deleted
	r.NoError(err)
	a.False(result.IsError)
	a.Equal("keep this keep this too\n", result.NewContent)
}
