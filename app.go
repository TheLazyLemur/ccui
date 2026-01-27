package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"ccui/backend"
	"ccui/backend/acp"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type SessionMode = backend.SessionMode // Wails binding compatibility

type SessionInfo struct{ ID, Name, CreatedAt, ModeID string }

type SessionState struct {
	ID, Name  string
	CreatedAt time.Time
	Client    *acp.Client
	EventChan chan backend.Event
}

type App struct {
	ctx             context.Context
	mcpServer       *UserQuestionServer
	mcpServerURL    string
	sessions        map[string]*SessionState
	activeSessionID string
	sessionMu       sync.RWMutex
	ptyManager      *PTYManager
}

func NewApp() *App { return &App{sessions: make(map[string]*SessionState)} }

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	a.mcpServer = NewUserQuestionServer(ctx)
	if url, err := a.mcpServer.Start(); err != nil {
		slog.Error("failed to start MCP server", "error", err)
	} else {
		a.mcpServerURL = url
	}
	runtime.EventsOn(ctx, "send_message", a.handleSendMessage)
	runtime.EventsOn(ctx, "permission_response", a.handlePermissionResponse)
	runtime.EventsOn(ctx, "user_answer", a.handleUserAnswer)
	runtime.EventsOn(ctx, "cancel", a.handleCancel)
	runtime.EventsOn(ctx, "submit_review", a.handleSubmitReview)
	a.StartTerminalListeners()
}

func (a *App) CreateSession(name string) (string, error) {
	cwd, _ := os.Getwd()
	sessionID := fmt.Sprintf("session-%d", time.Now().UnixNano())
	eventPrefix := fmt.Sprintf("session:%s:", sessionID)
	eventChan := make(chan backend.Event, 100)
	client, err := a.createClient(cwd, a.getMCPServers(), eventChan, false, false)
	if err != nil {
		close(eventChan)
		return "", fmt.Errorf("create ACP client: %w", err)
	}
	state := &SessionState{ID: sessionID, Name: name, CreatedAt: time.Now(), Client: client, EventChan: eventChan}
	go a.bridgeEvents(eventPrefix, eventChan, "chat_chunk")
	a.sessionMu.Lock()
	a.sessions[sessionID], a.activeSessionID = state, sessionID
	a.sessionMu.Unlock()
	runtime.EventsEmit(a.ctx, "sessions_updated", a.GetSessions())
	runtime.EventsEmit(a.ctx, "active_session_changed", sessionID)
	if modes := client.AvailableModes(); len(modes) > 0 {
		runtime.EventsEmit(a.ctx, eventPrefix+"modes_available", modes)
		runtime.EventsEmit(a.ctx, eventPrefix+"mode_changed", client.CurrentMode())
	}
	return sessionID, nil
}

func (a *App) createClient(cwd string, mcpServers []any, eventChan chan backend.Event, autoPermission, suppressToolEvents bool) (*acp.Client, error) {
	cmd := exec.CommandContext(a.ctx, "claude-code-acp")
	cmd.Env, cmd.Dir, cmd.Stderr = append(os.Environ(), "ANTHROPIC_API_KEY="+os.Getenv("ANTHROPIC_API_KEY")), cwd, os.Stderr
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("stdin pipe: %w", err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("stdout pipe: %w", err)
	}
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("start: %w", err)
	}
	client := acp.NewClient(acp.ClientConfig{
		Transport: acp.NewStdioTransport(stdin, stdout), EventChan: eventChan,
		AutoPermission: autoPermission, SuppressToolEvents: suppressToolEvents,
	})
	if err := client.Initialize(); err != nil {
		cmd.Process.Kill()
		return nil, fmt.Errorf("initialize: %w", err)
	}
	if err := client.NewSession(cwd, mcpServers); err != nil {
		cmd.Process.Kill()
		return nil, fmt.Errorf("new session: %w", err)
	}
	return client, nil
}

func (a *App) bridgeEvents(prefix string, eventChan <-chan backend.Event, chunkEventName string) {
	for event := range eventChan {
		switch event.Type {
		case backend.EventMessageChunk:
			runtime.EventsEmit(a.ctx, prefix+chunkEventName, event.Data)
		case backend.EventThoughtChunk:
			runtime.EventsEmit(a.ctx, prefix+"chat_thought", event.Data)
		case backend.EventToolState:
			runtime.EventsEmit(a.ctx, prefix+"tool_state", event.Data)
		case backend.EventModeChanged:
			runtime.EventsEmit(a.ctx, prefix+"mode_changed", event.Data)
		case backend.EventPlanUpdate:
			runtime.EventsEmit(a.ctx, prefix+"plan_update", event.Data)
		case backend.EventPromptComplete:
			runtime.EventsEmit(a.ctx, prefix+"prompt_complete", event.Data)
		case backend.EventFileChanges:
			runtime.EventsEmit(a.ctx, prefix+"file_changes_updated", event.Data)
		}
	}
}

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
	if state.EventChan != nil {
		close(state.EventChan)
	}
	delete(a.sessions, sessionID)
	if a.activeSessionID == sessionID {
		for id := range a.sessions {
			a.activeSessionID = id
			break
		}
		if len(a.sessions) == 0 {
			a.activeSessionID = ""
		}
	}
	runtime.EventsEmit(a.ctx, "sessions_updated", a.getSessionsLocked())
	runtime.EventsEmit(a.ctx, "active_session_changed", a.activeSessionID)
	return nil
}

func (a *App) GetSessions() []SessionInfo {
	a.sessionMu.RLock()
	defer a.sessionMu.RUnlock()
	return a.getSessionsLocked()
}

func (a *App) getSessionsLocked() []SessionInfo {
	result := make([]SessionInfo, 0, len(a.sessions))
	for _, s := range a.sessions {
		info := SessionInfo{ID: s.ID, Name: s.Name, CreatedAt: s.CreatedAt.Format(time.RFC3339)}
		if s.Client != nil {
			info.ModeID = s.Client.CurrentMode()
		}
		result = append(result, info)
	}
	return result
}

func (a *App) GetActiveSession() string {
	a.sessionMu.RLock()
	defer a.sessionMu.RUnlock()
	return a.activeSessionID
}

func (a *App) getActiveClient() *acp.Client {
	a.sessionMu.RLock()
	defer a.sessionMu.RUnlock()
	if state := a.sessions[a.activeSessionID]; state != nil {
		return state.Client
	}
	return nil
}

func (a *App) getActiveState() *SessionState {
	a.sessionMu.RLock()
	defer a.sessionMu.RUnlock()
	return a.sessions[a.activeSessionID]
}

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
		state := a.getActiveState()
		if state == nil || state.Client == nil {
			runtime.EventsEmit(a.ctx, "error", "No active session")
			return
		}
		eventPrefix := fmt.Sprintf("session:%s:", state.ID)
		if err := state.Client.SendPrompt(input, []string{"mcp__ccui__ccui_ask_user_question"}); err != nil {
			slog.Error("prompt failed", "error", err)
			runtime.EventsEmit(a.ctx, eventPrefix+"error", err.Error())
		}
	}()
}

func (a *App) handlePermissionResponse(data ...interface{}) {
	if optionID, ok := firstAs[string](data); ok {
		if client := a.getActiveClient(); client != nil {
			client.RespondToPermission(optionID)
		}
	}
}

func (a *App) handleUserAnswer(data ...interface{}) {
	if a.mcpServer == nil {
		return
	}
	if m, ok := firstAs[map[string]interface{}](data); ok {
		a.mcpServer.HandleUserAnswer(UserAnswer{RequestID: mapStr(m, "requestId"), Answer: mapStr(m, "answer")})
	}
}

func (a *App) handleCancel(data ...interface{}) {
	if client := a.getActiveClient(); client != nil {
		client.Cancel()
	}
}

func (a *App) handleSubmitReview(data ...interface{}) {
	if commentsRaw, ok := firstAs[[]interface{}](data); ok {
		a.SubmitReview(parseReviewComments(commentsRaw))
	}
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
		if s.EventChan != nil {
			close(s.EventChan)
		}
	}
	a.sessionMu.Unlock()
}

func (a *App) SetMode(modeID string) error {
	if client := a.getActiveClient(); client != nil {
		return client.SetMode(modeID)
	}
	return fmt.Errorf("no active session")
}

func (a *App) GetModes() []SessionMode {
	if client := a.getActiveClient(); client != nil {
		return client.AvailableModes()
	}
	return nil
}

func (a *App) GetCurrentMode() string {
	if client := a.getActiveClient(); client != nil {
		return client.CurrentMode()
	}
	return ""
}

type ReviewComment struct{ ID, Type, FilePath, Text string; LineNumber, HunkIndex int }

func (a *App) SubmitReview(comments []ReviewComment) {
	state := a.getActiveState()
	if state == nil || state.Client == nil {
		return
	}
	changes := state.Client.FileChangeStore().GetAll()
	if len(changes) == 0 && len(comments) == 0 {
		return
	}
	eventPrefix := fmt.Sprintf("session:%s:", state.ID)
	go func() {
		runtime.EventsEmit(a.ctx, eventPrefix+"review_agent_running", true)
		prompt := buildReviewPrompt(changes, comments)
		cwd, _ := os.Getwd()
		reviewEventChan := make(chan backend.Event, 100)
		reviewClient, err := a.createClient(cwd, []any{}, reviewEventChan, true, true)
		if err != nil {
			runtime.EventsEmit(a.ctx, eventPrefix+"review_agent_chunk", "Error: "+err.Error())
			runtime.EventsEmit(a.ctx, eventPrefix+"review_agent_complete", nil)
			close(reviewEventChan)
			return
		}
		reviewClient.SetFileChangeStore(state.Client.FileChangeStore())
		go a.bridgeEvents(eventPrefix, reviewEventChan, "review_agent_chunk")
		if err := reviewClient.SendPrompt(prompt, []string{}); err != nil {
			runtime.EventsEmit(a.ctx, eventPrefix+"review_agent_chunk", "\nError: "+err.Error())
		}
		runtime.EventsEmit(a.ctx, eventPrefix+"review_agent_complete", nil)
		go func() { reviewClient.Close(); close(reviewEventChan) }()
	}()
}

func buildReviewPrompt(changes []backend.FileChange, comments []ReviewComment) string {
	var b strings.Builder
	b.WriteString("Review feedback for recent changes:\n\n")
	for _, c := range changes {
		fmt.Fprintf(&b, "## File: %s\n```diff\n", c.FilePath)
		for _, h := range c.Hunks {
			fmt.Fprintf(&b, "@@ -%d,%d +%d,%d @@\n", h.OldStart, h.OldLines, h.NewStart, h.NewLines)
			for _, line := range h.Lines {
				b.WriteString(line + "\n")
			}
		}
		b.WriteString("```\n\n")
	}
	b.WriteString("## Review Comments:\n")
	for _, c := range comments {
		switch c.Type {
		case "line":
			fmt.Fprintf(&b, "- [%s:%d] %s\n", c.FilePath, c.LineNumber, c.Text)
		case "hunk":
			fmt.Fprintf(&b, "- [%s hunk %d] %s\n", c.FilePath, c.HunkIndex+1, c.Text)
		default:
			fmt.Fprintf(&b, "- [General] %s\n", c.Text)
		}
	}
	b.WriteString("\nPlease address this feedback by making the necessary changes.")
	return b.String()
}

func parseReviewComments(raw []interface{}) (comments []ReviewComment) {
	for _, c := range raw {
		if m, ok := c.(map[string]interface{}); ok {
			comments = append(comments, ReviewComment{
				ID: mapStr(m, "id"), Type: mapStr(m, "type"), Text: mapStr(m, "text"),
				FilePath: mapStr(m, "filePath"), LineNumber: mapInt(m, "lineNumber"), HunkIndex: mapInt(m, "hunkIndex"),
			})
		}
	}
	return
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

func firstAs[T any](data []interface{}) (T, bool) {
	var zero T
	if len(data) == 0 {
		return zero, false
	}
	v, ok := data[0].(T)
	return v, ok
}
