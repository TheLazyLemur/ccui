package automation

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestStore(t *testing.T) *Store {
	t.Helper()
	dir := t.TempDir()
	s, err := NewStore(dir)
	require.NoError(t, err)
	return s
}

func TestNewStore_CreatesDirectory(t *testing.T) {
	a := assert.New(t)
	dir := filepath.Join(t.TempDir(), "nested", "automations")

	s, err := NewStore(dir)
	a.NoError(err)
	a.NotNil(s)
	a.DirExists(dir)
}

func TestStore_ListEmpty(t *testing.T) {
	s := setupTestStore(t)
	assert.Empty(t, s.List())
}

func TestStore_CreateAndGet(t *testing.T) {
	a := assert.New(t)
	r := require.New(t)
	s := setupTestStore(t)

	// when
	created, err := s.Create(Automation{
		Name:            "daily review",
		Prompt:          "review recent changes",
		Schedule:        "0 9 * * *",
		ProjectDir:      "/tmp/project",
		BackendType:     "anthropic",
		PermissionLevel: PermReadOnly,
		Enabled:         true,
	})
	r.NoError(err)

	// then
	a.NotEmpty(created.ID)
	a.NotEmpty(created.CreatedAt)
	a.NotEmpty(created.UpdatedAt)
	a.Equal("daily review", created.Name)
	a.Equal(PermReadOnly, created.PermissionLevel)

	// get by id
	got, err := s.Get(created.ID)
	r.NoError(err)
	a.Equal(created.ID, got.ID)
	a.Equal("daily review", got.Name)
}

func TestStore_CreateMultipleAndList(t *testing.T) {
	a := assert.New(t)
	r := require.New(t)
	s := setupTestStore(t)

	_, err := s.Create(Automation{Name: "a1"})
	r.NoError(err)
	_, err = s.Create(Automation{Name: "a2"})
	r.NoError(err)

	list := s.List()
	a.Len(list, 2)
}

func TestStore_Update(t *testing.T) {
	a := assert.New(t)
	r := require.New(t)
	s := setupTestStore(t)

	created, err := s.Create(Automation{Name: "original", Enabled: false})
	r.NoError(err)

	// when
	created.Name = "updated"
	created.Enabled = true
	updated, err := s.Update(*created)
	r.NoError(err)

	// then
	a.Equal("updated", updated.Name)
	a.True(updated.Enabled)
	a.NotEmpty(updated.UpdatedAt)
	a.Equal(created.CreatedAt, updated.CreatedAt) // createdAt preserved
}

func TestStore_UpdateNotFound(t *testing.T) {
	s := setupTestStore(t)
	_, err := s.Update(Automation{ID: "nonexistent"})
	assert.Error(t, err)
}

func TestStore_Delete(t *testing.T) {
	a := assert.New(t)
	r := require.New(t)
	s := setupTestStore(t)

	created, err := s.Create(Automation{Name: "to-delete"})
	r.NoError(err)

	err = s.Delete(created.ID)
	r.NoError(err)

	a.Empty(s.List())

	_, err = s.Get(created.ID)
	a.Error(err)
}

func TestStore_DeleteNotFound(t *testing.T) {
	s := setupTestStore(t)
	err := s.Delete("nonexistent")
	assert.Error(t, err)
}

func TestStore_GetNotFound(t *testing.T) {
	s := setupTestStore(t)
	_, err := s.Get("nonexistent")
	assert.Error(t, err)
}

func TestStore_PersistsToDisk(t *testing.T) {
	a := assert.New(t)
	r := require.New(t)

	dir := t.TempDir()
	s1, err := NewStore(dir)
	r.NoError(err)

	created, err := s1.Create(Automation{Name: "persisted", Prompt: "test prompt"})
	r.NoError(err)

	// reload from disk
	s2, err := NewStore(dir)
	r.NoError(err)

	got, err := s2.Get(created.ID)
	r.NoError(err)
	a.Equal("persisted", got.Name)
	a.Equal("test prompt", got.Prompt)
}

func TestStore_PersistsAfterUpdate(t *testing.T) {
	a := assert.New(t)
	r := require.New(t)

	dir := t.TempDir()
	s1, err := NewStore(dir)
	r.NoError(err)

	created, err := s1.Create(Automation{Name: "v1"})
	r.NoError(err)
	created.Name = "v2"
	_, err = s1.Update(*created)
	r.NoError(err)

	// reload
	s2, err := NewStore(dir)
	r.NoError(err)
	got, err := s2.Get(created.ID)
	r.NoError(err)
	a.Equal("v2", got.Name)
}

func TestStore_PersistsAfterDelete(t *testing.T) {
	a := assert.New(t)
	r := require.New(t)

	dir := t.TempDir()
	s1, err := NewStore(dir)
	r.NoError(err)

	created, err := s1.Create(Automation{Name: "gone"})
	r.NoError(err)
	r.NoError(s1.Delete(created.ID))

	// reload
	s2, err := NewStore(dir)
	r.NoError(err)
	a.Empty(s2.List())
}

func TestStore_IndexFileCreated(t *testing.T) {
	dir := t.TempDir()
	_, err := NewStore(dir)
	require.NoError(t, err)

	_, err = os.Stat(filepath.Join(dir, "index.json"))
	assert.NoError(t, err)
}
