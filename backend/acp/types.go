package acp

import (
	"encoding/json"

	"ccui/backend"
)

// JSONRPCMessage represents a JSON-RPC 2.0 message
type JSONRPCMessage struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      *int            `json:"id,omitempty"`
	Method  string          `json:"method,omitempty"`
	Params  json.RawMessage `json:"params,omitempty"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *RPCError       `json:"error,omitempty"`
}

// RPCError represents a JSON-RPC error
type RPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// InitializeParams for initialize request
type InitializeParams struct {
	ProtocolVersion    int                `json:"protocolVersion"`
	ClientCapabilities ClientCapabilities `json:"clientCapabilities"`
}

// ClientCapabilities describes client capabilities
type ClientCapabilities struct {
	FS       *FSCapabilities `json:"fs,omitempty"`
	Terminal bool            `json:"terminal,omitempty"`
}

// FSCapabilities describes filesystem capabilities
type FSCapabilities struct {
	ReadTextFile  bool `json:"readTextFile"`
	WriteTextFile bool `json:"writeTextFile"`
}

// ModesInfo contains session mode information
type ModesInfo struct {
	CurrentModeID  string               `json:"currentModeId"`
	AvailableModes []backend.SessionMode `json:"availableModes"`
}

// SessionNewResult from session/new response
type SessionNewResult struct {
	SessionID string     `json:"sessionId"`
	Modes     *ModesInfo `json:"modes,omitempty"`
}

// PromptContent for prompts
type PromptContent struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
}

// SessionPromptParams for session/prompt request
type SessionPromptParams struct {
	SessionID    string          `json:"sessionId"`
	Prompt       []PromptContent `json:"prompt"`
	AllowedTools []string        `json:"allowedTools,omitempty"`
}

// SessionPromptResult from session/prompt response
type SessionPromptResult struct {
	SessionID  string `json:"sessionId"`
	StopReason string `json:"stopReason"`
}

// SessionUpdate notification params
type SessionUpdate struct {
	SessionID string        `json:"sessionId"`
	Update    UpdateContent `json:"update"`
}

// UpdateContent holds the update payload
type UpdateContent struct {
	SessionUpdate string          `json:"sessionUpdate,omitempty"`
	Content       json.RawMessage `json:"content,omitempty"`
	ToolCallID    string          `json:"toolCallId,omitempty"`
	Title         string          `json:"title,omitempty"`
	ToolKind      string          `json:"toolKind,omitempty"`
	Status        string          `json:"status,omitempty"`
	Input         map[string]any  `json:"input,omitempty"`
	Output        []backend.OutputBlock `json:"output,omitempty"`
	RawInput      map[string]any  `json:"rawInput,omitempty"`
	RawOutput     *ToolRawOutput  `json:"rawOutput,omitempty"`
	Meta          *MetaContent    `json:"_meta,omitempty"`
	ModeID        string          `json:"modeId,omitempty"`
	Entries       []backend.PlanEntry `json:"entries,omitempty"`
}

// MetaContent holds tool metadata
type MetaContent struct {
	ClaudeCode *ClaudeCodeMeta `json:"claudeCode,omitempty"`
}

// ClaudeCodeMeta for Claude Code specific metadata
type ClaudeCodeMeta struct {
	ToolName     string        `json:"toolName,omitempty"`
	ToolResponse *ToolResponse `json:"toolResponse,omitempty"`
}

// ToolResponse contains tool response data
type ToolResponse struct {
	FilePath        string              `json:"filePath,omitempty"`
	Content         string              `json:"content,omitempty"`
	OldString       string              `json:"oldString,omitempty"`
	NewString       string              `json:"newString,omitempty"`
	OriginalFile    string              `json:"originalFile,omitempty"`
	StructuredPatch []backend.PatchHunk `json:"structuredPatch,omitempty"`
	Type            string              `json:"type,omitempty"`
}

// ToolRawOutput holds raw tool output
type ToolRawOutput struct {
	Output   string              `json:"output,omitempty"`
	Metadata *ToolOutputMetadata `json:"metadata,omitempty"`
}

// ToolOutputMetadata for tool output
type ToolOutputMetadata struct {
	Diff      string    `json:"diff,omitempty"`
	Filediff  *FileDiff `json:"filediff,omitempty"`
	Filepath  string    `json:"filepath,omitempty"`
	Exists    bool      `json:"exists,omitempty"`
	Truncated bool      `json:"truncated,omitempty"`
}

// FileDiff for file changes
type FileDiff struct {
	File      string `json:"file,omitempty"`
	Before    string `json:"before,omitempty"`
	After     string `json:"after,omitempty"`
	Additions int    `json:"additions,omitempty"`
	Deletions int    `json:"deletions,omitempty"`
}

// PermissionRequest from session/request_permission
type PermissionRequest struct {
	SessionID string              `json:"sessionId"`
	ToolCall  ToolCallInfo        `json:"toolCall"`
	Options   []backend.PermOption `json:"options"`
}

// ToolCallInfo describes the tool requesting permission
type ToolCallInfo struct {
	ToolCallID string `json:"toolCallId"`
	Title      string `json:"title"`
	Kind       string `json:"kind"`
}

// PermissionResponse to send back
type PermissionResponse struct {
	Outcome PermissionOutcome `json:"outcome"`
}

// PermissionOutcome describes the selected option
type PermissionOutcome struct {
	Outcome  string `json:"outcome"`
	OptionID string `json:"optionId,omitempty"`
}
