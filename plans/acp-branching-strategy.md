# ACP Branching Strategy Plan

## Overview

Flip the backend architecture so that `main` branch uses only the **Anthropic API backend** (direct API), while preserving the **ACP backend** on a `feature/acp` branch for continued development until feature parity is achieved.

## Current State

| Aspect | Current (main) |
|--------|----------------|
| Default Backend | ACP (`claude-code-acp` npm package) |
| Opt-in Backend | Anthropic API (via `CCUI_BACKEND=anthropic`) |
| ACP Code Location | `backend/acp/` |
| API Code Location | `backend/anthropic/` |

## Target State

| Branch | Primary Backend | Secondary Backend |
|--------|-----------------|-------------------|
| `main` | Anthropic API | None (ACP removed) |
| `feature/acp` | Anthropic API | ACP (preserved, maintained) |

## Feature Parity Gap Analysis

The following features work in the **Anthropic API backend** but are **missing/incomplete in ACP**:

| Feature | API Status | ACP Status | Gap |
|---------|------------|------------|-----|
| Direct tool execution | ✅ Full control | ⚠️ Delegated to agent | ACP handles tools internally |
| Custom tool registry | ✅ Full registry | ❌ Limited | ACP uses built-in tools |
| Permission layer integration | ✅ Deep integration | ⚠️ Basic | ACP has own permission flow |
| Session modes (plan, acceptEdits, etc.) | ✅ Full support | ⚠️ Partial | ACP modes differ |
| File change tracking | ✅ Custom store | ⚠️ Adapter-based | Different architectures |
| Tool output formatting | ✅ Consistent | ⚠️ Adapter-dependent | ACP varies by adapter |
| MCP server integration | ✅ Full control | ⚠️ Config-based | ACP configures, agent manages |
| Streaming event granularity | ✅ Fine-grained | ⚠️ Coarse | ACP batches updates |
| Error handling | ✅ Direct | ⚠️ JSON-RPC layer | Extra abstraction in ACP |
| Review agent mode | ✅ Working | ❌ Needs testing | Unclear if fully functional |

## Migration Plan

### Phase 1: Prepare `feature/acp` Branch (Before touching `main`)

```bash
# From current main
git checkout -b feature/acp
git push -u origin feature/acp
```

This branch preserves ACP as-is, with both backends available.

### Phase 2: Simplify `main` to API-Only

#### Step 2.1: Update Backend Selection Logic (`app.go`)

**Current:**
```go
// Default to ACP, opt-in to Anthropic
bt := BackendACP
if os.Getenv("CCUI_BACKEND") == "anthropic" {
    bt = BackendAnthropic
}
```

**New:**
```go
// main branch: API only
bt := BackendAnthropic
```

Remove the `BackendType` selection logic entirely - main always uses Anthropic.

#### Step 2.2: Remove ACP Backend Code from `main`

Delete directory:
```
backend/acp/
├── backend.go
├── client.go
├── transport.go
├── types.go
├── adapters.go
├── transport_test.go
├── client_test.go
└── adapters_test.go
```

#### Step 2.3: Remove ACP Documentation from `main`

Delete or archive:
```
acp/
├── acp-overview.md
├── acp-protocol.md
└── claude-code-acp.md

ACP_FLOW_SUMMARY.md
```

#### Step 2.4: Update Imports and References

In `app.go`:
- Remove `"ccui/backend/acp"` import
- Remove ACP-related type assertions (e.g., `sess.(*acp.Client)`)
- Simplify `backendFactory` to only return Anthropic backend

**Current `handlePermissionResponse`:**
```go
func (a *App) handlePermissionResponse(data ...interface{}) {
    if optionID, ok := firstAs[string](data); ok {
        // ACP backend permission response
        if sess := a.getActiveSession(); sess != nil {
            if client, ok := sess.(*acp.Client); ok {
                client.RespondToPermission(optionID)
                return
            }
        }
        // Anthropic backend permission response
        // ...
    }
}
```

**New:**
```go
func (a *App) handlePermissionResponse(data ...interface{}) {
    if optionID, ok := firstAs[string](data); ok {
        // Anthropic backend only
        if a.permLayer != nil {
            if len(data) >= 2 {
                if m, ok := data[1].(map[string]interface{}); ok {
                    if toolCallID, ok := m["toolCallId"].(string); ok {
                        a.permLayer.Respond(toolCallID, optionID)
                        return
                    }
                }
            }
        }
    }
}
```

#### Step 2.5: Keep Abstractions

**Preserve in `main`:**
- `backend/interface.go` - `AgentBackend` and `Session` interfaces
- `backend/types.go` - Shared types (`ToolState`, `FileChange`, etc.)
- `backend/anthropic/` - Full API backend implementation
- `backend/tools/` - Tool implementations (used by API backend)

These abstractions allow ACP to be re-integrated later.

#### Step 2.6: Update Build/CI (if needed)

- Remove `claude-code-acp` npm dependency from build process
- Update documentation to reflect API-only backend

### Phase 3: Maintain `feature/acp` Branch

On the `feature/acp` branch, maintain ACP support:

1. **Keep both backends working**
2. **Regularly rebase onto main** to stay current:
   ```bash
   git checkout feature/acp
   git rebase main
   # Resolve conflicts in backend/acp/ as needed
   ```
3. **Document parity gaps** as they're discovered
4. **Work toward closing gaps** to prepare for eventual merge

### Phase 4: Future Re-integration (Option A)

When feature parity is achieved:

1. Merge `feature/acp` → `main`
2. Re-introduce backend selection logic
3. Make ACP opt-in or default based on maturity

## File Changes Summary

### Files Modified on `main`

| File | Change |
|------|--------|
| `app.go` | Remove ACP import, simplify backend selection, simplify permission handling |
| `go.mod` | Remove any ACP-specific dependencies (if any) |

### Files Deleted from `main`

```
backend/acp/*          (entire directory)
acp/*                  (entire directory)
ACP_FLOW_SUMMARY.md
```

### Files Preserved on `main`

```
backend/interface.go   (abstractions)
backend/types.go       (shared types)
backend/anthropic/*    (API backend)
backend/tools/*        (tool implementations)
```

### Files Preserved on `feature/acp`

```
backend/acp/*          (ACP implementation)
backend/anthropic/*    (API implementation)
app.go                 (with both backends)
```

## MCP Server Considerations

The `ask_user_question` MCP tool must work with both backends:

- **On `main`:** Works with Anthropic API backend only
- **On `feature/acp`:** Works with both backends

The MCP server is backend-agnostic - it just exposes the tool. Both backends can configure it in `MCPServers`.

## Testing Strategy

### On `main`:
- Focus all testing on Anthropic API backend
- Remove ACP-specific tests
- Ensure tool registry, permissions, sessions work correctly

### On `feature/acp`:
- Maintain tests for both backends
- Track which tests pass/fail on ACP vs API
- Use test gaps to identify parity issues

## Documentation Updates

### On `main`:
- Update README to reflect API-only backend
- Remove ACP setup instructions
- Document that ACP is on a branch for development

### On `feature/acp`:
- Keep all ACP documentation
- Add "Feature Parity Tracking" section
- Document known gaps and workarounds

## Rollback Plan

If issues arise on `main`:

1. Switch to `feature/acp` branch for ACP functionality
2. Fix issues on `main` with API backend
3. No need to revert - ACP is preserved on branch

## Success Criteria

- [ ] `main` branch builds and runs with only Anthropic API backend
- [ ] `feature/acp` branch maintains both backends with regular rebasing
- [ ] All existing functionality works on `main` (now via API)
- [ ] MCP server works on `main`
- [ ] Permission system works on `main`
- [ ] Review agent works on `main`
- [ ] Automation system works on `main`

## Timeline

| Phase | Estimated Time |
|-------|----------------|
| Phase 1: Create `feature/acp` branch | 30 minutes |
| Phase 2: Simplify `main` to API-only | 2-4 hours |
| Phase 3: Verify `main` functionality | 2-4 hours |
| Phase 4: Document and communicate | 1 hour |
| **Total** | **1-2 days** |

## Communication

After completion:
1. Update team on branch strategy
2. Document how to switch to ACP if needed (`git checkout feature/acp`)
3. Set up branch protection for `feature/acp` to prevent accidental deletion
