package main

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"mcp-atlassian-server/pkg/clients"
	"mcp-atlassian-server/pkg/utils"

	md "github.com/JohannesKaufmann/html-to-markdown"
	"github.com/ctreminiom/go-atlassian/v2/jira/agile"
	"github.com/ctreminiom/go-atlassian/v2/pkg/infra/models"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/sirupsen/logrus"
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

	s.AddTool(mcp.NewTool("confluence_ping",
		mcp.WithDescription("Ping Confluence API"),
	), confluencePingHandler)

	s.AddTool(mcp.NewTool("jira_ping",
		mcp.WithDescription("Ping Jira API"),
	), jiraPingHandler)

	s.AddTool(mcp.NewTool("confluence_search",
		mcp.WithDescription("Search Confluence content using simple terms or CQL"),
		mcp.WithString("query",
			mcp.Description("Search query - can be either a simple text or a CQL query string."),
			mcp.Required(),
		),
		mcp.WithNumber("limit",
			mcp.Description("Maximum number of results (1-50)"),
			mcp.DefaultNumber(10),
		),
		mcp.WithString("spaces_filter",
			mcp.Description("(Optional) Comma-separated list of space keys to filter results by."),
			mcp.DefaultString(""),
		),
	), confluenceSearchHandler)

	s.AddTool(mcp.NewTool("confluence_get_page",
		mcp.WithDescription("Get content of a specific Confluence page by its ID, or by its title and space key."),
		mcp.WithString("page_id",
			mcp.Description("Confluence page ID (numeric ID, can be found in the page URL). Provide this OR both 'title' and 'space_key'. If page_id is provided, title and space_key will be ignored."),
			mcp.DefaultString(""),
		),
		mcp.WithString("title",
			mcp.Description("The exact title of the Confluence page. Use this with 'space_key' if 'page_id' is not known."),
			mcp.DefaultString(""),
		),
		mcp.WithString("space_key",
			mcp.Description("The key of the Confluence space where the page resides (e.g., 'DEV', 'TEAM'). Required if using 'title'."),
			mcp.DefaultString(""),
		),
		mcp.WithBoolean("include_metadata",
			mcp.Description("Whether to include page metadata such as creation date, last update, version, and labels."),
			mcp.DefaultBool(true),
		),
		mcp.WithBoolean("convert_to_markdown",
			mcp.Description("Whether to convert page to markdown (true) or keep it in raw HTML format (false)."),
			mcp.DefaultBool(true),
		),
	), confluenceGetPageHandler)

	s.AddTool(mcp.NewTool("confluence_get_page_children",
		mcp.WithDescription("Get child pages of a specific Confluence page."),
		mcp.WithString("parent_id",
			mcp.Description("The ID of the parent page whose children you want to retrieve"),
			mcp.Required(),
		),
		mcp.WithString("expand",
			mcp.Description("Fields to expand in the response (e.g., 'version', 'body.storage')"),
			mcp.DefaultString("version"),
		),
		mcp.WithNumber("limit",
			mcp.Description("Maximum number of child pages to return (1-50)"),
			mcp.DefaultNumber(25),
		),
		mcp.WithBoolean("include_content",
			mcp.Description("Whether to include the page content in the response"),
			mcp.DefaultBool(false),
		),
		mcp.WithBoolean("convert_to_markdown",
			mcp.Description("Whether to convert page content to markdown (true) or keep it in raw HTML format (false). Only relevant if include_content is true."),
			mcp.DefaultBool(true),
		),
		mcp.WithNumber("start",
			mcp.Description("Starting index for pagination (0-based)"),
			mcp.DefaultNumber(0),
		),
	), confluenceGetPageChildrenHandler)

	s.AddTool(mcp.NewTool("confluence_get_comments",
		mcp.WithDescription("Get comments for a specific Confluence page."),
		mcp.WithString("page_id",
			mcp.Description("Confluence page ID (numeric ID, can be parsed from URL)"),
			mcp.Required(),
		),
	), confluenceGetCommentsHandler)

	s.AddTool(mcp.NewTool("confluence_get_labels",
		mcp.WithDescription("Get labels for a specific Confluence page."),
		mcp.WithString("page_id",
			mcp.Description("Confluence page ID (numeric ID, can be parsed from URL)"),
			mcp.Required(),
		),
	), confluenceGetLabelsHandler)

	s.AddTool(mcp.NewTool("confluence_add_label",
		mcp.WithDescription("Add label to an existing Confluence page."),
		mcp.WithString("page_id",
			mcp.Description("The ID of the page to update"),
			mcp.Required(),
		),
		mcp.WithString("name",
			mcp.Description("The name of the label"),
			mcp.Required(),
		),
	), confluenceAddLabelHandler)

	s.AddTool(mcp.NewTool("confluence_create_page",
		mcp.WithDescription("Create a new Confluence page."),
		mcp.WithString("space_key",
			mcp.Description("The key of the space to create the page in (usually a short uppercase code like 'DEV', 'TEAM', or 'DOC')"),
			mcp.Required(),
		),
		mcp.WithString("title",
			mcp.Description("The title of the page"),
			mcp.Required(),
		),
		mcp.WithString("content",
			mcp.Description("The content of the page in Markdown format. Supports headings, lists, tables, code blocks, and other Markdown syntax"),
			mcp.Required(),
		),
		mcp.WithString("parent_id",
			mcp.Description("(Optional) parent page ID. If provided, this page will be created as a child of the specified page"),
			mcp.DefaultString(""),
		),
	), confluenceCreatePageHandler)

	s.AddTool(mcp.NewTool("confluence_update_page",
		mcp.WithDescription("Update an existing Confluence page."),
		mcp.WithString("page_id",
			mcp.Description("The ID of the page to update"),
			mcp.Required(),
		),
		mcp.WithString("title",
			mcp.Description("The new title of the page"),
			mcp.Required(),
		),
		mcp.WithString("content",
			mcp.Description("The new content of the page in Markdown format"),
			mcp.Required(),
		),
		mcp.WithBoolean("is_minor_edit",
			mcp.Description("Whether this is a minor edit"),
			mcp.DefaultBool(false),
		),
		mcp.WithString("version_comment",
			mcp.Description("Optional comment for this version"),
			mcp.DefaultString(""),
		),
		mcp.WithString("parent_id",
			mcp.Description("Optional new parent page ID"),
			mcp.DefaultString(""),
		),
	), confluenceUpdatePageHandler)

	s.AddTool(mcp.NewTool("confluence_delete_page",
		mcp.WithDescription("Delete an existing Confluence page."),
		mcp.WithString("page_id",
			mcp.Description("The ID of the page to delete"),
			mcp.Required(),
		),
	), confluenceDeletePageHandler)

	s.AddTool(mcp.NewTool("confluence_add_comment",
		mcp.WithDescription("Add a comment to a Confluence page."),
		mcp.WithString("page_id",
			mcp.Description("The ID of the page to add a comment to"),
			mcp.Required(),
		),
		mcp.WithString("content",
			mcp.Description("The comment content in Markdown format"),
			mcp.Required(),
		),
	), confluenceAddCommentHandler)

	s.AddTool(mcp.NewTool("jira_get_user_profile",
		mcp.WithDescription("Retrieve profile information for a specific Jira user."),
		mcp.WithString("user_identifier",
			mcp.Description("Identifier for the user (e.g., email address, username, account ID, or key for Server/DC)."),
			mcp.Required(),
		),
	), jiraGetUserProfileHandler)

	s.AddTool(mcp.NewTool("jira_get_issue",
		mcp.WithDescription("Get details of a specific Jira issue."),
		mcp.WithString("issue_key", mcp.Description("Jira issue key (e.g., 'PROJ-123')"), mcp.Required()),
		mcp.WithString("fields", mcp.Description("Comma-separated list of fields to return (e.g., 'summary,status'). Use '*all' for all fields."), mcp.DefaultString("")),
		mcp.WithString("expand", mcp.Description("Fields to expand (e.g., 'renderedFields', 'transitions', 'changelog')"), mcp.DefaultString("")),
		mcp.WithNumber("comment_limit", mcp.Description("Maximum number of comments to include (0 for none)"), mcp.DefaultNumber(10)),
		mcp.WithString("properties", mcp.Description("Comma-separated list of issue properties to return"), mcp.DefaultString("")),
		mcp.WithBoolean("update_history", mcp.Description("Whether to update the issue view history for the requesting user"), mcp.DefaultBool(true)),
	), jiraGetIssueHandler)

	s.AddTool(mcp.NewTool("jira_search",
		mcp.WithDescription("Search Jira issues using JQL (Jira Query Language)."),
		mcp.WithString("jql", mcp.Description("JQL query string (e.g., 'project = PROJ AND status = \"In Progress\"')"), mcp.Required()),
		mcp.WithString("fields", mcp.Description("Comma-separated fields to return in the results. Use '*all' for all fields."), mcp.DefaultString("")),
		mcp.WithNumber("limit", mcp.Description("Maximum number of results (1-50)"), mcp.DefaultNumber(10)),
		mcp.WithNumber("start_at", mcp.Description("Starting index for pagination (0-based)"), mcp.DefaultNumber(0)),
		mcp.WithString("projects_filter", mcp.Description("Comma-separated list of project keys to filter results by."), mcp.DefaultString("")),
		mcp.WithString("expand", mcp.Description("Fields to expand (e.g., 'renderedFields', 'transitions', 'changelog')"), mcp.DefaultString("")),
	), jiraSearchHandler)

	s.AddTool(mcp.NewTool("jira_search_fields",
		mcp.WithDescription("Search Jira fields by keyword with fuzzy match."),
		mcp.WithString("keyword", mcp.Description("Keyword for fuzzy search. If left empty, lists the first 'limit' available fields in their default order."), mcp.DefaultString("")),
		mcp.WithNumber("limit", mcp.Description("Maximum number of results"), mcp.DefaultNumber(10)),
		mcp.WithBoolean("refresh", mcp.Description("Whether to force refresh the field list"), mcp.DefaultBool(false)),
	), jiraSearchFieldsHandler)

	s.AddTool(mcp.NewTool("jira_get_project_issues",
		mcp.WithDescription("Get all issues for a specific Jira project."),
		mcp.WithString("project_key", mcp.Description("The project key"), mcp.Required()),
		mcp.WithNumber("limit", mcp.Description("Maximum number of results (1-50)"), mcp.DefaultNumber(10)),
		mcp.WithNumber("start_at", mcp.Description("Starting index for pagination (0-based)"), mcp.DefaultNumber(0)),
	), jiraGetProjectIssuesHandler)

	s.AddTool(mcp.NewTool("jira_get_transitions",
		mcp.WithDescription("Get available status transitions for a Jira issue."),
		mcp.WithString("issue_key", mcp.Description("Jira issue key (e.g., 'PROJ-123')"), mcp.Required()),
	), jiraGetTransitionsHandler)

	s.AddTool(mcp.NewTool("jira_get_worklog",
		mcp.WithDescription("Get worklog entries for a Jira issue."),
		mcp.WithString("issue_key", mcp.Description("Jira issue key (e.g., 'PROJ-123')"), mcp.Required()),
	), jiraGetWorklogHandler)

	s.AddTool(mcp.NewTool("jira_get_agile_boards",
		mcp.WithDescription("Get Jira agile boards by name, project key, or type."),
		mcp.WithString("board_name", mcp.Description("(Optional) The name of board, support fuzzy search"), mcp.DefaultString("")),
		mcp.WithString("project_key", mcp.Description("(Optional) Jira project key (e.g., 'PROJ-123')"), mcp.DefaultString("")),
		mcp.WithString("board_type", mcp.Description("(Optional) The type of jira board (e.g., 'scrum', 'kanban')"), mcp.DefaultString("")),
		mcp.WithNumber("start_at", mcp.Description("Starting index for pagination (0-based)"), mcp.DefaultNumber(0)),
		mcp.WithNumber("limit", mcp.Description("Maximum number of results (1-50)"), mcp.DefaultNumber(10)),
	), jiraGetAgileBoardsHandler)

	s.AddTool(mcp.NewTool("jira_get_board_issues",
		mcp.WithDescription("Get all issues linked to a specific board filtered by JQL."),
		mcp.WithNumber("board_id", mcp.Description("The id of the board (e.g., '1001')"), mcp.Required()),
		mcp.WithString("jql", mcp.Description("JQL query string to filter issues."), mcp.Required()),
		mcp.WithString("fields", mcp.Description("Comma-separated fields to return in the results. Use '*all' for all fields."), mcp.DefaultString("")),
		mcp.WithNumber("start_at", mcp.Description("Starting index for pagination (0-based)"), mcp.DefaultNumber(0)),
		mcp.WithNumber("limit", mcp.Description("Maximum number of results (1-50)"), mcp.DefaultNumber(10)),
		mcp.WithString("expand", mcp.Description("Optional fields to expand in the response (e.g., 'changelog')."), mcp.DefaultString("version")),
	), jiraGetBoardIssuesHandler)

	s.AddTool(mcp.NewTool("jira_get_sprints_from_board",
		mcp.WithDescription("Get Jira sprints from board by state."),
		mcp.WithNumber("board_id", mcp.Description("The id of board (e.g., '1000')"), mcp.Required()),
		mcp.WithString("state", mcp.Description("Sprint state (e.g., 'active', 'future', 'closed')"), mcp.DefaultString("")),
		mcp.WithNumber("start_at", mcp.Description("Starting index for pagination (0-based)"), mcp.DefaultNumber(0)),
		mcp.WithNumber("limit", mcp.Description("Maximum number of results (1-50)"), mcp.DefaultNumber(10)),
	), jiraGetSprintsFromBoardHandler)

	s.AddTool(mcp.NewTool("jira_get_sprint_issues",
		mcp.WithDescription("Get Jira issues from sprint."),
		mcp.WithNumber("sprint_id", mcp.Description("The id of sprint (e.g., '10001')"), mcp.Required()),
		mcp.WithString("fields", mcp.Description("Comma-separated fields to return in the results. Use '*all' for all fields."), mcp.DefaultString("")),
		mcp.WithNumber("start_at", mcp.Description("Starting index for pagination (0-based)"), mcp.DefaultNumber(0)),
		mcp.WithNumber("limit", mcp.Description("Maximum number of results (1-50)"), mcp.DefaultNumber(10)),
	), jiraGetSprintIssuesHandler)

	s.AddTool(mcp.NewTool("jira_download_attachments",
		mcp.WithDescription("Download attachments from a Jira issue."),
		mcp.WithString("issue_key", mcp.Description("Jira issue key"), mcp.Required()),
		mcp.WithString("target_dir", mcp.Description("Directory to save attachments"), mcp.Required()),
	), jiraDownloadAttachmentsHandler)

	s.AddTool(mcp.NewTool("jira_get_link_types",
		mcp.WithDescription("Get all available issue link types."),
	), jiraGetLinkTypesHandler)

	s.AddTool(mcp.NewTool("jira_create_issue",
		mcp.WithDescription("Create a new Jira issue with optional Epic link or parent for subtasks."),
		mcp.WithString("project_key", mcp.Description("The JIRA project key"), mcp.Required()),
		mcp.WithString("summary", mcp.Description("Summary/title of the issue"), mcp.Required()),
		mcp.WithString("issue_type", mcp.Description("Issue type (e.g., 'Task', 'Bug', 'Story', 'Epic', 'Subtask')"), mcp.Required()),
		mcp.WithString("assignee", mcp.Description("Assignee's user identifier (email, display name, or account ID)"), mcp.DefaultString("")),
		mcp.WithString("description", mcp.Description("Issue description"), mcp.DefaultString("")),
		mcp.WithString("components", mcp.Description("Comma-separated list of component names"), mcp.DefaultString("")),
		mcp.WithString("additional_fields", mcp.Description("JSON string of additional fields"), mcp.DefaultString("")),
	), jiraCreateIssueHandler)

	s.AddTool(mcp.NewTool("jira_batch_create_issues",
		mcp.WithDescription("Create multiple Jira issues in a batch."),
		mcp.WithString("issues", mcp.Description("JSON array string of issue objects"), mcp.Required()),
		mcp.WithBoolean("validate_only", mcp.Description("If true, only validates without creating"), mcp.DefaultBool(false)),
	), jiraBatchCreateIssuesHandler)

	s.AddTool(mcp.NewTool("jira_batch_get_changelogs",
		mcp.WithDescription("Get changelogs for multiple Jira issues (Cloud only)."),
		mcp.WithString("issue_ids_or_keys", mcp.Description("Comma-separated list of issue IDs or keys"), mcp.Required()),
		mcp.WithString("fields", mcp.Description("Comma-separated list of fields to filter changelogs by. None for all fields."), mcp.DefaultString("")),
		mcp.WithNumber("limit", mcp.Description("Maximum changelogs per issue (-1 for all)"), mcp.DefaultNumber(-1)),
	), jiraBatchGetChangelogsHandler)

	s.AddTool(mcp.NewTool("jira_update_issue",
		mcp.WithDescription("Update an existing Jira issue including changing status, adding Epic links, updating fields, etc."),
		mcp.WithString("issue_key", mcp.Description("Jira issue key"), mcp.Required()),
		mcp.WithString("fields", mcp.Description("JSON string of fields to update"), mcp.Required()),
		mcp.WithString("additional_fields", mcp.Description("Optional JSON string of additional fields"), mcp.DefaultString("")),
		mcp.WithString("attachments", mcp.Description("Optional JSON array string or comma-separated list of file paths"), mcp.DefaultString("")),
	), jiraUpdateIssueHandler)

	s.AddTool(mcp.NewTool("jira_delete_issue",
		mcp.WithDescription("Delete an existing Jira issue."),
		mcp.WithString("issue_key", mcp.Description("Jira issue key"), mcp.Required()),
	), jiraDeleteIssueHandler)

	s.AddTool(mcp.NewTool("jira_add_comment",
		mcp.WithDescription("Add a comment to a Jira issue."),
		mcp.WithString("issue_key", mcp.Description("Jira issue key"), mcp.Required()),
		mcp.WithString("comment", mcp.Description("Comment text in Markdown"), mcp.Required()),
	), jiraAddCommentHandler)

	s.AddTool(mcp.NewTool("jira_add_worklog",
		mcp.WithDescription("Add a worklog entry to a Jira issue."),
		mcp.WithString("issue_key", mcp.Description("Jira issue key"), mcp.Required()),
		mcp.WithString("time_spent", mcp.Description("Time spent in Jira format (e.g., '2h 30m')"), mcp.Required()),
		mcp.WithString("comment", mcp.Description("Optional comment in Markdown"), mcp.DefaultString("")),
		mcp.WithString("started", mcp.Description("Optional start time in ISO format"), mcp.DefaultString("")),
		mcp.WithString("original_estimate", mcp.Description("Optional new original estimate"), mcp.DefaultString("")),
		mcp.WithString("remaining_estimate", mcp.Description("Optional new remaining estimate"), mcp.DefaultString("")),
	), jiraAddWorklogHandler)

	s.AddTool(mcp.NewTool("jira_link_to_epic",
		mcp.WithDescription("Link an existing issue to an epic."),
		mcp.WithString("issue_key", mcp.Description("The key of the issue to link"), mcp.Required()),
		mcp.WithString("epic_key", mcp.Description("The key of the epic to link to"), mcp.Required()),
	), jiraLinkToEpicHandler)

	s.AddTool(mcp.NewTool("jira_create_issue_link",
		mcp.WithDescription("Create a link between two Jira issues."),
		mcp.WithString("link_type", mcp.Description("The type of link (e.g., 'Blocks')"), mcp.Required()),
		mcp.WithString("inward_issue_key", mcp.Description("The key of the source issue"), mcp.Required()),
		mcp.WithString("outward_issue_key", mcp.Description("The key of the target issue"), mcp.Required()),
		mcp.WithString("comment", mcp.Description("Optional comment text"), mcp.DefaultString("")),
		mcp.WithString("comment_visibility", mcp.Description("Optional JSON string for comment visibility"), mcp.DefaultString("")),
	), jiraCreateIssueLinkHandler)

	s.AddTool(mcp.NewTool("jira_remove_issue_link",
		mcp.WithDescription("Remove a link between two Jira issues."),
		mcp.WithString("link_id", mcp.Description("The ID of the link to remove"), mcp.Required()),
	), jiraRemoveIssueLinkHandler)

	s.AddTool(mcp.NewTool("jira_transition_issue",
		mcp.WithDescription("Transition a Jira issue to a new status."),
		mcp.WithString("issue_key", mcp.Description("Jira issue key"), mcp.Required()),
		mcp.WithString("transition_id", mcp.Description("ID of the transition"), mcp.Required()),
		mcp.WithString("fields", mcp.Description("Optional JSON string of fields to update during transition"), mcp.DefaultString("")),
		mcp.WithString("comment", mcp.Description("Optional comment for the transition"), mcp.DefaultString("")),
	), jiraTransitionIssueHandler)

	s.AddTool(mcp.NewTool("jira_create_sprint",
		mcp.WithDescription("Create Jira sprint for a board."),
		mcp.WithNumber("board_id", mcp.Description("Board ID"), mcp.Required()),
		mcp.WithString("sprint_name", mcp.Description("Sprint name"), mcp.Required()),
		mcp.WithString("start_date", mcp.Description("Start date (ISO format)"), mcp.DefaultString("")),
		mcp.WithString("end_date", mcp.Description("End date (ISO format)"), mcp.DefaultString("")),
		mcp.WithString("goal", mcp.Description("Optional sprint goal"), mcp.DefaultString("")),
	), jiraCreateSprintHandler)

	s.AddTool(mcp.NewTool("jira_update_sprint",
		mcp.WithDescription("Update jira sprint."),
		mcp.WithNumber("sprint_id", mcp.Description("The ID of the sprint"), mcp.Required()),
		mcp.WithString("sprint_name", mcp.Description("Optional new name"), mcp.DefaultString("")),
		mcp.WithString("state", mcp.Description("Optional new state (future|active|closed)"), mcp.DefaultString("")),
		mcp.WithString("start_date", mcp.Description("Optional new start date"), mcp.DefaultString("")),
		mcp.WithString("end_date", mcp.Description("Optional new end date"), mcp.DefaultString("")),
		mcp.WithString("goal", mcp.Description("Optional new goal"), mcp.DefaultString("")),
	), jiraUpdateSprintHandler)

	if err := server.ServeStdio(s); err != nil {
		fmt.Printf("Server error: %v\n", err)
	}
}

func confluencePingHandler(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	client, err := clients.GetConfluenceClient()
	if err != nil {
		return mcp.NewToolResultError("Confluence client error: " + err.Error()), nil
	}
	// Use a simple content search as a health check
	_, resp, err := client.Search.Content(ctx, "type=page", &models.SearchContentOptions{Limit: 1})
	if err != nil || resp == nil || resp.StatusCode != 200 {
		return mcp.NewToolResultError("Confluence ping failed"), nil
	}
	return mcp.NewToolResultText("Confluence OK"), nil
}

func jiraPingHandler(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	client, err := clients.GetJiraClient()
	if err != nil {
		return mcp.NewToolResultError("Jira client error: " + err.Error()), nil
	}
	_, resp, err := client.MySelf.Details(ctx, []string{})
	if err != nil || resp == nil || resp.StatusCode != 200 {
		logrus.Error(err)
		return mcp.NewToolResultError("Jira ping failed:" + err.Error()), nil
	}
	return mcp.NewToolResultText("Jira OK"), nil
}

// Handler for confluence_search
func confluenceSearchHandler(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	query, err := req.RequireString("query")
	if err != nil {
		return mcp.NewToolResultError("Missing required parameter: query"), nil
	}
	limit := req.GetInt("limit", 10)
	if limit < 1 || limit > 50 {
		limit = 10
	}
	spacesFilter := req.GetString("spaces_filter", "")

	client, err := clients.GetConfluenceClient()
	if err != nil {
		return mcp.NewToolResultError("Confluence client error: " + err.Error()), nil
	}

	cql := query
	if spacesFilter != "" {
		spaceKeys := strings.Split(spacesFilter, ",")
		var spaceCQLs []string
		for _, key := range spaceKeys {
			key = strings.TrimSpace(key)
			if key != "" {
				spaceCQLs = append(spaceCQLs, fmt.Sprintf("space=\"%s\"", key))
			}
		}
		if len(spaceCQLs) > 0 {
			cql = fmt.Sprintf("(%s) AND (%s)", cql, strings.Join(spaceCQLs, " OR "))
		}
	}

	options := &models.SearchContentOptions{
		Limit: limit,
	}
	results, resp, err := client.Search.Content(ctx, cql, options)
	if err != nil || resp == nil || resp.StatusCode != 200 {
		return mcp.NewToolResultError("Confluence search failed: " + err.Error()), nil
	}
	jsonBytes, err := json.Marshal(results)
	if err != nil {
		return mcp.NewToolResultError("Failed to marshal results: " + err.Error()), nil
	}
	return mcp.NewToolResultText(string(jsonBytes)), nil
}

// Handler for confluence_get_page
func confluenceGetPageHandler(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	pageID := req.GetString("page_id", "")
	title := req.GetString("title", "")
	spaceKey := req.GetString("space_key", "")
	includeMetadata := req.GetBool("include_metadata", true)
	convertToMarkdown := req.GetBool("convert_to_markdown", true)
	client, err := clients.GetConfluenceClient()
	if err != nil {
		return mcp.NewToolResultError("Confluence client error: " + err.Error()), nil
	}

	if pageID != "" {
		// Use correct expands for v2 SDK
		expands := []string{"body.storage", "version", "metadata.labels"}
		page, resp, err := client.Content.Get(ctx, pageID, expands, 0)
		if err != nil || resp == nil || resp.StatusCode != 200 {
			return mcp.NewToolResultError("Failed to retrieve page by ID: " + err.Error()), nil
		}
		if !includeMetadata {
			if page.Body != nil && page.Body.Storage != nil {
				if convertToMarkdown {
					converter := md.NewConverter("", true, nil)
					markdown, err := converter.ConvertString(page.Body.Storage.Value)
					if err != nil {
						return mcp.NewToolResultError("Failed to convert HTML to Markdown: " + err.Error()), nil
					}
					jsonBytes, _ := json.Marshal(markdown)
					return mcp.NewToolResultText(string(jsonBytes)), nil
				} else {
					jsonBytes, _ := json.Marshal(page.Body.Storage.Value)
					return mcp.NewToolResultText(string(jsonBytes)), nil
				}
			}
		}
		jsonBytes, _ := json.Marshal(page)
		return mcp.NewToolResultText(string(jsonBytes)), nil
	} else if title != "" && spaceKey != "" {
		cql := fmt.Sprintf("title=\"%s\" AND space=\"%s\"", title, spaceKey)
		results, resp, err := client.Search.Content(ctx, cql, &models.SearchContentOptions{Limit: 1})
		if err != nil || resp == nil || resp.StatusCode != 200 || len(results.Results) == 0 {
			return mcp.NewToolResultError(fmt.Sprintf("Page with title '%s' not found in space '%s'", title, spaceKey)), nil
		}
		jsonBytes, _ := json.Marshal(results.Results[0])
		return mcp.NewToolResultText(string(jsonBytes)), nil
	} else {
		return mcp.NewToolResultError("Either 'page_id' OR both 'title' and 'space_key' must be provided."), nil
	}
}

// Handler for confluence_get_page_children
func confluenceGetPageChildrenHandler(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	parentID := req.GetString("parent_id", "")
	expand := req.GetString("expand", "version")
	limit := req.GetInt("limit", 25)
	start := req.GetInt("start", 0)

	client, err := clients.GetConfluenceClient()
	if err != nil {
		return mcp.NewToolResultError("Confluence client error: " + err.Error()), nil
	}

	expands := []string{}
	if expand != "" {
		expands = strings.Split(expand, ",")
		for i := range expands {
			expands[i] = strings.TrimSpace(expands[i])
		}
	}
	// Comment out containsBodyStorage usage if present
	// if includeContent && !containsBodyStorage(expand) {
	// 	expands = append(expands, "body.storage")
	// }
	children, resp, err := client.Content.ChildrenDescendant.ChildrenByType(ctx, parentID, "page", 0, expands, start, limit)
	if err != nil || resp == nil || resp.StatusCode != 200 {
		return mcp.NewToolResultError("Failed to get child pages: " + err.Error()), nil
	}
	jsonBytes, _ := json.Marshal(map[string]any{
		"parent_id":       parentID,
		"count":           len(children.Results),
		"limit_requested": limit,
		"start_requested": start,
		"results":         children.Results,
	})
	return mcp.NewToolResultText(string(jsonBytes)), nil
}

// Handler for confluence_get_comments
func confluenceGetCommentsHandler(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	pageID := req.GetString("page_id", "")
	client, err := clients.GetConfluenceClient()
	if err != nil {
		return mcp.NewToolResultError("Confluence client error: " + err.Error()), nil
	}
	// v2 SDK: expands and pagination
	comments, resp, err := client.Content.Comment.Gets(ctx, pageID, nil, nil, 0, 50)
	if err != nil || resp == nil || resp.StatusCode != 200 {
		return mcp.NewToolResultError("Failed to get comments: " + err.Error()), nil
	}
	jsonBytes, _ := json.Marshal(comments)
	return mcp.NewToolResultText(string(jsonBytes)), nil
}

// Handler for confluence_get_labels
func confluenceGetLabelsHandler(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	pageID := req.GetString("page_id", "")
	client, err := clients.GetConfluenceClient()
	if err != nil {
		return mcp.NewToolResultError("Confluence client error: " + err.Error()), nil
	}
	labels, resp, err := client.Content.Label.Gets(ctx, pageID, "", 0, 50)
	if err != nil || resp == nil || resp.StatusCode != 200 {
		return mcp.NewToolResultError("Failed to get labels: " + err.Error()), nil
	}
	jsonBytes, _ := json.Marshal(labels)
	return mcp.NewToolResultText(string(jsonBytes)), nil
}

// Handler for confluence_add_label
func confluenceAddLabelHandler(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	pageID := req.GetString("page_id", "")
	name := req.GetString("name", "")
	client, err := clients.GetConfluenceClient()
	if err != nil {
		return mcp.NewToolResultError("Confluence client error: " + err.Error()), nil
	}
	payload := []*models.ContentLabelPayloadScheme{{Prefix: "global", Name: name}}
	labels, resp, err := client.Content.Label.Add(ctx, pageID, payload, false)
	if err != nil || resp == nil || resp.StatusCode != 200 {
		return mcp.NewToolResultError("Failed to add label: " + err.Error()), nil
	}
	jsonBytes, _ := json.Marshal(labels)
	return mcp.NewToolResultText(string(jsonBytes)), nil
}

// Handler for confluence_create_page
func confluenceCreatePageHandler(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	spaceKey := req.GetString("space_key", "")
	title := req.GetString("title", "")
	content := req.GetString("content", "")
	parentID := req.GetString("parent_id", "")
	client, err := clients.GetConfluenceClient()
	if err != nil {
		return mcp.NewToolResultError("Confluence client error: " + err.Error()), nil
	}
	pagePayload := &models.ContentScheme{
		Type:  "page",
		Title: title,
		Space: &models.SpaceScheme{Key: spaceKey},
		Body: &models.BodyScheme{
			Storage: &models.BodyNodeScheme{
				Value:          content,
				Representation: "wiki",
			},
		},
	}
	if parentID != "" {
		pagePayload.Ancestors = []*models.ContentScheme{{ID: parentID}}
	}
	page, resp, err := client.Content.Create(ctx, pagePayload)
	if err != nil || resp == nil || (resp.StatusCode != 200 && resp.StatusCode != 201) {
		return mcp.NewToolResultError("Failed to create page: " + err.Error()), nil
	}
	jsonBytes, _ := json.Marshal(page)
	return mcp.NewToolResultText(string(jsonBytes)), nil
}

// Handler for confluence_update_page
func confluenceUpdatePageHandler(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	pageID := req.GetString("page_id", "")
	title := req.GetString("title", "")
	content := req.GetString("content", "")
	isMinorEdit := req.GetBool("is_minor_edit", false)
	versionComment := req.GetString("version_comment", "")
	parentID := req.GetString("parent_id", "")
	client, err := clients.GetConfluenceClient()
	if err != nil {
		return mcp.NewToolResultError("Confluence client error: " + err.Error()), nil
	}
	current, resp, err := client.Content.Get(ctx, pageID, []string{"version"}, 0)
	if err != nil || resp == nil || resp.StatusCode != 200 {
		return mcp.NewToolResultError("Failed to get current page: " + err.Error()), nil
	}
	newVersion := 1
	if current.Version != nil && current.Version.Number > 0 {
		newVersion = current.Version.Number + 1
	}
	updatePayload := &models.ContentScheme{
		ID:    pageID,
		Type:  "page",
		Title: title,
		Version: &models.ContentVersionScheme{
			Number:    newVersion,
			MinorEdit: isMinorEdit,
			Message:   versionComment,
		},
		Body: &models.BodyScheme{
			Storage: &models.BodyNodeScheme{
				Value:          content,
				Representation: "wiki",
			},
		},
	}
	if parentID != "" {
		updatePayload.Ancestors = []*models.ContentScheme{{ID: parentID}}
	}
	page, resp, err := client.Content.Update(ctx, pageID, updatePayload)
	if err != nil || resp == nil || resp.StatusCode != 200 {
		return mcp.NewToolResultError("Failed to update page: " + err.Error()), nil
	}
	jsonBytes, _ := json.Marshal(page)
	return mcp.NewToolResultText(string(jsonBytes)), nil
}

// Handler for confluence_delete_page
func confluenceDeletePageHandler(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	pageID := req.GetString("page_id", "")
	client, err := clients.GetConfluenceClient()
	if err != nil {
		return mcp.NewToolResultError("Confluence client error: " + err.Error()), nil
	}
	resp, err := client.Content.Delete(ctx, pageID, "current")
	if err != nil || resp == nil || resp.StatusCode != 204 {
		return mcp.NewToolResultError("Failed to delete page: " + err.Error()), nil
	}
	return mcp.NewToolResultText(fmt.Sprintf("Page %s deleted successfully", pageID)), nil
}

// Handler for confluence_add_comment
func confluenceAddCommentHandler(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	pageID := req.GetString("page_id", "")
	content := req.GetString("content", "")
	client, err := clients.GetConfluenceClient()
	if err != nil {
		return mcp.NewToolResultError("Confluence client error: " + err.Error()), nil
	}
	commentPayload := &models.ContentScheme{
		Type:      "comment",
		Title:     "Comment",
		Ancestors: []*models.ContentScheme{{ID: pageID}},
		Body: &models.BodyScheme{
			Storage: &models.BodyNodeScheme{
				Value:          content,
				Representation: "wiki",
			},
		},
	}
	comment, resp, err := client.Content.Create(ctx, commentPayload)
	if err != nil || resp == nil || (resp.StatusCode != 200 && resp.StatusCode != 201) {
		return mcp.NewToolResultError("Failed to add comment: " + err.Error()), nil
	}
	jsonBytes, _ := json.Marshal(comment)
	return mcp.NewToolResultText(string(jsonBytes)), nil
}

// Handler for jira_get_user_profile
func jiraGetUserProfileHandler(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	userIdentifier := req.GetString("user_identifier", "")
	client, err := clients.GetJiraClient()
	if err != nil {
		return mcp.NewToolResultError("Jira client error: " + err.Error()), nil
	}
	user, resp, err := client.User.Get(ctx, userIdentifier, nil)
	if err != nil || resp == nil || resp.StatusCode != 200 {
		return mcp.NewToolResultError("Failed to get user profile: " + err.Error()), nil
	}
	jsonBytes, _ := json.Marshal(user)
	return mcp.NewToolResultText(string(jsonBytes)), nil
}

// Handler for jira_get_issue
func jiraGetIssueHandler(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	issueKey := req.GetString("issue_key", "")
	fields := req.GetString("fields", "")
	expand := req.GetString("expand", "")
	commentLimit := req.GetInt("comment_limit", 10)
	client, err := clients.GetJiraClient()
	if err != nil {
		return mcp.NewToolResultError("Jira client error: " + err.Error()), nil
	}
	var fieldSlice, expandSlice []string
	if fields != "" {
		fieldSlice = utils.SplitAndTrim(fields)
	}
	if expand != "" {
		expandSlice = utils.SplitAndTrim(expand)
	}
	issue, resp, err := client.Issue.Get(ctx, issueKey, fieldSlice, expandSlice)
	if err != nil || resp == nil || resp.StatusCode != 200 {
		return mcp.NewToolResultError("Failed to get issue: " + err.Error()), nil
	}
	// If comments are present and commentLimit is set, trim the comments array
	if issue.Fields != nil && issue.Fields.Comment != nil && len(issue.Fields.Comment.Comments) > commentLimit && commentLimit > 0 {
		issue.Fields.Comment.Comments = issue.Fields.Comment.Comments[:commentLimit]
	}
	jsonBytes, _ := json.Marshal(issue)
	return mcp.NewToolResultText(string(jsonBytes)), nil
}

// Handler for jira_search
func jiraSearchHandler(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	jql := req.GetString("jql", "")
	fields := req.GetString("fields", "")
	limit := req.GetInt("limit", 10)
	startAt := req.GetInt("start_at", 0)
	projectsFilter := req.GetString("projects_filter", "")
	expand := req.GetString("expand", "")
	properties := req.GetString("properties", "")
	client, err := clients.GetJiraClient()
	if err != nil {
		return mcp.NewToolResultError("Jira client error: " + err.Error()), nil
	}
	var fieldSlice, expandSlice []string
	if fields != "" {
		fieldSlice = utils.SplitAndTrim(fields)
	}
	if expand != "" {
		expandSlice = utils.SplitAndTrim(expand)
	}
	// If projectsFilter is set, prepend a project filter to the JQL
	if projectsFilter != "" {
		projects := utils.SplitAndTrim(projectsFilter)
		if len(projects) > 0 {
			quoted := make([]string, len(projects))
			for i, p := range projects {
				quoted[i] = "'" + p + "'"
			}
			projectJQL := "project in (" + strings.Join(quoted, ",") + ")"
			if jql != "" {
				jql = projectJQL + " AND (" + jql + ")"
			} else {
				jql = projectJQL
			}
		}
	}
	issues, resp, err := client.Issue.Search.Post(ctx, jql, fieldSlice, expandSlice, startAt, limit, properties)
	if err != nil || resp == nil || resp.StatusCode != 200 {
		return mcp.NewToolResultError("Failed to search issues: " + err.Error()), nil
	}
	jsonBytes, _ := json.Marshal(issues)
	return mcp.NewToolResultText(string(jsonBytes)), nil
}

// Handler for jira_search_fields
func jiraSearchFieldsHandler(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	keyword := req.GetString("keyword", "")
	limit := req.GetInt("limit", 10)
	startAt := req.GetInt("start_at", 0)
	refresh := req.GetBool("refresh", false)
	_ = refresh // Not used in this handler
	client, err := clients.GetJiraClient()
	if err != nil {
		return mcp.NewToolResultError("Jira client error: " + err.Error()), nil
	}
	options := &models.FieldSearchOptionsScheme{Query: keyword}
	fieldsPage, resp, err := client.Issue.Field.Search(ctx, options, startAt, limit)
	if err != nil || resp == nil || resp.StatusCode != 200 {
		return mcp.NewToolResultError("Failed to search fields: " + err.Error()), nil
	}
	jsonBytes, _ := json.Marshal(fieldsPage.Values)
	return mcp.NewToolResultText(string(jsonBytes)), nil
}

// Handler for jira_get_project_issues
func jiraGetProjectIssuesHandler(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	projectKey := req.GetString("project_key", "")
	limit := req.GetInt("limit", 10)
	startAt := req.GetInt("start_at", 0)
	client, err := clients.GetJiraClient()
	if err != nil {
		return mcp.NewToolResultError("Jira client error: " + err.Error()), nil
	}

	jql := "project = '" + projectKey + "'"
	issues, resp, err := client.Issue.Search.Post(ctx, jql, nil, nil, startAt, limit, "")
	if err != nil || resp == nil || resp.StatusCode != 200 {
		return mcp.NewToolResultError("Failed to get project issues: " + err.Error()), nil
	}
	jsonBytes, _ := json.Marshal(issues)
	return mcp.NewToolResultText(string(jsonBytes)), nil
}

// --- Jira Tool Handler Implementations ---

func jiraGetTransitionsHandler(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	issueKey := req.GetString("issue_key", "")
	if issueKey == "" {
		return mcp.NewToolResultError("Missing required parameter: issue_key"), nil
	}
	client, err := clients.GetJiraClient()
	if err != nil {
		return mcp.NewToolResultError("Jira client error: " + err.Error()), nil
	}
	transitions, resp, err := client.Issue.Transitions(ctx, issueKey)
	if err != nil || resp == nil || resp.StatusCode != 200 {
		return mcp.NewToolResultError("Failed to get transitions: " + err.Error()), nil
	}
	jsonBytes, _ := json.Marshal(transitions)
	return mcp.NewToolResultText(string(jsonBytes)), nil
}

func jiraGetWorklogHandler(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	issueKey := req.GetString("issue_key", "")
	if issueKey == "" {
		return mcp.NewToolResultError("Missing required parameter: issue_key"), nil
	}
	client, err := clients.GetJiraClient()
	if err != nil {
		return mcp.NewToolResultError("Jira client error: " + err.Error()), nil
	}
	startAt := 0
	maxResults := 100
	after := 0
	var expand []string
	worklogs, resp, err := client.Issue.Worklog.Issue(ctx, issueKey, startAt, maxResults, after, expand)
	if err != nil || resp == nil || resp.StatusCode != 200 {
		return mcp.NewToolResultError("Failed to get worklogs: " + err.Error()), nil
	}
	jsonBytes, _ := json.Marshal(worklogs)
	return mcp.NewToolResultText(string(jsonBytes)), nil
}

func jiraGetAgileBoardsHandler(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	boardName := req.GetString("board_name", "")
	projectKey := req.GetString("project_key", "")
	boardType := req.GetString("board_type", "")
	startAt := req.GetInt("start_at", 0)
	limit := req.GetInt("limit", 10)
	agileClient, err := clients.GetAgileClient()
	if err != nil {
		return mcp.NewToolResultError("Jira client error: " + err.Error()), nil
	}
	opts := &models.GetBoardsOptions{
		BoardName:      boardName,
		BoardType:      boardType,
		ProjectKeyOrID: projectKey,
	}
	boards, resp, err := agileClient.Board.Gets(ctx, opts, startAt, limit)
	if err != nil || resp == nil || resp.StatusCode != 200 {
		return mcp.NewToolResultError("Failed to get agile boards: " + err.Error()), nil
	}
	jsonBytes, _ := json.Marshal(boards)
	return mcp.NewToolResultText(string(jsonBytes)), nil
}

func jiraGetBoardIssuesHandler(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	boardID, err := req.RequireInt("board_id")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	jql := req.GetString("jql", "")
	fields := req.GetString("fields", "")
	startAt := req.GetInt("start_at", 0)
	limit := req.GetInt("limit", 10)
	expand := req.GetString("expand", "version")
	jiraClient, err := clients.GetJiraClient()
	if err != nil {
		return mcp.NewToolResultError("Jira client error: " + err.Error()), nil
	}
	agileClient, err := agile.New(jiraClient.HTTP, jiraClient.Site.Host)
	if err != nil {
		return mcp.NewToolResultError("Failed to create agile client: " + err.Error()), nil
	}
	agileClient.Auth = jiraClient.Auth
	var fieldSlice, expandSlice []string
	if fields != "" {
		fieldSlice = utils.SplitAndTrim(fields)
	}
	if expand != "" {
		expandSlice = utils.SplitAndTrim(expand)
	}
	issues, resp, err := agileClient.Board.Issues(ctx, boardID, &models.IssueOptionScheme{
		JQL:    jql,
		Fields: fieldSlice,
		Expand: expandSlice,
	}, startAt, limit)
	if err != nil || resp == nil || resp.StatusCode != 200 {
		return mcp.NewToolResultError("Failed to get board issues: " + err.Error()), nil
	}
	jsonBytes, _ := json.Marshal(issues)
	return mcp.NewToolResultText(string(jsonBytes)), nil
}

func jiraGetSprintsFromBoardHandler(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	boardID, err := req.RequireInt("board_id")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	state := req.GetString("state", "")
	startAt := req.GetInt("start_at", 0)
	limit := req.GetInt("limit", 10)
	jiraClient, err := clients.GetJiraClient()
	if err != nil {
		return mcp.NewToolResultError("Jira client error: " + err.Error()), nil
	}
	agileClient, err := agile.New(jiraClient.HTTP, jiraClient.Site.Host)
	if err != nil {
		return mcp.NewToolResultError("Failed to create agile client: " + err.Error()), nil
	}
	agileClient.Auth = jiraClient.Auth
	sprints, resp, err := agileClient.Board.Sprints(ctx, boardID, startAt, limit, []string{state})
	if err != nil || resp == nil || resp.StatusCode != 200 {
		return mcp.NewToolResultError("Failed to get sprints from board: " + err.Error()), nil
	}
	jsonBytes, _ := json.Marshal(sprints)
	return mcp.NewToolResultText(string(jsonBytes)), nil
}

func jiraGetSprintIssuesHandler(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	sprintID, err := req.RequireInt("sprint_id")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	fields := req.GetString("fields", "")
	startAt := req.GetInt("start_at", 0)
	limit := req.GetInt("limit", 10)
	agileClient, err := clients.GetAgileClient()
	if err != nil {
		return mcp.NewToolResultError("Jira client error: " + err.Error()), nil
	}
	var fieldSlice []string
	if fields != "" {
		fieldSlice = utils.SplitAndTrim(fields)
	}
	issues, resp, err := agileClient.Sprint.Issues(ctx, sprintID, &models.IssueOptionScheme{
		Fields: fieldSlice,
	}, startAt, limit)
	if err != nil || resp == nil || resp.StatusCode != 200 {
		return mcp.NewToolResultError("Failed to get sprint issues: " + err.Error()), nil
	}
	jsonBytes, _ := json.Marshal(issues)
	return mcp.NewToolResultText(string(jsonBytes)), nil
}

func jiraDownloadAttachmentsHandler(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// Not supported by go-atlassian v2 SDK (no direct download to disk utility)
	return mcp.NewToolResultError("jira_download_attachments is not supported in this Go implementation. Please use the web UI or Python version."), nil
}

func jiraGetLinkTypesHandler(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	client, err := clients.GetJiraClient()
	if err != nil {
		return mcp.NewToolResultError("Jira client error: " + err.Error()), nil
	}
	linkTypes, resp, err := client.Issue.Link.Type.Gets(ctx)
	if err != nil || resp == nil || resp.StatusCode != 200 {
		return mcp.NewToolResultError("Failed to get link types: " + err.Error()), nil
	}
	jsonBytes, _ := json.Marshal(linkTypes)
	return mcp.NewToolResultText(string(jsonBytes)), nil
}

func jiraCreateIssueHandler(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	projectKey := req.GetString("project_key", "")
	summary := req.GetString("summary", "")
	issueType := req.GetString("issue_type", "")
	assignee := req.GetString("assignee", "")
	description := req.GetString("description", "")
	components := req.GetString("components", "")
	additionalFields := req.GetString("additional_fields", "")
	if projectKey == "" || summary == "" || issueType == "" {
		return mcp.NewToolResultError("Missing required parameters: project_key, summary, issue_type are required"), nil
	}
	client, err := clients.GetJiraClient()
	if err != nil {
		return mcp.NewToolResultError("Jira client error: " + err.Error()), nil
	}
	payload := &models.IssueSchemeV2{
		Fields: &models.IssueFieldsSchemeV2{},
	}
	payload.Fields.Project = &models.ProjectScheme{Key: projectKey}
	payload.Fields.Summary = summary
	payload.Fields.IssueType = &models.IssueTypeScheme{Name: issueType}
	payload.Fields.Description = description
	payload.Fields.Assignee = &models.UserScheme{Name: assignee}
	if components != "" {
		comps := utils.SplitAndTrim(components)
		var compObjs []*models.ComponentScheme
		for _, c := range comps {
			compObjs = append(compObjs, &models.ComponentScheme{Name: c})
		}
		payload.Fields.Components = compObjs
	}
	var cfields *models.CustomFields
	if additionalFields != "" {
		var add map[string]any
		if err := json.Unmarshal([]byte(additionalFields), &add); err != nil {
			return mcp.NewToolResultError("Failed to parse additional_fields: " + err.Error()), nil

		}
		if err := json.Unmarshal([]byte(additionalFields), &payload.Fields); err != nil {
			return mcp.NewToolResultError("Failed to parse additional_fields into issue fields: " + err.Error()), nil
		}
		cfields = &models.CustomFields{Fields: []map[string]any{add}}
	}

	issue, resp, err := client.Issue.Create(ctx, payload, cfields)
	if err != nil || resp == nil || (resp.StatusCode != 201 && resp.StatusCode != 200) {
		return mcp.NewToolResultError("Failed to create issue: " + err.Error()), nil
	}
	jsonBytes, _ := json.Marshal(issue)
	return mcp.NewToolResultText(string(jsonBytes)), nil
}

func jiraBatchCreateIssuesHandler(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// Not supported by go-atlassian v2 SDK (no batch create endpoint)
	return mcp.NewToolResultError("jira_batch_create_issues is not supported in this Go implementation. Please use the Python version or create issues one at a time."), nil
}

func jiraBatchGetChangelogsHandler(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// Not supported by go-atlassian v2 SDK (Cloud only feature)
	return mcp.NewToolResultError("jira_batch_get_changelogs is not supported by go-atlassian SDK (Cloud only feature)"), nil
}

func jiraLinkToEpicHandler(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	issueKey := req.GetString("issue_key", "")
	epicKey := req.GetString("epic_key", "")
	if issueKey == "" || epicKey == "" {
		return mcp.NewToolResultError("Missing required parameters: issue_key and epic_key are required"), nil
	}
	client, err := clients.GetAgileClient()
	if err != nil {
		return mcp.NewToolResultError("Jira client error: " + err.Error()), nil
	}
	resp, err := client.Epic.Move(ctx, epicKey, []string{issueKey})
	if err != nil || resp == nil || resp.StatusCode != 204 {
		return mcp.NewToolResultError("Failed to link to epic: " + err.Error()), nil
	}
	return mcp.NewToolResultText("Issue linked to epic successfully"), nil
}

func jiraCreateIssueLinkHandler(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	linkType := req.GetString("link_type", "")
	inwardKey := req.GetString("inward_issue_key", "")
	outwardKey := req.GetString("outward_issue_key", "")
	comment := req.GetString("comment", "")
	commentVisibility := req.GetString("comment_visibility", "")
	if linkType == "" || inwardKey == "" || outwardKey == "" {
		return mcp.NewToolResultError("Missing required parameters: link_type, inward_issue_key, outward_issue_key are required"), nil
	}
	client, err := clients.GetJiraClient()
	if err != nil {
		return mcp.NewToolResultError("Jira client error: " + err.Error()), nil
	}
	payload := &models.LinkPayloadSchemeV2{
		Type:         &models.LinkTypeScheme{Name: linkType},
		InwardIssue:  &models.LinkedIssueScheme{Key: inwardKey},
		OutwardIssue: &models.LinkedIssueScheme{Key: outwardKey},
	}
	if comment != "" {
		payload.Comment = &models.CommentPayloadSchemeV2{Body: comment}
		if commentVisibility != "" {
			payload.Comment.Visibility = &models.CommentVisibilityScheme{Type: commentVisibility}
		}
	}
	resp, err := client.Issue.Link.Create(ctx, payload)
	if err != nil || resp == nil || (resp.StatusCode != 201 && resp.StatusCode != 200) {
		return mcp.NewToolResultError("Failed to create issue link: " + err.Error()), nil
	}
	return mcp.NewToolResultText("Issue link created successfully"), nil
}

func jiraRemoveIssueLinkHandler(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	linkID := req.GetString("link_id", "")
	if linkID == "" {
		return mcp.NewToolResultError("Missing required parameter: link_id"), nil
	}
	client, err := clients.GetJiraClient()
	if err != nil {
		return mcp.NewToolResultError("Jira client error: " + err.Error()), nil
	}
	resp, err := client.Issue.Link.Delete(ctx, linkID)
	if err != nil || resp == nil || resp.StatusCode != 204 {
		return mcp.NewToolResultError("Failed to remove issue link: " + err.Error()), nil
	}
	return mcp.NewToolResultText("Issue link removed successfully"), nil
}

func jiraTransitionIssueHandler(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	issueKey := req.GetString("issue_key", "")
	transitionID := req.GetString("transition_id", "")
	fieldsStr := req.GetString("fields", "")
	comment := req.GetString("comment", "")
	if issueKey == "" || transitionID == "" {
		return mcp.NewToolResultError("Missing required parameters: issue_key and transition_id are required"), nil
	}
	var fields map[string]any
	if fieldsStr != "" {
		if err := json.Unmarshal([]byte(fieldsStr), &fields); err != nil {
			return mcp.NewToolResultError("Invalid fields JSON: " + err.Error()), nil
		}
	}
	client, err := clients.GetJiraClient()
	if err != nil {
		return mcp.NewToolResultError("Jira client error: " + err.Error()), nil
	}
	payload := &models.IssueSchemeV2{
		Transitions: []*models.IssueTransitionScheme{
			{
				ID: transitionID,
			},
		},
		Fields: &models.IssueFieldsSchemeV2{},
	}

	if len(fields) > 0 {
		json.Unmarshal([]byte(fieldsStr), &payload.Fields)
	}
	cfields := &models.CustomFields{Fields: []map[string]any{fields}}
	if comment != "" {
		payload.Fields.Comment = &models.IssueCommentPageSchemeV2{
			Comments: []*models.IssueCommentSchemeV2{
				{
					Body: comment,
				},
			},
		}
	}
	resp, err := client.Issue.Update(ctx, issueKey, false, payload, cfields, nil)
	if err != nil || resp == nil || resp.StatusCode != 204 {
		return mcp.NewToolResultError("Failed to transition issue: " + err.Error()), nil
	}
	return mcp.NewToolResultText("Issue transitioned successfully"), nil
}

func jiraCreateSprintHandler(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	boardID, err := req.RequireInt("board_id")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	sprintName := req.GetString("sprint_name", "")
	startDate := req.GetString("start_date", "")
	endDate := req.GetString("end_date", "")
	goal := req.GetString("goal", "")
	if sprintName == "" {
		return mcp.NewToolResultError("Missing required parameter: sprint_name"), nil
	}
	agileClient, err := clients.GetAgileClient()
	if err != nil {
		return mcp.NewToolResultError("Jira agile client error: " + err.Error()), nil
	}
	payload := &models.SprintPayloadScheme{Name: sprintName, OriginBoardID: boardID}
	if startDate != "" {
		payload.StartDate = startDate
	}
	if endDate != "" {
		payload.EndDate = endDate
	}
	if goal != "" {
		payload.Goal = goal
	}
	result, resp, err := agileClient.Sprint.Create(ctx, payload)
	if err != nil || resp == nil || (resp.StatusCode != 201 && resp.StatusCode != 200) {
		return mcp.NewToolResultError("Failed to create sprint: " + err.Error()), nil
	}
	jsonBytes, _ := json.Marshal(result)
	return mcp.NewToolResultText(string(jsonBytes)), nil
}

func jiraUpdateSprintHandler(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	sprintID, err := req.RequireInt("sprint_id")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	sprintName := req.GetString("sprint_name", "")
	state := req.GetString("state", "")
	startDate := req.GetString("start_date", "")
	endDate := req.GetString("end_date", "")
	goal := req.GetString("goal", "")
	agileClient, err := clients.GetAgileClient()
	if err != nil {
		return mcp.NewToolResultError("Jira agile client error: " + err.Error()), nil
	}
	payload := &models.SprintPayloadScheme{}
	if sprintName != "" {
		payload.Name = sprintName
	}
	if state != "" {
		payload.State = state
	}
	if startDate != "" {
		payload.StartDate = startDate
	}
	if endDate != "" {
		payload.EndDate = endDate
	}
	if goal != "" {
		payload.Goal = goal
	}
	result, resp, err := agileClient.Sprint.Update(ctx, sprintID, payload)
	if err != nil || resp == nil || resp.StatusCode != 200 {
		return mcp.NewToolResultError("Failed to update sprint: " + err.Error()), nil
	}
	jsonBytes, _ := json.Marshal(result)
	return mcp.NewToolResultText(string(jsonBytes)), nil
}

// Handler for jira_add_comment
func jiraAddCommentHandler(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	issueKey := req.GetString("issue_key", "")
	comment := req.GetString("comment", "")
	if issueKey == "" || comment == "" {
		return mcp.NewToolResultError("Missing required parameters: issue_key and comment are required"), nil
	}
	client, err := clients.GetJiraClient()
	if err != nil {
		return mcp.NewToolResultError("Jira client error: " + err.Error()), nil
	}
	payload := &models.CommentPayloadSchemeV2{
		Body: comment,
	}
	result, resp, err := client.Issue.Comment.Add(ctx, issueKey, payload, nil)
	if err != nil || resp == nil || (resp.StatusCode != 201 && resp.StatusCode != 200) {
		return mcp.NewToolResultError("Failed to add comment: " + err.Error()), nil
	}
	jsonBytes, _ := json.Marshal(result)
	return mcp.NewToolResultText(string(jsonBytes)), nil
}

// Handler for jira_add_worklog
func jiraAddWorklogHandler(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	issueKey := req.GetString("issue_key", "")
	timeSpent := req.GetString("time_spent", "")
	comment := req.GetString("comment", "")
	started := req.GetString("started", "")
	originalEstimate := req.GetString("original_estimate", "")
	remainingEstimate := req.GetString("remaining_estimate", "")
	if issueKey == "" || timeSpent == "" {
		return mcp.NewToolResultError("Missing required parameters: issue_key and time_spent are required"), nil
	}
	client, err := clients.GetJiraClient()
	if err != nil {
		return mcp.NewToolResultError("Jira client error: " + err.Error()), nil
	}
	payload := &models.WorklogRichTextPayloadScheme{
		TimeSpent: timeSpent,
	}
	if comment != "" {
		payload.Comment = &models.CommentPayloadSchemeV2{Body: comment}
	}
	if started != "" {
		payload.Started = started
	}
	options := &models.WorklogOptionsScheme{}
	if originalEstimate != "" {
		options.AdjustEstimate = "new"
		options.NewEstimate = originalEstimate
	}
	if remainingEstimate != "" {
		options.AdjustEstimate = "manual"
		options.ReduceBy = remainingEstimate
	}
	result, resp, err := client.Issue.Worklog.Add(ctx, issueKey, payload, options)
	if err != nil || resp == nil || (resp.StatusCode != 201 && resp.StatusCode != 200) {
		return mcp.NewToolResultError("Failed to add worklog: " + err.Error()), nil
	}
	jsonBytes, _ := json.Marshal(result)
	return mcp.NewToolResultText(string(jsonBytes)), nil
}

func jiraUpdateIssueHandler(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	issueKey := req.GetString("issue_key", "")
	fieldsStr := req.GetString("fields", "")
	additionalFields := req.GetString("additional_fields", "")
	if issueKey == "" || fieldsStr == "" {
		return mcp.NewToolResultError("Missing required parameters: issue_key and fields are required"), nil
	}
	client, err := clients.GetJiraClient()
	if err != nil {
		return mcp.NewToolResultError("Jira client error: " + err.Error()), nil
	}
	var fields map[string]any
	if err := json.Unmarshal([]byte(fieldsStr), &fields); err != nil {
		return mcp.NewToolResultError("Invalid fields JSON: " + err.Error()), nil
	}
	// Merge additional_fields if provided
	if additionalFields != "" {
		var add map[string]any
		if err := json.Unmarshal([]byte(additionalFields), &add); err == nil {
			for k, v := range add {
				fields[k] = v
			}
		}
	}
	payload := &models.IssueSchemeV2{
		Fields: &models.IssueFieldsSchemeV2{},
	}
	fieldsBytes, _ := json.Marshal(fields)
	json.Unmarshal(fieldsBytes, &payload.Fields)
	cfields := &models.CustomFields{Fields: []map[string]any{fields}}
	resp, err := client.Issue.Update(ctx, issueKey, false, payload, cfields, nil)
	if err != nil || resp == nil || resp.StatusCode != 204 {
		return mcp.NewToolResultError("Failed to update issue: " + err.Error()), nil
	}
	return mcp.NewToolResultText("Issue updated successfully"), nil
}

func jiraDeleteIssueHandler(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	issueKey := req.GetString("issue_key", "")
	if issueKey == "" {
		return mcp.NewToolResultError("Missing required parameter: issue_key"), nil
	}
	client, err := clients.GetJiraClient()
	if err != nil {
		return mcp.NewToolResultError("Jira client error: " + err.Error()), nil
	}
	resp, err := client.Issue.Delete(ctx, issueKey, false)
	if err != nil || resp == nil || resp.StatusCode != 204 {
		return mcp.NewToolResultError("Failed to delete issue: " + err.Error()), nil
	}
	return mcp.NewToolResultText("Issue deleted successfully"), nil
}
