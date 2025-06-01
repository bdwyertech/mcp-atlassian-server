package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"

	log "github.com/sirupsen/logrus"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"mcp-atlassian-server/pkg/clients"
	"mcp-atlassian-server/pkg/tools/confluence"
	"mcp-atlassian-server/pkg/tools/jira"
)

func init() {
	if os.Getenv("DEBUG") != "" {
		log.SetLevel(log.DebugLevel)
	}
}

const (
	envEnabledTools  = "ENABLED_TOOLS"
	envDisabledTools = "DISABLED_TOOLS"
	envMCPMode       = "MCP_MODE"
	envMCPHTTP       = "MCP_HTTP"
	envMCP_SSE       = "MCP_SSE"
)

func main() {
	hooks := &server.Hooks{}
	s := server.NewMCPServer(
		"Atlassian MCP - Provides tools for interacting with Atlassian Jira & Confluence",
		"0.1.0",
		server.WithToolCapabilities(true),
		server.WithInstructions("Provides tools for interacting with Atlassian Jira & Confluence."),
		server.WithHooks(hooks),
		server.WithRecovery(),
		server.WithLogging(),
		server.WithToolFilter(toolFilter),
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

	if os.Getenv("MCP_HTTP") != "" {
		svr := server.NewStreamableHTTPServer(s, server.WithHTTPContextFunc(svrCtxFunc))
		log.Info("Listening on :8080/mcp")
		if err := svr.Start(":8080"); err != nil {
			log.Fatal(err)
		}
	} else if os.Getenv("MCP_SSE") != "" {
		svr := server.NewSSEServer(s, server.WithSSEContextFunc(svrCtxFunc))
		log.Info("Listening on :8080/mcp")
		if err := svr.Start(":8080"); err != nil {
			log.Fatal(err)
		}
	} else {
		if err := server.ServeStdio(s); err != nil {
			fmt.Printf("Server error: %v\n", err)
		}
	}
}

func svrCtxFunc(ctx context.Context, r *http.Request) context.Context {
	for key, value := range r.Header {
		if strings.EqualFold(key, "JIRA_PERSONAL_TOKEN") {
			ctx = context.WithValue(ctx, clients.JiraPersonalTokenKey, value[0])
		}
		if strings.EqualFold(key, "CONFLUENCE_PERSONAL_TOKEN") {
			ctx = context.WithValue(ctx, clients.ConfluencePersonalTokenKey, value[0])
		}
	}
	return ctx
}

func toolFilter(ctx context.Context, tools []mcp.Tool) []mcp.Tool {
	enabledTools := []mcp.Tool{}
	enabledEnv := os.Getenv(envEnabledTools)
	disabledEnv := os.Getenv(envDisabledTools)

	if enabledEnv != "" {
		enabledSet := map[string]struct{}{}
		for _, name := range strings.Split(enabledEnv, ",") {
			enabledSet[strings.TrimSpace(name)] = struct{}{}
		}
		for _, t := range tools {
			if _, ok := enabledSet[t.Name]; ok {
				enabledTools = append(enabledTools, t)
			}
		}
		return enabledTools
	} else if disabledEnv != "" {
		disabledTools := strings.Split(disabledEnv, ",")
		for _, t := range tools {
			enabled := true
			for _, disabled := range disabledTools {
				if t.Name == strings.TrimSpace(disabled) {
					enabled = false
					break
				}
			}
			if enabled {
				enabledTools = append(enabledTools, t)
			}
		}
		return enabledTools
	}
	// If neither is set, return all tools
	return tools
}
