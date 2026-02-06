package backend

import "context"

// EventType for backend events
type EventType string

const (
	EventMessageChunk      EventType = "message_chunk"
	EventThoughtChunk      EventType = "thought_chunk"
	EventToolState         EventType = "tool_state"
	EventModeChanged       EventType = "mode_changed"
	EventPlanUpdate        EventType = "plan_update"
	EventPermissionRequest EventType = "permission_request"
	EventPromptComplete    EventType = "prompt_complete"
	EventFileChanges       EventType = "file_changes"
)

// Event from the backend
type Event struct {
	Type EventType
	Data any
}

// SessionOpts for creating sessions
type SessionOpts struct {
	CWD        string
	MCPServers []any
	EventChan  chan<- Event // where to send events

	// Review-mode configuration
	AutoPermission     bool             // auto-approve all permissions
	SuppressToolEvents bool             // don't emit tool state events
	FileChangeStore    *FileChangeStore // optional shared store
}

// Session represents an active agent session
type Session interface {
	SendPrompt(text string, allowedTools []string) error
	SetMode(modeID string) error
	Cancel()
	Close() error

	SessionID() string
	CurrentMode() string
	AvailableModes() []SessionMode
	FileChangeStore() *FileChangeStore
}

// AgentBackend creates and manages sessions
type AgentBackend interface {
	NewSession(ctx context.Context, opts SessionOpts) (Session, error)
}
