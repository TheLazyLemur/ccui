# claude-code-acp Integration (Go)

ACP adapter for Claude Code. Spawns as subprocess, communicates via stdio JSON-RPC.

## Package

```
npm: @zed-industries/claude-code-acp
binary: claude-code-acp (or cc-acp via bun)
version: 0.13.1
license: Apache 2.0
repo: github.com/zed-industries/claude-code-acp
```

## Install

```bash
npm install -g @zed-industries/claude-code-acp
```

**WARNING:** There's an old unrelated package called `claude-code-acp` (v0.1.1) - don't use it. Always use `@zed-industries/claude-code-acp`.

## Environment

```bash
ANTHROPIC_API_KEY=sk-...  # Required
# Optional backends:
CLAUDE_CODE_USE_BEDROCK=1
CLAUDE_CODE_USE_VERTEX=1
```

## Go Integration

### Types

```go
type JSONRPCMessage struct {
    JSONRPC string          `json:"jsonrpc"`
    ID      *int            `json:"id,omitempty"`      // nil for notifications
    Method  string          `json:"method,omitempty"`
    Params  json.RawMessage `json:"params,omitempty"`
    Result  json.RawMessage `json:"result,omitempty"`
    Error   *RPCError       `json:"error,omitempty"`
}

type RPCError struct {
    Code    int    `json:"code"`
    Message string `json:"message"`
}

type InitializeParams struct {
    ProtocolVersion    int                `json:"protocolVersion"` // Currently 1
    ClientCapabilities ClientCapabilities `json:"clientCapabilities"`
}

type ClientCapabilities struct {
    FS       *FSCapabilities `json:"fs,omitempty"`
    Terminal bool            `json:"terminal,omitempty"`
}

type FSCapabilities struct {
    ReadTextFile  bool `json:"readTextFile"`
    WriteTextFile bool `json:"writeTextFile"`
}

// NOTE: ClientCapabilities controls WHO handles operations:
// - If true: Agent sends requests to YOUR client (fs/read_text_file, fs/write_text_file)
// - If false/omitted: Agent handles it internally via its own tools
// For simple clients, set all to false and let agent handle everything.

type SessionNewParams struct {
    Cwd        string      `json:"cwd"`
    McpServers []McpServer `json:"mcpServers"` // Required (can be empty [])
}

type McpServer struct {
    Name    string      `json:"name"`
    Command string      `json:"command"`
    Args    []string    `json:"args"`
    Env     []EnvVar    `json:"env"`
}

type EnvVar struct {
    Name  string `json:"name"`
    Value string `json:"value"`
}

type SessionNewResult struct {
    SessionID string `json:"sessionId"`
}

type PromptContent struct {
    Type     string `json:"type"` // text|resource|image
    Text     string `json:"text,omitempty"`
    URI      string `json:"uri,omitempty"`
    MimeType string `json:"mimeType,omitempty"`
    Data     string `json:"data,omitempty"` // base64 for images
}

type SessionPromptParams struct {
    SessionID string          `json:"sessionId"`
    Prompt    []PromptContent `json:"prompt"`
}

type SessionUpdate struct {
    SessionID string       `json:"sessionId"`
    Update    UpdateContent `json:"update"`
}

type UpdateContent struct {
    // Message streaming
    SessionUpdate string       `json:"sessionUpdate,omitempty"` // agent_message_chunk|agent_thought_chunk
    Content *TextContent `json:"content,omitempty"`

    // Tool calls
    SessionUpdate string         `json:"sessionUpdate,omitempty"` // tool_call|tool_call_update
    ToolCallID    string         `json:"toolCallId,omitempty"`
    Title         string         `json:"title,omitempty"`
    ToolKind      string         `json:"toolKind,omitempty"` // read|edit|execute|search|fetch|think|other
    Status        string         `json:"status,omitempty"`   // pending|in_progress|completed|failed|cancelled
    Input         map[string]any `json:"input,omitempty"`
    Output        []OutputBlock  `json:"output,omitempty"`

    // Plan updates
    Plan *AgentPlan `json:"plan,omitempty"`
}

type OutputBlock struct {
    Type       string       `json:"type"` // content|diff|terminal
    Content    *TextContent `json:"content,omitempty"`
    Path       string       `json:"path,omitempty"`
    OldContent string       `json:"oldContent,omitempty"`
    NewContent string       `json:"newContent,omitempty"`
    TerminalID string       `json:"terminalId,omitempty"`
    ExitCode   *int         `json:"exitCode,omitempty"`
}

type AgentPlan struct {
    Description string   `json:"description"`
    Steps       []string `json:"steps"`
}

type TextContent struct {
    Type string `json:"type"`
    Text string `json:"text"`
}

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

type PermissionResponse struct {
    Outcome PermissionOutcome `json:"outcome"`
}

type PermissionOutcome struct {
    Outcome  string `json:"outcome"`            // selected|cancelled
    OptionID string `json:"optionId,omitempty"` // required if selected
}

// Prompt response (final result after streaming)
type SessionPromptResult struct {
    SessionID  string `json:"sessionId"`
    StopReason string `json:"stopReason"` // end_turn|max_tokens|cancelled|refusal
}

// Audio content (requires agent audio capability)
type AudioContent struct {
    Type     string `json:"type"` // audio
    Data     string `json:"data"` // base64
    MimeType string `json:"mimeType"` // audio/wav|audio/mp3|audio/ogg
}

// Resource link (agent fetches independently)
type ResourceLink struct {
    Type        string `json:"type"` // resourceLink
    Name        string `json:"name"`
    URI         string `json:"uri"`
    Description string `json:"description,omitempty"`
    MimeType    string `json:"mimeType,omitempty"`
    Size        int64  `json:"size,omitempty"`
}

// Embedded resource (full content included)
type Resource struct {
    Type     string `json:"type"` // resource
    URI      string `json:"uri"`
    Text     string `json:"text,omitempty"` // for text resources
    Blob     string `json:"blob,omitempty"` // base64 for binary
    MimeType string `json:"mimeType,omitempty"`
}
```

### Client Implementation

```go
type ACPClient struct {
    cmd       *exec.Cmd
    stdin     io.WriteCloser
    stdout    *bufio.Scanner
    sessionID string
    msgID     int
    mu        sync.Mutex
    callbacks map[int]chan JSONRPCMessage
    updates   chan SessionUpdate
}

func NewACPClient(ctx context.Context, cwd string) (*ACPClient, error) {
    cmd := exec.CommandContext(ctx, "cc-acp")
    cmd.Env = append(os.Environ(), "ANTHROPIC_API_KEY="+os.Getenv("ANTHROPIC_API_KEY"))

    stdin, _ := cmd.StdinPipe()
    stdout, _ := cmd.StdoutPipe()

    if err := cmd.Start(); err != nil {
        return nil, err
    }

    c := &ACPClient{
        cmd:       cmd,
        stdin:     stdin,
        stdout:    bufio.NewScanner(stdout),
        callbacks: make(map[int]chan JSONRPCMessage),
        updates:   make(chan SessionUpdate, 100),
    }

    go c.readLoop()

    // Initialize
    if err := c.initialize(); err != nil {
        return nil, err
    }

    // Create session
    if err := c.newSession(cwd); err != nil {
        return nil, err
    }

    return c, nil
}

func (c *ACPClient) readLoop() {
    for c.stdout.Scan() {
        line := c.stdout.Bytes()
        // Skip non-JSON lines (debug output)
        if len(line) == 0 || line[0] != '{' {
            continue
        }

        var msg JSONRPCMessage
        if err := json.Unmarshal(line, &msg); err != nil {
            continue
        }

        // CRITICAL: Check Method BEFORE ID!
        // Requests from agent have BOTH Method AND ID.
        // If you check ID first, permission requests get routed as responses and dropped.
        if msg.Method != "" {
            // Request or notification from agent
            if msg.Method == "session/update" {
                var update SessionUpdate
                json.Unmarshal(msg.Params, &update)
                c.updates <- update
            } else if msg.Method == "session/request_permission" {
                c.respondPermission(msg)
            }
            // Add handlers for fs/read_text_file, fs/write_text_file if you
            // advertised those capabilities (otherwise agent handles them)
        } else if msg.ID != nil {
            // Response to OUR request
            c.mu.Lock()
            if ch, ok := c.callbacks[*msg.ID]; ok {
                ch <- msg
                delete(c.callbacks, *msg.ID)
            }
            c.mu.Unlock()
        }
    }
}

func (c *ACPClient) send(method string, params any) (JSONRPCMessage, error) {
    c.mu.Lock()
    c.msgID++
    id := c.msgID
    ch := make(chan JSONRPCMessage, 1)
    c.callbacks[id] = ch
    c.mu.Unlock()

    paramsJSON, _ := json.Marshal(params)
    msg := JSONRPCMessage{
        JSONRPC: "2.0",
        ID:      &id,
        Method:  method,
        Params:  paramsJSON,
    }

    data, _ := json.Marshal(msg)
    c.stdin.Write(append(data, '\n'))

    return <-ch, nil
}

func (c *ACPClient) notify(method string, params any) {
    paramsJSON, _ := json.Marshal(params)
    msg := JSONRPCMessage{
        JSONRPC: "2.0",
        Method:  method,
        Params:  paramsJSON,
    }
    data, _ := json.Marshal(msg)
    c.stdin.Write(append(data, '\n'))
}

func (c *ACPClient) initialize() error {
    _, err := c.send("initialize", InitializeParams{
        ProtocolVersion: 1,
        ClientCapabilities: ClientCapabilities{
            FS:       &FSCapabilities{ReadTextFile: true, WriteTextFile: true},
            Terminal: true,
        },
    })
    return err
}

func (c *ACPClient) newSession(cwd string) error {
    resp, err := c.send("session/new", SessionNewParams{
        Cwd:        cwd,
        McpServers: []McpServer{}, // Required even if empty
    })
    if err != nil {
        return err
    }

    var result SessionNewResult
    json.Unmarshal(resp.Result, &result)
    c.sessionID = result.SessionID
    return nil
}

func (c *ACPClient) SendPrompt(text string) (SessionPromptResult, error) {
    resp, err := c.send("session/prompt", SessionPromptParams{
        SessionID: c.sessionID,
        Prompt:    []PromptContent{{Type: "text", Text: text}},
    })
    if err != nil {
        return SessionPromptResult{}, err
    }
    var result SessionPromptResult
    json.Unmarshal(resp.Result, &result)
    return result, nil
}

// SendPromptWithImage sends text + image
func (c *ACPClient) SendPromptWithImage(text string, imageData []byte, mimeType string) (SessionPromptResult, error) {
    resp, err := c.send("session/prompt", SessionPromptParams{
        SessionID: c.sessionID,
        Prompt: []PromptContent{
            {Type: "text", Text: text},
            {Type: "image", Data: base64.StdEncoding.EncodeToString(imageData), MimeType: mimeType},
        },
    })
    if err != nil {
        return SessionPromptResult{}, err
    }
    var result SessionPromptResult
    json.Unmarshal(resp.Result, &result)
    return result, nil
}

// SendPromptWithFile sends text + file content
func (c *ACPClient) SendPromptWithFile(text string, uri string, content string, mimeType string) (SessionPromptResult, error) {
    resp, err := c.send("session/prompt", SessionPromptParams{
        SessionID: c.sessionID,
        Prompt: []PromptContent{
            {Type: "text", Text: text},
            {Type: "resource", URI: uri, Text: content, MimeType: mimeType},
        },
    })
    if err != nil {
        return SessionPromptResult{}, err
    }
    var result SessionPromptResult
    json.Unmarshal(resp.Result, &result)
    return result, nil
}

func (c *ACPClient) Updates() <-chan SessionUpdate {
    return c.updates
}

func (c *ACPClient) Cancel() {
    c.notify("session/cancel", map[string]string{"sessionId": c.sessionID})
}

func (c *ACPClient) Close() error {
    c.stdin.Close()
    return c.cmd.Wait()
}

// respondPermission - Interactive version (prompts user)
func (c *ACPClient) respondPermission(msg JSONRPCMessage) {
    var req PermissionRequest
    json.Unmarshal(msg.Params, &req)

    // Show permission prompt
    fmt.Fprintf(os.Stderr, "\n┌─ Permission Request ─────────────────\n")
    fmt.Fprintf(os.Stderr, "│ %s\n", req.ToolCall.Title)
    fmt.Fprintf(os.Stderr, "├──────────────────────────────────────\n")
    for i, opt := range req.Options {
        fmt.Fprintf(os.Stderr, "│ %d) %s\n", i+1, opt.Name)
    }
    fmt.Fprintf(os.Stderr, "└──────────────────────────────────────\n")
    fmt.Fprintf(os.Stderr, "Select [1-%d]: ", len(req.Options))

    // Read user choice
    var choice int
    fmt.Scanf("%d", &choice)

    var optionID string
    if choice >= 1 && choice <= len(req.Options) {
        optionID = req.Options[choice-1].OptionID
    } else {
        // Default to first allow_once
        for _, opt := range req.Options {
            if opt.Kind == "allow_once" {
                optionID = opt.OptionID
                break
            }
        }
    }

    c.sendPermissionResponse(msg.ID, optionID)
}

// Auto-approve version (for autonomous mode)
func (c *ACPClient) respondPermissionAuto(msg JSONRPCMessage) {
    var req PermissionRequest
    json.Unmarshal(msg.Params, &req)

    // Find first allow option
    var optionID string
    for _, opt := range req.Options {
        if opt.Kind == "allow_always" {
            optionID = opt.OptionID
            break
        }
        if opt.Kind == "allow_once" && optionID == "" {
            optionID = opt.OptionID
        }
    }

    c.sendPermissionResponse(msg.ID, optionID)
}

func (c *ACPClient) sendPermissionResponse(id *int, optionID string) {
    resp := JSONRPCMessage{
        JSONRPC: "2.0",
        ID:      id,
    }
    result, _ := json.Marshal(PermissionResponse{
        Outcome: PermissionOutcome{Outcome: "selected", OptionID: optionID},
    })
    resp.Result = result
    data, _ := json.Marshal(resp)
    c.stdin.Write(append(data, '\n'))
}
```

### Wails Integration

```go
// app.go
type App struct {
    ctx    context.Context
    client *ACPClient
}

func (a *App) StartChat(cwd string) error {
    client, err := NewACPClient(a.ctx, cwd)
    if err != nil {
        return err
    }
    a.client = client

    // Forward updates to frontend via Wails events
    go func() {
        for update := range client.Updates() {
            u := update.Update

            // Text streaming
            if u.SessionUpdate == "agent_message_chunk" && u.Content != nil {
                runtime.EventsEmit(a.ctx, "chat:chunk", u.Content.Text)
            }

            // Thinking/reasoning
            if u.Kind == "agent_thought_chunk" && u.Content != nil {
                runtime.EventsEmit(a.ctx, "chat:thought", u.Content.Text)
            }

            // Tool call started
            if u.SessionUpdate == "tool_call" {
                runtime.EventsEmit(a.ctx, "chat:tool", map[string]any{
                    "id":     u.ToolCallID,
                    "title":  u.Title,
                    "kind":   u.ToolKind,
                    "status": u.Status,
                    "input":  u.Input,
                })
            }

            // Tool call progress/completion
            if u.SessionUpdate == "tool_call_update" {
                runtime.EventsEmit(a.ctx, "chat:tool:update", map[string]any{
                    "id":     u.ToolCallID,
                    "status": u.Status,
                    "output": u.Output,
                })
            }

            // Plan update
            if u.Plan != nil {
                runtime.EventsEmit(a.ctx, "chat:plan", map[string]any{
                    "description": u.Plan.Description,
                    "steps":       u.Plan.Steps,
                })
            }
        }
    }()

    return nil
}

func (a *App) SendMessage(text string) (string, error) {
    if a.client == nil {
        return "", errors.New("chat not started")
    }
    result, err := a.client.SendPrompt(text)
    if err != nil {
        return "", err
    }
    return result.StopReason, nil
}

func (a *App) CancelChat() {
    if a.client != nil {
        a.client.Cancel()
    }
}

func (a *App) StopChat() error {
    if a.client != nil {
        return a.client.Close()
    }
    return nil
}
```

### Frontend (Svelte)

```svelte
<script>
import { EventsOn } from '../wailsjs/runtime/runtime';
import { StartChat, SendMessage, CancelChat, StopChat } from '../wailsjs/go/main/App';

let messages = [];
let currentChunk = '';
let tools = new Map();
let plan = null;
let input = '';
let loading = false;

// Text streaming (accumulate chunks)
EventsOn('chat:chunk', (text) => {
    currentChunk += text;
});

// Thinking/reasoning
EventsOn('chat:thought', (text) => {
    // Optionally show reasoning
});

// Tool started
EventsOn('chat:tool', (tool) => {
    tools.set(tool.id, tool);
    tools = tools; // trigger reactivity
});

// Tool progress/completion
EventsOn('chat:tool:update', (update) => {
    const tool = tools.get(update.id);
    if (tool) {
        tool.status = update.status;
        tool.output = update.output;
        tools = tools;
    }
});

// Plan update
EventsOn('chat:plan', (p) => {
    plan = p;
});

async function send() {
    if (!input.trim() || loading) return;

    // Flush previous chunk to messages
    if (currentChunk) {
        messages = [...messages, { role: 'assistant', text: currentChunk }];
        currentChunk = '';
    }

    messages = [...messages, { role: 'user', text: input }];
    const prompt = input;
    input = '';
    loading = true;
    tools.clear();

    const stopReason = await SendMessage(prompt);

    // Flush final chunk
    if (currentChunk) {
        messages = [...messages, { role: 'assistant', text: currentChunk }];
        currentChunk = '';
    }

    loading = false;
}

function cancel() {
    CancelChat();
}
</script>
```

## Client Capabilities

Controls WHO handles operations - your client or the agent internally.

| Capability | If `true` | If `false`/omitted |
|------------|-----------|---------------------|
| `fs.readTextFile` | Agent sends `fs/read_text_file` requests to you | Agent uses its own Read tool |
| `fs.writeTextFile` | Agent sends `fs/write_text_file` requests to you | Agent uses its own Write tool |
| `terminal` | Agent sends terminal requests to you | Agent uses its own Bash tool |

**For simple clients:** Set all capabilities to `false` (or omit them). The agent handles everything internally, you just handle permissions and display output.

**For rich clients (like Zed):** Set to `true` to intercept operations - show diffs before writing, run commands in integrated terminal, etc.

If you advertise a capability but don't handle the corresponding requests, operations will hang!

## Permission Modes

| Mode | Behavior |
|------|----------|
| `default` | Prompts for all tool executions |
| `acceptEdits` | Auto-approves file ops, prompts others |
| `bypassPermissions` | Full autonomous (dangerous) |
| `plan` | Planning only, no execution |

## Available Tools

- Read, Write, Edit (file ops)
- Bash (shell)
- Glob, Grep (search)
- WebSearch, WebFetch (web)
- Task (subagents)

Restrict via `allowedTools` param.

## Error Handling

```go
// Check for RPC errors in responses
func (c *ACPClient) handleError(msg JSONRPCMessage) error {
    if msg.Error != nil {
        return fmt.Errorf("ACP error %d: %s", msg.Error.Code, msg.Error.Message)
    }
    return nil
}

// Retry logic for recoverable errors
func shouldRetry(code int) bool {
    switch code {
    case -32000, // Connection closed
         -32001, // Timeout
         -32003: // Session expired
        return true
    }
    return false
}
```

| Code | Name | Retry |
|------|------|-------|
| -32700 | Parse Error | No |
| -32600 | Invalid Request | No |
| -32601 | Method Not Found | No |
| -32602 | Invalid Params | No |
| -32000 | Connection Closed | Yes (reconnect) |
| -32001 | Request Timeout | Yes (backoff) |
| -32002 | Permission Denied | No |
| -32003 | Session Expired | Yes (new session) |

## Resources

- npm: https://www.npmjs.com/package/@zed-industries/claude-code-acp
- repo: https://github.com/zed-industries/claude-code-acp
- ACP spec: https://agentclientprotocol.com
