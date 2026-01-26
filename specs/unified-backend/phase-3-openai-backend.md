# Phase 3: OpenAI Direct API Backend

Implement AgentBackend for direct OpenAI API calls.

## Scope

- OpenAI Chat Completions API
- Function calling (parallel tool calls)
- Streaming support
- Reuses PermissionLayer, ToolExecutor, tool definitions from Phase 1-2

## Differences from Anthropic

| Aspect | Anthropic | OpenAI |
|--------|-----------|--------|
| Tool format | `tool_use` blocks | `function_call` / `tool_calls` |
| Parallel tools | Sequential | Can be parallel |
| Streaming | SSE with deltas | SSE with deltas |
| Tool result | `tool_result` content | `tool` role message |

## TODO

- [ ] Translate tool schemas to OpenAI format
- [ ] Handle parallel tool calls
- [ ] Map response format to unified Event type
