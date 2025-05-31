package main

import (
	"context"
	"fmt"

	"mcp-atlassian-server/pkg/tools/confluence"
	"mcp-atlassian-server/pkg/tools/jira"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func main() {
	var hooks server.Hooks
	hooks.AddBeforeAny(func(ctx context.Context, id any, method mcp.MCPMethod, message any) {

	})

	s := server.NewMCPServer(
		"Atlassian MCP - Provides tools for interacting with Atlassian Jira & Confluence",
		"0.1.0",
		server.WithToolCapabilities(true),
		server.WithInstructions("Provides tools for interacting with Atlassian Jira."),
	)

	confluence.AddTools(s)
	jira.AddTools(s)

	if err := server.ServeStdio(s); err != nil {
		fmt.Printf("Server error: %v\n", err)
	}
}
