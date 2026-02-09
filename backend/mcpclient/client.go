package mcpclient

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
)

// MCPClient wraps mcp-go/client with higher-level operations for CCUI
type MCPClient struct {
	client     *client.Client
	name       string
	baseURL    string
	tools      []mcp.Tool
	initialized bool
}

// NewSSEMCPClient creates a client for SSE-based MCP server
func NewSSEMCPClient(baseURL string) (*MCPClient, error) {
	c, err := client.NewSSEMCPClient(baseURL)
	if err != nil {
		return nil, fmt.Errorf("create sse mcp client: %w", err)
	}
	return &MCPClient{
		client:  c,
		baseURL: baseURL,
		name:    extractServerName(baseURL),
	}, nil
}

// extractServerName extracts a server name from the URL
func extractServerName(baseURL string) string {
	// For now, use a simple heuristic - can be made smarter later
	// URLs like http://127.0.0.1:12345/sse -> "local"
	// Could parse hostname or use a provided name
	return "ccui"
}

// SetName allows overriding the default server name
func (m *MCPClient) SetName(name string) {
	m.name = name
}

// Name returns the server name
func (m *MCPClient) Name() string {
	return m.name
}

// Initialize connects to the MCP server and discovers available tools
func (m *MCPClient) Initialize(ctx context.Context) error {
	if m.initialized {
		return nil
	}

	// Start the client connection
	if err := m.client.Start(ctx); err != nil {
		return fmt.Errorf("start mcp client: %w", err)
	}

	// Call initialize to negotiate protocol version and capabilities
	result, err := m.client.Initialize(ctx, mcp.InitializeRequest{
		Request: mcp.Request{Method: "initialize"},
		Params: struct {
			ProtocolVersion string             `json:"protocolVersion"`
			Capabilities    mcp.ClientCapabilities `json:"capabilities"`
			ClientInfo      mcp.Implementation `json:"clientInfo"`
		}{
			ProtocolVersion: mcp.LATEST_PROTOCOL_VERSION,
			Capabilities:    mcp.ClientCapabilities{},
			ClientInfo: mcp.Implementation{
				Name:    "ccui",
				Version: "1.0.0",
			},
		},
	})
	if err != nil {
		return fmt.Errorf("initialize mcp connection: %w", err)
	}

	slog.Info("mcp client initialized",
		"server", result.ServerInfo.Name,
		"version", result.ServerInfo.Version,
		"protocol", result.ProtocolVersion,
	)

	// Discover available tools
	if err := m.discoverTools(ctx); err != nil {
		return fmt.Errorf("discover tools: %w", err)
	}

	m.initialized = true
	return nil
}

// discoverTools fetches the list of available tools from the server
func (m *MCPClient) discoverTools(ctx context.Context) error {
	result, err := m.client.ListTools(ctx, mcp.ListToolsRequest{})
	if err != nil {
		return fmt.Errorf("list tools: %w", err)
	}

	m.tools = result.Tools
	slog.Info("discovered mcp tools", "count", len(m.tools), "server", m.name)
	for _, tool := range m.tools {
		slog.Debug("mcp tool", "name", tool.Name, "server", m.name)
	}

	return nil
}

// GetTools returns the discovered tools
func (m *MCPClient) GetTools() []mcp.Tool {
	return m.tools
}

// ExecuteTool calls an MCP tool with the given name and arguments
func (m *MCPClient) ExecuteTool(ctx context.Context, name string, arguments map[string]any) (*mcp.CallToolResult, error) {
	if !m.initialized {
		return nil, fmt.Errorf("mcp client not initialized")
	}

	req := mcp.CallToolRequest{
		Request: mcp.Request{Method: "tools/call"},
		Params: struct {
			Name      string         `json:"name"`
			Arguments map[string]any `json:"arguments,omitempty"`
			Meta      *struct {
				ProgressToken mcp.ProgressToken `json:"progressToken,omitempty"`
			} `json:"_meta,omitempty"`
		}{
			Name:      name,
			Arguments: arguments,
		},
	}

	result, err := m.client.CallTool(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("call tool %s: %w", name, err)
	}

	return result, nil
}

// Close closes the MCP client connection
func (m *MCPClient) Close() error {
	if m.client != nil {
		return m.client.Close()
	}
	return nil
}

// IsInitialized returns whether the client has been initialized
func (m *MCPClient) IsInitialized() bool {
	return m.initialized
}
