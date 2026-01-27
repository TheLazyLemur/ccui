package backend

import "sync"

// PatchHunk represents a single hunk in a unified diff
type PatchHunk struct {
	OldStart int      `json:"oldStart"`
	OldLines int      `json:"oldLines"`
	NewStart int      `json:"newStart"`
	NewLines int      `json:"newLines"`
	Lines    []string `json:"lines"`
}

// DiffBlock represents a diff content block
type DiffBlock struct {
	Type    string `json:"type"`
	Path    string `json:"path,omitempty"`
	OldText string `json:"oldText,omitempty"`
	NewText string `json:"newText,omitempty"`
}

// OutputBlock represents tool output content
type OutputBlock struct {
	Type       string       `json:"type"`
	Content    *TextContent `json:"content,omitempty"`
	Path       string       `json:"path,omitempty"`
	OldContent string       `json:"oldContent,omitempty"`
	NewContent string       `json:"newContent,omitempty"`
}

// TextContent represents text content in messages
type TextContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// PermOption represents a permission option
type PermOption struct {
	OptionID string `json:"optionId"`
	Name     string `json:"name"`
	Kind     string `json:"kind"`
}

// SessionMode represents an agent session mode
type SessionMode struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

// PlanEntry represents a plan item
type PlanEntry struct {
	Content  string `json:"content"`
	Priority string `json:"priority"` // high, medium, low
	Status   string `json:"status"`   // pending, in_progress, completed
}

// ToolState tracks unified state for a single tool call
type ToolState struct {
	ID                string         `json:"id"`
	Status            string         `json:"status"` // pending, awaiting_permission, running, completed, error
	Title             string         `json:"title"`
	Kind              string         `json:"kind"`
	ToolName          string         `json:"toolName,omitempty"`
	ParentID          string         `json:"parentId,omitempty"`
	Input             map[string]any `json:"input,omitempty"`
	Output            []OutputBlock  `json:"output,omitempty"`
	Diff              map[string]any `json:"diff,omitempty"`
	Diffs             []DiffBlock    `json:"diffs,omitempty"`
	PermissionOptions []PermOption   `json:"permissionOptions,omitempty"`
}

// ToolCallManager tracks all active tool calls
type ToolCallManager struct {
	tools       map[string]*ToolState
	parentStack []string // stack of active Task tool IDs
	mu          sync.RWMutex
}

// NewToolCallManager creates a new ToolCallManager
func NewToolCallManager() *ToolCallManager {
	return &ToolCallManager{tools: make(map[string]*ToolState)}
}

// Get returns the tool state for the given ID
func (m *ToolCallManager) Get(id string) *ToolState {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.tools[id]
}

// Set stores a tool state
func (m *ToolCallManager) Set(state *ToolState) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.tools[state.ID] = state
}

// Update applies a function to update a tool state
func (m *ToolCallManager) Update(id string, fn func(*ToolState)) *ToolState {
	m.mu.Lock()
	defer m.mu.Unlock()
	if s, ok := m.tools[id]; ok {
		fn(s)
		return s
	}
	return nil
}

// PushParent adds a parent tool ID to the stack
func (m *ToolCallManager) PushParent(id string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.parentStack = append(m.parentStack, id)
}

// PopParent removes a parent tool ID from the stack
func (m *ToolCallManager) PopParent(id string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	// Remove the specific ID from stack (may not be at top if nested)
	for i := len(m.parentStack) - 1; i >= 0; i-- {
		if m.parentStack[i] == id {
			m.parentStack = append(m.parentStack[:i], m.parentStack[i+1:]...)
			return
		}
	}
}

// CurrentParent returns the current parent tool ID
func (m *ToolCallManager) CurrentParent() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if len(m.parentStack) > 0 {
		return m.parentStack[len(m.parentStack)-1]
	}
	return ""
}

// FileChange tracks a file's changes during the session
type FileChange struct {
	FilePath        string      `json:"filePath"`
	OriginalContent string      `json:"originalContent"`
	CurrentContent  string      `json:"currentContent"`
	Hunks           []PatchHunk `json:"hunks"`
}

// FileChangeStore accumulates file changes, coalesces to latest state
type FileChangeStore struct {
	changes map[string]*FileChange
	mu      sync.RWMutex
}

// NewFileChangeStore creates a new FileChangeStore
func NewFileChangeStore() *FileChangeStore {
	return &FileChangeStore{changes: make(map[string]*FileChange)}
}

// RecordChange records a file change, coalescing with existing changes
func (s *FileChangeStore) RecordChange(filePath, originalContent, currentContent string, hunks []PatchHunk) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if existing, ok := s.changes[filePath]; ok {
		// Coalesce: keep original, update current
		existing.CurrentContent = currentContent
		existing.Hunks = hunks
	} else {
		s.changes[filePath] = &FileChange{
			FilePath:        filePath,
			OriginalContent: originalContent,
			CurrentContent:  currentContent,
			Hunks:           hunks,
		}
	}
}

// Get returns the file change for the given path
func (s *FileChangeStore) Get(filePath string) *FileChange {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.changes[filePath]
}

// GetAll returns all file changes
func (s *FileChangeStore) GetAll() []FileChange {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]FileChange, 0, len(s.changes))
	for _, c := range s.changes {
		result = append(result, *c)
	}
	return result
}

// Clear removes all file changes
func (s *FileChangeStore) Clear() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.changes = make(map[string]*FileChange)
}
