# Phase 1: Architecture Refactor

Extract layered architecture from current monolithic app.go.

## Current State

```
app.go (~1200 lines)
├── ACPClient struct
├── JSON-RPC handling
├── Permission handling (coupled)
├── Tool state management
├── Session management
└── Wails bindings
```

## Target State

```
backend/
├── interface.go      # AgentBackend, Session interfaces
├── acp/
│   └── client.go     # ACPClient implementation
├── types.go          # Shared types (ToolState, Message, etc)

permission/
├── layer.go          # PermissionLayer interface + impl
├── rules.go          # Deterministic permission rules

executor/
├── executor.go       # ToolExecutor interface
├── fs.go             # File operations
├── bash.go           # Shell execution
├── search.go         # Glob, grep

app.go                # Wails bindings, wires layers together
```

## Interfaces

### AgentBackend

```go
// backend/interface.go
type AgentBackend interface {
    NewSession(ctx context.Context, opts SessionOpts) (Session, error)
}

type Session interface {
    ID() string
    Prompt(ctx context.Context, prompt []Content) error
    Events() <-chan Event
    Cancel()
    Close() error
}

type SessionOpts struct {
    Cwd        string
    McpServers []McpServerConfig
}

type Event struct {
    Type         EventType // message_chunk, tool_call, tool_update, permission_request, done
    Content      *TextContent
    ToolCall     *ToolCall
    ToolUpdate   *ToolUpdate
    PermissionReq *PermissionRequest
    StopReason   string
}
```

### PermissionLayer

```go
// permission/layer.go
type Decision int
const (
    Allow Decision = iota
    Deny
    Ask
)

type PermissionLayer interface {
    // Check returns immediate decision based on rules
    Check(tool string, input map[string]any) Decision

    // Request emits to frontend, blocks until response
    Request(ctx context.Context, req PermissionRequest) (string, error)

    // Respond handles frontend response
    Respond(optionID string)
}

type PermissionRequest struct {
    ToolCallID string
    Title      string
    Kind       string // read, write, execute, etc
    Input      map[string]any
    Options    []PermOption
}
```

### ToolExecutor

```go
// executor/executor.go
type ToolExecutor interface {
    Execute(ctx context.Context, name string, input map[string]any) (any, error)
    Available() []ToolDef
}

type ToolDef struct {
    Name        string
    Description string
    Parameters  map[string]any // JSON Schema
}
```

## Tasks

- [ ] Create `backend/interface.go` with interfaces
- [ ] Create `backend/types.go` with shared types
- [ ] Extract `backend/acp/client.go` from app.go
- [ ] Create `permission/layer.go`
- [ ] Create `permission/rules.go` with deterministic rules
- [ ] Create `executor/executor.go` interface
- [ ] Create `executor/fs.go` (read, write, edit)
- [ ] Create `executor/bash.go`
- [ ] Refactor `app.go` to wire layers
- [ ] Update tests

## Permission Rules (Initial)

```go
// permission/rules.go
func DefaultRules() RuleSet {
    return RuleSet{
        "Read":   Allow,  // always allow reads
        "Glob":   Allow,  // always allow search
        "Grep":   Allow,  // always allow search
        "Write":  Ask,    // ask user
        "Edit":   Ask,    // ask user
        "Bash":   func(input) Decision { /* check command */ },
    }
}
```

## Migration Strategy

1. Create new packages alongside existing code
2. ACPClient implements AgentBackend interface
3. Gradually move logic from app.go to layers
4. Keep Wails bindings in app.go
5. Tests pass throughout

## Success Criteria

- [ ] app.go < 400 lines
- [ ] ACPClient implements AgentBackend
- [ ] Permission logic extracted and testable
- [ ] Existing functionality unchanged
- [ ] Tests pass
