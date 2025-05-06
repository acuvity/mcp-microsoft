package collection

import (
	"context"

	"github.com/mark3labs/mcp-go/mcp"
)

// Tool is runtime information for the tool
type Tool struct {
	Name      string
	Tool      mcp.Tool
	Processor func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error)
}

// toolsMap organizes tools in a map
type toolsMap map[string]*Tool

// Tools is a map of tool name to tool
var Tools toolsMap

func init() {
	Tools = make(toolsMap)
}

// RegisterTool register a test in the main suite.
func RegisterTool(t Tool) {
	if Tools == nil {
		panic("tools map is not initialized")
	}
	if Tools[t.Name] != nil {
		panic("tool already registered")
	}
	Tools[t.Name] = &t
}
