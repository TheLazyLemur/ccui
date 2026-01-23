# JSON-RPC Type Definitions

Define all protocol types as Go structs with `json` tags inline in the file that uses them.

```go
type JSONRPCMessage struct {
    JSONRPC string          `json:"jsonrpc"`
    ID      *int            `json:"id,omitempty"`
    Method  string          `json:"method,omitempty"`
    Params  json.RawMessage `json:"params,omitempty"`
}
```

- Use `json.RawMessage` for polymorphic fields that need deferred parsing
- Use `*int` (pointer) for optional numeric IDs to distinguish zero from absent
- Group related types together (e.g., all request types, all response types)
