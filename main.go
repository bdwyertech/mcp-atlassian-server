package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"

	log "github.com/sirupsen/logrus"

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

func main() {
	hooks := &server.Hooks{}
	s := server.NewMCPServer(
		"Atlassian MCP - Provides tools for interacting with Atlassian Jira & Confluence",
		"0.1.0",
		server.WithToolCapabilities(true),
		server.WithInstructions("Provides tools for interacting with Atlassian Jira & Confluence."),
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

	// Choose serving mode: SSE or stdio
	if os.Getenv("MCP_HTTP") != "" {
		svr := server.NewStreamableHTTPServer(s, server.WithHTTPContextFunc(func(ctx context.Context, r *http.Request) context.Context {
			for key, value := range r.Header {
				fmt.Println(key)
				if key == "JIRA_PERSONAL_TOKEN" {
					fmt.Println("JIRA_PERSONAL_TOKEN")
					ctx = context.WithValue(ctx, clients.JiraPersonalTokenKey, value)
				}
				if key == "CONFLUENCE_PERSONAL_TOKEN" {
					fmt.Println("CONFLUENCE_PERSONAL_TOKEN")
					ctx = context.WithValue(ctx, clients.ConfluencePersonalTokenKey, value)
				}
			}
			return ctx
		}))
		log.Info("Listening on :8080/mcp")
		if err := svr.Start(":8080"); err != nil {
			log.Fatal(err)
		}
	} else if os.Getenv("MCP_SSE") != "" {
		svr := server.NewSSEServer(s, server.WithSSEContextFunc(func(ctx context.Context, r *http.Request) context.Context {
			for key, value := range r.Header {
				fmt.Println(key)
				if key == "JIRA_PERSONAL_TOKEN" {
					fmt.Println("JIRA_PERSONAL_TOKEN")
					ctx = context.WithValue(ctx, clients.JiraPersonalTokenKey, value)
				}
				if key == "CONFLUENCE_PERSONAL_TOKEN" {
					fmt.Println("CONFLUENCE_PERSONAL_TOKEN")
					ctx = context.WithValue(ctx, clients.ConfluencePersonalTokenKey, value)
				}
			}
			return ctx
		}))
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
