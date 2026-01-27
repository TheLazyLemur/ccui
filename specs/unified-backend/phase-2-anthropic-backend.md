# Phase 2: Anthropic Direct API Backend

Implement AgentBackend for direct Anthropic API calls.

## Scope

- Direct Anthropic Messages API integration
- Tool use handling (we execute tools)
- Streaming support
- Uses shared PermissionLayer and ToolExecutor from Phase 1

## Flow

```
User prompt
    │
    ▼
AnthropicBackend.Prompt()
    │
    ├─► API call with tools
    │
    ▼
Stream response
    │
    ├─► text chunk → emit Event{Type: message_chunk}
    │
    ├─► tool_use block →
    │       │
    │       ├─► PermissionLayer.Check()
    │       │       │
    │       │       ├─► Allow → ToolExecutor.Execute() → continue
    │       │       ├─► Deny → send error result → continue
    │       │       └─► Ask → PermissionLayer.Request() → wait → execute or deny
    │       │
    │       └─► send tool_result back to API
    │
    └─► stop_reason → emit Event{Type: done}
```

## Phase 1 Context (Jump-off Point)

### Interface to Implement

```go
// backend/interface.go
type AgentBackend interface {
    NewSession(ctx context.Context, opts SessionOpts) (Session, error)
}

type Session interface {
    SendPrompt(text string, allowedTools []string) error
    SetMode(modeID string) error  // Can be no-op for direct API
    Cancel()
    Close() error
    SessionID() string
    CurrentMode() string
    AvailableModes() []SessionMode
}

type SessionOpts struct {
    CWD        string
    MCPServers []any
    EventChan  chan<- Event
}
```

### Events to Emit

```go
// backend/interface.go - use these event types
const (
    EventMessageChunk      EventType = "message_chunk"
    EventToolState         EventType = "tool_state"
    EventPermissionRequest EventType = "permission_request"  // emitted by PermissionLayer
    EventPromptComplete    EventType = "prompt_complete"
    EventFileChanges       EventType = "file_changes"
)
```

### Permission Layer (Ready to Use)

```go
// permission/layer.go
type Layer struct { ... }

func (l *Layer) Check(toolName, input string) Decision  // Allow, Ask, Deny
func (l *Layer) Request(toolCallID, toolName string, options []backend.PermOption) (string, error)  // Blocks
func (l *Layer) Respond(toolCallID, optionID string)  // Called by frontend
```

Default rules (permission/rules.go):
- **Allow**: Read, Glob, Grep, WebSearch, WebFetch
- **Ask**: Write, Edit, NotebookEdit, Bash
- **Deny**: anything else

### Types to Reuse

```go
// backend/types.go
type ToolState struct {
    ID                string
    Status            string  // pending, awaiting_permission, running, completed, error
    Title             string
    ToolName          string
    Input             map[string]any
    Output            []OutputBlock
    Diffs             []DiffBlock
    PermissionOptions []PermOption
}

type ToolCallManager   // tracks tool states, parent stack
type FileChangeStore   // coalesces file changes (original → current)
```

### What's Missing (Must Create)

**ToolExecutor** - not yet implemented. Need:

```go
// backend/tools/executor.go (proposed)
type ToolExecutor interface {
    Execute(ctx context.Context, toolName string, input map[string]any) (ToolResult, error)
}

type ToolResult struct {
    Content  string
    IsError  bool
    FileDiff *DiffBlock  // for Write/Edit
}
```

Tools to implement:
- Read, Write, Edit (file ops)
- Bash (command execution)
- Glob, Grep (search)
- WebSearch, WebFetch (optional, can skip initially)

## Implementation Plan

### 1. ToolExecutor Interface + Implementations

```
backend/tools/
├── executor.go      # ToolExecutor interface
├── read.go          # Read tool
├── write.go         # Write tool
├── edit.go          # Edit tool (string replace)
├── bash.go          # Bash tool
├── glob.go          # Glob tool
└── grep.go          # Grep tool
```

### 2. Anthropic Backend

```
backend/anthropic/
├── client.go        # AnthropicBackend + Session impl
├── tools.go         # Tool schema definitions (Anthropic format)
└── stream.go        # Streaming response handler
```

### 3. Integration

- Wire up in `app.go` alongside ACP backend
- Share PermissionLayer, ToolCallManager, FileChangeStore
- Same event emission pattern as ACP client

## TODO

- [ ] Define ToolExecutor interface
- [ ] Implement core tools (Read, Write, Edit, Bash, Glob, Grep)
- [ ] Define tool schemas (translate to Anthropic format)
- [ ] Implement AnthropicBackend.NewSession()
- [ ] Implement streaming with tool loop
- [ ] Handle multi-turn tool use
- [ ] Error handling / retries

## Key Files to Reference

| File | Purpose |
|------|---------|
| `backend/interface.go` | AgentBackend interface (48 LOC) |
| `backend/types.go` | Shared types (202 LOC) |
| `backend/acp/client.go` | Reference impl (415 LOC) |
| `permission/layer.go` | Permission layer (80 LOC) |
| `permission/rules.go` | Default rules (43 LOC) |
