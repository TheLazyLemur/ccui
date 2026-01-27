package permission

import (
	"ccui/backend"
	"sync"
)

// EventEmitter abstracts event emission (decoupled from Wails)
type EventEmitter interface {
	Emit(eventName string, data any)
}

// PermissionRequest is emitted when user permission is needed
type PermissionRequest struct {
	ToolCallID string               `json:"toolCallId"`
	ToolName   string               `json:"toolName"`
	Options    []backend.PermOption `json:"options"`
}

// Layer handles permission checks and user permission requests
type Layer struct {
	rules   *RuleSet
	emitter EventEmitter

	mu       sync.Mutex
	pending  map[string]chan string // toolCallID -> response channel
}

// NewLayer creates a new permission layer
func NewLayer(rules *RuleSet, emitter EventEmitter) *Layer {
	return &Layer{
		rules:   rules,
		emitter: emitter,
		pending: make(map[string]chan string),
	}
}

// Check returns the permission decision for a tool
func (l *Layer) Check(toolName, input string) Decision {
	return l.rules.Check(toolName, input)
}

// Request blocks until user grants/denies permission
// Returns the selected option ID
func (l *Layer) Request(toolCallID, toolName string, options []backend.PermOption) (string, error) {
	// Create response channel
	respCh := make(chan string, 1)
	l.mu.Lock()
	l.pending[toolCallID] = respCh
	l.mu.Unlock()

	// Emit permission request event
	l.emitter.Emit("permission_request", PermissionRequest{
		ToolCallID: toolCallID,
		ToolName:   toolName,
		Options:    options,
	})

	// Block waiting for response
	optionID := <-respCh

	// Cleanup
	l.mu.Lock()
	delete(l.pending, toolCallID)
	l.mu.Unlock()

	return optionID, nil
}

// Respond unblocks a pending permission request
func (l *Layer) Respond(toolCallID, optionID string) {
	l.mu.Lock()
	ch, ok := l.pending[toolCallID]
	l.mu.Unlock()

	if ok {
		ch <- optionID
	}
}
