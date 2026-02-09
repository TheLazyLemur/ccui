package automation

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/google/uuid"
)

// Store persists automations as JSON
type Store struct {
	dir  string
	mu   sync.RWMutex
	data []Automation
}

// NewStore creates or loads an automation store from dir
func NewStore(dir string) (*Store, error) {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("create store dir: %w", err)
	}
	s := &Store{dir: dir}
	if err := s.load(); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *Store) indexPath() string {
	return filepath.Join(s.dir, "index.json")
}

func (s *Store) load() error {
	data, err := os.ReadFile(s.indexPath())
	if os.IsNotExist(err) {
		s.data = []Automation{}
		return s.save()
	}
	if err != nil {
		return fmt.Errorf("read index: %w", err)
	}
	return json.Unmarshal(data, &s.data)
}

func (s *Store) save() error {
	data, err := json.MarshalIndent(s.data, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal index: %w", err)
	}
	return os.WriteFile(s.indexPath(), data, 0644)
}

// List returns all automations
func (s *Store) List() []Automation {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]Automation, len(s.data))
	copy(out, s.data)
	return out
}

// Get returns an automation by ID
func (s *Store) Get(id string) (*Automation, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for i := range s.data {
		if s.data[i].ID == id {
			a := s.data[i]
			return &a, nil
		}
	}
	return nil, fmt.Errorf("automation not found: %s", id)
}

// Create adds a new automation, generating ID and timestamps
func (s *Store) Create(a Automation) (*Automation, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().UTC().Format(time.RFC3339)
	a.ID = uuid.New().String()
	a.CreatedAt = now
	a.UpdatedAt = now

	s.data = append(s.data, a)
	if err := s.save(); err != nil {
		s.data = s.data[:len(s.data)-1]
		return nil, err
	}
	return &a, nil
}

// Update replaces an existing automation
func (s *Store) Update(a Automation) (*Automation, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i := range s.data {
		if s.data[i].ID == a.ID {
			a.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
			a.CreatedAt = s.data[i].CreatedAt
			s.data[i] = a
			if err := s.save(); err != nil {
				return nil, err
			}
			return &a, nil
		}
	}
	return nil, fmt.Errorf("automation not found: %s", a.ID)
}

// Delete removes an automation by ID
func (s *Store) Delete(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i := range s.data {
		if s.data[i].ID == id {
			s.data = append(s.data[:i], s.data[i+1:]...)
			return s.save()
		}
	}
	return fmt.Errorf("automation not found: %s", id)
}
