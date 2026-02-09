package automation

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestRunStore(t *testing.T) *RunStore {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "runs")
	s, err := NewRunStore(dir)
	require.NoError(t, err)
	return s
}

func TestRunStore_CreateAndGet(t *testing.T) {
	a := assert.New(t)
	r := require.New(t)
	s := setupTestRunStore(t)

	run, err := s.Create("auto-1", Run{
		Status: RunStatusRunning,
		Output: "",
	})
	r.NoError(err)

	a.NotEmpty(run.ID)
	a.Equal("auto-1", run.AutomationID)
	a.Equal(RunStatusRunning, run.Status)
	a.NotEmpty(run.StartedAt)

	got, err := s.Get("auto-1", run.ID)
	r.NoError(err)
	a.Equal(run.ID, got.ID)
}

func TestRunStore_Update(t *testing.T) {
	a := assert.New(t)
	r := require.New(t)
	s := setupTestRunStore(t)

	run, err := s.Create("auto-1", Run{Status: RunStatusRunning})
	r.NoError(err)

	run.Status = RunStatusCompleted
	run.Output = "found 3 issues"
	run.HasFindings = true
	err = s.Update(*run)
	r.NoError(err)

	got, err := s.Get("auto-1", run.ID)
	r.NoError(err)
	a.Equal(RunStatusCompleted, got.Status)
	a.Equal("found 3 issues", got.Output)
	a.True(got.HasFindings)
}

func TestRunStore_ListByAutomation(t *testing.T) {
	a := assert.New(t)
	r := require.New(t)
	s := setupTestRunStore(t)

	_, err := s.Create("auto-1", Run{Status: RunStatusCompleted})
	r.NoError(err)
	_, err = s.Create("auto-1", Run{Status: RunStatusCompleted})
	r.NoError(err)
	_, err = s.Create("auto-2", Run{Status: RunStatusCompleted})
	r.NoError(err)

	runs, err := s.ListByAutomation("auto-1")
	r.NoError(err)
	a.Len(runs, 2)

	runs2, err := s.ListByAutomation("auto-2")
	r.NoError(err)
	a.Len(runs2, 1)
}

func TestRunStore_ListByAutomation_Empty(t *testing.T) {
	a := assert.New(t)
	r := require.New(t)
	s := setupTestRunStore(t)

	runs, err := s.ListByAutomation("nonexistent")
	r.NoError(err)
	a.Empty(runs)
}

func TestRunStore_GetNotFound(t *testing.T) {
	s := setupTestRunStore(t)
	_, err := s.Get("auto-1", "nonexistent")
	assert.Error(t, err)
}

func TestRunStore_UnreadWithFindings(t *testing.T) {
	a := assert.New(t)
	r := require.New(t)
	s := setupTestRunStore(t)

	// create runs: one with findings unread, one with findings read, one without findings
	run1, err := s.Create("auto-1", Run{Status: RunStatusCompleted})
	r.NoError(err)
	run1.HasFindings = true
	r.NoError(s.Update(*run1))

	run2, err := s.Create("auto-1", Run{Status: RunStatusCompleted})
	r.NoError(err)
	run2.HasFindings = true
	run2.Read = true
	r.NoError(s.Update(*run2))

	run3, err := s.Create("auto-2", Run{Status: RunStatusCompleted})
	r.NoError(err)
	run3.HasFindings = false
	r.NoError(s.Update(*run3))

	triage, err := s.UnreadWithFindings()
	r.NoError(err)
	a.Len(triage, 1)
	a.Equal(run1.ID, triage[0].ID)
}

func TestRunStore_PersistsToDisk(t *testing.T) {
	a := assert.New(t)
	r := require.New(t)

	dir := filepath.Join(t.TempDir(), "runs")
	s1, err := NewRunStore(dir)
	r.NoError(err)

	run, err := s1.Create("auto-1", Run{Status: RunStatusCompleted, Output: "persisted"})
	r.NoError(err)

	// reload
	s2, err := NewRunStore(dir)
	r.NoError(err)

	got, err := s2.Get("auto-1", run.ID)
	r.NoError(err)
	a.Equal("persisted", got.Output)
}
