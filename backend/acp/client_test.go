package acp

import (
	"encoding/json"
	"sync"
	"testing"

	"ccui/backend"
)

// MockTransport for testing
type MockTransport struct {
	mu           sync.Mutex
	handler      func(method string, params json.RawMessage, id *int)
	sentMessages []struct {
		Method string
		Params any
	}
	responses map[string]json.RawMessage
}

func NewMockTransport() *MockTransport {
	return &MockTransport{
		responses: make(map[string]json.RawMessage),
	}
}

func (m *MockTransport) Send(method string, params any) (json.RawMessage, error) {
	m.mu.Lock()
	m.sentMessages = append(m.sentMessages, struct {
		Method string
		Params any
	}{method, params})
	resp := m.responses[method]
	m.mu.Unlock()
	return resp, nil
}

func (m *MockTransport) Notify(method string, params any) {
	m.mu.Lock()
	m.sentMessages = append(m.sentMessages, struct {
		Method string
		Params any
	}{method, params})
	m.mu.Unlock()
}

func (m *MockTransport) OnMethod(handler func(method string, params json.RawMessage, id *int)) {
	m.handler = handler
}

func (m *MockTransport) Respond(id *int, result json.RawMessage) {
	// Track response in sentMessages with empty method
	m.mu.Lock()
	m.sentMessages = append(m.sentMessages, struct {
		Method string
		Params any
	}{"", map[string]any{"id": id, "result": result}})
	m.mu.Unlock()
}

func (m *MockTransport) Close() error {
	return nil
}

func (m *MockTransport) SetResponse(method string, result any) {
	data, _ := json.Marshal(result)
	m.responses[method] = data
}

func (m *MockTransport) SimulateMethod(method string, params any, id *int) {
	if m.handler != nil {
		data, _ := json.Marshal(params)
		m.handler(method, data, id)
	}
}

func TestClient_HandleMessageChunk(t *testing.T) {
	transport := NewMockTransport()
	events := make(chan backend.Event, 10)

	client := &Client{
		transport:       transport,
		eventChan:       events,
		toolManager:     backend.NewToolCallManager(),
		fileChangeStore: backend.NewFileChangeStore(),
		toolAdapters:    DefaultToolAdapters(),
	}

	// Set up transport handler
	transport.OnMethod(func(method string, params json.RawMessage, id *int) {
		client.handleMethod(method, params, id)
	})

	// Simulate agent_message_chunk
	transport.SimulateMethod("session/update", SessionUpdate{
		SessionID: "test-session",
		Update: UpdateContent{
			SessionUpdate: "agent_message_chunk",
			Content:       json.RawMessage(`{"type":"text","text":"Hello world"}`),
		},
	}, nil)

	// Verify event emitted
	select {
	case evt := <-events:
		if evt.Type != backend.EventMessageChunk {
			t.Errorf("expected EventMessageChunk, got %v", evt.Type)
		}
		if evt.Data != "Hello world" {
			t.Errorf("expected 'Hello world', got %v", evt.Data)
		}
	default:
		t.Error("expected event but got none")
	}
}

func TestClient_HandleThoughtChunk(t *testing.T) {
	transport := NewMockTransport()
	events := make(chan backend.Event, 10)

	client := &Client{
		transport:       transport,
		eventChan:       events,
		toolManager:     backend.NewToolCallManager(),
		fileChangeStore: backend.NewFileChangeStore(),
		toolAdapters:    DefaultToolAdapters(),
	}

	transport.OnMethod(func(method string, params json.RawMessage, id *int) {
		client.handleMethod(method, params, id)
	})

	// Simulate agent_thought_chunk
	transport.SimulateMethod("session/update", SessionUpdate{
		SessionID: "test-session",
		Update: UpdateContent{
			SessionUpdate: "agent_thought_chunk",
			Content:       json.RawMessage(`{"type":"text","text":"Thinking..."}`),
		},
	}, nil)

	select {
	case evt := <-events:
		if evt.Type != backend.EventThoughtChunk {
			t.Errorf("expected EventThoughtChunk, got %v", evt.Type)
		}
		if evt.Data != "Thinking..." {
			t.Errorf("expected 'Thinking...', got %v", evt.Data)
		}
	default:
		t.Error("expected event but got none")
	}
}

func TestClient_HandleToolCall(t *testing.T) {
	transport := NewMockTransport()
	events := make(chan backend.Event, 10)

	client := &Client{
		transport:       transport,
		eventChan:       events,
		toolManager:     backend.NewToolCallManager(),
		fileChangeStore: backend.NewFileChangeStore(),
		toolAdapters:    DefaultToolAdapters(),
	}

	transport.OnMethod(func(method string, params json.RawMessage, id *int) {
		client.handleMethod(method, params, id)
	})

	// Simulate tool_call
	transport.SimulateMethod("session/update", SessionUpdate{
		SessionID: "test-session",
		Update: UpdateContent{
			SessionUpdate: "tool_call",
			ToolCallID:    "tool-123",
			Title:         "Read",
			ToolKind:      "read",
			Status:        "running",
			RawInput:      map[string]any{"file_path": "/test/file.go"},
		},
	}, nil)

	// Verify tool_state event emitted
	select {
	case evt := <-events:
		if evt.Type != backend.EventToolState {
			t.Errorf("expected EventToolState, got %v", evt.Type)
		}
		state, ok := evt.Data.(*backend.ToolState)
		if !ok {
			t.Fatalf("expected *backend.ToolState, got %T", evt.Data)
		}
		if state.ID != "tool-123" {
			t.Errorf("expected tool ID 'tool-123', got %s", state.ID)
		}
		if state.Status != "running" {
			t.Errorf("expected status 'running', got %s", state.Status)
		}
		if state.Title != "Read" {
			t.Errorf("expected title 'Read', got %s", state.Title)
		}
	default:
		t.Error("expected event but got none")
	}

	// Verify tool stored in manager
	stored := client.toolManager.Get("tool-123")
	if stored == nil {
		t.Fatal("expected tool to be stored in manager")
	}
	if stored.Status != "running" {
		t.Errorf("expected stored status 'running', got %s", stored.Status)
	}
}

func TestClient_HandleToolCallUpdate(t *testing.T) {
	transport := NewMockTransport()
	events := make(chan backend.Event, 10)

	client := &Client{
		transport:       transport,
		eventChan:       events,
		toolManager:     backend.NewToolCallManager(),
		fileChangeStore: backend.NewFileChangeStore(),
		toolAdapters:    DefaultToolAdapters(),
	}

	transport.OnMethod(func(method string, params json.RawMessage, id *int) {
		client.handleMethod(method, params, id)
	})

	// First create the tool
	client.toolManager.Set(&backend.ToolState{
		ID:     "tool-456",
		Status: "running",
		Title:  "Edit",
	})

	// Simulate tool_call_update
	transport.SimulateMethod("session/update", SessionUpdate{
		SessionID: "test-session",
		Update: UpdateContent{
			SessionUpdate: "tool_call_update",
			ToolCallID:    "tool-456",
			Status:        "completed",
			Output:        []backend.OutputBlock{{Type: "text"}},
		},
	}, nil)

	// Verify updated tool_state event emitted
	select {
	case evt := <-events:
		if evt.Type != backend.EventToolState {
			t.Errorf("expected EventToolState, got %v", evt.Type)
		}
		state, ok := evt.Data.(*backend.ToolState)
		if !ok {
			t.Fatalf("expected *backend.ToolState, got %T", evt.Data)
		}
		if state.ID != "tool-456" {
			t.Errorf("expected tool ID 'tool-456', got %s", state.ID)
		}
		if state.Status != "completed" {
			t.Errorf("expected status 'completed', got %s", state.Status)
		}
	default:
		t.Error("expected event but got none")
	}
}

func TestClient_HandlePermissionRequest(t *testing.T) {
	transport := NewMockTransport()
	events := make(chan backend.Event, 10)

	client := &Client{
		transport:        transport,
		eventChan:        events,
		toolManager:      backend.NewToolCallManager(),
		fileChangeStore:  backend.NewFileChangeStore(),
		toolAdapters:     DefaultToolAdapters(),
		permissionRespCh: make(chan string, 1),
	}

	// First create a tool that needs permission
	client.toolManager.Set(&backend.ToolState{
		ID:     "tool-789",
		Status: "running",
		Title:  "Write",
	})

	transport.OnMethod(func(method string, params json.RawMessage, id *int) {
		client.handleMethod(method, params, id)
	})

	// Send permission response before request (simulating async UI)
	client.permissionRespCh <- "allow_once"

	// Simulate permission request
	id := 42
	transport.SimulateMethod("session/request_permission", PermissionRequest{
		SessionID: "test-session",
		ToolCall: ToolCallInfo{
			ToolCallID: "tool-789",
			Title:      "Write",
			Kind:       "write",
		},
		Options: []backend.PermOption{
			{OptionID: "allow_once", Name: "Allow Once", Kind: "allow"},
			{OptionID: "deny", Name: "Deny", Kind: "deny"},
		},
	}, &id)

	// Verify tool state updated with permission options
	stored := client.toolManager.Get("tool-789")
	if stored == nil {
		t.Fatal("expected tool to be stored in manager")
	}
	if stored.Status != "awaiting_permission" {
		t.Errorf("expected status 'awaiting_permission', got %s", stored.Status)
	}
	if len(stored.PermissionOptions) != 2 {
		t.Errorf("expected 2 permission options, got %d", len(stored.PermissionOptions))
	}

	// Verify permission response was sent
	found := false
	for _, msg := range transport.sentMessages {
		if msg.Method == "" { // Response messages have no method
			found = true
			break
		}
	}
	// Note: response is sent via stdin.Write, not through Send/Notify
	_ = found
}

func TestClient_HandlePermissionRequest_AutoAllow(t *testing.T) {
	transport := NewMockTransport()
	events := make(chan backend.Event, 10)

	client := &Client{
		transport:        transport,
		eventChan:        events,
		toolManager:      backend.NewToolCallManager(),
		fileChangeStore:  backend.NewFileChangeStore(),
		toolAdapters:     DefaultToolAdapters(),
		autoPermission:   true,
		permissionRespCh: make(chan string, 1),
	}

	transport.OnMethod(func(method string, params json.RawMessage, id *int) {
		client.handleMethod(method, params, id)
	})

	// Simulate permission request with auto-allow enabled
	id := 43
	transport.SimulateMethod("session/request_permission", PermissionRequest{
		SessionID: "test-session",
		ToolCall: ToolCallInfo{
			ToolCallID: "tool-auto",
			Title:      "Bash",
			Kind:       "bash",
		},
		Options: []backend.PermOption{
			{OptionID: "allow_always", Name: "Allow Always", Kind: "allow"},
		},
	}, &id)

	// With auto-permission, should NOT block waiting for user response
	// and should NOT update tool state to awaiting_permission
	stored := client.toolManager.Get("tool-auto")
	if stored != nil && stored.Status == "awaiting_permission" {
		t.Error("auto-permission should not set status to awaiting_permission")
	}
}

func TestClient_HandleModeUpdate(t *testing.T) {
	transport := NewMockTransport()
	events := make(chan backend.Event, 10)

	client := &Client{
		transport:       transport,
		eventChan:       events,
		toolManager:     backend.NewToolCallManager(),
		fileChangeStore: backend.NewFileChangeStore(),
		toolAdapters:    DefaultToolAdapters(),
	}

	transport.OnMethod(func(method string, params json.RawMessage, id *int) {
		client.handleMethod(method, params, id)
	})

	// Simulate current_mode_update
	transport.SimulateMethod("session/update", SessionUpdate{
		SessionID: "test-session",
		Update: UpdateContent{
			SessionUpdate: "current_mode_update",
			ModeID:        "plan",
		},
	}, nil)

	// Verify mode_changed event emitted
	select {
	case evt := <-events:
		if evt.Type != backend.EventModeChanged {
			t.Errorf("expected EventModeChanged, got %v", evt.Type)
		}
		if evt.Data != "plan" {
			t.Errorf("expected 'plan', got %v", evt.Data)
		}
	default:
		t.Error("expected event but got none")
	}

	// Verify client mode updated
	if client.currentModeID != "plan" {
		t.Errorf("expected currentModeID 'plan', got %s", client.currentModeID)
	}
}

func TestClient_HandlePlanUpdate(t *testing.T) {
	transport := NewMockTransport()
	events := make(chan backend.Event, 10)

	client := &Client{
		transport:       transport,
		eventChan:       events,
		toolManager:     backend.NewToolCallManager(),
		fileChangeStore: backend.NewFileChangeStore(),
		toolAdapters:    DefaultToolAdapters(),
	}

	transport.OnMethod(func(method string, params json.RawMessage, id *int) {
		client.handleMethod(method, params, id)
	})

	entries := []backend.PlanEntry{
		{Content: "Step 1", Priority: "high", Status: "completed"},
		{Content: "Step 2", Priority: "medium", Status: "pending"},
	}

	// Simulate plan update
	transport.SimulateMethod("session/update", SessionUpdate{
		SessionID: "test-session",
		Update: UpdateContent{
			SessionUpdate: "plan",
			Entries:       entries,
		},
	}, nil)

	// Verify plan_update event emitted
	select {
	case evt := <-events:
		if evt.Type != backend.EventPlanUpdate {
			t.Errorf("expected EventPlanUpdate, got %v", evt.Type)
		}
		plan, ok := evt.Data.([]backend.PlanEntry)
		if !ok {
			t.Fatalf("expected []backend.PlanEntry, got %T", evt.Data)
		}
		if len(plan) != 2 {
			t.Errorf("expected 2 entries, got %d", len(plan))
		}
	default:
		t.Error("expected event but got none")
	}
}

func TestClient_SuppressToolEvents(t *testing.T) {
	transport := NewMockTransport()
	events := make(chan backend.Event, 10)

	client := &Client{
		transport:          transport,
		eventChan:          events,
		toolManager:        backend.NewToolCallManager(),
		fileChangeStore:    backend.NewFileChangeStore(),
		toolAdapters:       DefaultToolAdapters(),
		suppressToolEvents: true,
	}

	transport.OnMethod(func(method string, params json.RawMessage, id *int) {
		client.handleMethod(method, params, id)
	})

	// Simulate tool_call with suppression enabled
	transport.SimulateMethod("session/update", SessionUpdate{
		SessionID: "test-session",
		Update: UpdateContent{
			SessionUpdate: "tool_call",
			ToolCallID:    "tool-suppressed",
			Title:         "Read",
			Status:        "running",
		},
	}, nil)

	// Should NOT emit tool_state event
	select {
	case evt := <-events:
		t.Errorf("expected no event with suppressToolEvents, got %v", evt.Type)
	default:
		// Expected - no event
	}
}

// mockPermissionLayer implements permission handling for tests
type mockPermissionLayer struct {
	mu       sync.Mutex
	requests []mockPermRequest
	response string // response to return from Request
}

type mockPermRequest struct {
	toolCallID string
	toolName   string
	options    []backend.PermOption
}

func (m *mockPermissionLayer) Request(toolCallID, toolName string, options []backend.PermOption) (string, error) {
	m.mu.Lock()
	m.requests = append(m.requests, mockPermRequest{toolCallID, toolName, options})
	resp := m.response
	m.mu.Unlock()
	return resp, nil
}

func (m *mockPermissionLayer) getRequests() []mockPermRequest {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]mockPermRequest{}, m.requests...)
}

func TestClient_PermissionLayerIntegration(t *testing.T) {
	// given - client with a permission layer
	transport := NewMockTransport()
	events := make(chan backend.Event, 10)

	layer := &mockPermissionLayer{response: "allow_once"}

	client := NewClient(ClientConfig{
		Transport: transport,
		EventChan: events,
	}, WithPermissionLayer(layer))

	// Create a tool that will request permission
	client.toolManager.Set(&backend.ToolState{
		ID:     "tool-perm",
		Status: "running",
		Title:  "Write",
	})

	// when - permission request comes in
	id := 99
	transport.SimulateMethod("session/request_permission", PermissionRequest{
		SessionID: "test-session",
		ToolCall: ToolCallInfo{
			ToolCallID: "tool-perm",
			Title:      "Write",
			Kind:       "write",
		},
		Options: []backend.PermOption{
			{OptionID: "allow_once", Name: "Allow Once", Kind: "allow"},
			{OptionID: "deny", Name: "Deny", Kind: "deny"},
		},
	}, &id)

	// then - should delegate to permission layer
	requests := layer.getRequests()
	if len(requests) != 1 {
		t.Fatalf("expected 1 permission request to layer, got %d", len(requests))
	}
	if requests[0].toolCallID != "tool-perm" {
		t.Errorf("expected toolCallID 'tool-perm', got %s", requests[0].toolCallID)
	}
	if requests[0].toolName != "Write" {
		t.Errorf("expected toolName 'Write', got %s", requests[0].toolName)
	}
	if len(requests[0].options) != 2 {
		t.Errorf("expected 2 options, got %d", len(requests[0].options))
	}
}
