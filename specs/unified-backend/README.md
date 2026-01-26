# CCUI Unified Agent Backend Spec

## Problem

Current architecture is tightly coupled to ACP (Agent Client Protocol):

1. **Vendor lock-in** - Only works with ACP-compatible agents (claude-code-acp, opencode-acp)
2. **No direct API support** - Can't use Anthropic/OpenAI APIs directly
3. **Inconsistent tool handling** - claude-code-acp respects ClientCapabilities, opencode-acp ignores them
4. **Coupled permission logic** - Permission handling embedded in ACP message handling
5. **No tool executor** - Agents execute their own tools, we can't intercept

## Goal

Unified backend interface that supports:
- ACP backends (claude-code, opencode, future agents)
- Direct API backends (Anthropic, OpenAI)
- Consistent permission flow regardless of backend
- Optional client-side tool execution

## Architecture

```
┌─────────────────────────────────────────────────────────────────────┐
│                           Frontend                                   │
│  - ToolCard, ChatContent (unchanged)                                │
│  - Emits 'permission_response'                                      │
└─────────────────────────────────┬───────────────────────────────────┘
                                  │ Wails Events
┌─────────────────────────────────┴───────────────────────────────────┐
│                        Permission Layer                              │
│  - Deterministic rules (read=auto, write=ask, bash=check)           │
│  - Emits permission requests to frontend                            │
│  - Waits for user response                                          │
└─────────────────────────────────┬───────────────────────────────────┘
                                  │
┌─────────────────────────────────┴───────────────────────────────────┐
│                        Tool Executor                                 │
│  - fs read/write, bash, glob, grep, etc                             │
│  - Used by Direct API backend                                        │
│  - Optionally used by ACP via MCP                                   │
└─────────────────────────────────┬───────────────────────────────────┘
                                  │
┌─────────────────────────────────┴───────────────────────────────────┐
│                      AgentBackend Interface                          │
│  NewSession(ctx, opts) (Session, error)                             │
│  Session: Prompt(), Events(), Cancel(), Close()                     │
└─────────────────────────────────┬───────────────────────────────────┘
                                  │
          ┌───────────────────────┼───────────────────────┐
          ▼                       ▼                       ▼
   ACP (claude-code)       ACP (opencode)          Direct API
   - Permission: agent     - Permission: agent     - Permission: us
   - Execution: agent      - Execution: agent      - Execution: us
```

## Phases

| Phase | Description | Status |
|-------|-------------|--------|
| 1 | Architecture refactor - extract layers | Planned |
| 2 | Direct API backend - Anthropic | Planned |
| 3 | Direct API backend - OpenAI | Planned |
| 4 | MCP tool execution (optional) | Planned |

## Key Decisions

1. **Permission flow stays the same** - Frontend UI unchanged, same events
2. **ACP agents keep their tool execution** - We don't fight their architecture
3. **Direct API = we execute tools** - We control the loop
4. **Shared types** - ToolState, PermissionOptions work across all backends
