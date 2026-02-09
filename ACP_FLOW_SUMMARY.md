# ACP (Agent Client Protocol) Flow - Comprehensive Overview

## What is ACP?

ACP (Agent Client Protocol) is a JSON-RPC 2.0 protocol over stdio that standardizes communication between editor/app and AI agents. It's often described as "LSP for AI agents."

## Architecture Overview

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                              CCUI Application                                │
│  ┌──────────────┐    ┌──────────────┐    ┌──────────────────────────────┐  │
│  │   Frontend   │    │   App (Go)   │    │      ACP Backend Package      │  │
│  │  (Svelte)    │◄──►│   (Wails)    │◄──►│  ┌────────────────────────┐  │  │
│  │              │    │              │    │  │    ACPBackend          │  │  │
│  │ - App.svelte │    │ - Session    │    │  │  ┌──────────────────┐  │  │  │
│  │ - ToolCard   │    │   management │    │  │  │  StdioTransport  │  │  │  │
│  │ - Terminal   │    │ - Event      │    │  │  │  ┌────────────┐  │  │  │  │
│  │              │    │   bridging   │    │  │  │  │   Client   │  │  │  │  │
│  └──────────────┘    └──────────────┘    │  │  │  │  ┌──────┐  │  │  │  │  │
│         ▲                                    │  │  │  │  Tool │  │  │  │  │  │
│         │ Wails Events                       │  │  │  │  Mgr  │  │  │  │  │  │
│         │                                    │  │  │  └──┬───┘  │  │  │  │  │
│         └────────────────────────────────────┘  │  └─────┼──────┘  │  │  │  │
│                                                  │        │         │  │  │  │
└──────────────────────────────────────────────────┼────────┼─────────┼──┼──┼──┘
                                                   │        │         │  │  │
                                                   │    ┌───┴───┐     │  │  │
                                                   │    │       │     │  │  │
                                                   ▼    ▼       ▼     ▼  ▼  ▼
                                              ┌────────────────────────────────┐
                                              │     claude-code-acp (v0.13.1)   │
                                              │      (@zed-industries)          │
                                              │  ┌────────────────────────────┐  │
                                              │  │  JSON-RPC 2.0 over stdio   │  │
                                              │  │  - initialize              │  │
                                              │  │  - session/new             │  │
                                              │  │  - session/prompt          │  │
                                              │  │  - session/update          │  │
                                              │  │  - session/request_perm    │  │
                                              │  │  - session/cancel          │  │
                                              │  └────────────────────────────┘  │
                                              │              │                   │
                                              │              ▼                   │
                                              │  ┌────────────────────────────┐  │
                                              │  │      Claude LLM API        │  │
                                              │  │    (Anthropic/Bedrock)     │  │
                                              │  └────────────────────────────┘  │
                                              └────────────────────────────────┘
```

## Core Components

### 1. Transport Layer (`backend/acp/transport.go`)

The `StdioTransport` handles all JSON-RPC communication:

```go
type Transport interface {
    Send(method string, params any) (json.RawMessage, error)  // Request-response
    Notify(method string, params any)                          // One-way notification
    Respond(id *int, result json.RawMessage)                  // Send response
    OnMethod(handler func(method string, params json.RawMessage, id *int))
    Close() error
}
```

**Key Features:**
- Bidirectional communication over stdin/stdout pipes
- Request-response correlation via message IDs
- Concurrent request handling with callback routing
- **Critical**: Checks `Method` BEFORE `ID` to handle permission requests correctly

### 2. Client (`backend/acp/client.go`)

The `Client` implements the `backend.Session` interface and manages:

```go
type Client struct {
    transport       Transport
    sessionID       string
    eventChan       chan<- backend.Event
    toolManager     *backend.ToolCallManager
    fileChangeStore *backend.FileChangeStore
    toolAdapters    []ToolEventAdapter
    permissionRespCh chan string
    autoPermission   bool
    currentModeID    string
    availableModes   []backend.SessionMode
}
```

### 3. Backend (`backend/acp/backend.go`)

Factory that creates ACP sessions:

```go
type ACPBackend struct {
    ctx    context.Context
    apiKey string
}

func (b *ACPBackend) NewSession(ctx context.Context, opts backend.SessionOpts) (backend.Session, error) {
    // 1. Spawn claude-code-acp subprocess
    // 2. Create StdioTransport
    // 3. Create Client
    // 4. Initialize handshake
    // 5. Create session
}
```

### 4. Adapters (`backend/acp/adapters.go`)

Tool event adapters handle different ACP backend formats:

- **ClaudeCodeAdapter**: Handles Claude Code specific metadata (`_meta.claudeCode`)
- **OpenCodeAdapter**: Handles OpenCode format with diff blocks

## Protocol Flow

### 1. Initialization Sequence

```
Client                              Agent (claude-code-acp)
  │                                         │
  ├── initialize ──────────────────────────►│
  │   {"protocolVersion":1,"clientCapabilities":{}}│
  │                                         │
  │◄────────────────── initialize result ──┤
  │   {"protocolVersion":1,"agentCapabilities":{...},
  │    "agentInfo":{"name":"@zed-industries/claude-code-acp",
  │                 "version":"0.13.1"}}    │
  │                                         │
  ├── session/new ─────────────────────────►│
  │   {"cwd":"/path","mcpServers":[...]}    │
  │                                         │
  │◄────────────────── session/new result ─┤
  │   {"sessionId":"uuid","modes":{...}}    │
  │                                         │
  │◄── session/update (available_commands)─┤
  │   {"sessionUpdate":"available_commands_update",
  │    "availableCommands":[...]}           │
```

### 2. Prompt Flow

```
Client                              Agent
  │                                         │
  ├── session/prompt ──────────────────────►│
  │   {"sessionId":"uuid","prompt":[{"type":"text",
  │    "text":"Hello"}],"allowedTools":[...]}│
  │                                         │
  │◄── session/update ─────────────────────┤
  │   {"sessionUpdate":"agent_message_chunk",
  │    "content":{"type":"text","text":"Hi"}} │
  │                                         │
  │◄── session/update ─────────────────────┤ (repeated)
  │   ...more chunks...                     │
  │                                         │
  │◄── session/update ─────────────────────┤ (if tool needed)
  │   {"sessionUpdate":"tool_call",          │
  │    "toolCallId":"...","title":"Read",    │
  │    "status":"pending","kind":"read"}     │
  │                                         │
  │◄── session/request_permission ─────────┤ (if permission needed)
  │   {"toolCall":{...},"options":[...]}     │
  │                                         │
  ├── response ────────────────────────────►│ (user allows)
  │   {"outcome":{"outcome":"selected",      │
  │    "optionId":"allow_once"}}            │
  │                                         │
  │◄── session/update ─────────────────────┤
  │   {"sessionUpdate":"tool_call_update",   │
  │    "status":"in_progress"}              │
  │                                         │
  │◄── session/update ─────────────────────┤
  │   {"sessionUpdate":"tool_call_update",   │
  │    "status":"completed","output":[...]}  │
  │                                         │
  │◄── session/prompt result ──────────────┤
  │   {"sessionId":"...","stopReason":"end_turn"}
```

### 3. Message Types

#### Client → Agent (Requests)

| Method | Purpose |
|--------|---------|
| `initialize` | Protocol handshake, capability negotiation |
| `session/new` | Create new session with cwd and MCP servers |
| `session/prompt` | Send user message/prompt |
| `session/cancel` | Cancel current operation |
| `session/set_mode` | Change session mode |

#### Agent → Client (Notifications/Requests)

| Method | Purpose |
|--------|---------|
| `session/update` | Streaming updates (text chunks, tool calls, plan updates) |
| `session/request_permission` | Request user approval for tool execution |

### 4. Session Update Types

```go
// Text streaming
{"sessionUpdate": "agent_message_chunk", "content": {"type": "text", "text": "..."}}
{"sessionUpdate": "agent_thought_chunk", "content": {"type": "text", "text": "..."}}

// Tool calls
{"sessionUpdate": "tool_call", "toolCallId": "...", "title": "Read", 
 "kind": "read", "status": "pending", "_meta": {"claudeCode": {"toolName": "Read"}}}

{"sessionUpdate": "tool_call_update", "toolCallId": "...", "status": "completed",
 "output": [...], "_meta": {"claudeCode": {"toolResponse": {...}}}}

// Plan updates
{"sessionUpdate": "plan", "entries": [{"content": "...", "priority": "high", "status": "pending"}]}

// Mode changes
{"sessionUpdate": "current_mode_update", "modeId": "plan"}

// Available commands
{"sessionUpdate": "available_commands_update", "availableCommands": [...]}
```

## Event Flow to Frontend

```
┌─────────────┐     ┌─────────────┐     ┌─────────────┐     ┌─────────────┐
│  ACP Agent  │────►│  Transport  │────►│   Client    │────►│   Events    │
│             │     │  readLoop   │     │ handleMethod│     │  eventChan  │
└─────────────┘     └─────────────┘     └──────┬──────┘     └──────┬──────┘
                                               │                    │
                                               ▼                    ▼
                                         ┌─────────────┐     ┌─────────────┐
                                         │ ToolManager │     │  App.bridge │
                                         │  (tracking) │     │   Events    │
                                         └─────────────┘     └──────┬──────┘
                                                                    │
                                                                    ▼
                                                              ┌─────────────┐
                                                              │ Wails Events │
                                                              │  (frontend)  │
                                                              └─────────────┘
```

### Event Types

```go
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
```

## Permission Flow

```
┌─────────┐     ┌──────────┐     ┌────────────┐     ┌──────────┐
│  Agent  │────►│  Client  │────►│   Events   │────►│ Frontend │
│         │     │          │     │            │     │          │
└─────────┘     └────┬─────┘     └────────────┘     └────┬─────┘
                     │                                    │
                     │ handlePermissionRequest            │
                     │                                    │
                     ├── Update tool state                │
                     │   (status: awaiting_permission)    │
                     │                                    │
                     ├── Emit permission request event    │
                     │                                    │
                     ├── Wait for permissionRespCh ◄─────┤ User clicks
                     │                                    │ allow/deny
                     │                                    │
                     ├── sendPermissionResponse ─────────►│
                     │   (outcome: selected)              │
                     │                                    │
                     ▼                                    ▼
```

## File Change Tracking

The `FileChangeStore` accumulates file modifications across tool calls:

```go
type FileChange struct {
    FilePath        string
    OriginalContent string
    CurrentContent  string
    Hunks           []PatchHunk
}

// Coalesces multiple edits to the same file
func (s *FileChangeStore) RecordChange(filePath, original, current string, hunks []PatchHunk) {
    if existing, ok := s.changes[filePath]; ok {
        existing.CurrentContent = current  // Update to latest
        existing.Hunks = hunks
    } else {
        s.changes[filePath] = &FileChange{...}
    }
}
```

## Session Modes

ACP supports multiple session modes:

| Mode | Description |
|------|-------------|
| `default` | Standard behavior, prompts for dangerous operations |
| `acceptEdits` | Auto-accept file edit operations |
| `plan` | Planning mode, no actual tool execution |
| `dontAsk` | Don't prompt for permissions, deny if not pre-approved |
| `bypassPermissions` | Bypass all permission checks |

## MCP Integration

MCP servers are configured in `session/new`:

```go
McpServers: []McpServer{
    {
        Name:    "ccui",
        Type:    "sse",
        URL:     "http://127.0.0.1:PORT/sse",
        Headers: []Header{},
    },
}
```

The CCUI app exposes its own MCP server for `ask_user_question` functionality.

## Key Implementation Details

### 1. Message Routing (Critical!)

```go
// WRONG - drops permission requests
if msg.ID != nil { ... }  // Response
else if msg.Method != "" { ... }  // Request/notification

// CORRECT - handles both
if msg.Method != "" { ... }  // Request or notification from agent
else if msg.ID != nil { ... }  // Response to our request
```

### 2. Tool Call Hierarchy

Task tools create parent-child relationships:

```go
if toolName == "Task" {
    c.toolManager.PushParent(u.ToolCallID)  // Children attach here
}

// On completion
if state.ToolName == "Task" && isTerminalStatus(u.Status) {
    c.toolManager.PopParent(u.ToolCallID)
}
```

### 3. Auto-Permission Mode

For automation/review agents:

```go
if c.autoPermission {
    c.sendPermissionResponse(id, "allow_always")
    return
}
```

### 4. Suppressed Tool Events

For review mode (tracks file changes without UI noise):

```go
if c.suppressToolEvents {
    if toolResponse != nil {
        c.trackFileChange(toolName, toolResponse)  // Still track changes
    }
    return  // But don't emit tool_state events
}
```

## Testing

The ACP package includes comprehensive tests:

- `transport_test.go`: Tests JSON-RPC routing, callbacks, errors
- `client_test.go`: Tests message handling, tool calls, permissions, modes
- `adapters_test.go`: Tests tool event parsing

## Log Files

ACP logs are stored in `.acp-logs/` with detailed protocol traces:

```json
{"type":"send","data":{"jsonrpc":"2.0","id":1,"method":"initialize",...}}
{"type":"raw_message","data":{"jsonrpc":"2.0","id":1,"result":{...}}}
{"type":"method","data":{"method":"session/update","params":{...}}}
{"type":"session_update","data":{"sessionUpdate":"agent_message_chunk",...}}
```

## Package Location

```
backend/
├── acp/
│   ├── backend.go       # ACPBackend factory
│   ├── client.go        # Client implementation
│   ├── transport.go     # StdioTransport
│   ├── types.go         # ACP protocol types
│   ├── adapters.go      # Tool event adapters
│   └── *_test.go        # Tests
├── interface.go         # Backend/Session interfaces
└── types.go             # Shared types (ToolState, FileChange, etc.)
```

## References

- ACP Spec: https://agentclientprotocol.com
- NPM Package: `@zed-industries/claude-code-acp` (v0.13.1)
- GitHub: https://github.com/zed-industries/claude-code-acp
