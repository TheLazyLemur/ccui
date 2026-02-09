package automation

import (
	"context"
	"fmt"
	"path/filepath"
	"sync"
	"testing"

	"ccui/backend"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockSession implements backend.Session
type mockSession struct {
	eventChan chan<- backend.Event
	response  string
	sendErr   error
	closed    bool
	mu        sync.Mutex
}

func (m *mockSession) SendPrompt(text string, allowedTools []string) error {
	if m.sendErr != nil {
		return m.sendErr
	}
	m.eventChan <- backend.Event{Type: backend.EventMessageChunk, Data: m.response}
	m.eventChan <- backend.Event{Type: backend.EventPromptComplete, Data: nil}
	return nil
}
func (m *mockSession) SetMode(string) error           { return nil }
func (m *mockSession) Cancel()                         {}
func (m *mockSession) Close() error                    { m.mu.Lock(); m.closed = true; m.mu.Unlock(); return nil }
func (m *mockSession) SessionID() string               { return "mock-session" }
func (m *mockSession) CurrentMode() string             { return "" }
func (m *mockSession) AvailableModes() []backend.SessionMode { return nil }
func (m *mockSession) FileChangeStore() *backend.FileChangeStore { return nil }

// mockBackend implements backend.AgentBackend
type mockBackend struct {
	session *mockSession
}

func (b *mockBackend) NewSession(ctx context.Context, opts backend.SessionOpts) (backend.Session, error) {
	b.session.eventChan = opts.EventChan
	return b.session, nil
}

// mockEmitter captures emitted events
type mockEmitter struct {
	mu     sync.Mutex
	events []emittedEvent
}

type emittedEvent struct {
	name string
	data any
}

func (e *mockEmitter) Emit(name string, data any) {
	e.mu.Lock()
	e.events = append(e.events, emittedEvent{name, data})
	e.mu.Unlock()
}

func (e *mockEmitter) getEvents() []emittedEvent {
	e.mu.Lock()
	defer e.mu.Unlock()
	out := make([]emittedEvent, len(e.events))
	copy(out, e.events)
	return out
}

func setupEngine(t *testing.T, session *mockSession) (*Engine, *mockEmitter) {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "runs")
	runStore, err := NewRunStore(dir)
	require.NoError(t, err)

	emitter := &mockEmitter{}
	mb := &mockBackend{session: session}
	factory := func(backendType string) (backend.AgentBackend, error) {
		return mb, nil
	}

	engine := NewEngine(factory, runStore, emitter, nil)
	return engine, emitter
}

func TestEngine_Execute_Success(t *testing.T) {
	a := assert.New(t)
	r := require.New(t)

	session := &mockSession{response: "Found 2 potential issues"}
	engine, emitter := setupEngine(t, session)

	auto := Automation{
		ID:              "auto-1",
		Name:            "test",
		Prompt:          "review code",
		ProjectDir:      t.TempDir(),
		BackendType:     "acp",
		PermissionLevel: PermReadOnly,
	}

	run, err := engine.Execute(context.Background(), auto)
	r.NoError(err)

	a.Equal(RunStatusCompleted, run.Status)
	a.Equal("Found 2 potential issues", run.Output)
	a.True(run.HasFindings)
	a.NotEmpty(run.CompletedAt)

	// verify events were emitted
	events := emitter.getEvents()
	a.True(len(events) >= 2, "expected at least 2 events, got %d", len(events))

	// first event should be run_started
	a.Contains(events[0].name, "run_started")
}

func TestEngine_Execute_EmptyOutput(t *testing.T) {
	a := assert.New(t)
	r := require.New(t)

	session := &mockSession{response: ""}
	engine, _ := setupEngine(t, session)

	auto := Automation{
		ID:              "auto-1",
		Prompt:          "check things",
		ProjectDir:      t.TempDir(),
		BackendType:     "acp",
		PermissionLevel: PermReadOnly,
	}

	run, err := engine.Execute(context.Background(), auto)
	r.NoError(err)
	a.Equal(RunStatusCompleted, run.Status)
	a.False(run.HasFindings)
}

func TestEngine_Execute_SendError(t *testing.T) {
	a := assert.New(t)

	session := &mockSession{sendErr: fmt.Errorf("connection refused")}
	engine, _ := setupEngine(t, session)

	auto := Automation{
		ID:              "auto-1",
		Prompt:          "review",
		ProjectDir:      t.TempDir(),
		BackendType:     "acp",
		PermissionLevel: PermReadOnly,
	}

	run, err := engine.Execute(context.Background(), auto)
	a.Error(err)
	a.Equal(RunStatusFailed, run.Status)
	a.Contains(run.Error, "connection refused")
}

func TestEngine_Execute_BackendFactoryError(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "runs")
	runStore, err := NewRunStore(dir)
	require.NoError(t, err)

	factory := func(string) (backend.AgentBackend, error) {
		return nil, fmt.Errorf("unknown backend")
	}

	engine := NewEngine(factory, runStore, nil, nil)

	auto := Automation{
		ID:          "auto-1",
		Prompt:      "review",
		ProjectDir:  t.TempDir(),
		BackendType: "invalid",
	}

	run, err := engine.Execute(context.Background(), auto)
	assert.Error(t, err)
	assert.Equal(t, RunStatusFailed, run.Status)
}

func TestEngine_CancelRun(t *testing.T) {
	// just verify CancelRun doesn't panic on unknown IDs
	dir := filepath.Join(t.TempDir(), "runs")
	runStore, _ := NewRunStore(dir)
	engine := NewEngine(nil, runStore, nil, nil)
	engine.CancelRun("nonexistent") // should not panic
}
