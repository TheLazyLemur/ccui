package mcpclient

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"ccui/backend/anthropic"
	"ccui/backend/tools"

	"github.com/mark3labs/mcp-go/mcp"
)

// Provider implements tools.ToolExecutor for MCP servers
type Provider struct {
	client *MCPClient
	name   string
}

// NewProvider creates a new MCP tool provider
func NewProvider(client *MCPClient, name string) *Provider {
	if name != "" {
		client.SetName(name)
	}
	return &Provider{
		client: client,
		name:   client.Name(),
	}
}

// Name returns the provider name
func (p *Provider) Name() string {
	return p.name
}

// GetTools returns tools in Anthropic-compatible format
func (p *Provider) GetTools() []anthropic.Tool {
	mcpTools := p.client.GetTools()
	anthropicTools := make([]anthropic.Tool, 0, len(mcpTools))

	for _, tool := range mcpTools {
		anthropicTools = append(anthropicTools, convertMCPToolToAnthropic(tool, p.name))
	}

	return anthropicTools
}

// Execute runs an MCP tool by name
// The name should be in the format "mcp__{server}__{tool_name}"
func (p *Provider) Execute(ctx context.Context, name string, input map[string]any) (tools.ToolResult, error) {
	// Extract the actual tool name from the prefixed name
	toolName := extractToolName(name, p.name)

	slog.Debug("executing mcp tool", "provider", p.name, "tool", toolName, "fullName", name)

	result, err := p.client.ExecuteTool(ctx, toolName, input)
	if err != nil {
		return tools.ToolResult{
			Content: fmt.Sprintf("MCP tool execution failed: %v", err),
			IsError: true,
		}, nil // Return as tool result error, not execution error
	}

	// Convert MCP result content to string
	content := extractContentText(result)

	return tools.ToolResult{
		Content: content,
		IsError: result.IsError,
	}, nil
}

// extractToolName extracts the actual tool name from the prefixed name
// Format: "mcp__{server}__{tool_name}" -> "tool_name"
func extractToolName(prefixedName, serverName string) string {
	prefix := fmt.Sprintf("mcp__%s__", serverName)
	if len(prefixedName) > len(prefix) && prefixedName[:len(prefix)] == prefix {
		return prefixedName[len(prefix):]
	}
	// If not prefixed, return as-is (might be a direct call)
	return prefixedName
}

// extractContentText extracts text content from MCP result
func extractContentText(result *mcp.CallToolResult) string {
	var text string
	for _, content := range result.Content {
		switch c := content.(type) {
		case mcp.TextContent:
			text += c.Text
		case *mcp.TextContent:
			if c != nil {
				text += c.Text
			}
		default:
			// Try to marshal other content types as JSON
			if jsonBytes, err := json.Marshal(content); err == nil {
				text += string(jsonBytes)
			}
		}
	}
	return text
}

// convertMCPToolToAnthropic converts an MCP tool definition to Anthropic format
func convertMCPToolToAnthropic(mcpTool mcp.Tool, serverName string) anthropic.Tool {
	return anthropic.Tool{
		Name:        fmt.Sprintf("mcp__%s__%s", serverName, mcpTool.Name),
		Description: mcpTool.Description,
		InputSchema: convertInputSchema(mcpTool.InputSchema),
	}
}

// convertInputSchema converts MCP ToolInputSchema to Anthropic InputSchema
func convertInputSchema(mcpSchema mcp.ToolInputSchema) anthropic.InputSchema {
	schema := anthropic.InputSchema{
		Type:       mcpSchema.Type,
		Properties: make(map[string]anthropic.Property),
		Required:   mcpSchema.Required,
	}

	// Convert properties - MCP uses map[string]any for properties
	for name, propAny := range mcpSchema.Properties {
		if propMap, ok := propAny.(map[string]any); ok {
			schema.Properties[name] = convertPropertyMap(propMap)
		}
	}

	return schema
}

// convertPropertyMap converts a property map to Anthropic Property
func convertPropertyMap(propMap map[string]any) anthropic.Property {
	prop := anthropic.Property{}

	if t, ok := propMap["type"].(string); ok {
		prop.Type = t
	}
	if desc, ok := propMap["description"].(string); ok {
		prop.Description = desc
	}
	if enum, ok := propMap["enum"].([]any); ok {
		for _, e := range enum {
			if s, ok := e.(string); ok {
				prop.Enum = append(prop.Enum, s)
			}
		}
	}

	// Handle array items
	if items, ok := propMap["items"].(map[string]any); ok {
		converted := convertPropertyMap(items)
		prop.Items = &converted
	}

	// Handle nested object properties
	if nestedProps, ok := propMap["properties"].(map[string]any); ok {
		prop.Properties = make(map[string]anthropic.Property)
		for name, nestedAny := range nestedProps {
			if nestedMap, ok := nestedAny.(map[string]any); ok {
				prop.Properties[name] = convertPropertyMap(nestedMap)
			}
		}
	}

	// Handle nested required fields
	if required, ok := propMap["required"].([]any); ok {
		for _, r := range required {
			if s, ok := r.(string); ok {
				prop.Required = append(prop.Required, s)
			}
		}
	}

	return prop
}

// GetClient returns the underlying MCP client
func (p *Provider) GetClient() *MCPClient {
	return p.client
}

// ToolProvider interface check
var _ interface {
	Name() string
	GetTools() []anthropic.Tool
	Execute(ctx context.Context, name string, input map[string]any) (tools.ToolResult, error)
} = (*Provider)(nil)
