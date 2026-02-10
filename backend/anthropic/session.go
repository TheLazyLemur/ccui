package anthropic

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"

	"ccui/backend"
	"ccui/backend/tools"
	"ccui/permission"

	"github.com/google/uuid"
)

const maxContextTokens = 200_000

// AnthropicSession implements backend.Session for direct API calls
type AnthropicSession struct {
	id          string
	ctx         context.Context
	cancel      context.CancelFunc
	backend     *AnthropicBackend
	opts        backend.SessionOpts
	history     []Message
	toolManager *backend.ToolCallManager
	fileStore   *backend.FileChangeStore
	inputTokens int
	mu          sync.Mutex

	// AskUser callback, nil when not available (automation/review sessions)
	askUser func(ctx context.Context, input map[string]any) (string, error)

	// Review-mode configuration
	autoPermission     bool
	suppressToolEvents bool
}

func newAnthropicSession(ctx context.Context, b *AnthropicBackend, opts backend.SessionOpts) *AnthropicSession {
	ctx, cancel := context.WithCancel(ctx)

	fileStore := opts.FileChangeStore
	if fileStore == nil {
		fileStore = backend.NewFileChangeStore()
	}

	return &AnthropicSession{
		id:                 uuid.New().String(),
		ctx:                ctx,
		cancel:             cancel,
		backend:            b,
		opts:               opts,
		history:            make([]Message, 0),
		toolManager:        backend.NewToolCallManager(),
		fileStore:          fileStore,
		askUser:            opts.AskUser,
		autoPermission:     opts.AutoPermission,
		suppressToolEvents: opts.SuppressToolEvents,
	}
}

// SessionID returns the unique session identifier
func (s *AnthropicSession) SessionID() string {
	return s.id
}

// CurrentMode returns empty string (direct API has no modes)
func (s *AnthropicSession) CurrentMode() string {
	return ""
}

// AvailableModes returns nil (direct API has no modes)
func (s *AnthropicSession) AvailableModes() []backend.SessionMode {
	return nil
}

// SetMode is a no-op for direct API
func (s *AnthropicSession) SetMode(modeID string) error {
	return nil
}

// FileChangeStore returns the file change store
func (s *AnthropicSession) FileChangeStore() *backend.FileChangeStore {
	return s.fileStore
}

// Cancel cancels the current operation
func (s *AnthropicSession) Cancel() {
	s.cancel()
}

// Close closes the session
func (s *AnthropicSession) Close() error {
	s.cancel()
	return nil
}

// SendPrompt sends a prompt to the Anthropic API
func (s *AnthropicSession) SendPrompt(text string, allowedTools []string) error {
	// Check token limit before adding message
	s.mu.Lock()
	tokens := s.inputTokens
	s.mu.Unlock()
	if tokens >= maxContextTokens {
		s.emit(backend.Event{
			Type: backend.EventContextFull,
			Data: map[string]any{"inputTokens": s.inputTokens, "limit": maxContextTokens},
		})
		return fmt.Errorf("context full: %d input tokens exceeds limit of %d", s.inputTokens, maxContextTokens)
	}

	s.mu.Lock()
	// Add user message to history
	s.history = append(s.history, Message{
		Role:    "user",
		Content: []ContentBlock{{Type: BlockTypeText, Text: text}},
	})
	s.mu.Unlock()

	// Get all tools (local + MCP providers + conditional AskUserQuestion)
	allTools := s.backend.getAllTools()
	if s.askUser != nil {
		allTools = append(allTools, AskUserQuestionTool())
	}

	// Tool loop
	for {
		select {
		case <-s.ctx.Done():
			return s.ctx.Err()
		default:
		}

		stopReason, err := s.doRequest(allTools)
		if err != nil {
			return err
		}

		if stopReason != StopReasonToolUse {
			// Done - emit prompt complete
			s.emit(backend.Event{
				Type: backend.EventPromptComplete,
				Data: map[string]any{"stopReason": stopReason},
			})
			return nil
		}

		// Check token limit before continuing tool loop
		s.mu.Lock()
		tokens := s.inputTokens
		s.mu.Unlock()
		if tokens >= maxContextTokens {
			s.emit(backend.Event{
				Type: backend.EventContextFull,
				Data: map[string]any{"inputTokens": s.inputTokens, "limit": maxContextTokens},
			})
			return fmt.Errorf("context full: %d input tokens exceeds limit of %d", s.inputTokens, maxContextTokens)
		}
	}
}

// doRequest makes a single API request and processes the response
func (s *AnthropicSession) doRequest(tools []Tool) (string, error) {
	s.mu.Lock()
	req := MessagesRequest{
		Model:     s.backend.model,
		Messages:  s.history,
		MaxTokens: s.backend.maxTokens,
		Tools:     tools,
		Stream:    true,
	}
	s.mu.Unlock()

	body, err := json.Marshal(req)
	if err != nil {
		return "", fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(s.ctx, "POST", s.backend.baseURL+"/v1/messages", bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", s.backend.apiKey)
	httpReq.Header.Set("anthropic-version", "2023-06-01")

	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("http request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("API error %d: %s", resp.StatusCode, string(bodyBytes))
	}

	return s.processStream(resp.Body)
}

// contentBlockState tracks in-progress content blocks during streaming
type contentBlockState struct {
	index       int
	blockType   string
	toolID      string
	toolName    string
	textBuilder strings.Builder
	jsonBuilder strings.Builder
}

// processStream processes SSE events and returns the stop reason
func (s *AnthropicSession) processStream(body io.ReadCloser) (string, error) {
	reader := NewStreamReader(body)
	defer reader.Close()

	var stopReason string
	blocks := make(map[int]*contentBlockState)
	var assistantContent []ContentBlock

	for {
		ev, err := reader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", fmt.Errorf("stream error: %w", err)
		}

		switch ev.Type {
		case EventMessageStart:
			if ev.MessageStart != nil {
				s.mu.Lock()
				s.inputTokens = ev.MessageStart.Message.Usage.InputTokens
				s.mu.Unlock()
			}

		case EventContentBlockStart:
			if ev.ContentBlockStart == nil {
				continue
			}
			idx := ev.ContentBlockStart.Index
			cb := ev.ContentBlockStart.ContentBlock
			blocks[idx] = &contentBlockState{
				index:     idx,
				blockType: cb.Type,
				toolID:    cb.ID,
				toolName:  cb.Name,
			}

			// If tool_use, create pending tool state
			if cb.Type == BlockTypeToolUse {
				state := &backend.ToolState{
					ID:       cb.ID,
					Status:   "pending",
					Title:    cb.Name,
					Kind:     "tool",
					ToolName: cb.Name,
					ParentID: s.toolManager.CurrentParent(),
				}
				s.toolManager.Set(state)
				s.emitToolState(state)
			}

		case EventContentBlockDelta:
			if ev.ContentBlockDelta == nil {
				continue
			}
			idx := ev.ContentBlockDelta.Index
			delta := ev.ContentBlockDelta.Delta
			block, ok := blocks[idx]
			if !ok {
				continue
			}

			switch delta.Type {
			case DeltaTypeText:
				block.textBuilder.WriteString(delta.Text)
				s.emit(backend.Event{
					Type: backend.EventMessageChunk,
					Data: delta.Text,
				})
			case DeltaTypeInputJSON:
				block.jsonBuilder.WriteString(delta.PartialJSON)
			case DeltaTypeThinking:
				s.emit(backend.Event{
					Type: backend.EventThoughtChunk,
					Data: delta.Thinking,
				})
			}

		case EventContentBlockStop:
			if ev.ContentBlockStop == nil {
				continue
			}
			idx := ev.ContentBlockStop.Index
			block, ok := blocks[idx]
			if !ok {
				continue
			}

			// Finalize block
			switch block.blockType {
			case BlockTypeText:
				assistantContent = append(assistantContent, ContentBlock{
					Type: BlockTypeText,
					Text: block.textBuilder.String(),
				})
			case BlockTypeToolUse:
				// Parse accumulated JSON input
				var input map[string]any
				if block.jsonBuilder.Len() > 0 {
					json.Unmarshal([]byte(block.jsonBuilder.String()), &input)
				}
				assistantContent = append(assistantContent, ContentBlock{
					Type:  BlockTypeToolUse,
					ID:    block.toolID,
					Name:  block.toolName,
					Input: input,
				})

				// Update tool state with input
				s.toolManager.Update(block.toolID, func(ts *backend.ToolState) {
					ts.Input = input
				})
			}
			delete(blocks, idx)

		case EventMessageDelta:
			if ev.MessageDelta != nil {
				stopReason = ev.MessageDelta.Delta.StopReason
			}

		case EventError:
			if ev.Error != nil {
				return "", fmt.Errorf("API error: %s", ev.Error.Error.Message)
			}
		}
	}

	// Add assistant message to history
	if len(assistantContent) > 0 {
		s.mu.Lock()
		s.history = append(s.history, Message{
			Role:    "assistant",
			Content: assistantContent,
		})
		s.mu.Unlock()
	}

	// Execute tools if stop_reason is tool_use
	if stopReason == StopReasonToolUse {
		if err := s.executeTools(assistantContent); err != nil {
			return "", err
		}
	}

	return stopReason, nil
}

// executeTools processes tool_use blocks and adds results to history
func (s *AnthropicSession) executeTools(content []ContentBlock) error {
	var toolResults []ContentBlock

	for _, block := range content {
		if block.Type != BlockTypeToolUse {
			continue
		}

		result, err := s.executeTool(block.ID, block.Name, block.Input)
		if err != nil {
			return err
		}
		toolResults = append(toolResults, result)
	}

	if len(toolResults) > 0 {
		s.mu.Lock()
		s.history = append(s.history, Message{
			Role:    "user",
			Content: toolResults,
		})
		s.mu.Unlock()
	}

	return nil
}

// executeTool executes a single tool with permission checking
func (s *AnthropicSession) executeTool(id, name string, input map[string]any) (ContentBlock, error) {
	inputJSON, _ := json.Marshal(input)

	// Skip permission check if auto-permission enabled
	if !s.autoPermission {
		// Check permission
		decision := s.backend.permLayer.Check(name, string(inputJSON))

		switch decision {
		case permission.Deny:
			return s.toolError(id, "Permission denied")

			case permission.Ask:
			// Update state to awaiting_permission
			state := s.toolManager.Update(id, func(ts *backend.ToolState) {
				ts.Status = "awaiting_permission"
				ts.PermissionOptions = []backend.PermOption{
					{OptionID: "allow", Name: "Allow", Kind: "allow"},
					{OptionID: "deny", Name: "Deny", Kind: "deny"},
				}
			})
			if state != nil {
				s.emitToolState(state)
			}

			// Request permission (blocks until user responds)
			optionID, err := s.backend.permLayer.Request(id, name, []backend.PermOption{
				{OptionID: "allow", Name: "Allow", Kind: "allow"},
				{OptionID: "deny", Name: "Deny", Kind: "deny"},
			})
			if err != nil {
				return s.toolError(id, fmt.Sprintf("Permission request failed: %v", err))
			}

			if optionID != "allow" {
				s.toolManager.Update(id, func(ts *backend.ToolState) {
					ts.Status = "error"
				})
				return s.toolError(id, "User denied permission")
			}
		}
	}

	// Update state to running
	s.toolManager.Update(id, func(ts *backend.ToolState) {
		ts.Status = "running"
	})
	s.emitToolState(s.toolManager.Get(id))

	// Execute the tool (local or MCP)
	result, err := s.executeWithProvider(name, input)
	if err != nil {
		s.toolManager.Update(id, func(ts *backend.ToolState) {
			ts.Status = "error"
		})
		return s.toolError(id, fmt.Sprintf("Execution failed: %v", err))
	}

	// Track file changes (only for Write/Edit tools)
	if result.FilePath != "" && (name == "Write" || name == "Edit") {
		s.fileStore.RecordChange(result.FilePath, result.OldContent, result.NewContent, result.Hunks)
		s.emit(backend.Event{
			Type: backend.EventFileChanges,
			Data: s.fileStore.GetAll(),
		})
	}

	// Update state to completed
	state := s.toolManager.Update(id, func(ts *backend.ToolState) {
		ts.Status = "completed"
		if result.Content != "" {
			ts.Output = []backend.OutputBlock{{
				Type:    "text",
				Content: &backend.TextContent{Type: "text", Text: result.Content},
			}}
		}
	})
	if state != nil {
		s.emitToolState(state)
	}

	// Build tool_result block
	return ContentBlock{
		Type:      BlockTypeToolResult,
		ToolUseID: id,
		Content:   result.Content,
		IsError:   result.IsError,
	}, nil
}

// toolError creates a tool_result error block
func (s *AnthropicSession) toolError(id, msg string) (ContentBlock, error) {
	return ContentBlock{
		Type:      BlockTypeToolResult,
		ToolUseID: id,
		Content:   msg,
		IsError:   true,
	}, nil
}

// emit sends an event to the event channel
func (s *AnthropicSession) emit(ev backend.Event) {
	if s.opts.EventChan != nil {
		select {
		case s.opts.EventChan <- ev:
		case <-s.ctx.Done():
		}
	}
}

// emitToolState emits a copy of the tool state to avoid mutation issues
func (s *AnthropicSession) emitToolState(state *backend.ToolState) {
	if state == nil || s.suppressToolEvents {
		return
	}
	// Copy the state to avoid race conditions with later mutations
	copy := &backend.ToolState{
		ID:                state.ID,
		Status:            state.Status,
		Title:             state.Title,
		Kind:              state.Kind,
		ToolName:          state.ToolName,
		ParentID:          state.ParentID,
		Input:             state.Input,
		Output:            state.Output,
		Diff:              state.Diff,
		Diffs:             state.Diffs,
		PermissionOptions: state.PermissionOptions,
	}
	s.emit(backend.Event{Type: backend.EventToolState, Data: copy})
}

// executeWithProvider executes a tool using the appropriate provider (local or MCP)
func (s *AnthropicSession) executeWithProvider(name string, input map[string]any) (tools.ToolResult, error) {
	// Native AskUserQuestion — direct function call, no MCP roundtrip
	if name == "AskUserQuestion" && s.askUser != nil {
		answer, err := s.askUser(s.ctx, input)
		if err != nil {
			return tools.ToolResult{Content: err.Error(), IsError: true}, nil
		}
		return tools.ToolResult{Content: answer}, nil
	}

	// Check if this is an MCP tool
	if provider := s.backend.findProvider(name); provider != nil {
		return provider.Execute(s.ctx, name, input)
	}

	// Fall back to local executor
	return s.backend.executor.Execute(s.ctx, name, input)
}
