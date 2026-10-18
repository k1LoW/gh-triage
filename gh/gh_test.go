package gh

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/google/go-github/v71/github"
	"github.com/k1LoW/gh-triage/profile"
)

func TestTriageAuthenticatedUser(t *testing.T) {
	for _, tt := range []struct {
		name      string
		empty     bool
		failFirst bool
		disabled  bool
	}{
		{name: "multiple pages"},
		{name: "empty inbox", empty: true},
		{name: "no enabled actions", disabled: true},
		{name: "retry after lookup error", failFirst: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			var userRequests, issueRequests atomic.Int64
			var output bytes.Buffer
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				switch {
				case r.URL.Path == "/user":
					count := userRequests.Add(1)
					if tt.failFirst && count == 1 {
						w.WriteHeader(http.StatusBadGateway)
						return
					}
					// A second lookup during a run must not be needed.
					if !tt.failFirst && count > 1 {
						w.WriteHeader(http.StatusBadGateway)
						return
					}
					_, _ = fmt.Fprint(w, `{"login":"reviewer"}`)
				case r.URL.Path == "/notifications":
					if tt.empty || r.URL.Query().Get("page") == "3" {
						_, _ = fmt.Fprint(w, `[]`)
						return
					}
					var notifications []map[string]any
					for i := range 2 {
						notifications = append(notifications, map[string]any{
							"subject":    map[string]string{"type": "Issue", "title": "notification", "url": fmt.Sprintf("https://api.github.com/repos/example/project/issues/%d", i+1)},
							"repository": map[string]any{"name": "project", "owner": map[string]string{"login": "example"}},
						})
					}
					_ = json.NewEncoder(w).Encode(notifications)
				case strings.HasPrefix(r.URL.Path, "/repos/example/project/issues/"):
					issueRequests.Add(1)
					_, _ = fmt.Fprint(w, `{"number":1,"state":"open","html_url":"https://github.com/example/project/issues/1","user":{"login":"reviewer"}}`)
				default:
					t.Errorf("unexpected request: %s %s", r.Method, r.URL)
					w.WriteHeader(http.StatusNotFound)
				}
			}))
			t.Cleanup(server.Close)
			client := github.NewClient(server.Client())
			client.BaseURL, _ = url.Parse(server.URL + "/")
			c := &Client{
				client: client,
				config: &profile.Profile{List: profile.Action{Max: 100, Conditions: []string{"author == me"}}},
				w:      &output,
			}
			if tt.disabled {
				c.config.List.Max = 0
			}
			if tt.failFirst {
				if err := c.Triage(t.Context()); err == nil || !strings.Contains(err.Error(), "failed to get authenticated user") {
					t.Fatalf("lookup error = %v", err)
				}
				if issueRequests.Load() != 0 {
					t.Fatal("processed notifications after failed user lookup")
				}
			}
			if err := c.Triage(t.Context()); err != nil {
				t.Fatal(err)
			}
			wantUsers, wantIssues := int64(1), int64(4)
			if tt.empty || tt.disabled {
				wantUsers, wantIssues = 0, 0
			} else if tt.failFirst {
				wantUsers = 2
			}
			if got := userRequests.Load(); got != wantUsers {
				t.Errorf("user requests = %d, want %d", got, wantUsers)
			}
			if got := issueRequests.Load(); got != wantIssues {
				t.Errorf("issue requests = %d, want %d", got, wantIssues)
			}
			if got := strings.Count(output.String(), "notification"); got != int(wantIssues) {
				t.Errorf("listed notifications = %d, want %d", got, wantIssues)
			}
		})
	}
}
