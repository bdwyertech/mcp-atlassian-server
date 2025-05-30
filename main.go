package main

import (
	"context"
	"fmt"

	"mcp-atlassian-server/pkg/handlers/confluence"
	"mcp-atlassian-server/pkg/handlers/jira"

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

	s.AddTool(mcp.NewTool("confluence_ping",
		mcp.WithDescription("Ping Confluence API"),
	), confluence.PingHandler)

	s.AddTool(mcp.NewTool("jira_ping",
		mcp.WithDescription("Ping Jira API"),
	), jira.PingHandler)

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
	), confluence.SearchHandler)

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
	), confluence.GetPageHandler)

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
	), confluence.GetPageChildrenHandler)

	s.AddTool(mcp.NewTool("confluence_get_comments",
		mcp.WithDescription("Get comments for a specific Confluence page."),
		mcp.WithString("page_id",
			mcp.Description("Confluence page ID (numeric ID, can be parsed from URL)"),
			mcp.Required(),
		),
	), confluence.GetCommentsHandler)

	s.AddTool(mcp.NewTool("confluence_get_labels",
		mcp.WithDescription("Get labels for a specific Confluence page."),
		mcp.WithString("page_id",
			mcp.Description("Confluence page ID (numeric ID, can be parsed from URL)"),
			mcp.Required(),
		),
	), confluence.GetLabelsHandler)

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
	), confluence.AddLabelHandler)

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
	), confluence.CreatePageHandler)

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
	), confluence.UpdatePageHandler)

	s.AddTool(mcp.NewTool("confluence_delete_page",
		mcp.WithDescription("Delete an existing Confluence page."),
		mcp.WithString("page_id",
			mcp.Description("The ID of the page to delete"),
			mcp.Required(),
		),
	), confluence.DeletePageHandler)

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
	), confluence.AddCommentHandler)

	s.AddTool(mcp.NewTool("jira_get_user_profile",
		mcp.WithDescription("Retrieve profile information for a specific Jira user."),
		mcp.WithString("user_identifier",
			mcp.Description("Identifier for the user (e.g., email address, username, account ID, or key for Server/DC)."),
			mcp.Required(),
		),
	), jira.GetUserProfileHandler)

	s.AddTool(mcp.NewTool("jira_get_issue",
		mcp.WithDescription("Get details of a specific Jira issue."),
		mcp.WithString("issue_key", mcp.Description("Jira issue key (e.g., 'PROJ-123')"), mcp.Required()),
		mcp.WithString("fields", mcp.Description("Comma-separated list of fields to return (e.g., 'summary,status'). Use '*all' for all fields."), mcp.DefaultString("")),
		mcp.WithString("expand", mcp.Description("Fields to expand (e.g., 'renderedFields', 'transitions', 'changelog')"), mcp.DefaultString("")),
		mcp.WithNumber("comment_limit", mcp.Description("Maximum number of comments to include (0 for none)"), mcp.DefaultNumber(10)),
		mcp.WithString("properties", mcp.Description("Comma-separated list of issue properties to return"), mcp.DefaultString("")),
		mcp.WithBoolean("update_history", mcp.Description("Whether to update the issue view history for the requesting user"), mcp.DefaultBool(true)),
	), jira.GetIssueHandler)

	s.AddTool(mcp.NewTool("jira_search",
		mcp.WithDescription("Search Jira issues using JQL (Jira Query Language)."),
		mcp.WithString("jql", mcp.Description("JQL query string (e.g., 'project = PROJ AND status = \"In Progress\"')"), mcp.Required()),
		mcp.WithString("fields", mcp.Description("Comma-separated fields to return in the results. Use '*all' for all fields."), mcp.DefaultString("")),
		mcp.WithNumber("limit", mcp.Description("Maximum number of results (1-50)"), mcp.DefaultNumber(10)),
		mcp.WithNumber("start_at", mcp.Description("Starting index for pagination (0-based)"), mcp.DefaultNumber(0)),
		mcp.WithString("projects_filter", mcp.Description("Comma-separated list of project keys to filter results by."), mcp.DefaultString("")),
		mcp.WithString("expand", mcp.Description("Fields to expand (e.g., 'renderedFields', 'transitions', 'changelog')"), mcp.DefaultString("")),
	), jira.SearchHandler)

	s.AddTool(mcp.NewTool("jira_search_fields",
		mcp.WithDescription("Search Jira fields by keyword with fuzzy match."),
		mcp.WithString("keyword", mcp.Description("Keyword for fuzzy search. If left empty, lists the first 'limit' available fields in their default order."), mcp.DefaultString("")),
		mcp.WithNumber("limit", mcp.Description("Maximum number of results"), mcp.DefaultNumber(10)),
		mcp.WithBoolean("refresh", mcp.Description("Whether to force refresh the field list"), mcp.DefaultBool(false)),
	), jira.SearchFieldsHandler)

	s.AddTool(mcp.NewTool("jira_get_project_issues",
		mcp.WithDescription("Get all issues for a specific Jira project."),
		mcp.WithString("project_key", mcp.Description("The project key"), mcp.Required()),
		mcp.WithNumber("limit", mcp.Description("Maximum number of results (1-50)"), mcp.DefaultNumber(10)),
		mcp.WithNumber("start_at", mcp.Description("Starting index for pagination (0-based)"), mcp.DefaultNumber(0)),
	), jira.GetProjectIssuesHandler)

	s.AddTool(mcp.NewTool("jira_get_transitions",
		mcp.WithDescription("Get available status transitions for a Jira issue."),
		mcp.WithString("issue_key", mcp.Description("Jira issue key (e.g., 'PROJ-123')"), mcp.Required()),
	), jira.GetTransitionsHandler)

	s.AddTool(mcp.NewTool("jira_get_worklog",
		mcp.WithDescription("Get worklog entries for a Jira issue."),
		mcp.WithString("issue_key", mcp.Description("Jira issue key (e.g., 'PROJ-123')"), mcp.Required()),
	), jira.GetWorklogHandler)

	s.AddTool(mcp.NewTool("jira_get_agile_boards",
		mcp.WithDescription("Get Jira agile boards by name, project key, or type."),
		mcp.WithString("board_name", mcp.Description("(Optional) The name of board, support fuzzy search"), mcp.DefaultString("")),
		mcp.WithString("project_key", mcp.Description("(Optional) Jira project key (e.g., 'PROJ-123')"), mcp.DefaultString("")),
		mcp.WithString("board_type", mcp.Description("(Optional) The type of jira board (e.g., 'scrum', 'kanban')"), mcp.DefaultString("")),
		mcp.WithNumber("start_at", mcp.Description("Starting index for pagination (0-based)"), mcp.DefaultNumber(0)),
		mcp.WithNumber("limit", mcp.Description("Maximum number of results (1-50)"), mcp.DefaultNumber(10)),
	), jira.GetAgileBoardsHandler)

	s.AddTool(mcp.NewTool("jira_get_board_issues",
		mcp.WithDescription("Get all issues linked to a specific board filtered by JQL."),
		mcp.WithNumber("board_id", mcp.Description("The id of the board (e.g., '1001')"), mcp.Required()),
		mcp.WithString("jql", mcp.Description("JQL query string to filter issues."), mcp.Required()),
		mcp.WithString("fields", mcp.Description("Comma-separated fields to return in the results. Use '*all' for all fields."), mcp.DefaultString("")),
		mcp.WithNumber("start_at", mcp.Description("Starting index for pagination (0-based)"), mcp.DefaultNumber(0)),
		mcp.WithNumber("limit", mcp.Description("Maximum number of results (1-50)"), mcp.DefaultNumber(10)),
		mcp.WithString("expand", mcp.Description("Optional fields to expand in the response (e.g., 'changelog')."), mcp.DefaultString("version")),
	), jira.GetBoardIssuesHandler)

	s.AddTool(mcp.NewTool("jira_get_sprints_from_board",
		mcp.WithDescription("Get Jira sprints from board by state."),
		mcp.WithNumber("board_id", mcp.Description("The id of board (e.g., '1000')"), mcp.Required()),
		mcp.WithString("state", mcp.Description("Sprint state (e.g., 'active', 'future', 'closed')"), mcp.DefaultString("")),
		mcp.WithNumber("start_at", mcp.Description("Starting index for pagination (0-based)"), mcp.DefaultNumber(0)),
		mcp.WithNumber("limit", mcp.Description("Maximum number of results (1-50)"), mcp.DefaultNumber(10)),
	), jira.GetSprintsFromBoardHandler)

	s.AddTool(mcp.NewTool("jira_get_sprint_issues",
		mcp.WithDescription("Get Jira issues from sprint."),
		mcp.WithNumber("sprint_id", mcp.Description("The id of sprint (e.g., '10001')"), mcp.Required()),
		mcp.WithString("fields", mcp.Description("Comma-separated fields to return in the results. Use '*all' for all fields."), mcp.DefaultString("")),
		mcp.WithNumber("start_at", mcp.Description("Starting index for pagination (0-based)"), mcp.DefaultNumber(0)),
		mcp.WithNumber("limit", mcp.Description("Maximum number of results (1-50)"), mcp.DefaultNumber(10)),
	), jira.GetSprintIssuesHandler)

	s.AddTool(mcp.NewTool("jira_download_attachments",
		mcp.WithDescription("Download attachments from a Jira issue."),
		mcp.WithString("issue_key", mcp.Description("Jira issue key"), mcp.Required()),
		mcp.WithString("target_dir", mcp.Description("Directory to save attachments"), mcp.Required()),
	), jira.DownloadAttachmentsHandler)

	s.AddTool(mcp.NewTool("jira_get_link_types",
		mcp.WithDescription("Get all available issue link types."),
	), jira.GetLinkTypesHandler)

	s.AddTool(mcp.NewTool("jira_create_issue",
		mcp.WithDescription("Create a new Jira issue with optional Epic link or parent for subtasks."),
		mcp.WithString("project_key", mcp.Description("The JIRA project key"), mcp.Required()),
		mcp.WithString("summary", mcp.Description("Summary/title of the issue"), mcp.Required()),
		mcp.WithString("issue_type", mcp.Description("Issue type (e.g., 'Task', 'Bug', 'Story', 'Epic', 'Subtask')"), mcp.Required()),
		mcp.WithString("assignee", mcp.Description("Assignee's user identifier (email, display name, or account ID)"), mcp.DefaultString("")),
		mcp.WithString("description", mcp.Description("Issue description"), mcp.DefaultString("")),
		mcp.WithString("components", mcp.Description("Comma-separated list of component names"), mcp.DefaultString("")),
		mcp.WithString("additional_fields", mcp.Description("JSON string of additional fields"), mcp.DefaultString("")),
	), jira.CreateIssueHandler)

	s.AddTool(mcp.NewTool("jira_batch_create_issues",
		mcp.WithDescription("Create multiple Jira issues in a batch."),
		mcp.WithString("issues", mcp.Description("JSON array string of issue objects"), mcp.Required()),
		mcp.WithBoolean("validate_only", mcp.Description("If true, only validates without creating"), mcp.DefaultBool(false)),
	), jira.BatchCreateIssuesHandler)

	s.AddTool(mcp.NewTool("jira_batch_get_changelogs",
		mcp.WithDescription("Get changelogs for multiple Jira issues (Cloud only)."),
		mcp.WithString("issue_ids_or_keys", mcp.Description("Comma-separated list of issue IDs or keys"), mcp.Required()),
		mcp.WithString("fields", mcp.Description("Comma-separated list of fields to filter changelogs by. None for all fields."), mcp.DefaultString("")),
		mcp.WithNumber("limit", mcp.Description("Maximum changelogs per issue (-1 for all)"), mcp.DefaultNumber(-1)),
	), jira.BatchGetChangelogsHandler)

	s.AddTool(mcp.NewTool("jira_update_issue",
		mcp.WithDescription("Update an existing Jira issue including changing status, adding Epic links, updating fields, etc."),
		mcp.WithString("issue_key", mcp.Description("Jira issue key"), mcp.Required()),
		mcp.WithString("fields", mcp.Description("JSON string of fields to update"), mcp.Required()),
		mcp.WithString("additional_fields", mcp.Description("Optional JSON string of additional fields"), mcp.DefaultString("")),
		mcp.WithString("attachments", mcp.Description("Optional JSON array string or comma-separated list of file paths"), mcp.DefaultString("")),
	), jira.UpdateIssueHandler)

	s.AddTool(mcp.NewTool("jira_delete_issue",
		mcp.WithDescription("Delete an existing Jira issue."),
		mcp.WithString("issue_key", mcp.Description("Jira issue key"), mcp.Required()),
	), jira.DeleteIssueHandler)

	s.AddTool(mcp.NewTool("jira_add_comment",
		mcp.WithDescription("Add a comment to a Jira issue."),
		mcp.WithString("issue_key", mcp.Description("Jira issue key"), mcp.Required()),
		mcp.WithString("comment", mcp.Description("Comment text in Markdown"), mcp.Required()),
	), jira.AddCommentHandler)

	s.AddTool(mcp.NewTool("jira_add_worklog",
		mcp.WithDescription("Add a worklog entry to a Jira issue."),
		mcp.WithString("issue_key", mcp.Description("Jira issue key"), mcp.Required()),
		mcp.WithString("time_spent", mcp.Description("Time spent in Jira format (e.g., '2h 30m')"), mcp.Required()),
		mcp.WithString("comment", mcp.Description("Optional comment in Markdown"), mcp.DefaultString("")),
		mcp.WithString("started", mcp.Description("Optional start time in ISO format"), mcp.DefaultString("")),
		mcp.WithString("original_estimate", mcp.Description("Optional new original estimate"), mcp.DefaultString("")),
		mcp.WithString("remaining_estimate", mcp.Description("Optional new remaining estimate"), mcp.DefaultString("")),
	), jira.AddWorklogHandler)

	s.AddTool(mcp.NewTool("jira_link_to_epic",
		mcp.WithDescription("Link an existing issue to an epic."),
		mcp.WithString("issue_key", mcp.Description("The key of the issue to link"), mcp.Required()),
		mcp.WithString("epic_key", mcp.Description("The key of the epic to link to"), mcp.Required()),
	), jira.LinkToEpicHandler)

	s.AddTool(mcp.NewTool("jira_create_issue_link",
		mcp.WithDescription("Create a link between two Jira issues."),
		mcp.WithString("link_type", mcp.Description("The type of link (e.g., 'Blocks')"), mcp.Required()),
		mcp.WithString("inward_issue_key", mcp.Description("The key of the source issue"), mcp.Required()),
		mcp.WithString("outward_issue_key", mcp.Description("The key of the target issue"), mcp.Required()),
		mcp.WithString("comment", mcp.Description("Optional comment text"), mcp.DefaultString("")),
		mcp.WithString("comment_visibility", mcp.Description("Optional JSON string for comment visibility"), mcp.DefaultString("")),
	), jira.CreateIssueLinkHandler)

	s.AddTool(mcp.NewTool("jira_remove_issue_link",
		mcp.WithDescription("Remove a link between two Jira issues."),
		mcp.WithString("link_id", mcp.Description("The ID of the link to remove"), mcp.Required()),
	), jira.RemoveIssueLinkHandler)

	s.AddTool(mcp.NewTool("jira_transition_issue",
		mcp.WithDescription("Transition a Jira issue to a new status."),
		mcp.WithString("issue_key", mcp.Description("Jira issue key"), mcp.Required()),
		mcp.WithString("transition_id", mcp.Description("ID of the transition"), mcp.Required()),
		mcp.WithString("fields", mcp.Description("Optional JSON string of fields to update during transition"), mcp.DefaultString("")),
		mcp.WithString("comment", mcp.Description("Optional comment for the transition"), mcp.DefaultString("")),
	), jira.TransitionIssueHandler)

	s.AddTool(mcp.NewTool("jira_create_sprint",
		mcp.WithDescription("Create Jira sprint for a board."),
		mcp.WithNumber("board_id", mcp.Description("Board ID"), mcp.Required()),
		mcp.WithString("sprint_name", mcp.Description("Sprint name"), mcp.Required()),
		mcp.WithString("start_date", mcp.Description("Start date (ISO format)"), mcp.DefaultString("")),
		mcp.WithString("end_date", mcp.Description("End date (ISO format)"), mcp.DefaultString("")),
		mcp.WithString("goal", mcp.Description("Optional sprint goal"), mcp.DefaultString("")),
	), jira.CreateSprintHandler)

	s.AddTool(mcp.NewTool("jira_update_sprint",
		mcp.WithDescription("Update jira sprint."),
		mcp.WithNumber("sprint_id", mcp.Description("The ID of the sprint"), mcp.Required()),
		mcp.WithString("sprint_name", mcp.Description("Optional new name"), mcp.DefaultString("")),
		mcp.WithString("state", mcp.Description("Optional new state (future|active|closed)"), mcp.DefaultString("")),
		mcp.WithString("start_date", mcp.Description("Optional new start date"), mcp.DefaultString("")),
		mcp.WithString("end_date", mcp.Description("Optional new end date"), mcp.DefaultString("")),
		mcp.WithString("goal", mcp.Description("Optional new goal"), mcp.DefaultString("")),
	), jira.UpdateSprintHandler)

	if err := server.ServeStdio(s); err != nil {
		fmt.Printf("Server error: %v\n", err)
	}
}
