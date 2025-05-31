package clients

type contextKey string

const (
	JiraPersonalTokenKey       contextKey = "JIRA_PERSONAL_TOKEN"
	ConfluencePersonalTokenKey contextKey = "CONFLUENCE_PERSONAL_TOKEN"
)
