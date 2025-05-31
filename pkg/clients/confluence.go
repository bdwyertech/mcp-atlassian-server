package clients

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/ctreminiom/go-atlassian/v2/confluence"
)

// ConfluenceRoundTripper modifies requests for on-prem compatibility
type ConfluenceRoundTripper struct {
	rt http.RoundTripper
}

func (w *ConfluenceRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	req.URL.Path = strings.TrimPrefix(req.URL.Path, "/wiki")
	// fmt.Printf("Request: %s %s\n", req.Method, req.URL)
	return w.rt.RoundTrip(req)
}

// GetConfluenceClient returns a new Confluence client using environment variables.
func GetConfluenceClient(ctx context.Context) (*confluence.Client, error) {
	baseURL := os.Getenv("CONFLUENCE_URL")
	apiToken := os.Getenv("CONFLUENCE_PERSONAL_TOKEN")
	if token := ctx.Value(ConfluencePersonalTokenKey); token != nil {
		apiToken = token.(string)
	}
	if baseURL == "" || apiToken == "" {
		return nil, fmt.Errorf("missing Confluence credentials in environment variables")
	}
	c := &http.Client{
		Timeout:   http.DefaultClient.Timeout,
		Transport: &ConfluenceRoundTripper{http.DefaultTransport},
	}
	api, err := confluence.New(c, baseURL)
	if err != nil {
		return nil, err
	}
	api.Auth.SetBearerToken(apiToken)
	return api, nil
}
