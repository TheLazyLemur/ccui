# Phase 4: MCP Tool Execution (Optional)

Expose ToolExecutor as MCP server for ACP backends.

## Motivation

- Unified tool execution across all backends
- ACP agents use our tools via MCP instead of built-in
- Full control over execution, permissions, logging

## Architecture

```
ACP Agent (claude-code / opencode)
    │
    ├─► Disable built-in tools (if supported)
    │
    ├─► Register our MCP server
    │
    └─► Agent calls MCP tools
            │
            ▼
      MCP Server (SSE)
            │
            ├─► PermissionLayer.Check()
            │
            └─► ToolExecutor.Execute()
```

## Extend mcpserver.go

Current: Only `ccui_ask_user_question`

Add:
- `ccui_read_file`
- `ccui_write_file`
- `ccui_edit_file`
- `ccui_bash`
- `ccui_glob`
- `ccui_grep`

## Challenges

- Tool call timeouts vs permission wait times
- Need to check if opencode/claude-code support disabling built-ins
- May not be worth it if ACP permission flow works fine

## Status

Low priority - ACP's built-in tools work. Only pursue if:
1. Need identical tool behavior across all backends
2. ACP backends support disabling built-ins
