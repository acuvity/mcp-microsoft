package mcp

import (
	"fmt"

	"github.com/mark3labs/mcp-go/server"
	"github.com/acuvity/mcp-microsoft/baggage"
	"github.com/acuvity/mcp-microsoft/client"
	"github.com/acuvity/mcp-microsoft/collection"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func Run(cmd *cobra.Command, args []string) error {

	cl, err := client.GetClient(
		viper.GetString("tenant-id"),     // Tenant ID
		viper.GetString("client-id"),     // Client ID
		viper.GetString("client-secret"), // Client Secret
	)
	if err != nil {
		return fmt.Errorf("error creating client: %v", err)
	}

	// Create a new MCP server
	s := server.NewMCPServer(
		"Microsoft MCP Server",
		"1.0.0",
	)

	for _, tool := range collection.Tools {
		s.AddTool(tool.Tool, tool.Processor)
	}

	// Start the server
	switch viper.GetString("transport") {
	case "stdio":
		if err := server.ServeStdio(s, server.WithStdioContextFunc(baggage.WithInfomation(cl))); err != nil {
			return fmt.Errorf("server error: %v", err)
		}
	case "sse":
		server := server.NewSSEServer(s, server.WithBaseURL("http://localhost:8000"), server.WithSSEContextFunc(baggage.WithInfomationFromRequest(cl)))
		if server == nil {
			return fmt.Errorf("server error: %v", err)
		}
		if err := server.Start(":8000"); err != nil {
			return fmt.Errorf("server error: %v", err)
		}
	default:
		return fmt.Errorf("invalid transport type: '%s'. Must be 'stdio' or 'sse'", viper.GetString("transport"))
	}
	return nil
}
