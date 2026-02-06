package acp

import (
	"encoding/json"
	"strings"
	"sync"

	"ccui/backend"
)

// PermissionLayer abstracts permission request handling
type PermissionLayer interface {
	Request(toolCallID, toolName string, options []backend.PermOption) (string, error)
}

// Client manages communication with an ACP subprocess
type Client struct {
	transport Transport
	sessionID string
	eventChan chan<- backend.Event

	// Tool tracking
	toolManager     *backend.ToolCallManager
	fileChangeStore *backend.FileChangeStore
	toolAdapters    []ToolEventAdapter

	// Permission handling
	permissionRespCh  chan string
	permissionMu      sync.Mutex
	permissionMsgID   *int
	permissionLayer   PermissionLayer

	// Config
	autoPermission     bool
	suppressToolEvents bool

	// Session modes
	currentModeID  string
	availableModes []backend.SessionMode
}

// ClientOption for configuring a Client
type ClientOption func(*Client)

// WithPermissionLayer sets the permission layer for delegation
func WithPermissionLayer(layer PermissionLayer) ClientOption {
	return func(c *Client) {
		c.permissionLayer = layer
	}
}

// ClientConfig for creating a Client
type ClientConfig struct {
	Transport          Transport
	EventChan          chan<- backend.Event
	AutoPermission     bool
	SuppressToolEvents bool
	FileChangeStore    *backend.FileChangeStore // optional shared store
}

// NewClient creates a Client with the given transport
func NewClient(cfg ClientConfig, opts ...ClientOption) *Client {
	fileStore := cfg.FileChangeStore
	if fileStore == nil {
		fileStore = backend.NewFileChangeStore()
	}

	c := &Client{
		transport:          cfg.Transport,
		eventChan:          cfg.EventChan,
		toolManager:        backend.NewToolCallManager(),
		fileChangeStore:    fileStore,
		toolAdapters:       DefaultToolAdapters(),
		permissionRespCh:   make(chan string, 1),
		autoPermission:     cfg.AutoPermission,
		suppressToolEvents: cfg.SuppressToolEvents,
	}

	// Apply options
	for _, opt := range opts {
		opt(c)
	}

	// Register method handler
	c.transport.OnMethod(c.handleMethod)

	return c
}

// Initialize performs the ACP initialize handshake
func (c *Client) Initialize() error {
	_, err := c.transport.Send("initialize", InitializeParams{
		ProtocolVersion: 1,
		ClientCapabilities: ClientCapabilities{
			Terminal: false,
		},
	})
	return err
}

// NewSession creates a new ACP session
func (c *Client) NewSession(cwd string, mcpServers []any) error {
	resp, err := c.transport.Send("session/new", map[string]any{
		"cwd":        cwd,
		"mcpServers": mcpServers,
	})
	if err != nil {
		return err
	}

	var result SessionNewResult
	json.Unmarshal(resp, &result)
	c.sessionID = result.SessionID
	if result.Modes != nil {
		c.currentModeID = result.Modes.CurrentModeID
		c.availableModes = result.Modes.AvailableModes
	}
	return nil
}

// SendPrompt implements backend.Session
func (c *Client) SendPrompt(text string, allowedTools []string) error {
	resp, err := c.transport.Send("session/prompt", SessionPromptParams{
		SessionID:    c.sessionID,
		Prompt:       []PromptContent{{Type: "text", Text: text}},
		AllowedTools: allowedTools,
	})
	if err != nil {
		return err
	}

	var result SessionPromptResult
	json.Unmarshal(resp, &result)

	c.emit(backend.EventPromptComplete, result.StopReason)
	return nil
}

// SetMode implements backend.Session
func (c *Client) SetMode(modeID string) error {
	_, err := c.transport.Send("session/set_mode", map[string]string{
		"sessionId": c.sessionID,
		"modeId":    modeID,
	})
	if err != nil {
		return err
	}
	c.currentModeID = modeID
	c.emit(backend.EventModeChanged, modeID)
	return nil
}

// Cancel implements backend.Session
func (c *Client) Cancel() {
	c.transport.Notify("session/cancel", map[string]string{"sessionId": c.sessionID})
}

// Close implements backend.Session
func (c *Client) Close() error {
	return c.transport.Close()
}

// SessionID implements backend.Session
func (c *Client) SessionID() string {
	return c.sessionID
}

// CurrentMode implements backend.Session
func (c *Client) CurrentMode() string {
	return c.currentModeID
}

// AvailableModes implements backend.Session
func (c *Client) AvailableModes() []backend.SessionMode {
	return c.availableModes
}

// RespondToPermission sends a permission response
func (c *Client) RespondToPermission(optionID string) {
	c.permissionRespCh <- optionID
}

// FileChangeStore returns the file change store
func (c *Client) FileChangeStore() *backend.FileChangeStore {
	return c.fileChangeStore
}

// SetFileChangeStore sets the file change store (for sharing between clients)
func (c *Client) SetFileChangeStore(store *backend.FileChangeStore) {
	c.fileChangeStore = store
}

func (c *Client) emit(eventType backend.EventType, data any) {
	if c.eventChan != nil {
		c.eventChan <- backend.Event{Type: eventType, Data: data}
	}
}

func (c *Client) handleMethod(method string, params json.RawMessage, id *int) {
	switch method {
	case "session/update":
		var update SessionUpdate
		json.Unmarshal(params, &update)
		c.handleSessionUpdate(update)

	case "session/request_permission":
		var req PermissionRequest
		json.Unmarshal(params, &req)
		c.handlePermissionRequest(req, id)
	}
}

func (c *Client) handleSessionUpdate(update SessionUpdate) {
	u := update.Update

	switch u.SessionUpdate {
	case "agent_message_chunk":
		var content backend.TextContent
		if len(u.Content) > 0 {
			json.Unmarshal(u.Content, &content)
		}
		c.emit(backend.EventMessageChunk, content.Text)

	case "agent_thought_chunk":
		var content backend.TextContent
		if len(u.Content) > 0 {
			json.Unmarshal(u.Content, &content)
		}
		c.emit(backend.EventThoughtChunk, content.Text)

	case "tool_call":
		if c.suppressToolEvents {
			return
		}
		c.handleToolCall(u)

	case "tool_call_update":
		c.handleToolCallUpdate(u)

	case "current_mode_update":
		c.currentModeID = u.ModeID
		c.emit(backend.EventModeChanged, u.ModeID)

	case "plan":
		c.emit(backend.EventPlanUpdate, u.Entries)
	}
}

func (c *Client) handleToolCall(u UpdateContent) {
	adapter := c.adapterFor(u)
	toolName := ResolveToolName(adapter, u)
	var diffs []backend.DiffBlock
	if adapter != nil {
		diffs = adapter.DiffBlocks(u)
	}

	// Update existing tool if present
	if existing := c.toolManager.Get(u.ToolCallID); existing != nil {
		existing.Status = u.Status
		existing.Title = u.Title
		if existing.ToolName == "" {
			existing.ToolName = toolName
		}
		if u.RawInput != nil {
			existing.Input = u.RawInput
		}
		c.emit(backend.EventToolState, existing)
		return
	}

	state := &backend.ToolState{
		ID:       u.ToolCallID,
		Status:   u.Status,
		Title:    u.Title,
		Kind:     u.ToolKind,
		ToolName: toolName,
		ParentID: c.toolManager.CurrentParent(),
		Input:    u.RawInput,
		Diffs:    diffs,
	}

	if toolName == "Task" {
		c.toolManager.PushParent(u.ToolCallID)
	}

	c.toolManager.Set(state)
	c.emit(backend.EventToolState, state)
}

func (c *Client) handleToolCallUpdate(u UpdateContent) {
	adapter := c.adapterFor(u)
	toolName := ResolveToolName(adapter, u)
	var diffs []backend.DiffBlock
	var toolResponse *ToolResponse
	if adapter != nil {
		diffs = adapter.DiffBlocks(u)
		toolResponse = adapter.ToolResponse(u)
	}

	// Suppressed mode: only track file changes
	if c.suppressToolEvents {
		if toolResponse != nil {
			c.trackFileChange(toolName, toolResponse)
		}
		return
	}

	state := c.toolManager.Update(u.ToolCallID, func(s *backend.ToolState) {
		s.Status = u.Status
		s.Output = u.Output
		if u.RawInput != nil {
			s.Input = u.RawInput
		}
		if s.ToolName == "" {
			s.ToolName = toolName
		}
		if toolResponse != nil {
			if len(toolResponse.StructuredPatch) > 0 || toolResponse.OldString != "" || toolResponse.NewString != "" || toolResponse.Content != "" {
				s.Diff = map[string]any{
					"filePath":        toolResponse.FilePath,
					"oldString":       toolResponse.OldString,
					"newString":       toolResponse.NewString,
					"originalFile":    toolResponse.OriginalFile,
					"structuredPatch": toolResponse.StructuredPatch,
					"type":            toolResponse.Type,
					"content":         toolResponse.Content,
				}
			}
			c.trackFileChange(s.ToolName, toolResponse)
		} else if len(diffs) > 0 && s.Diff == nil {
			s.Diffs = diffs
		}
	})

	if state == nil {
		return
	}
	if state.ToolName == "Task" && isTerminalStatus(u.Status) {
		c.toolManager.PopParent(u.ToolCallID)
	}
	c.emit(backend.EventToolState, state)
}

func (c *Client) handlePermissionRequest(req PermissionRequest, id *int) {
	// Auto-allow our MCP ask user question tool
	if req.ToolCall.Title == "mcp__ccui__ccui_ask_user_question" {
		c.sendPermissionResponse(id, "allow_always")
		return
	}

	// Auto-allow all permissions if configured
	if c.autoPermission {
		c.sendPermissionResponse(id, "allow_always")
		return
	}

	// Delegate to permission layer if present
	if c.permissionLayer != nil {
		optionID, _ := c.permissionLayer.Request(req.ToolCall.ToolCallID, req.ToolCall.Title, req.Options)
		c.sendPermissionResponse(id, optionID)
		return
	}

	// Fallback: channel-based approach
	// Update tool state with permission options
	state := c.toolManager.Update(req.ToolCall.ToolCallID, func(s *backend.ToolState) {
		s.Status = "awaiting_permission"
		s.PermissionOptions = req.Options
	})
	if state != nil {
		c.emit(backend.EventToolState, state)
	}

	// Emit permission request event
	c.emit(backend.EventPermissionRequest, req)

	// Store message ID for response
	c.permissionMu.Lock()
	c.permissionMsgID = id
	c.permissionMu.Unlock()

	// Wait for response from UI
	optionID := <-c.permissionRespCh
	c.sendPermissionResponse(id, optionID)
}

func (c *Client) sendPermissionResponse(id *int, optionID string) {
	result, _ := json.Marshal(PermissionResponse{
		Outcome: PermissionOutcome{Outcome: "selected", OptionID: optionID},
	})
	c.transport.Respond(id, result)
}

func (c *Client) trackFileChange(toolName string, tr *ToolResponse) {
	if tr.FilePath == "" || (toolName != "Edit" && toolName != "Write") {
		return
	}
	currentContent := tr.Content
	if toolName == "Edit" && tr.Content == "" {
		base := tr.OriginalFile
		if existing := c.fileChangeStore.Get(tr.FilePath); existing != nil {
			base = existing.CurrentContent
		}
		currentContent = strings.Replace(base, tr.OldString, tr.NewString, 1)
	}
	c.fileChangeStore.RecordChange(tr.FilePath, tr.OriginalFile, currentContent, tr.StructuredPatch)
	c.emit(backend.EventFileChanges, c.fileChangeStore.GetAll())
}

func (c *Client) adapterFor(update UpdateContent) ToolEventAdapter {
	for _, adapter := range c.toolAdapters {
		if adapter.CanHandle(update) {
			return adapter
		}
	}
	return nil
}

func isTerminalStatus(status string) bool {
	return status == "completed" || status == "error" || status == "failed"
}
