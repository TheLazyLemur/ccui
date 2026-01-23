# Agent Client Protocol (ACP)

"LSP for AI agents" - standardizes editor/app to AI agent communication.

## Two Protocols Named ACP

1. **Agent Client Protocol** (Editor-Agent) - JSON-RPC 2.0 over stdio
2. **Agent Communication Protocol** (Agent-to-Agent) - HTTP REST

For chat app integration, we use #1.

## Core Architecture

```
Chat App (Go/Wails)
       ↓
  JSON-RPC 2.0 over stdio
       ↓
claude-code-acp (subprocess)
       ↓
Claude LLM
```

## JSON-RPC 2.0 Message Format

**Request:**
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "session/prompt",
  "params": { "sessionId": "abc", "prompt": [...] }
}
```

**Response:**
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": { "sessionId": "abc", "stopReason": "end_turn" }
}
```

**Notification (no id, one-way):**
```json
{
  "jsonrpc": "2.0",
  "method": "session/update",
  "params": { "update": { "type": "text", "text": "..." } }
}
```

## Protocol Flow

```
1. Client → Agent: initialize (capability negotiation)
2. Client → Agent: session/new (create session)
3. Client → Agent: session/prompt (send user message)
4. Agent → Client: session/update (streaming responses, notifications)
5. Agent → Client: session/request_permission (tool approvals)
6. Client → Agent: session/cancel (interrupt)
```

## Key Methods

| Method | Direction | Purpose |
|--------|-----------|---------|
| `initialize` | Client→Agent | Negotiate capabilities |
| `session/new` | Client→Agent | Create new session |
| `session/prompt` | Client→Agent | Send user message |
| `session/update` | Agent→Client | Stream response/tool updates |
| `session/request_permission` | Agent→Client | Request tool approval |
| `session/cancel` | Client→Agent | Cancel current operation |

## Capability Negotiation

```json
{
  "method": "initialize",
  "params": {
    "protocolVersion": "0.12.0",
    "clientCapabilities": {
      "fs": { "readTextFile": true, "writeTextFile": true },
      "terminal": true
    }
  }
}
```

## Session Update Types

**Text content:**
```json
{ "content": { "type": "text", "text": "..." } }
```

**Tool call:**
```json
{
  "sessionUpdate": "tool_call",
  "toolCallId": "123",
  "title": "Read",
  "kind": "read",
  "status": "pending"
}
```

## Permission Request Flow

Agent asks:
```json
{
  "method": "session/request_permission",
  "id": 5,
  "params": {
    "title": "Execute bash",
    "description": "Run: npm test",
    "options": [
      { "id": "allow", "label": "Allow" },
      { "id": "deny", "label": "Deny" }
    ]
  }
}
```

Client responds:
```json
{
  "id": 5,
  "result": { "outcome": { "outcome": "selected", "optionId": "allow" } }
}
```

## Error Codes

| Code | Meaning |
|------|---------|
| -32700 | Parse error |
| -32600 | Invalid request |
| -32601 | Method not found |
| -32602 | Invalid params |
| -32001 | Resource not found |
| -32002 | Permission denied |

## ACP vs MCP

| | ACP | MCP |
|-|-----|-----|
| Purpose | Editor↔Agent | Agent↔Tools |
| Transport | stdio JSON-RPC | stdio JSON-RPC |
| Direction | Bidirectional | Agent calls tools |
| Use case | Chat UI integration | Tool access |

## Common Gotchas

1. **Wrong package**: Use `@zed-industries/claude-code-acp` (v0.13.1), NOT `claude-code-acp` (v0.1.1 - old/broken)

2. **readLoop message routing**: Check `Method` BEFORE `ID`! Requests from agent have BOTH:
   ```go
   // WRONG - drops permission requests
   if msg.ID != nil { ... }
   else if msg.Method != "" { ... }

   // CORRECT
   if msg.Method != "" { ... }  // requests + notifications
   else if msg.ID != nil { ... } // responses only
   ```

3. **ClientCapabilities**: If you set `fs.writeTextFile: true`, YOU must handle `fs/write_text_file` requests. Otherwise operations hang. For simple clients, omit capabilities.

## Related Docs

- [acp-protocol.md](./acp-protocol.md) - Full protocol reference (types, errors, streaming)
- [claude-code-acp.md](./claude-code-acp.md) - Go client impl + Wails integration

## Resources

- Spec: https://agentclientprotocol.com
- GitHub: https://github.com/agentclientprotocol/agent-client-protocol
- claude-code-acp: https://www.npmjs.com/package/@zed-industries/claude-code-acp
