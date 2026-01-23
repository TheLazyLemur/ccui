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

type SessionNewResult struct {
	SessionID string `json:"sessionId"`
}

type PromptContent struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
}

type SessionPromptParams struct {
	SessionID string          `json:"sessionId"`
	Prompt    []PromptContent `json:"prompt"`
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
	toolManager *ToolCallManager

	// Permission handling
	permissionRespCh chan string
	permissionMsg    *JSONRPCMessage
}

// App struct
type App struct {
	ctx    context.Context
	client *ACPClient
}

func NewApp() *App {
	return &App{}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx

	// Listen for frontend events
	runtime.EventsOn(ctx, "send_message", a.handleSendMessage)
	runtime.EventsOn(ctx, "permission_response", a.handlePermissionResponse)
	runtime.EventsOn(ctx, "cancel", a.handleCancel)
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
		// Initialize client if needed
		if a.client == nil {
			cwd, _ := os.Getwd()
			client, err := NewACPClient(a.ctx, cwd)
			if err != nil {
				slog.Error("failed to create ACP client", "error", err)
				runtime.EventsEmit(a.ctx, "error", err.Error())
				return
			}
			a.client = client

			// Start update listener
			go a.listenForUpdates()
		}

		// Send prompt
		result, err := a.client.SendPrompt(input)
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

func (a *App) handleCancel(data ...interface{}) {
	if a.client != nil {
		a.client.Cancel()
	}
}

func (a *App) shutdown(ctx context.Context) {
	if a.client != nil {
		a.client.Close()
	}
}

// NewACPClient creates a new ACP client
func NewACPClient(ctx context.Context, cwd string) (*ACPClient, error) {
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
		permissionRespCh: make(chan string, 1),
	}

	go c.readLoop()

	if err := c.initialize(); err != nil {
		cmd.Process.Kill()
		return nil, fmt.Errorf("initialize: %w", err)
	}

	if err := c.newSession(cwd); err != nil {
		cmd.Process.Kill()
		return nil, fmt.Errorf("new session: %w", err)
	}

	return c, nil
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

func (c *ACPClient) readLoop() {
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
		if u.SessionUpdate == "agent_message_chunk" {
			runtime.EventsEmit(c.ctx, "chat_chunk", content.Text)
		} else {
			runtime.EventsEmit(c.ctx, "chat_thought", content.Text)
		}

	case "tool_call":
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
			}
		})
		if state != nil {
			// Pop from parent stack if Task tool completed
			if state.ToolName == "Task" && (u.Status == "completed" || u.Status == "error" || u.Status == "failed") {
				c.toolManager.PopParent(u.ToolCallID)
			}
			runtime.EventsEmit(c.ctx, "tool_state", state)
		}
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

	resp := <-ch
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

func (c *ACPClient) newSession(cwd string) error {
	resp, err := c.send("session/new", map[string]any{
		"cwd":        cwd,
		"mcpServers": []any{},
	})
	if err != nil {
		return err
	}

	var result SessionNewResult
	json.Unmarshal(resp.Result, &result)
	c.sessionID = result.SessionID
	return nil
}

func (c *ACPClient) SendPrompt(text string) (SessionPromptResult, error) {
	resp, err := c.send("session/prompt", SessionPromptParams{
		SessionID: c.sessionID,
		Prompt:    []PromptContent{{Type: "text", Text: text}},
	})
	if err != nil {
		return SessionPromptResult{}, err
	}
	var result SessionPromptResult
	json.Unmarshal(resp.Result, &result)
	return result, nil
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
