package anthropic

import "encoding/json"

// MessagesRequest for POST /v1/messages
type MessagesRequest struct {
	Model       string          `json:"model"`
	Messages    []Message       `json:"messages"`
	MaxTokens   int             `json:"max_tokens"`
	System      string          `json:"system,omitempty"`
	Tools       []Tool          `json:"tools,omitempty"`
	ToolChoice  *ToolChoice     `json:"tool_choice,omitempty"`
	Stream      bool            `json:"stream,omitempty"`
	Thinking    *ThinkingConfig `json:"thinking,omitempty"`
	Metadata    *Metadata       `json:"metadata,omitempty"`
}

// ToolChoice specifies how tools should be used
type ToolChoice struct {
	Type string `json:"type"` // "auto", "any", "tool", "none"
	Name string `json:"name,omitempty"` // required when type="tool"
}

// ThinkingConfig enables extended thinking
type ThinkingConfig struct {
	Type         string `json:"type"`          // "enabled"
	BudgetTokens int    `json:"budget_tokens"` // max tokens for thinking
}

// Metadata for request tracking
type Metadata struct {
	UserID string `json:"user_id,omitempty"`
}

// MessagesResponse for non-streaming response
type MessagesResponse struct {
	ID           string         `json:"id"`
	Type         string         `json:"type"` // "message"
	Role         string         `json:"role"` // "assistant"
	Content      []ContentBlock `json:"content"`
	Model        string         `json:"model"`
	StopReason   string         `json:"stop_reason,omitempty"` // end_turn, tool_use, max_tokens, stop_sequence
	StopSequence string         `json:"stop_sequence,omitempty"`
	Usage        Usage          `json:"usage"`
}

// Message in the conversation
type Message struct {
	Role    string         `json:"role"` // "user" or "assistant"
	Content []ContentBlock `json:"content"`
}

// ContentBlock types: text, tool_use, tool_result, thinking, server_tool_use, web_search_tool_result
type ContentBlock struct {
	Type string `json:"type"` // "text", "tool_use", "tool_result", "thinking", "server_tool_use", "web_search_tool_result"

	// text block
	Text string `json:"text,omitempty"`

	// tool_use / server_tool_use block
	ID    string         `json:"id,omitempty"`
	Name  string         `json:"name,omitempty"`
	Input map[string]any `json:"input,omitempty"`

	// tool_result block
	ToolUseID string `json:"tool_use_id,omitempty"`
	Content   any    `json:"content,omitempty"` // string or []ContentBlock
	IsError   bool   `json:"is_error,omitempty"`

	// thinking block
	Thinking  string `json:"thinking,omitempty"`
	Signature string `json:"signature,omitempty"`
}

// Usage tracks token usage
type Usage struct {
	InputTokens              int             `json:"input_tokens"`
	OutputTokens             int             `json:"output_tokens"`
	CacheCreationInputTokens int             `json:"cache_creation_input_tokens,omitempty"`
	CacheReadInputTokens     int             `json:"cache_read_input_tokens,omitempty"`
	ServerToolUse            *ServerToolUse  `json:"server_tool_use,omitempty"`
}

// ServerToolUse tracks server-side tool usage metrics
type ServerToolUse struct {
	WebSearchRequests int `json:"web_search_requests,omitempty"`
}

// Tool definition for Anthropic API
type Tool struct {
	Name        string      `json:"name"`
	Description string      `json:"description,omitempty"`
	InputSchema InputSchema `json:"input_schema"`
}

// InputSchema is JSON Schema for tool input
type InputSchema struct {
	Type       string              `json:"type"` // "object"
	Properties map[string]Property `json:"properties,omitempty"`
	Required   []string            `json:"required,omitempty"`
}

// Property in JSON Schema
type Property struct {
	Type        string              `json:"type"`
	Description string              `json:"description,omitempty"`
	Enum        []string            `json:"enum,omitempty"`
	Items       *Property           `json:"items,omitempty"`       // for arrays
	Properties  map[string]Property `json:"properties,omitempty"`  // for nested objects
	Required    []string            `json:"required,omitempty"`    // for nested objects
	Default     any                 `json:"default,omitempty"`
}

// SSE event types for streaming

// SSEEvent wraps all streaming event types
type SSEEvent struct {
	Type    string          `json:"type"`
	Message json.RawMessage `json:"message,omitempty"`
	Index   int             `json:"index,omitempty"`
	Delta   json.RawMessage `json:"delta,omitempty"`
	Usage   *Usage          `json:"usage,omitempty"`
	Error   *APIError       `json:"error,omitempty"`
}

// MessageStartEvent: type="message_start"
type MessageStartEvent struct {
	Type    string           `json:"type"` // "message_start"
	Message MessagesResponse `json:"message"`
}

// ContentBlockStartEvent: type="content_block_start"
type ContentBlockStartEvent struct {
	Type         string       `json:"type"` // "content_block_start"
	Index        int          `json:"index"`
	ContentBlock ContentBlock `json:"content_block"`
}

// ContentBlockDeltaEvent: type="content_block_delta"
type ContentBlockDeltaEvent struct {
	Type  string     `json:"type"` // "content_block_delta"
	Index int        `json:"index"`
	Delta BlockDelta `json:"delta"`
}

// BlockDelta for content_block_delta
type BlockDelta struct {
	Type        string `json:"type"` // "text_delta", "input_json_delta", "thinking_delta", "signature_delta"
	Text        string `json:"text,omitempty"`
	PartialJSON string `json:"partial_json,omitempty"`
	Thinking    string `json:"thinking,omitempty"`
	Signature   string `json:"signature,omitempty"`
}

// ContentBlockStopEvent: type="content_block_stop"
type ContentBlockStopEvent struct {
	Type  string `json:"type"` // "content_block_stop"
	Index int    `json:"index"`
}

// MessageDeltaEvent: type="message_delta"
type MessageDeltaEvent struct {
	Type  string       `json:"type"` // "message_delta"
	Delta MessageDelta `json:"delta"`
	Usage Usage        `json:"usage"`
}

// MessageDelta for message_delta event
type MessageDelta struct {
	StopReason   string `json:"stop_reason,omitempty"`
	StopSequence string `json:"stop_sequence,omitempty"`
}

// MessageStopEvent: type="message_stop"
type MessageStopEvent struct {
	Type string `json:"type"` // "message_stop"
}

// PingEvent: type="ping"
type PingEvent struct {
	Type string `json:"type"` // "ping"
}

// ErrorEvent: type="error"
type ErrorEvent struct {
	Type  string   `json:"type"` // "error"
	Error APIError `json:"error"`
}

// APIError from Anthropic API
type APIError struct {
	Type    string `json:"type"`
	Message string `json:"message"`
}

// SSE event type constants
const (
	EventMessageStart     = "message_start"
	EventContentBlockStart = "content_block_start"
	EventContentBlockDelta = "content_block_delta"
	EventContentBlockStop  = "content_block_stop"
	EventMessageDelta     = "message_delta"
	EventMessageStop      = "message_stop"
	EventPing             = "ping"
	EventError            = "error"
)

// Stop reason constants
const (
	StopReasonEndTurn      = "end_turn"
	StopReasonToolUse      = "tool_use"
	StopReasonMaxTokens    = "max_tokens"
	StopReasonStopSequence = "stop_sequence"
)

// Content block type constants
const (
	BlockTypeText              = "text"
	BlockTypeToolUse           = "tool_use"
	BlockTypeToolResult        = "tool_result"
	BlockTypeThinking          = "thinking"
	BlockTypeServerToolUse     = "server_tool_use"
	BlockTypeWebSearchResult   = "web_search_tool_result"
)

// Delta type constants
const (
	DeltaTypeText      = "text_delta"
	DeltaTypeInputJSON = "input_json_delta"
	DeltaTypeThinking  = "thinking_delta"
	DeltaTypeSignature = "signature_delta"
)
