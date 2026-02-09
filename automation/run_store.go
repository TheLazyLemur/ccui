package automation

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

// RunStore persists runs as individual JSON files per automation
type RunStore struct {
	dir string
	mu  sync.RWMutex
}

// NewRunStore creates or loads a run store
func NewRunStore(dir string) (*RunStore, error) {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("create run store dir: %w", err)
	}
	return &RunStore{dir: dir}, nil
}

func (s *RunStore) automationDir(automationID string) string {
	return filepath.Join(s.dir, automationID)
}

func (s *RunStore) runPath(automationID, runID string) string {
	return filepath.Join(s.automationDir(automationID), runID+".json")
}

// Create adds a new run, generating ID and timestamp
func (s *RunStore) Create(automationID string, r Run) (*Run, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	r.ID = uuid.New().String()
	r.AutomationID = automationID
	r.StartedAt = time.Now().UTC().Format(time.RFC3339)

	dir := s.automationDir(automationID)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("create automation run dir: %w", err)
	}

	if err := s.writeRun(r); err != nil {
		return nil, err
	}
	return &r, nil
}

// Get returns a run by automation and run ID
func (s *RunStore) Get(automationID, runID string) (*Run, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.readRun(automationID, runID)
}

// Update persists changes to a run
func (s *RunStore) Update(r Run) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.writeRun(r)
}

// ListByAutomation returns all runs for an automation, newest first
func (s *RunStore) ListByAutomation(automationID string) ([]Run, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	dir := s.automationDir(automationID)
	entries, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("list runs: %w", err)
	}

	var runs []Run
	for _, e := range entries {
		if !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		runID := strings.TrimSuffix(e.Name(), ".json")
		r, err := s.readRun(automationID, runID)
		if err != nil {
			continue
		}
		runs = append(runs, *r)
	}

	sort.Slice(runs, func(i, j int) bool {
		return runs[i].StartedAt > runs[j].StartedAt
	})
	return runs, nil
}

// UnreadWithFindings returns all unread runs that have findings, across all automations
func (s *RunStore) UnreadWithFindings() ([]Run, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	autoDirs, err := os.ReadDir(s.dir)
	if err != nil {
		return nil, fmt.Errorf("list automation dirs: %w", err)
	}

	var results []Run
	for _, ad := range autoDirs {
		if !ad.IsDir() {
			continue
		}
		autoID := ad.Name()
		entries, err := os.ReadDir(s.automationDir(autoID))
		if err != nil {
			continue
		}
		for _, e := range entries {
			if !strings.HasSuffix(e.Name(), ".json") {
				continue
			}
			runID := strings.TrimSuffix(e.Name(), ".json")
			r, err := s.readRun(autoID, runID)
			if err != nil {
				continue
			}
			if r.HasFindings && !r.Read {
				results = append(results, *r)
			}
		}
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].StartedAt > results[j].StartedAt
	})
	return results, nil
}

func (s *RunStore) readRun(automationID, runID string) (*Run, error) {
	data, err := os.ReadFile(s.runPath(automationID, runID))
	if err != nil {
		return nil, fmt.Errorf("read run %s: %w", runID, err)
	}
	var r Run
	if err := json.Unmarshal(data, &r); err != nil {
		return nil, fmt.Errorf("unmarshal run %s: %w", runID, err)
	}
	return &r, nil
}

func (s *RunStore) writeRun(r Run) error {
	data, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal run: %w", err)
	}
	return os.WriteFile(s.runPath(r.AutomationID, r.ID), data, 0644)
}
