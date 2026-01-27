package permission

import (
	"sync"
	"testing"
	"time"

	"ccui/backend"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockEmitter captures emitted events for testing
type mockEmitter struct {
	mu     sync.Mutex
	events []emittedEvent
}

type emittedEvent struct {
	name string
	data any
}

func (m *mockEmitter) Emit(eventName string, data any) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.events = append(m.events, emittedEvent{name: eventName, data: data})
}

func (m *mockEmitter) getEvents() []emittedEvent {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]emittedEvent{}, m.events...)
}

func TestPermissionLayer_CheckDelegates(t *testing.T) {
	a := assert.New(t)

	// given - a layer with default rules
	emitter := &mockEmitter{}
	layer := NewLayer(DefaultRules(), emitter)

	// when/then - Check should delegate to RuleSet
	a.Equal(Allow, layer.Check("Read", ""))
	a.Equal(Ask, layer.Check("Write", ""))
	a.Equal(Deny, layer.Check("UnknownTool", ""))
}

func TestPermissionLayer_RequestBlocks(t *testing.T) {
	r := require.New(t)
	a := assert.New(t)

	// given - a layer with an Ask-requiring tool
	emitter := &mockEmitter{}
	layer := NewLayer(DefaultRules(), emitter)

	options := []backend.PermOption{
		{OptionID: "allow", Name: "Allow", Kind: "allow"},
		{OptionID: "deny", Name: "Deny", Kind: "deny"},
	}

	// when - Request is called in a goroutine
	resultCh := make(chan string, 1)
	errCh := make(chan error, 1)
	go func() {
		optionID, err := layer.Request("call-123", "Write", options)
		if err != nil {
			errCh <- err
		} else {
			resultCh <- optionID
		}
	}()

	// then - should block (no result yet)
	select {
	case <-resultCh:
		t.Fatal("Request should block until Respond is called")
	case <-errCh:
		t.Fatal("Request should not error")
	case <-time.After(50 * time.Millisecond):
		// expected - still blocking
	}

	// then - emitter should have received permission_request event
	events := emitter.getEvents()
	r.Len(events, 1)
	a.Equal("permission_request", events[0].name)
	req, ok := events[0].data.(PermissionRequest)
	r.True(ok)
	a.Equal("call-123", req.ToolCallID)
	a.Equal("Write", req.ToolName)
	a.Equal(options, req.Options)

	// cleanup - respond to unblock
	layer.Respond("call-123", "allow")
	select {
	case result := <-resultCh:
		a.Equal("allow", result)
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Request should unblock after Respond")
	}
}

func TestPermissionLayer_RespondUnblocks(t *testing.T) {
	a := assert.New(t)

	// given - a layer with a pending request
	emitter := &mockEmitter{}
	layer := NewLayer(DefaultRules(), emitter)

	options := []backend.PermOption{
		{OptionID: "allow", Name: "Allow", Kind: "allow"},
		{OptionID: "deny", Name: "Deny", Kind: "deny"},
	}

	resultCh := make(chan string, 1)
	go func() {
		optionID, _ := layer.Request("call-456", "Edit", options)
		resultCh <- optionID
	}()

	// wait for request to be pending
	time.Sleep(20 * time.Millisecond)

	// when - Respond is called with the deny option
	layer.Respond("call-456", "deny")

	// then - Request should return the selected option
	select {
	case result := <-resultCh:
		a.Equal("deny", result)
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Request should unblock after Respond")
	}
}
