package confluence

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"

	md "github.com/JohannesKaufmann/html-to-markdown"
	"github.com/ctreminiom/go-atlassian/v2/pkg/infra/models"
	"github.com/mark3labs/mcp-go/mcp"

	"mcp-atlassian-server/pkg/clients"
)

func PingHandler(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
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

// Handler for confluence_search
func SearchHandler(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
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
func GetPageHandler(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
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
func GetPageChildrenHandler(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
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
func GetCommentsHandler(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
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
func GetLabelsHandler(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
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
func AddLabelHandler(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
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
func CreatePageHandler(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
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
func UpdatePageHandler(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
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
func DeletePageHandler(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
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
func AddCommentHandler(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	pageID := req.GetString("page_id", "")
	content := req.GetString("content", "")
	client, err := clients.GetConfluenceClient()
	if err != nil {
		return mcp.NewToolResultError("Confluence client error: " + err.Error()), nil
	}
	commentPayload := &ConfluenceComment{
		Type: "comment",
		Container: &Container{
			Type: "page",
			ID:   pageID,
			Status: "current",
		},
		Body: &models.BodyScheme{
			Storage: &models.BodyNodeScheme{
				Value:          content,
				Representation: "storage",
			},
		},
	}
	payloadBytes, err := json.Marshal(commentPayload)
	if err != nil {
		return mcp.NewToolResultError("Failed to marshal comment payload: " + err.Error()), nil
	}

	// Build the URL
	url := fmt.Sprintf("%s://%s/rest/api/content", client.Site.Scheme, client.Site.Host)
	reqHttp, err := client.NewRequest(ctx, "POST", url, "application/json", bytes.NewReader(payloadBytes))
	if err != nil {
		return mcp.NewToolResultError("Failed to create HTTP request: " + err.Error()), nil
	}
	var structure any
	resp, err := client.Call(reqHttp, &structure)
	if err != nil {
		return mcp.NewToolResultError("Failed to add comment: " + err.Error() + resp.Bytes.String()), nil
	}
	if resp.StatusCode != 201 && resp.StatusCode != 200 {
		return mcp.NewToolResultError("Failed to add comment: " + string(resp.Bytes.String())), nil
	}
	return mcp.NewToolResultText(string(resp.Bytes.String())), nil
}

type ConfluenceComment struct {
	Type      string             `json:"type"` // should be "comment"
	Container *Container         `json:"container"`
	Body      *models.BodyScheme `json:"body"`
}

type Container struct {
	Type   string `json:"type"`             // should be "page"
	ID     string `json:"id"`               // page or blog post ID
	Status string `json:"status,omitempty"` // optional, e.g., "current"
}
