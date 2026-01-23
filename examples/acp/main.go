package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"sync"
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

// Initialize types
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

// Session types
type SessionNewParams struct {
	Cwd            string `json:"cwd"`
	PermissionMode string `json:"permissionMode,omitempty"`
}

type SessionNewResult struct {
	SessionID string `json:"sessionId"`
}

type SessionPromptParams struct {
	SessionID string          `json:"sessionId"`
	Prompt    []PromptContent `json:"prompt"`
}

type PromptContent struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
}

type SessionPromptResult struct {
	SessionID  string `json:"sessionId"`
	StopReason string `json:"stopReason"`
}

// Streaming types
type SessionUpdate struct {
	SessionID string        `json:"sessionId"`
	Update    UpdateContent `json:"update"`
}

type UpdateContent struct {
	SessionUpdate string       `json:"sessionUpdate,omitempty"`
	Content       *TextContent `json:"content,omitempty"`
}

type TextContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// Permission types
type PermissionRequest struct {
	SessionID string        `json:"sessionId"`
	ToolCall  ToolCallInfo  `json:"toolCall"`
	Options   []PermOption  `json:"options"`
}

type ToolCallInfo struct {
	ToolCallID string `json:"toolCallId"`
	Title      string `json:"title"`
	Kind       string `json:"kind"` // read|edit|delete|move|search|execute|think|fetch|other
}

type PermOption struct {
	OptionID string `json:"optionId"`
	Name     string `json:"name"`
	Kind     string `json:"kind"` // allow_once|allow_always|reject_once|reject_always
}

type PermissionResponse struct {
	Outcome PermissionOutcome `json:"outcome"`
}

type PermissionOutcome struct {
	Outcome  string `json:"outcome"`            // selected|cancelled
	OptionID string `json:"optionId,omitempty"` // required if selected
}

// ACPClient manages subprocess + JSON-RPC
type ACPClient struct {
	cmd       *exec.Cmd
	stdin     io.WriteCloser
	stdout    *bufio.Scanner
	sessionID string
	msgID     int
	mu        sync.Mutex
	callbacks map[int]chan JSONRPCMessage
}

func NewACPClient(ctx context.Context, cwd string) (*ACPClient, error) {
	cmd := exec.CommandContext(ctx, "claude-code-acp")
	cmd.Env = append(os.Environ(), "ANTHROPIC_API_KEY="+os.Getenv("ANTHROPIC_API_KEY"))

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

	c := &ACPClient{
		cmd:       cmd,
		stdin:     stdin,
		stdout:    bufio.NewScanner(stdout),
		callbacks: make(map[int]chan JSONRPCMessage),
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

func (c *ACPClient) readLoop() {
	for c.stdout.Scan() {
		line := c.stdout.Bytes()
		// Skip non-JSON lines (cc-acp debug output)
		if len(line) == 0 || line[0] != '{' {
			continue
		}

		var msg JSONRPCMessage
		if err := json.Unmarshal(line, &msg); err != nil {
			fmt.Fprintf(os.Stderr, "parse error: %v: %s\n", err, line)
			continue
		}

		if msg.Method != "" {
			// Request or notification from agent
			fmt.Fprintf(os.Stderr, "[method] %s\n", msg.Method)

			if msg.Method == "session/update" {
				var update SessionUpdate
				json.Unmarshal(msg.Params, &update)
				if update.Update.SessionUpdate == "agent_message_chunk" && update.Update.Content != nil {
					fmt.Print(update.Update.Content.Text)
				}
			} else if msg.Method == "session/request_permission" {
				c.respondPermission(msg)
			}
		} else if msg.ID != nil {
			// Response to our request
			c.mu.Lock()
			if ch, ok := c.callbacks[*msg.ID]; ok {
				ch <- msg
				delete(c.callbacks, *msg.ID)
			}
			c.mu.Unlock()
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
	if _, err := c.stdin.Write(append(data, '\n')); err != nil {
		return JSONRPCMessage{}, err
	}

	resp := <-ch
	if resp.Error != nil {
		return resp, fmt.Errorf("rpc error %d: %s", resp.Error.Code, resp.Error.Message)
	}
	return resp, nil
}

func (c *ACPClient) initialize() error {
	_, err := c.send("initialize", InitializeParams{
		ProtocolVersion: 1,
		ClientCapabilities: ClientCapabilities{
			// FS capabilities disabled - agent handles file ops itself
			Terminal: false,
		},
	})
	return err
}

func (c *ACPClient) newSession(cwd string) error {
	resp, err := c.send("session/new", map[string]any{
		"cwd":        cwd,
		"mcpServers": []any{}, // Must be array, not object
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

func (c *ACPClient) respondPermission(msg JSONRPCMessage) {
	var req PermissionRequest
	json.Unmarshal(msg.Params, &req)

	// Show permission prompt
	fmt.Fprintf(os.Stderr, "\n┌─ Permission Request ─────────────────\n")
	fmt.Fprintf(os.Stderr, "│ %s\n", req.ToolCall.Title)
	fmt.Fprintf(os.Stderr, "├──────────────────────────────────────\n")
	for i, opt := range req.Options {
		fmt.Fprintf(os.Stderr, "│ %d) %s\n", i+1, opt.Name)
	}
	fmt.Fprintf(os.Stderr, "└──────────────────────────────────────\n")
	fmt.Fprintf(os.Stderr, "Select [1-%d]: ", len(req.Options))

	// Read user choice
	var choice int
	fmt.Scanf("%d", &choice)

	var optionID string
	if choice >= 1 && choice <= len(req.Options) {
		optionID = req.Options[choice-1].OptionID
	} else {
		// Default to first allow_once option
		for _, opt := range req.Options {
			if opt.Kind == "allow_once" {
				optionID = opt.OptionID
				break
			}
		}
	}

	resp := JSONRPCMessage{
		JSONRPC: "2.0",
		ID:      msg.ID,
	}
	result, _ := json.Marshal(PermissionResponse{
		Outcome: PermissionOutcome{Outcome: "selected", OptionID: optionID},
	})
	resp.Result = result
	data, _ := json.Marshal(resp)
	c.stdin.Write(append(data, '\n'))
}

func (c *ACPClient) Close() error {
	c.stdin.Close()
	return c.cmd.Wait()
}

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	cwd, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "getwd: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Starting ACP client...")
	client, err := NewACPClient(ctx, cwd)
	if err != nil {
		fmt.Fprintf(os.Stderr, "acp client: %v\n", err)
		os.Exit(1)
	}
	defer client.Close()

	fmt.Println("Ready. Type your prompt (Ctrl+C to exit):")
	scanner := bufio.NewScanner(os.Stdin)
	for {
		fmt.Print("\n> ")
		if !scanner.Scan() {
			break
		}
		input := scanner.Text()
		if input == "" {
			continue
		}

		result, err := client.SendPrompt(input)
		if err != nil {
			fmt.Fprintf(os.Stderr, "\nerror: %v\n", err)
			continue
		}
		fmt.Printf("\n[%s]\n", result.StopReason)
	}
}
