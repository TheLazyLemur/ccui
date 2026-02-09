package automation

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestScheduler(t *testing.T) (*Scheduler, *Store) {
	t.Helper()
	dir := t.TempDir()
	store, err := NewStore(dir)
	require.NoError(t, err)

	runDir := filepath.Join(dir, "runs")
	runStore, err := NewRunStore(runDir)
	require.NoError(t, err)

	// no-op factory since we're testing scheduling not execution
	engine := NewEngine(nil, runStore, nil, nil)
	scheduler := NewScheduler(context.Background(), store, engine)
	return scheduler, store
}

func TestScheduler_SyncAddsEntries(t *testing.T) {
	a := assert.New(t)
	scheduler, store := setupTestScheduler(t)
	defer scheduler.Stop()

	_, err := store.Create(Automation{
		Name:     "daily review",
		Schedule: "0 9 * * *",
		Enabled:  true,
	})
	require.NoError(t, err)

	scheduler.Sync()

	scheduler.mu.Lock()
	a.Len(scheduler.entries, 1)
	scheduler.mu.Unlock()
}

func TestScheduler_SyncRemovesDisabled(t *testing.T) {
	a := assert.New(t)
	scheduler, store := setupTestScheduler(t)
	defer scheduler.Stop()

	created, err := store.Create(Automation{
		Name:     "daily review",
		Schedule: "0 9 * * *",
		Enabled:  true,
	})
	require.NoError(t, err)

	scheduler.Sync()
	scheduler.mu.Lock()
	a.Len(scheduler.entries, 1)
	scheduler.mu.Unlock()

	// disable
	created.Enabled = false
	_, err = store.Update(*created)
	require.NoError(t, err)

	scheduler.Sync()
	scheduler.mu.Lock()
	a.Len(scheduler.entries, 0)
	scheduler.mu.Unlock()
}

func TestScheduler_SyncIgnoresEmptySchedule(t *testing.T) {
	a := assert.New(t)
	scheduler, store := setupTestScheduler(t)
	defer scheduler.Stop()

	_, err := store.Create(Automation{
		Name:    "manual only",
		Enabled: true,
		// no schedule
	})
	require.NoError(t, err)

	scheduler.Sync()

	scheduler.mu.Lock()
	a.Len(scheduler.entries, 0)
	scheduler.mu.Unlock()
}

func TestScheduler_SyncSkipsInvalidCron(t *testing.T) {
	a := assert.New(t)
	scheduler, store := setupTestScheduler(t)
	defer scheduler.Stop()

	_, err := store.Create(Automation{
		Name:     "bad schedule",
		Schedule: "not a cron expression",
		Enabled:  true,
	})
	require.NoError(t, err)

	scheduler.Sync()

	scheduler.mu.Lock()
	a.Len(scheduler.entries, 0)
	scheduler.mu.Unlock()
}

func TestScheduler_SyncHandlesDelete(t *testing.T) {
	a := assert.New(t)
	scheduler, store := setupTestScheduler(t)
	defer scheduler.Stop()

	created, err := store.Create(Automation{
		Name:     "to delete",
		Schedule: "0 9 * * *",
		Enabled:  true,
	})
	require.NoError(t, err)

	scheduler.Sync()
	scheduler.mu.Lock()
	a.Len(scheduler.entries, 1)
	scheduler.mu.Unlock()

	require.NoError(t, store.Delete(created.ID))

	scheduler.Sync()
	scheduler.mu.Lock()
	a.Len(scheduler.entries, 0)
	scheduler.mu.Unlock()
}

func TestScheduler_StartAndStop(t *testing.T) {
	scheduler, _ := setupTestScheduler(t)

	// should not panic
	scheduler.Start()
	scheduler.Stop()
}
