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

## TODO

- [ ] Define tool schemas (translate to Anthropic format)
- [ ] Implement streaming
- [ ] Handle multi-turn tool use
- [ ] Error handling / retries
