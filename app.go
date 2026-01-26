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
	RawOutput     *ToolRawOutput  `json:"rawOutput,omitempty"`
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

type ToolRawOutput struct {
	Output   string              `json:"output,omitempty"`
	Metadata *ToolOutputMetadata `json:"metadata,omitempty"`
}

type ToolOutputMetadata struct {
	Diff      string    `json:"diff,omitempty"`
	Filediff  *FileDiff `json:"filediff,omitempty"`
	Filepath  string    `json:"filepath,omitempty"`
	Exists    bool      `json:"exists,omitempty"`
	Truncated bool      `json:"truncated,omitempty"`
}

type FileDiff struct {
	File      string `json:"file,omitempty"`
	Before    string `json:"before,omitempty"`
	After     string `json:"after,omitempty"`
	Additions int    `json:"additions,omitempty"`
	Deletions int    `json:"deletions,omitempty"`
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
	toolAdapters    []ToolEventAdapter

	// Permission handling
	permissionRespCh chan string
	permissionMsg    *JSONRPCMessage

	// Event name config
	eventPrefix        string // e.g. "session:abc123:"
	chunkEventName     string // defaults to "chat_chunk"
	autoPermission     bool   // auto-allow all permissions (for review agent)
	suppressToolEvents bool   // don't emit tool_state events (for review agent)

	// Session modes
	currentModeID  string
	availableModes []SessionMode
}

// SessionInfo for frontend
type SessionInfo struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	CreatedAt string `json:"createdAt"`
	ModeID    string `json:"modeId"`
}

// SessionState holds per-session data
type SessionState struct {
	ID        string
	Name      string
	CreatedAt time.Time
	Client    *ACPClient
}

// App struct
type App struct {
	ctx             context.Context
	mcpServer       *UserQuestionServer
	mcpServerURL    string
	sessions        map[string]*SessionState
	activeSessionID string
	sessionMu       sync.RWMutex
	ptyManager      *PTYManager
}

func NewApp() *App {
	return &App{
		sessions: make(map[string]*SessionState),
	}
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

	// Listen for frontend events
	runtime.EventsOn(ctx, "send_message", a.handleSendMessage)
	runtime.EventsOn(ctx, "permission_response", a.handlePermissionResponse)
	runtime.EventsOn(ctx, "user_answer", a.handleUserAnswer)
	runtime.EventsOn(ctx, "cancel", a.handleCancel)
	runtime.EventsOn(ctx, "submit_review", a.handleSubmitReview)

	// Terminal PTY support
	a.StartTerminalListeners()
}

// CreateSession creates a new session with the given name
func (a *App) CreateSession(name string) (string, error) {
	cwd, _ := os.Getwd()
	mcpServers := a.getMCPServers()
	sessionID := fmt.Sprintf("session-%d", time.Now().UnixNano())
	eventPrefix := fmt.Sprintf("session:%s:", sessionID)

	client, err := NewACPClientWithConfig(a.ctx, ACPClientConfig{
		CWD:         cwd,
		MCPServers:  mcpServers,
		EventPrefix: eventPrefix,
	})
	if err != nil {
		return "", fmt.Errorf("create ACP client: %w", err)
	}

	state := &SessionState{
		ID:        sessionID,
		Name:      name,
		CreatedAt: time.Now(),
		Client:    client,
	}

	a.sessionMu.Lock()
	a.sessions[sessionID] = state
	a.activeSessionID = sessionID
	a.sessionMu.Unlock()

	// Emit session events
	runtime.EventsEmit(a.ctx, "sessions_updated", a.GetSessions())
	runtime.EventsEmit(a.ctx, "active_session_changed", sessionID)

	// Emit modes for this session
	if len(client.availableModes) > 0 {
		runtime.EventsEmit(a.ctx, eventPrefix+"modes_available", client.availableModes)
		runtime.EventsEmit(a.ctx, eventPrefix+"mode_changed", client.currentModeID)
	}

	return sessionID, nil
}

// SwitchSession switches to an existing session
func (a *App) SwitchSession(sessionID string) error {
	a.sessionMu.Lock()
	defer a.sessionMu.Unlock()

	if a.sessions[sessionID] == nil {
		return fmt.Errorf("session not found: %s", sessionID)
	}
	a.activeSessionID = sessionID
	runtime.EventsEmit(a.ctx, "active_session_changed", sessionID)
	return nil
}

// CloseSession closes and removes a session
func (a *App) CloseSession(sessionID string) error {
	a.sessionMu.Lock()
	defer a.sessionMu.Unlock()

	state := a.sessions[sessionID]
	if state == nil {
		return fmt.Errorf("session not found: %s", sessionID)
	}

	if state.Client != nil {
		go state.Client.Close()
	}
	delete(a.sessions, sessionID)

	// Switch to another session if closing active
	if a.activeSessionID == sessionID {
		a.activeSessionID = a.pickNextSession()
	}

	runtime.EventsEmit(a.ctx, "sessions_updated", a.getSessionsLocked())
	runtime.EventsEmit(a.ctx, "active_session_changed", a.activeSessionID)
	return nil
}

// pickNextSession returns any remaining session ID or empty
func (a *App) pickNextSession() string {
	for id := range a.sessions {
		return id
	}
	return ""
}

// GetSessions returns info for all sessions
func (a *App) GetSessions() []SessionInfo {
	a.sessionMu.RLock()
	defer a.sessionMu.RUnlock()
	return a.getSessionsLocked()
}

func (a *App) getSessionsLocked() []SessionInfo {
	result := make([]SessionInfo, 0, len(a.sessions))
	for _, s := range a.sessions {
		info := SessionInfo{
			ID:        s.ID,
			Name:      s.Name,
			CreatedAt: s.CreatedAt.Format(time.RFC3339),
		}
		if s.Client != nil {
			info.ModeID = s.Client.currentModeID
		}
		result = append(result, info)
	}
	return result
}

// GetActiveSession returns the active session ID
func (a *App) GetActiveSession() string {
	a.sessionMu.RLock()
	defer a.sessionMu.RUnlock()
	return a.activeSessionID
}

// getActiveClient returns the active session's client
func (a *App) getActiveClient() *ACPClient {
	a.sessionMu.RLock()
	defer a.sessionMu.RUnlock()
	if state := a.sessions[a.activeSessionID]; state != nil {
		return state.Client
	}
	return nil
}

// getMCPServers returns MCP server config or empty slice
func (a *App) getMCPServers() []any {
	if a.mcpServerURL != "" {
		return MCPServerConfig(a.mcpServerURL)
	}
	return []any{}
}

func (a *App) handleSendMessage(data ...interface{}) {
	input, ok := firstAs[string](data)
	if !ok {
		return
	}

	go func() {
		client := a.getActiveClient()
		if client == nil {
			runtime.EventsEmit(a.ctx, "error", "No active session")
			return
		}

		// Send prompt with auto-allowed MCP tools
		allowedTools := []string{
			"mcp__ccui__ccui_ask_user_question", // Auto-allow our ask user question tool
		}
		result, err := client.SendPrompt(input, allowedTools)
		if err != nil {
			slog.Error("prompt failed", "error", err)
			runtime.EventsEmit(a.ctx, client.eventPrefix+"error", err.Error())
			return
		}

		runtime.EventsEmit(a.ctx, client.eventPrefix+"prompt_complete", result.StopReason)
	}()
}

func (a *App) handlePermissionResponse(data ...interface{}) {
	optionID, ok := firstAs[string](data)
	if !ok {
		return
	}
	if client := a.getActiveClient(); client != nil {
		client.permissionRespCh <- optionID
	}
}

// firstAs extracts and casts first element from variadic data
func firstAs[T any](data []interface{}) (T, bool) {
	var zero T
	if len(data) == 0 {
		return zero, false
	}
	v, ok := data[0].(T)
	return v, ok
}

func (a *App) handleUserAnswer(data ...interface{}) {
	if a.mcpServer == nil {
		return
	}
	answerMap, ok := firstAs[map[string]interface{}](data)
	if !ok {
		return
	}
	a.mcpServer.HandleUserAnswer(UserAnswer{
		RequestID: mapStr(answerMap, "requestId"),
		Answer:    mapStr(answerMap, "answer"),
	})
}

func (a *App) handleCancel(data ...interface{}) {
	if client := a.getActiveClient(); client != nil {
		client.Cancel()
	}
}

func (a *App) handleSubmitReview(data ...interface{}) {
	commentsRaw, ok := firstAs[[]interface{}](data)
	if !ok {
		return
	}
	comments := parseReviewComments(commentsRaw)
	a.SubmitReview(comments)
}

func parseReviewComments(raw []interface{}) []ReviewComment {
	var comments []ReviewComment
	for _, c := range raw {
		m, ok := c.(map[string]interface{})
		if !ok {
			continue
		}
		comment := ReviewComment{
			ID:         mapStr(m, "id"),
			Type:       mapStr(m, "type"),
			Text:       mapStr(m, "text"),
			FilePath:   mapStr(m, "filePath"),
			LineNumber: mapInt(m, "lineNumber"),
			HunkIndex:  mapInt(m, "hunkIndex"),
		}
		comments = append(comments, comment)
	}
	return comments
}

func mapStr(m map[string]interface{}, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}

func mapInt(m map[string]interface{}, key string) int {
	if v, ok := m[key].(float64); ok {
		return int(v)
	}
	return 0
}

func (a *App) shutdown(ctx context.Context) {
	if a.mcpServer != nil {
		a.mcpServer.Stop()
	}
	if a.ptyManager != nil {
		a.ptyManager.StopAll()
	}
	a.sessionMu.Lock()
	for _, s := range a.sessions {
		if s.Client != nil {
			s.Client.Close()
		}
	}
	a.sessionMu.Unlock()
}

// SetMode changes the agent's session mode
func (a *App) SetMode(modeID string) error {
	client := a.getActiveClient()
	if client == nil {
		return fmt.Errorf("no active session")
	}
	return client.SetMode(modeID)
}

// GetModes returns available session modes
func (a *App) GetModes() []SessionMode {
	client := a.getActiveClient()
	if client == nil {
		return nil
	}
	return client.availableModes
}

// GetCurrentMode returns the current mode ID
func (a *App) GetCurrentMode() string {
	client := a.getActiveClient()
	if client == nil {
		return ""
	}
	return client.currentModeID
}

// NewACPClient creates a new ACP client
func NewACPClient(ctx context.Context, cwd string, mcpServers []any) (*ACPClient, error) {
	cmd := exec.CommandContext(ctx, "claude-code-acp")
	// cmd := exec.CommandContext(ctx, "opencode", "acp")
	// cmd := exec.CommandContext(ctx, "/Users/danrousseau/Programming/ai-agents/acp-glm/acp-glm")
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
		toolAdapters:     defaultToolAdapters(),
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
	EventPrefix        string // e.g. "session:abc123:"
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
	client.eventPrefix = cfg.EventPrefix
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

// emit emits an event with the session prefix
func (c *ACPClient) emit(eventName string, data any) {
	runtime.EventsEmit(c.ctx, c.eventPrefix+eventName, data)
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
			c.emit("tool_state", state)
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
		var content TextContent
		if len(u.Content) > 0 {
			json.Unmarshal(u.Content, &content)
		}
		if u.SessionUpdate == "agent_message_chunk" {
			eventName := c.chunkEventName
			if eventName == "" {
				eventName = "chat_chunk"
			}
			c.emit(eventName, content.Text)
		} else {
			c.emit("chat_thought", content.Text)
		}

	case "tool_call":
		if c.suppressToolEvents {
			return
		}
		c.handleToolCall(u)

	case "tool_call_update":
		c.handleToolCallUpdate(u)

	case "current_mode_update":
		c.currentModeID = u.ModeID
		c.emit("mode_changed", u.ModeID)

	case "plan":
		c.emit("plan_update", u.Entries)
	}
}

func (c *ACPClient) handleToolCall(u UpdateContent) {
	adapter := c.adapterFor(u)
	toolName := resolveToolName(adapter, u)
	var diffs []DiffBlock
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
		c.emit("tool_state", existing)
		return
	}
	state := &ToolState{
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
	c.emit("tool_state", state)
}

func (c *ACPClient) handleToolCallUpdate(u UpdateContent) {
	adapter := c.adapterFor(u)
	toolName := resolveToolName(adapter, u)
	var diffs []DiffBlock
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
	state := c.toolManager.Update(u.ToolCallID, func(s *ToolState) {
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
	c.emit("tool_state", state)
}

func (c *ACPClient) trackFileChange(toolName string, tr *ToolResponse) {
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
	c.emit("file_changes_updated", c.fileChangeStore.GetAll())
}

func isTerminalStatus(status string) bool {
	return status == "completed" || status == "error" || status == "failed"
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
	if err != nil {
		return err
	}
	c.currentModeID = modeID
	c.emit("mode_changed", modeID)
	return nil
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
	client := a.getActiveClient()
	if client == nil {
		return
	}

	changes := client.fileChangeStore.GetAll()
	if len(changes) == 0 && len(comments) == 0 {
		return
	}

	eventPrefix := client.eventPrefix

	go func() {
		runtime.EventsEmit(a.ctx, eventPrefix+"review_agent_running", true)

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
			EventPrefix:        eventPrefix,
			ChunkEventName:     "review_agent_chunk",
			AutoPermission:     true,
			SuppressToolEvents: true,
		})
		if err != nil {
			runtime.EventsEmit(a.ctx, eventPrefix+"review_agent_chunk", "Error: "+err.Error())
			runtime.EventsEmit(a.ctx, eventPrefix+"review_agent_complete", nil)
			return
		}

		// Share fileChangeStore with main client so review changes are tracked
		reviewClient.fileChangeStore = client.fileChangeStore

		// SendPrompt blocks until complete (callback-based)
		_, err = reviewClient.SendPrompt(prompt.String(), []string{})

		if err != nil {
			runtime.EventsEmit(a.ctx, eventPrefix+"review_agent_chunk", "\nError: "+err.Error())
		}

		// Emit complete BEFORE Close() which may block on cmd.Wait()
		runtime.EventsEmit(a.ctx, eventPrefix+"review_agent_complete", nil)

		// Close in background to avoid blocking
		go reviewClient.Close()
	}()
}
