package gh

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/k1LoW/gh-triage/profile"
	"github.com/k1LoW/go-github-client/v71/factory"
)

func TestNewUsesEnterpriseEndpoints(t *testing.T) {
	t.Setenv("GH_HOST", "git.example.com")
	t.Setenv("GH_ENTERPRISE_TOKEN", "synthetic-test-token")
	transport := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if got := req.URL.String(); got != "https://git.example.com/api/graphql" {
			t.Errorf("GraphQL endpoint = %q", got)
		}
		return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`{"data":{"viewer":{"login":"test"}}}`))}, nil
	})
	client, err := factory.NewGithubClient(factory.HTTPClient(&http.Client{Transport: transport}))
	if err != nil {
		t.Fatal(err)
	}
	c := newClient(&profile.Profile{}, client, io.Discard, false)
	if got := c.client.BaseURL.String(); got != "https://git.example.com/api/v3/" {
		t.Fatalf("REST endpoint = %q", got)
	}
	var query struct{ Viewer struct{ Login string } }
	if err := c.v4Client.Query(context.Background(), &query, nil); err != nil {
		t.Fatal(err)
	}
	if query.Viewer.Login != "test" {
		t.Fatal("GraphQL response not decoded")
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestNewPreservesCustomGraphQLEndpoint(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/custom/graphql" {
			t.Errorf("unexpected GraphQL path %q", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"data":{"viewer":{"login":"test"}}}`)
	}))
	defer server.Close()
	t.Setenv("GH_HOST", "github.com")
	t.Setenv("GH_TOKEN", "synthetic-test-token")
	t.Setenv("GITHUB_GRAPHQL_URL", server.URL+"/custom/graphql")
	c, err := New(&profile.Profile{}, io.Discard, false)
	if err != nil {
		t.Fatal(err)
	}
	var query struct{ Viewer struct{ Login string } }
	if err := c.v4Client.Query(context.Background(), &query, nil); err != nil {
		t.Fatal(err)
	}
}
