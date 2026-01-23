package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// JSON-RPC types
type JSONRPCMessage struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      *int            `json:"id,omitempty"`
	Method  string          `json:"method,omitempty"`
	Params  json.RawMessage `json:"params,omitempty"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *RPCError       `json:"error,omitempty"`
}

type RPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type InitializeParams struct {
	ProtocolVersion    int                `json:"protocolVersion"`
	ClientCapabilities ClientCapabilities `json:"clientCapabilities"`
}

type ClientCapabilities struct {
	FS       *FSCapabilities `json:"fs,omitempty"`
	Terminal bool            `json:"terminal,omitempty"`
}

type FSCapabilities struct {
	ReadTextFile  bool `json:"readTextFile"`
	WriteTextFile bool `json:"writeTextFile"`
}

type ModesInfo struct {
	CurrentModeID  string        `json:"currentModeId"`
	AvailableModes []SessionMode `json:"availableModes"`
}

type SessionNewResult struct {
	SessionID string     `json:"sessionId"`
	Modes     *ModesInfo `json:"modes,omitempty"`
}

type PromptContent struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
}

type SessionPromptParams struct {
	SessionID    string          `json:"sessionId"`
	Prompt       []PromptContent `json:"prompt"`
	AllowedTools []string        `json:"allowedTools,omitempty"`
}

type SessionPromptResult struct {
	SessionID  string `json:"sessionId"`
	StopReason string `json:"stopReason"`
}

type SessionUpdate struct {
	SessionID string        `json:"sessionId"`
	Update    UpdateContent `json:"update"`
}

type UpdateContent struct {
	SessionUpdate string          `json:"sessionUpdate,omitempty"`
	Content       json.RawMessage `json:"content,omitempty"` // Can be TextContent or []DiffBlock
	ToolCallID    string          `json:"toolCallId,omitempty"`
	Title         string          `json:"title,omitempty"`
	ToolKind      string          `json:"toolKind,omitempty"`
	Status        string          `json:"status,omitempty"`
	Input         map[string]any  `json:"input,omitempty"`
	Output        []OutputBlock   `json:"output,omitempty"`
	RawInput      map[string]any  `json:"rawInput,omitempty"`
	Meta          *MetaContent    `json:"_meta,omitempty"`
	// Mode updates
	ModeID string `json:"modeId,omitempty"`
	// Plan updates
	Entries []PlanEntry `json:"entries,omitempty"`
}

type MetaContent struct {
	ClaudeCode *ClaudeCodeMeta `json:"claudeCode,omitempty"`
}

type ClaudeCodeMeta struct {
	ToolName     string        `json:"toolName,omitempty"`
	ToolResponse *ToolResponse `json:"toolResponse,omitempty"`
}

type ToolResponse struct {
	FilePath        string      `json:"filePath,omitempty"`
	Content         string      `json:"content,omitempty"`
	OldString       string      `json:"oldString,omitempty"`
	NewString       string      `json:"newString,omitempty"`
	OriginalFile    string      `json:"originalFile,omitempty"`
	StructuredPatch []PatchHunk `json:"structuredPatch,omitempty"`
	Type            string      `json:"type,omitempty"`
}

type PatchHunk struct {
	OldStart int      `json:"oldStart"`
	OldLines int      `json:"oldLines"`
	NewStart int      `json:"newStart"`
	NewLines int      `json:"newLines"`
	Lines    []string `json:"lines"`
}

type DiffBlock struct {
	Type    string `json:"type"`
	Path    string `json:"path,omitempty"`
	OldText string `json:"oldText,omitempty"`
	NewText string `json:"newText,omitempty"`
}

type OutputBlock struct {
	Type       string       `json:"type"`
	Content    *TextContent `json:"content,omitempty"`
	Path       string       `json:"path,omitempty"`
	OldContent string       `json:"oldContent,omitempty"`
	NewContent string       `json:"newContent,omitempty"`
}

type TextContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type PermissionRequest struct {
	SessionID string       `json:"sessionId"`
	ToolCall  ToolCallInfo `json:"toolCall"`
	Options   []PermOption `json:"options"`
}

type ToolCallInfo struct {
	ToolCallID string `json:"toolCallId"`
	Title      string `json:"title"`
	Kind       string `json:"kind"`
}

type PermOption struct {
	OptionID string `json:"optionId"`
	Name     string `json:"name"`
	Kind     string `json:"kind"`
}

type PermissionResponse struct {
	Outcome PermissionOutcome `json:"outcome"`
}

type PermissionOutcome struct {
	Outcome  string `json:"outcome"`
	OptionID string `json:"optionId,omitempty"`
}

// Session modes
type SessionMode struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

// Plan entries
type PlanEntry struct {
	Content  string `json:"content"`
	Priority string `json:"priority"` // high, medium, low
	Status   string `json:"status"`   // pending, in_progress, completed
}

// ToolState tracks unified state for a single tool call
type ToolState struct {
	ID                string         `json:"id"`
	Status            string         `json:"status"` // pending, awaiting_permission, running, completed, error
	Title             string         `json:"title"`
	Kind              string         `json:"kind"`
	ToolName          string         `json:"toolName,omitempty"`
	ParentID          string         `json:"parentId,omitempty"`
	Input             map[string]any `json:"input,omitempty"`
	Output            []OutputBlock  `json:"output,omitempty"`
	Diff              map[string]any `json:"diff,omitempty"`
	Diffs             []DiffBlock    `json:"diffs,omitempty"`
	PermissionOptions []PermOption   `json:"permissionOptions,omitempty"`
}

// ToolCallManager tracks all active tool calls
type ToolCallManager struct {
	tools       map[string]*ToolState
	parentStack []string // stack of active Task tool IDs
	mu          sync.RWMutex
}

// FileChange tracks a file's changes during the session
type FileChange struct {
	FilePath        string      `json:"filePath"`
	OriginalContent string      `json:"originalContent"`
	CurrentContent  string      `json:"currentContent"`
	Hunks           []PatchHunk `json:"hunks"`
}

// FileChangeStore accumulates file changes, coalesces to latest state
type FileChangeStore struct {
	changes map[string]*FileChange
	mu      sync.RWMutex
}

func NewFileChangeStore() *FileChangeStore {
	return &FileChangeStore{changes: make(map[string]*FileChange)}
}

func (s *FileChangeStore) RecordChange(filePath, originalContent, currentContent string, hunks []PatchHunk) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if existing, ok := s.changes[filePath]; ok {
		// Coalesce: keep original, update current
		existing.CurrentContent = currentContent
		existing.Hunks = hunks
	} else {
		s.changes[filePath] = &FileChange{
			FilePath:        filePath,
			OriginalContent: originalContent,
			CurrentContent:  currentContent,
			Hunks:           hunks,
		}
	}
}

func (s *FileChangeStore) Get(filePath string) *FileChange {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.changes[filePath]
}

func (s *FileChangeStore) GetAll() []FileChange {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]FileChange, 0, len(s.changes))
	for _, c := range s.changes {
		result = append(result, *c)
	}
	return result
}

func (s *FileChangeStore) Clear() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.changes = make(map[string]*FileChange)
}

func NewToolCallManager() *ToolCallManager {
	return &ToolCallManager{tools: make(map[string]*ToolState)}
}

func (m *ToolCallManager) Get(id string) *ToolState {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.tools[id]
}

func (m *ToolCallManager) Set(state *ToolState) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.tools[state.ID] = state
}

func (m *ToolCallManager) Update(id string, fn func(*ToolState)) *ToolState {
	m.mu.Lock()
	defer m.mu.Unlock()
	if s, ok := m.tools[id]; ok {
		fn(s)
		return s
	}
	return nil
}

func (m *ToolCallManager) PushParent(id string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.parentStack = append(m.parentStack, id)
}

func (m *ToolCallManager) PopParent(id string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	// Remove the specific ID from stack (may not be at top if nested)
	for i := len(m.parentStack) - 1; i >= 0; i-- {
		if m.parentStack[i] == id {
			m.parentStack = append(m.parentStack[:i], m.parentStack[i+1:]...)
			return
		}
	}
}

func (m *ToolCallManager) CurrentParent() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if len(m.parentStack) > 0 {
		return m.parentStack[len(m.parentStack)-1]
	}
	return ""
}

// ACPClient manages the claude-code-acp subprocess
type ACPClient struct {
	cmd       *exec.Cmd
	stdin     io.WriteCloser
	stdout    *bufio.Scanner
	sessionID string
	msgID     int
	mu        sync.Mutex
	callbacks map[int]chan JSONRPCMessage
	ctx       context.Context
	logFile   *os.File

	// Tool tracking
	toolManager     *ToolCallManager
	fileChangeStore *FileChangeStore

	// Permission handling
	permissionRespCh chan string
	permissionMsg    *JSONRPCMessage

	// Event name config
	chunkEventName     string // defaults to "chat_chunk"
	autoPermission     bool   // auto-allow all permissions (for review agent)
	suppressToolEvents bool   // don't emit tool_state events (for review agent)

	// Session modes
	currentModeID  string
	availableModes []SessionMode
}

// App struct
type App struct {
	ctx          context.Context
	client       *ACPClient
	mcpServer    *UserQuestionServer
	mcpServerURL string
}

func NewApp() *App {
	return &App{}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx

	// Start MCP server for AskUserQuestion
	a.mcpServer = NewUserQuestionServer(ctx)
	url, err := a.mcpServer.Start()
	if err != nil {
		slog.Error("failed to start MCP server", "error", err)
	} else {
		a.mcpServerURL = url
		slog.Info("MCP server started", "url", url)
	}

	// Create ACP client on startup to get modes immediately
	go func() {
		cwd, _ := os.Getwd()
		var mcpServers []any
		if a.mcpServerURL != "" {
			mcpServers = MCPServerConfig(a.mcpServerURL)
		} else {
			mcpServers = []any{}
		}
		client, err := NewACPClient(ctx, cwd, mcpServers)
		if err != nil {
			slog.Error("failed to create ACP client", "error", err)
			return
		}
		a.client = client

		// Emit modes to frontend
		if len(client.availableModes) > 0 {
			runtime.EventsEmit(ctx, "modes_available", client.availableModes)
			runtime.EventsEmit(ctx, "mode_changed", client.currentModeID)
		}

		// Start update listener
		go a.listenForUpdates()
	}()

	// Listen for frontend events
	runtime.EventsOn(ctx, "send_message", a.handleSendMessage)
	runtime.EventsOn(ctx, "permission_response", a.handlePermissionResponse)
	runtime.EventsOn(ctx, "user_answer", a.handleUserAnswer)
	runtime.EventsOn(ctx, "cancel", a.handleCancel)
	runtime.EventsOn(ctx, "submit_review", a.handleSubmitReview)
}

func (a *App) handleSendMessage(data ...interface{}) {
	if len(data) == 0 {
		return
	}
	input, ok := data[0].(string)
	if !ok {
		return
	}

	go func() {
		if a.client == nil {
			runtime.EventsEmit(a.ctx, "error", "Client not ready, please wait...")
			return
		}

		// Send prompt with auto-allowed MCP tools
		allowedTools := []string{
			"mcp__ccui__ccui_ask_user_question", // Auto-allow our ask user question tool
		}
		result, err := a.client.SendPrompt(input, allowedTools)
		if err != nil {
			slog.Error("prompt failed", "error", err)
			runtime.EventsEmit(a.ctx, "error", err.Error())
			return
		}

		runtime.EventsEmit(a.ctx, "prompt_complete", result.StopReason)
	}()
}

func (a *App) listenForUpdates() {
	<-a.ctx.Done()
}

func (a *App) handlePermissionResponse(data ...interface{}) {
	if a.client == nil || len(data) == 0 {
		return
	}
	optionID, ok := data[0].(string)
	if !ok {
		return
	}
	a.client.permissionRespCh <- optionID
}

func (a *App) handleUserAnswer(data ...interface{}) {
	if a.mcpServer == nil || len(data) == 0 {
		return
	}
	// Expect {requestId, answer} map
	answerMap, ok := data[0].(map[string]interface{})
	if !ok {
		return
	}
	requestID, _ := answerMap["requestId"].(string)
	answer, _ := answerMap["answer"].(string)
	a.mcpServer.HandleUserAnswer(UserAnswer{
		RequestID: requestID,
		Answer:    answer,
	})
}

func (a *App) handleCancel(data ...interface{}) {
	if a.client != nil {
		a.client.Cancel()
	}
}

func (a *App) handleSubmitReview(data ...interface{}) {
	if len(data) == 0 {
		return
	}
	// Parse comments from frontend
	commentsRaw, ok := data[0].([]interface{})
	if !ok {
		return
	}
	var comments []ReviewComment
	for _, c := range commentsRaw {
		cMap, ok := c.(map[string]interface{})
		if !ok {
			continue
		}
		comment := ReviewComment{
			ID:   cMap["id"].(string),
			Type: cMap["type"].(string),
			Text: cMap["text"].(string),
		}
		if fp, ok := cMap["filePath"].(string); ok {
			comment.FilePath = fp
		}
		if ln, ok := cMap["lineNumber"].(float64); ok {
			comment.LineNumber = int(ln)
		}
		if hi, ok := cMap["hunkIndex"].(float64); ok {
			comment.HunkIndex = int(hi)
		}
		comments = append(comments, comment)
	}
	a.SubmitReview(comments)
}

func (a *App) shutdown(ctx context.Context) {
	if a.mcpServer != nil {
		a.mcpServer.Stop()
	}
	if a.client != nil {
		a.client.Close()
	}
}

// SetMode changes the agent's session mode
func (a *App) SetMode(modeID string) error {
	if a.client == nil {
		return fmt.Errorf("no active session")
	}
	return a.client.SetMode(modeID)
}

// GetModes returns available session modes
func (a *App) GetModes() []SessionMode {
	if a.client == nil {
		return nil
	}
	return a.client.availableModes
}

// GetCurrentMode returns the current mode ID
func (a *App) GetCurrentMode() string {
	if a.client == nil {
		return ""
	}
	return a.client.currentModeID
}

// NewACPClient creates a new ACP client
func NewACPClient(ctx context.Context, cwd string, mcpServers []any) (*ACPClient, error) {
	cmd := exec.CommandContext(ctx, "claude-code-acp")
	cmd.Env = append(os.Environ(), "ANTHROPIC_API_KEY="+os.Getenv("ANTHROPIC_API_KEY"))
	cmd.Dir = cwd

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("stdin pipe: %w", err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("stdout pipe: %w", err)
	}
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("start: %w", err)
	}

	// Create log file
	logDir := filepath.Join(cwd, ".acp-logs")
	os.MkdirAll(logDir, 0755)
	logFile, err := os.Create(filepath.Join(logDir, fmt.Sprintf("events-%s.jsonl", time.Now().Format("2006-01-02-150405"))))
	if err != nil {
		slog.Warn("failed to create log file", "error", err)
	}

	c := &ACPClient{
		cmd:              cmd,
		stdin:            stdin,
		stdout:           bufio.NewScanner(stdout),
		callbacks:        make(map[int]chan JSONRPCMessage),
		ctx:              ctx,
		logFile:          logFile,
		toolManager:      NewToolCallManager(),
		fileChangeStore:  NewFileChangeStore(),
		permissionRespCh: make(chan string, 1),
	}

	go c.readLoop()

	if err := c.initialize(); err != nil {
		cmd.Process.Kill()
		return nil, fmt.Errorf("initialize: %w", err)
	}

	if err := c.newSession(cwd, mcpServers); err != nil {
		cmd.Process.Kill()
		return nil, fmt.Errorf("new session: %w", err)
	}

	return c, nil
}

// ACPClientConfig for customizing ACPClient behavior
type ACPClientConfig struct {
	CWD                string
	MCPServers         []any
	ChunkEventName     string // defaults to "chat_chunk"
	AutoPermission     bool   // auto-allow all permissions
	SuppressToolEvents bool   // don't emit tool_state events
}

// NewACPClientWithConfig creates a new ACP client with custom config
func NewACPClientWithConfig(ctx context.Context, cfg ACPClientConfig) (*ACPClient, error) {
	client, err := NewACPClient(ctx, cfg.CWD, cfg.MCPServers)
	if err != nil {
		return nil, err
	}
	if cfg.ChunkEventName != "" {
		client.chunkEventName = cfg.ChunkEventName
	}
	client.autoPermission = cfg.AutoPermission
	client.suppressToolEvents = cfg.SuppressToolEvents
	return client, nil
}

func (c *ACPClient) logEvent(eventType string, data any) {
	if c.logFile == nil {
		return
	}
	entry := map[string]any{
		"timestamp": time.Now().Format(time.RFC3339Nano),
		"type":      eventType,
		"data":      data,
	}
	if b, err := json.Marshal(entry); err == nil {
		c.logFile.Write(append(b, '\n'))
	}
}

func (c *ACPClient) closeAllCallbacks() {
	c.mu.Lock()
	defer c.mu.Unlock()
	for id, ch := range c.callbacks {
		close(ch)
		delete(c.callbacks, id)
	}
}

func (c *ACPClient) readLoop() {
	defer c.closeAllCallbacks()
	for c.stdout.Scan() {
		line := c.stdout.Bytes()
		if len(line) == 0 || line[0] != '{' {
			continue
		}

		var msg JSONRPCMessage
		if err := json.Unmarshal(line, &msg); err != nil {
			continue
		}

		// Log raw message
		c.logEvent("raw_message", json.RawMessage(line))

		// Check Method BEFORE ID - requests have both
		if msg.Method != "" {
			c.handleMethod(msg)
		} else if msg.ID != nil {
			c.mu.Lock()
			if ch, ok := c.callbacks[*msg.ID]; ok {
				ch <- msg
				delete(c.callbacks, *msg.ID)
			}
			c.mu.Unlock()
		}
	}
}

func (c *ACPClient) handleMethod(msg JSONRPCMessage) {
	c.logEvent("method", map[string]any{"method": msg.Method, "params": msg.Params})

	switch msg.Method {
	case "session/update":
		var update SessionUpdate
		json.Unmarshal(msg.Params, &update)
		c.handleSessionUpdate(update)

	case "session/request_permission":
		var req PermissionRequest
		json.Unmarshal(msg.Params, &req)
		c.logEvent("permission_request", req)

		// Auto-allow our MCP ask user question tool
		if req.ToolCall.Title == "mcp__ccui__ccui_ask_user_question" {
			c.sendPermissionResponse(msg.ID, "allow_always")
			return
		}

		// Auto-allow all permissions for review agent
		if c.autoPermission {
			c.sendPermissionResponse(msg.ID, "allow_always")
			return
		}

		// Update tool state with permission options
		state := c.toolManager.Update(req.ToolCall.ToolCallID, func(s *ToolState) {
			s.Status = "awaiting_permission"
			s.PermissionOptions = req.Options
		})
		if state != nil {
			runtime.EventsEmit(c.ctx, "tool_state", state)
		}

		// Store msg for response
		c.permissionMsg = &msg

		// Wait for response from frontend
		optionID := <-c.permissionRespCh
		c.sendPermissionResponse(msg.ID, optionID)
	}
}

func (c *ACPClient) handleSessionUpdate(update SessionUpdate) {
	u := update.Update
	c.logEvent("session_update", u)

	switch u.SessionUpdate {
	case "agent_message_chunk", "agent_thought_chunk":
		// Parse content as TextContent
		var content TextContent
		if len(u.Content) > 0 {
			json.Unmarshal(u.Content, &content)
		}
		eventName := c.chunkEventName
		if eventName == "" {
			eventName = "chat_chunk"
		}
		if u.SessionUpdate == "agent_message_chunk" {
			runtime.EventsEmit(c.ctx, eventName, content.Text)
		} else {
			runtime.EventsEmit(c.ctx, "chat_thought", content.Text)
		}

	case "tool_call":
		// Skip tool_state events for review agent
		if c.suppressToolEvents {
			return
		}

		// Check if we already have this tool (tool_call events can come multiple times)
		existing := c.toolManager.Get(u.ToolCallID)
		if existing != nil {
			// Update existing tool state
			existing.Status = u.Status
			existing.Title = u.Title
			if u.RawInput != nil {
				existing.Input = u.RawInput
			}
			runtime.EventsEmit(c.ctx, "tool_state", existing)
			return
		}

		// Parse content as []DiffBlock for diffs
		var diffs []DiffBlock
		if len(u.Content) > 0 && u.Content[0] == '[' {
			json.Unmarshal(u.Content, &diffs)
		}

		// Extract toolName from meta
		var toolName string
		if u.Meta != nil && u.Meta.ClaudeCode != nil {
			toolName = u.Meta.ClaudeCode.ToolName
		}

		// Get current parent (if any) before potentially pushing this tool
		parentID := c.toolManager.CurrentParent()

		// Create new tool state
		state := &ToolState{
			ID:       u.ToolCallID,
			Status:   u.Status,
			Title:    u.Title,
			Kind:     u.ToolKind,
			ToolName: toolName,
			ParentID: parentID,
			Input:    u.RawInput,
			Diffs:    diffs,
		}

		// If this is a Task tool, push it as a parent for subsequent tools
		if toolName == "Task" {
			c.toolManager.PushParent(u.ToolCallID)
		}

		c.toolManager.Set(state)
		runtime.EventsEmit(c.ctx, "tool_state", state)

	case "tool_call_update":
		// For suppressed tool events, only track file changes
		if c.suppressToolEvents {
			if u.Meta != nil && u.Meta.ClaudeCode != nil && u.Meta.ClaudeCode.ToolResponse != nil {
				tr := u.Meta.ClaudeCode.ToolResponse
				toolName := ""
				if u.Meta.ClaudeCode.ToolName != "" {
					toolName = u.Meta.ClaudeCode.ToolName
				}
				if tr.FilePath != "" && (toolName == "Edit" || toolName == "Write") {
					currentContent := tr.Content
					if toolName == "Edit" && tr.Content == "" {
						base := tr.OriginalFile
						if existing := c.fileChangeStore.Get(tr.FilePath); existing != nil {
							base = existing.CurrentContent
						}
						currentContent = strings.Replace(base, tr.OldString, tr.NewString, 1)
					}
					c.fileChangeStore.RecordChange(tr.FilePath, tr.OriginalFile, currentContent, tr.StructuredPatch)
					runtime.EventsEmit(c.ctx, "file_changes_updated", c.fileChangeStore.GetAll())
				}
			}
			return
		}

		// Update existing tool state
		state := c.toolManager.Update(u.ToolCallID, func(s *ToolState) {
			s.Status = u.Status
			s.Output = u.Output

			// Extract diff from _meta
			if u.Meta != nil && u.Meta.ClaudeCode != nil && u.Meta.ClaudeCode.ToolResponse != nil {
				tr := u.Meta.ClaudeCode.ToolResponse
				if len(tr.StructuredPatch) > 0 || tr.OldString != "" || tr.NewString != "" {
					s.Diff = map[string]any{
						"filePath":        tr.FilePath,
						"oldString":       tr.OldString,
						"newString":       tr.NewString,
						"originalFile":    tr.OriginalFile,
						"structuredPatch": tr.StructuredPatch,
						"type":            tr.Type,
						"content":         tr.Content,
					}
				}

				// Track file changes for review
				if tr.FilePath != "" && (s.ToolName == "Edit" || s.ToolName == "Write") {
					currentContent := tr.Content
					if s.ToolName == "Edit" && tr.Content == "" {
						// Edit tool: compute current from original + edit
						base := tr.OriginalFile
						if existing := c.fileChangeStore.Get(tr.FilePath); existing != nil {
							base = existing.CurrentContent
						}
						currentContent = strings.Replace(base, tr.OldString, tr.NewString, 1)
					}
					c.fileChangeStore.RecordChange(tr.FilePath, tr.OriginalFile, currentContent, tr.StructuredPatch)
					runtime.EventsEmit(c.ctx, "file_changes_updated", c.fileChangeStore.GetAll())
				}
			}
		})
		if state != nil {
			// Pop from parent stack if Task tool completed
			if state.ToolName == "Task" && (u.Status == "completed" || u.Status == "error" || u.Status == "failed") {
				c.toolManager.PopParent(u.ToolCallID)
			}
			runtime.EventsEmit(c.ctx, "tool_state", state)
		}

	case "current_mode_update":
		c.currentModeID = u.ModeID
		runtime.EventsEmit(c.ctx, "mode_changed", u.ModeID)

	case "plan":
		runtime.EventsEmit(c.ctx, "plan_update", u.Entries)
	}
}

func (c *ACPClient) send(method string, params any) (JSONRPCMessage, error) {
	c.mu.Lock()
	c.msgID++
	id := c.msgID
	ch := make(chan JSONRPCMessage, 1)
	c.callbacks[id] = ch
	c.mu.Unlock()

	paramsJSON, _ := json.Marshal(params)
	msg := JSONRPCMessage{
		JSONRPC: "2.0",
		ID:      &id,
		Method:  method,
		Params:  paramsJSON,
	}

	data, _ := json.Marshal(msg)
	c.logEvent("send", msg)

	if _, err := c.stdin.Write(append(data, '\n')); err != nil {
		return JSONRPCMessage{}, err
	}

	resp, ok := <-ch
	if !ok {
		return JSONRPCMessage{}, fmt.Errorf("connection closed")
	}
	if resp.Error != nil {
		return resp, fmt.Errorf("rpc error %d: %s", resp.Error.Code, resp.Error.Message)
	}
	return resp, nil
}

func (c *ACPClient) notify(method string, params any) {
	paramsJSON, _ := json.Marshal(params)
	msg := JSONRPCMessage{
		JSONRPC: "2.0",
		Method:  method,
		Params:  paramsJSON,
	}
	data, _ := json.Marshal(msg)
	c.logEvent("notify", msg)
	c.stdin.Write(append(data, '\n'))
}

func (c *ACPClient) initialize() error {
	_, err := c.send("initialize", InitializeParams{
		ProtocolVersion: 1,
		ClientCapabilities: ClientCapabilities{
			// Let agent handle file/terminal ops internally
			Terminal: false,
		},
	})
	return err
}

func (c *ACPClient) newSession(cwd string, mcpServers []any) error {
	resp, err := c.send("session/new", map[string]any{
		"cwd":        cwd,
		"mcpServers": mcpServers,
	})
	if err != nil {
		return err
	}

	var result SessionNewResult
	json.Unmarshal(resp.Result, &result)
	c.sessionID = result.SessionID
	if result.Modes != nil {
		c.currentModeID = result.Modes.CurrentModeID
		c.availableModes = result.Modes.AvailableModes
	}
	return nil
}

func (c *ACPClient) SendPrompt(text string, allowedTools []string) (SessionPromptResult, error) {
	resp, err := c.send("session/prompt", SessionPromptParams{
		SessionID:    c.sessionID,
		Prompt:       []PromptContent{{Type: "text", Text: text}},
		AllowedTools: allowedTools,
	})
	if err != nil {
		return SessionPromptResult{}, err
	}
	var result SessionPromptResult
	json.Unmarshal(resp.Result, &result)
	return result, nil
}

func (c *ACPClient) SetMode(modeID string) error {
	_, err := c.send("session/set_mode", map[string]string{
		"sessionId": c.sessionID,
		"modeId":    modeID,
	})
	if err == nil {
		c.currentModeID = modeID
		runtime.EventsEmit(c.ctx, "mode_changed", modeID)
	}
	return err
}

func (c *ACPClient) sendPermissionResponse(id *int, optionID string) {
	resp := JSONRPCMessage{
		JSONRPC: "2.0",
		ID:      id,
	}
	result, _ := json.Marshal(PermissionResponse{
		Outcome: PermissionOutcome{Outcome: "selected", OptionID: optionID},
	})
	resp.Result = result
	data, _ := json.Marshal(resp)
	c.logEvent("permission_response", resp)
	c.stdin.Write(append(data, '\n'))
}

func (c *ACPClient) Cancel() {
	c.notify("session/cancel", map[string]string{"sessionId": c.sessionID})
}

func (c *ACPClient) Close() error {
	if c.logFile != nil {
		c.logFile.Close()
	}
	c.stdin.Close()
	return c.cmd.Wait()
}

// ReviewComment from frontend
type ReviewComment struct {
	ID         string `json:"id"`
	Type       string `json:"type"` // line, hunk, general
	FilePath   string `json:"filePath,omitempty"`
	LineNumber int    `json:"lineNumber,omitempty"`
	HunkIndex  int    `json:"hunkIndex,omitempty"`
	Text       string `json:"text"`
}

// SubmitReview spawns a fresh acp subprocess to address review feedback
func (a *App) SubmitReview(comments []ReviewComment) {
	if a.client == nil {
		return
	}

	changes := a.client.fileChangeStore.GetAll()
	if len(changes) == 0 && len(comments) == 0 {
		return
	}

	go func() {
		runtime.EventsEmit(a.ctx, "review_agent_running", true)

		// Build prompt
		var prompt strings.Builder
		prompt.WriteString("Review feedback for recent changes:\n\n")

		for _, c := range changes {
			prompt.WriteString(fmt.Sprintf("## File: %s\n", c.FilePath))
			prompt.WriteString("```diff\n")
			for _, h := range c.Hunks {
				prompt.WriteString(fmt.Sprintf("@@ -%d,%d +%d,%d @@\n", h.OldStart, h.OldLines, h.NewStart, h.NewLines))
				for _, line := range h.Lines {
					prompt.WriteString(line + "\n")
				}
			}
			prompt.WriteString("```\n\n")
		}

		prompt.WriteString("## Review Comments:\n")
		for _, c := range comments {
			switch c.Type {
			case "line":
				prompt.WriteString(fmt.Sprintf("- [%s:%d] %s\n", c.FilePath, c.LineNumber, c.Text))
			case "hunk":
				prompt.WriteString(fmt.Sprintf("- [%s hunk %d] %s\n", c.FilePath, c.HunkIndex+1, c.Text))
			default:
				prompt.WriteString(fmt.Sprintf("- [General] %s\n", c.Text))
			}
		}
		prompt.WriteString("\nPlease address this feedback by making the necessary changes.")

		// Create fresh ACPClient for review agent
		cwd, _ := os.Getwd()
		reviewClient, err := NewACPClientWithConfig(a.ctx, ACPClientConfig{
			CWD:                cwd,
			MCPServers:         []any{},
			ChunkEventName:     "review_agent_chunk",
			AutoPermission:     true,
			SuppressToolEvents: true,
		})
		if err != nil {
			runtime.EventsEmit(a.ctx, "review_agent_chunk", "Error: "+err.Error())
			runtime.EventsEmit(a.ctx, "review_agent_complete", nil)
			return
		}

		// Share fileChangeStore with main client so review changes are tracked
		reviewClient.fileChangeStore = a.client.fileChangeStore

		// SendPrompt blocks until complete (callback-based)
		_, err = reviewClient.SendPrompt(prompt.String(), []string{})

		if err != nil {
			runtime.EventsEmit(a.ctx, "review_agent_chunk", "\nError: "+err.Error())
		}

		// Emit complete BEFORE Close() which may block on cmd.Wait()
		runtime.EventsEmit(a.ctx, "review_agent_complete", nil)

		// Close in background to avoid blocking
		go reviewClient.Close()
	}()
}
