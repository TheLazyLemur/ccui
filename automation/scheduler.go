package automation

import (
	"context"
	"log/slog"
	"sync"

	"github.com/robfig/cron/v3"
)

// Scheduler manages cron jobs for automations
type Scheduler struct {
	cron   *cron.Cron
	store  *Store
	engine *Engine
	ctx    context.Context

	mu      sync.Mutex
	entries map[string]cron.EntryID // automationID -> cron entry
}

// NewScheduler creates a scheduler
func NewScheduler(ctx context.Context, store *Store, engine *Engine) *Scheduler {
	return &Scheduler{
		cron:    cron.New(),
		store:   store,
		engine:  engine,
		ctx:     ctx,
		entries: make(map[string]cron.EntryID),
	}
}

// Start begins the cron scheduler and syncs initial entries
func (s *Scheduler) Start() {
	s.Sync()
	s.cron.Start()
}

// Stop halts the cron scheduler
func (s *Scheduler) Stop() {
	s.cron.Stop()
}

// Sync reconciles cron entries with the automation store
func (s *Scheduler) Sync() {
	s.mu.Lock()
	defer s.mu.Unlock()

	automations := s.store.List()

	// build set of wanted automations
	wanted := make(map[string]Automation)
	for _, a := range automations {
		if a.Enabled && a.Schedule != "" {
			wanted[a.ID] = a
		}
	}

	// remove entries no longer wanted
	for id, entryID := range s.entries {
		if _, ok := wanted[id]; !ok {
			s.cron.Remove(entryID)
			delete(s.entries, id)
		}
	}

	// add entries not yet scheduled
	for id, auto := range wanted {
		if _, exists := s.entries[id]; exists {
			continue
		}
		a := auto // capture for closure
		entryID, err := s.cron.AddFunc(a.Schedule, func() {
			slog.Info("automation triggered", "id", a.ID, "name", a.Name)
			if _, err := s.engine.Execute(s.ctx, a); err != nil {
				slog.Error("automation run failed", "id", a.ID, "error", err)
			}
		})
		if err != nil {
			slog.Error("invalid cron schedule", "id", a.ID, "schedule", a.Schedule, "error", err)
			continue
		}
		s.entries[id] = entryID
	}
}
