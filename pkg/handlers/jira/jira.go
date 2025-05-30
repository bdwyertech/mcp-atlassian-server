package jira

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"strings"

	"github.com/ctreminiom/go-atlassian/v2/jira/agile"
	"github.com/ctreminiom/go-atlassian/v2/pkg/infra/models"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/sirupsen/logrus"

	"mcp-atlassian-server/pkg/clients"
	"mcp-atlassian-server/pkg/utils"
)

func PingHandler(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
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

// Handler for jira_get_user_profile
func GetUserProfileHandler(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
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
func GetIssueHandler(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
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
func SearchHandler(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
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
func SearchFieldsHandler(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
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
func GetProjectIssuesHandler(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	projectKey := req.GetString("project_key", "")
	limit := req.GetInt("limit", 10)
	startAt := req.GetInt("start_at", 0)
	client, err := clients.GetJiraClient()
	if err != nil {
		return mcp.NewToolResultError("Jira client error: " + err.Error()), nil
	}

	jql := "project = '" + projectKey + "'"
	_, resp, err := client.Issue.Search.Post(ctx, jql, nil, nil, startAt, limit, "")
	if err != nil || resp == nil || resp.StatusCode != 200 {
		return mcp.NewToolResultError("Failed to get project issues: " + err.Error()), nil
	}
	return mcp.NewToolResultText(resp.Bytes.String()), nil
}

// --- Jira Tool Handler Implementations ---

func GetTransitionsHandler(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
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

func GetWorklogHandler(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
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

func GetAgileBoardsHandler(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
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

func GetBoardIssuesHandler(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
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

func GetSprintsFromBoardHandler(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
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

func GetSprintIssuesHandler(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
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

func DownloadAttachmentsHandler(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// Not supported by go-atlassian v2 SDK (no direct download to disk utility)
	return mcp.NewToolResultError("jira_download_attachments is not supported in this Go implementation. Please use the web UI or Python version."), nil
}

func GetLinkTypesHandler(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
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

func CreateIssueHandler(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
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

func BatchCreateIssuesHandler(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// Not supported by go-atlassian v2 SDK (no batch create endpoint)
	return mcp.NewToolResultError("jira_batch_create_issues is not supported in this Go implementation. Please use the Python version or create issues one at a time."), nil
}

func BatchGetChangelogsHandler(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// Not supported by go-atlassian v2 SDK (Cloud only feature)
	return mcp.NewToolResultError("jira_batch_get_changelogs is not supported by go-atlassian SDK (Cloud only feature)"), nil
}

func LinkToEpicHandler(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
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

func CreateIssueLinkHandler(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
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

func RemoveIssueLinkHandler(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
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

func TransitionIssueHandler(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
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

func CreateSprintHandler(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
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

func UpdateSprintHandler(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
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
func AddCommentHandler(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
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
func AddWorklogHandler(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
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

	// Build the payload as Jira expects: comment as a string, not an object
	payload := map[string]interface{}{
		"timeSpent": timeSpent,
	}
	if comment != "" {
		payload["comment"] = comment
	}
	if started != "" {
		payload["started"] = started
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return mcp.NewToolResultError("Failed to marshal worklog payload: " + err.Error()), nil
	}

	// Build the URL
	url := fmt.Sprintf("%s://%s/rest/api/2/issue/%s/worklog", client.Site.Scheme, client.Site.Host, issueKey)

	// Add estimate adjustment if needed
	params := ""
	if originalEstimate != "" {
		params = fmt.Sprintf("?adjustEstimate=new&newEstimate=%s", originalEstimate)
	} else if remainingEstimate != "" {
		params = fmt.Sprintf("?adjustEstimate=manual&reduceBy=%s", remainingEstimate)
	}
	url += params

	reqHttp, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(payloadBytes))
	if err != nil {
		return mcp.NewToolResultError("Failed to create HTTP request: " + err.Error()), nil
	}
	reqHttp.Header.Set("Content-Type", "application/json")
	reqHttp.Header.Set("Authorization", "Bearer "+client.Auth.GetBearerToken())
	resp, err := client.HTTP.Do(reqHttp)
	if err != nil {
		return mcp.NewToolResultError("Failed to add worklog: " + err.Error()), nil
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 201 && resp.StatusCode != 200 {
		return mcp.NewToolResultError("Failed to add worklog: " + string(respBody)), nil
	}
	return mcp.NewToolResultText(string(respBody)), nil
}

func UpdateIssueHandler(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
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

func DeleteIssueHandler(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
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
