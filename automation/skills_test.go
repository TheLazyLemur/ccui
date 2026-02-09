package automation

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupSkillsDir(t *testing.T) string {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "skills")
	require.NoError(t, os.MkdirAll(dir, 0755))
	return dir
}

func TestSkillStore_List(t *testing.T) {
	a := assert.New(t)
	r := require.New(t)
	dir := setupSkillsDir(t)

	r.NoError(os.WriteFile(filepath.Join(dir, "code-review.md"), []byte("Review code for bugs"), 0644))
	r.NoError(os.WriteFile(filepath.Join(dir, "security.md"), []byte("Check for vulnerabilities"), 0644))

	store := NewSkillStore(dir)
	skills := store.List()
	a.Len(skills, 2)
}

func TestSkillStore_Get(t *testing.T) {
	a := assert.New(t)
	r := require.New(t)
	dir := setupSkillsDir(t)

	r.NoError(os.WriteFile(filepath.Join(dir, "code-review.md"), []byte("Review code for bugs"), 0644))

	store := NewSkillStore(dir)
	content, err := store.Get("code-review")
	r.NoError(err)
	a.Equal("Review code for bugs", content)
}

func TestSkillStore_GetNotFound(t *testing.T) {
	dir := setupSkillsDir(t)
	store := NewSkillStore(dir)
	_, err := store.Get("nonexistent")
	assert.Error(t, err)
}

func TestSkillStore_ResolvePrompt(t *testing.T) {
	a := assert.New(t)
	r := require.New(t)
	dir := setupSkillsDir(t)

	r.NoError(os.WriteFile(filepath.Join(dir, "code-review.md"), []byte("Review all recent changes for bugs."), 0644))

	store := NewSkillStore(dir)
	resolved, err := store.ResolvePrompt("$code-review Focus on the auth module.")
	r.NoError(err)
	a.Equal("Review all recent changes for bugs.\nFocus on the auth module.", resolved)
}

func TestSkillStore_ResolvePrompt_NoSkillRef(t *testing.T) {
	a := assert.New(t)
	dir := setupSkillsDir(t)

	store := NewSkillStore(dir)
	resolved, err := store.ResolvePrompt("Just a normal prompt")
	a.NoError(err)
	a.Equal("Just a normal prompt", resolved)
}

func TestSkillStore_ResolvePrompt_UnknownSkill(t *testing.T) {
	dir := setupSkillsDir(t)
	store := NewSkillStore(dir)
	_, err := store.ResolvePrompt("$unknown-skill do something")
	assert.Error(t, err)
}

func TestSkillStore_ListEmpty(t *testing.T) {
	dir := setupSkillsDir(t)
	store := NewSkillStore(dir)
	assert.Empty(t, store.List())
}

func TestSkillStore_ListNonexistentDir(t *testing.T) {
	store := NewSkillStore("/nonexistent/path")
	assert.Empty(t, store.List())
}
