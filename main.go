package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	log "github.com/sirupsen/logrus"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"mcp-atlassian-server/pkg/tools/confluence"
	"mcp-atlassian-server/pkg/tools/jira"
)

func init() {
	if os.Getenv("DEBUG") != "" {
		log.SetLevel(log.DebugLevel)
	}
}

func main() {
	hooks := &server.Hooks{}
	hooks.AddBeforeAny(func(ctx context.Context, id any, method mcp.MCPMethod, message any) {

	})

	s := server.NewMCPServer(
		"Atlassian MCP - Provides tools for interacting with Atlassian Jira & Confluence",
		"0.1.0",
		server.WithToolCapabilities(true),
		server.WithInstructions("Provides tools for interacting with Atlassian Jira."),
		server.WithHooks(hooks),
	)

	switch strings.ToUpper(os.Getenv("MCP_MODE")) {
	case "JIRA":
		jira.AddTools(s)
	case "CONFLUENCE":
		confluence.AddTools(s)
	case "":
		confluence.AddTools(s)
		jira.AddTools(s)
	default:
		log.Fatal("Unknown MCP_MODE value")
	}

	if err := server.ServeStdio(s); err != nil {
		fmt.Printf("Server error: %v\n", err)
	}
}
