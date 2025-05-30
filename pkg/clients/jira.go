package clients

import (
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/ctreminiom/go-atlassian/v2/jira/agile"
	jira "github.com/ctreminiom/go-atlassian/v2/jira/v2"
)

// JiraRoundTripper modifies requests for on-prem compatibility
type JiraRoundTripper struct {
	rt http.RoundTripper
}

func (w *JiraRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	// Swap accountId query param to username
	if strings.HasPrefix(req.URL.Path, "/rest/api/2/user") {
		q := req.URL.Query()
		if accountId := q.Get("accountId"); accountId != "" {
			q.Del("accountId")
			q.Set("username", accountId)
			req.URL.RawQuery = q.Encode()
		}
	}
	// fmt.Printf("Request: %s %s\n", req.Method, req.URL)
	return w.rt.RoundTrip(req)
}

// GetJiraClient returns a new Jira client using environment variables.
func GetJiraClient() (*jira.Client, error) {
	baseURL := os.Getenv("JIRA_URL")
	apiToken := os.Getenv("JIRA_PERSONAL_TOKEN")
	if baseURL == "" || apiToken == "" {
		return nil, fmt.Errorf("missing Jira credentials in environment variables")
	}
	c := &http.Client{
		Timeout:   http.DefaultClient.Timeout,
		Transport: &JiraRoundTripper{http.DefaultTransport},
	}
	api, err := jira.New(c, baseURL)
	if err != nil {
		return nil, err
	}
	api.Auth.SetBearerToken(apiToken)
	return api, nil
}

// GetAgileClient returns a new Jira client using environment variables.
func GetAgileClient() (*agile.Client, error) {
	baseURL := os.Getenv("JIRA_URL")
	apiToken := os.Getenv("JIRA_PERSONAL_TOKEN")
	if baseURL == "" || apiToken == "" {
		return nil, fmt.Errorf("missing Jira credentials in environment variables")
	}
	api, err := agile.New(nil, baseURL)
	if err != nil {
		return nil, err
	}
	api.Auth.SetBearerToken(apiToken)
	return api, nil
}
