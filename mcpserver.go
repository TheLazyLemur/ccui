package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// UserQuestionServer wraps an MCP server with AskUserQuestion tool
type UserQuestionServer struct {
	mcpServer  *server.MCPServer
	httpServer *http.Server
	listener   net.Listener
	ctx        context.Context
	responseCh chan UserAnswer
}

// UserQuestion is emitted to frontend
type UserQuestion struct {
	RequestID string   `json:"requestId"`
	Question  string   `json:"question"`
	Options   []Option `json:"options,omitempty"`
}

type Option struct {
	Label       string `json:"label"`
	Description string `json:"description,omitempty"`
}

// UserAnswer received from frontend
type UserAnswer struct {
	RequestID string `json:"requestId"`
	Answer    string `json:"answer"`
}

// NewUserQuestionServer creates a new MCP server for user questions
func NewUserQuestionServer(ctx context.Context) *UserQuestionServer {
	s := &UserQuestionServer{
		ctx:        ctx,
		responseCh: make(chan UserAnswer, 1),
	}

	s.mcpServer = server.NewMCPServer(
		"ccui-mcp",
		"1.0.0",
		server.WithToolCapabilities(false),
	)

	// Register AskUserQuestion tool
	askTool := mcp.NewTool("ccui_ask_user_question",
		mcp.WithDescription(`Ask the user a question and wait for their response.

Use this tool when you need clarification or input from the user.

Args:
  - question (string, required): The question to ask the user
  - options (array, optional): List of suggested options with label and description

Returns: The user's text response.`),
		mcp.WithString("question",
			mcp.Required(),
			mcp.Description("The question to ask the user"),
		),
		mcp.WithArray("options",
			mcp.Description("Optional list of suggested options for the user"),
		),
		mcp.WithToolAnnotation(mcp.ToolAnnotation{
			Title:           "Ask User Question",
			ReadOnlyHint:    boolPtr(true),
			DestructiveHint: boolPtr(false),
			IdempotentHint:  boolPtr(false),
			OpenWorldHint:   boolPtr(true),
		}),
	)

	s.mcpServer.AddTool(askTool, s.handleAskUserQuestion)

	return s
}

func boolPtr(b bool) *bool {
	return &b
}

func (s *UserQuestionServer) handleAskUserQuestion(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	question, ok := req.Params.Arguments["question"].(string)
	if !ok || question == "" {
		return mcp.NewToolResultError("question is required"), nil
	}

	// Parse options if provided
	var options []Option
	if opts, ok := req.Params.Arguments["options"].([]interface{}); ok {
		for _, opt := range opts {
			if optMap, ok := opt.(map[string]interface{}); ok {
				o := Option{}
				if l, ok := optMap["label"].(string); ok {
					o.Label = l
				}
				if d, ok := optMap["description"].(string); ok {
					o.Description = d
				}
				if o.Label != "" {
					options = append(options, o)
				}
			}
		}
	}

	// Generate request ID
	requestID := fmt.Sprintf("uq-%d", ctx.Value("request_id"))

	// Emit question to frontend
	uq := UserQuestion{
		RequestID: requestID,
		Question:  question,
		Options:   options,
	}
	runtime.EventsEmit(s.ctx, "user_question", uq)

	// Block waiting for response
	answer := <-s.responseCh

	return mcp.NewToolResultText(answer.Answer), nil
}

// HandleUserAnswer processes response from frontend
func (s *UserQuestionServer) HandleUserAnswer(answer UserAnswer) {
	select {
	case s.responseCh <- answer:
	default:
		// Channel full, discard
	}
}

// Start binds to localhost random port and returns URL
func (s *UserQuestionServer) Start() (string, error) {
	// Bind to random port on localhost only
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return "", fmt.Errorf("listen: %w", err)
	}
	s.listener = listener

	addr := listener.Addr().(*net.TCPAddr)
	baseURL := fmt.Sprintf("http://127.0.0.1:%d", addr.Port)

	// Create SSE server - default endpoints are /sse and /message
	sseServer := server.NewSSEServer(s.mcpServer, server.WithBaseURL(baseURL))

	// Route SSE and message endpoints
	mux := http.NewServeMux()
	mux.Handle("/sse", sseServer)
	mux.Handle("/message", sseServer)

	s.httpServer = &http.Server{Handler: mux}

	go func() {
		if err := s.httpServer.Serve(listener); err != nil && err != http.ErrServerClosed {
			fmt.Printf("MCP server error: %v\n", err)
		}
	}()

	// Return SSE endpoint URL for ACP config
	return baseURL + "/sse", nil
}

// Stop shuts down the HTTP server
func (s *UserQuestionServer) Stop() error {
	if s.httpServer != nil {
		return s.httpServer.Shutdown(context.Background())
	}
	return nil
}

// MCPServerConfig returns config for session/new
func MCPServerConfig(url string) []any {
	return []any{
		map[string]any{
			"name":    "ccui",
			"type":    "sse",
			"url":     url,
			"headers": []any{},
		},
	}
}

// SerializeQuestion for logging
func (q UserQuestion) JSON() string {
	b, _ := json.Marshal(q)
	return string(b)
}
