# ACP Protocol Reference

Complete Agent Client Protocol specification for Go integration.

## 1. Initialization

### Handshake Sequence

```
Client                              Agent
  │                                   │
  ├── initialize ────────────────────>│
  │   {protocolVersion, capabilities} │
  │                                   │
  │<───────────── initialize result ──┤
  │   {protocolVersion, capabilities} │
  │                                   │
  ├── session/new ───────────────────>│
  │   {cwd, mcpServers, ...}          │
  │                                   │
  │<───────────── session/new result ─┤
  │   {sessionId}                     │
  │                                   │
  │   Ready for session/prompt        │
```

### Initialize Request

```go
type InitializeParams struct {
    ProtocolVersion    int                `json:"protocolVersion"` // Currently 1
    ClientInfo         *ClientInfo        `json:"clientInfo,omitempty"`
    ClientCapabilities ClientCapabilities `json:"clientCapabilities"`
}

type ClientInfo struct {
    Name    string `json:"name"`
    Title   string `json:"title,omitempty"`
    Version string `json:"version"`
}

type ClientCapabilities struct {
    FS       *FSCapabilities `json:"fs,omitempty"`
    Terminal bool            `json:"terminal,omitempty"`
}

type FSCapabilities struct {
    ReadTextFile  bool `json:"readTextFile"`
    WriteTextFile bool `json:"writeTextFile"`
}
```

### Initialize Response

```go
type InitializeResult struct {
    ProtocolVersion   int               `json:"protocolVersion"`
    AgentInfo         *AgentInfo        `json:"agentInfo,omitempty"`
    AgentCapabilities AgentCapabilities `json:"agentCapabilities"`
}

type AgentInfo struct {
    Name    string `json:"name"`
    Title   string `json:"title,omitempty"`
    Version string `json:"version"`
}

type AgentCapabilities struct {
    LoadSession        bool                `json:"loadSession,omitempty"`
    PromptCapabilities *PromptCapabilities `json:"promptCapabilities,omitempty"`
    MCP                *MCPCapabilities    `json:"mcp,omitempty"`
}

type PromptCapabilities struct {
    Content []string `json:"content"` // text, image, audio, embeddedContext
}

type MCPCapabilities struct {
    HTTPTransport bool `json:"httpTransport,omitempty"`
    SSETransport  bool `json:"sseTransport,omitempty"`
}
```

**Rule**: Omitted capabilities = unsupported. No defaults assumed.

---

## 2. Session Management

### session/new

```go
type SessionNewParams struct {
    Cwd        string      `json:"cwd"`        // Required
    McpServers []McpServer `json:"mcpServers"` // Required (can be empty [])
}

type McpServer struct {
    Name    string   `json:"name"`
    Command string   `json:"command"`
    Args    []string `json:"args"`
    Env     []EnvVar `json:"env"`
}

type EnvVar struct {
    Name  string `json:"name"`
    Value string `json:"value"`
}

type SessionNewResult struct {
    SessionID string `json:"sessionId"`
}
```

### session/load (Resume)

Only if agent advertises `loadSession: true`:

```go
type SessionLoadParams struct {
    SessionID  string         `json:"sessionId"`
    Cwd        string         `json:"cwd"`
    McpServers map[string]any `json:"mcpServers,omitempty"`
}
```

Agent replays history via `session/update` notifications before responding.

### session/cancel

Notification (no response):

```go
type SessionCancelParams struct {
    SessionID string `json:"sessionId"`
}
```

Agent returns `stopReason: "cancelled"` in pending prompt response.

---

## 3. Content Types

### Prompt Content (Client → Agent)

```go
type PromptContent struct {
    Type     string `json:"type"`               // text|image|audio|resource|resourceLink
    Text     string `json:"text,omitempty"`     // for text
    Data     string `json:"data,omitempty"`     // base64 for image/audio
    MimeType string `json:"mimeType,omitempty"` // image/png, audio/wav, etc.
    URI      string `json:"uri,omitempty"`      // for resource/resourceLink
    Name     string `json:"name,omitempty"`     // for resourceLink
    Size     int64  `json:"size,omitempty"`     // for resourceLink
}
```

### Content Type Requirements

| Type | Required Fields | Capability |
|------|-----------------|------------|
| text | text | none (mandatory) |
| image | data, mimeType | `image` |
| audio | data, mimeType | `audio` |
| resource | uri, (text OR blob) | `embeddedContext` |
| resourceLink | name, uri | none (mandatory) |

### Supported MIME Types

**Images**: image/png, image/jpeg, image/gif, image/webp, image/svg+xml
**Audio**: audio/wav, audio/mp3, audio/ogg, audio/flac, audio/aac

---

## 4. Tool Calls

### Tool Call Lifecycle

```
pending → in_progress → completed|failed|cancelled
```

### Tool Call Update (Agent → Client)

```go
type ToolCallUpdate struct {
    Kind       string         `json:"kind"`       // tool_call | tool_call_update
    ToolCallID string         `json:"toolCallId"`
    Title      string         `json:"title,omitempty"`
    ToolKind   string         `json:"toolKind,omitempty"` // read|edit|delete|move|search|execute|think|fetch|other
    Status     string         `json:"status"`             // pending|in_progress|completed|failed|cancelled
    Input      map[string]any `json:"input,omitempty"`
    Output     []OutputBlock  `json:"output,omitempty"`
}

type OutputBlock struct {
    Type    string       `json:"type"` // content|diff|terminal
    Content *TextContent `json:"content,omitempty"`
    Path    string       `json:"path,omitempty"`      // for diff
    OldContent string    `json:"oldContent,omitempty"`
    NewContent string    `json:"newContent,omitempty"`
}
```

### Tool Kinds (UI Hints)

| Kind | Purpose |
|------|---------|
| read | File reading |
| edit | File modifications |
| delete | Deletions |
| move | Moving/renaming |
| search | Grep/glob |
| execute | Bash commands |
| think | Internal reasoning |
| fetch | API/web requests |
| other | Custom |

---

## 5. Permissions

### Permission Request (Agent → Client)

```go
type PermissionRequest struct {
    SessionID string        `json:"sessionId"`
    ToolCall  ToolCallInfo  `json:"toolCall"`
    Options   []PermOption  `json:"options"`
}

type ToolCallInfo struct {
    ToolCallID string `json:"toolCallId"`
    Title      string `json:"title"`
    Kind       string `json:"kind"` // read|edit|delete|move|search|execute|think|fetch|other
}

type PermOption struct {
    OptionID string `json:"optionId"`
    Name     string `json:"name"`
    Kind     string `json:"kind"` // allow_once|allow_always|reject_once|reject_always
}
```

### Permission Response (Client → Agent)

```go
type PermissionResponse struct {
    Outcome PermissionOutcome `json:"outcome"`
}

type PermissionOutcome struct {
    Outcome  string `json:"outcome"`            // selected|cancelled
    OptionID string `json:"optionId,omitempty"` // required if selected
}
```

### Permission Modes

| Mode | Auto-Approve | Prompt For |
|------|--------------|------------|
| `default` | Nothing | All tools |
| `acceptEdits` | File ops (edit/write/mkdir/rm/mv) | Bash, external |
| `bypassPermissions` | Everything | Nothing (dangerous) |
| `plan` | Nothing | Nothing (no execution) |

### Allowed Tools Patterns

```go
AllowedTools: []string{
    "Bash(git *)",    // Wildcard: git commands only
    "Bash(npm test)", // Specific command
    "Read",           // All reads
    "Grep",           // All grep
}
```

---

## 6. Streaming

### session/update Notification

```go
type SessionUpdateParams struct {
    SessionID string        `json:"sessionId"`
    Update    SessionUpdate `json:"update"`
}

type SessionUpdate struct {
    // Text chunk
    SessionUpdate string       `json:"sessionUpdate,omitempty"` // agent_message_chunk|user_message_chunk|agent_thought_chunk
    Content *TextContent `json:"content,omitempty"`

    // Tool call (see Tool Calls section)
    SessionUpdate string `json:"sessionUpdate,omitempty"` // tool_call|tool_call_update
    ToolCallID    string `json:"toolCallId,omitempty"`
    Title         string `json:"title,omitempty"`
    Status        string `json:"status,omitempty"`

    // Plan
    Plan *AgentPlan `json:"plan,omitempty"`
}

type AgentPlan struct {
    Description string   `json:"description"`
    Steps       []string `json:"steps"`
}
```

### Prompt Response

```go
type SessionPromptResult struct {
    SessionID  string    `json:"sessionId"`
    Messages   []Message `json:"messages,omitempty"`
    StopReason string    `json:"stopReason"` // end_turn|max_tokens|cancelled|refusal
}
```

### Stop Reasons

| Reason | Meaning |
|--------|---------|
| `end_turn` | Completed normally |
| `max_tokens` | Hit token limit |
| `cancelled` | Client cancelled |
| `refusal` | Agent refused |

---

## 7. MCP Integration

### MCP Server Config in session/new

```go
McpServers: map[string]any{
    "filesystem": map[string]any{
        "type":    "stdio",           // stdio (required) | http | sse
        "command": "/path/to/server",
        "args":    []string{"--stdio"},
        "env":     map[string]string{},
    },
    "api": map[string]any{
        "type":    "http",
        "url":     "https://mcp.example.com",
        "headers": map[string]string{"Authorization": "Bearer xxx"},
    },
}
```

### ACP vs MCP

| Aspect | ACP | MCP |
|--------|-----|-----|
| Purpose | Editor ↔ Agent | Agent ↔ Tools |
| Direction | Bidirectional | Agent initiates |
| Transport | stdio | stdio/http/sse |
| Lifecycle | Wraps agent | Provides tools |

MCP servers are embedded within ACP sessions. Agent manages MCP connections internally.

---

## 8. Error Handling

### Error Response

```go
type RPCError struct {
    Code    int    `json:"code"`
    Message string `json:"message"`
    Data    any    `json:"data,omitempty"`
}
```

### Error Codes

| Code | Name | Recoverable | Action |
|------|------|-------------|--------|
| -32700 | Parse Error | No | Fix JSON |
| -32600 | Invalid Request | No | Fix structure |
| -32601 | Method Not Found | No | Check method |
| -32602 | Invalid Params | No | Fix params |
| -32603 | Internal Error | Maybe | Investigate |
| -32000 | Connection Closed | Yes | Reconnect |
| -32001 | Request Timeout | Yes | Retry w/ backoff |
| -32002 | Permission Denied | No | User rejected |
| -32003 | Session Expired | Yes | New session |

### Retry Strategy

```go
func shouldRetry(code int) bool {
    return code == -32000 || code == -32001 || code == -32003
}

// Exponential backoff: 1s, 2s, 4s, 8s (max 5 attempts)
delay := time.Duration(math.Pow(2, float64(attempt))) * time.Second
```

---

## 9. Complete Message Flow

```
1. Client spawns claude-code-acp subprocess
2. Client sends: initialize
3. Agent responds: initialize result (capabilities)
4. Client sends: session/new {cwd, mcpServers}
5. Agent responds: {sessionId}
6. Client sends: session/prompt {sessionId, prompt}
7. Agent sends: session/update (streaming chunks)
8. Agent sends: session/update (tool_call, status: pending)
9. Agent sends: session/request_permission
10. Client responds: {outcome: selected, optionId: allow_once}
11. Agent sends: session/update (tool_call_update, status: in_progress)
12. Agent sends: session/update (tool_call_update, status: completed)
13. Agent sends: session/update (more text chunks)
14. Agent responds to prompt: {stopReason: end_turn}
15. Loop to step 6 for next prompt
```

---

## Resources

- Spec: https://agentclientprotocol.com
- GitHub: https://github.com/agentclientprotocol/agent-client-protocol
- MCP: https://modelcontextprotocol.io
