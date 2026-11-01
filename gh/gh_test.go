package gh

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/google/go-github/v71/github"
	"github.com/k1LoW/gh-triage/profile"
)

// handlerTransport serves requests in process so that tests do not need to listen on a port.
type handlerTransport struct {
	handler http.Handler
}

func (h handlerTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	rec := httptest.NewRecorder()
	h.handler.ServeHTTP(rec, r)
	res := rec.Result()
	res.Request = r
	return res, nil
}

func TestTriageDependabotAlertsThread(t *testing.T) {
	const htmlURL = "https://github.example/octo/app/security/dependabot"

	tests := []struct {
		name         string
		alertsStatus int
		alertsHeader map[string]string
		alertsBody   string
		wantState    string
		wantListed   bool
		wantDone     bool
		wantErr      bool
	}{
		{
			name:         "open alerts",
			alertsStatus: http.StatusOK,
			alertsBody:   `[{"number":1,"state":"open"}]`,
			wantState:    "open",
			wantListed:   true,
		},
		{
			name:         "no open alerts",
			alertsStatus: http.StatusOK,
			alertsBody:   `[]`,
			wantState:    "closed",
			wantListed:   true,
			wantDone:     true,
		},
		{
			name:         "alerts disabled",
			alertsStatus: http.StatusForbidden,
			alertsBody:   `{"message":"Dependabot alerts are disabled for this repository."}`,
		},
		{
			name:         "repository not found",
			alertsStatus: http.StatusNotFound,
			alertsBody:   `{"message":"Not Found"}`,
		},
		{
			name:         "server error",
			alertsStatus: http.StatusInternalServerError,
			alertsBody:   `{"message":"Server Error"}`,
			wantErr:      true,
		},
		{
			name:         "rate limited",
			alertsStatus: http.StatusForbidden,
			alertsHeader: map[string]string{
				"X-RateLimit-Limit":     "5000",
				"X-RateLimit-Remaining": "0",
				"X-RateLimit-Reset":     strconv.FormatInt(time.Now().Add(time.Hour).Unix(), 10),
			},
			alertsBody: `{"message":"API rate limit exceeded"}`,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			done := false
			mux := http.NewServeMux()
			mux.HandleFunc("GET /notifications", func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Query().Get("page") != "1" {
					fmt.Fprint(w, `[]`)
					return
				}
				fmt.Fprint(w, `[{
					"id": "1",
					"repository": {
						"name": "app",
						"owner": {"login": "octo"},
						"html_url": "https://github.example/octo/app"
					},
					"subject": {
						"title": "Your repository has dependencies with security vulnerabilities",
						"type": "RepositoryDependabotAlertsThread",
						"url": null
					}
				}]`)
			})
			mux.HandleFunc("GET /user", func(w http.ResponseWriter, _ *http.Request) {
				fmt.Fprint(w, `{"login":"octo"}`)
			})
			mux.HandleFunc("GET /repos/octo/app/dependabot/alerts", func(w http.ResponseWriter, r *http.Request) {
				if got := r.URL.Query().Get("state"); got != "open" {
					t.Errorf("state query = %q, want open", got)
				}
				if got := r.URL.Query().Get("page"); got != "" {
					t.Errorf("page query = %q, want empty", got)
				}
				if got := r.URL.Query().Get("per_page"); got != "1" {
					t.Errorf("per_page query = %q, want 1", got)
				}
				for k, v := range tt.alertsHeader {
					w.Header().Set(k, v)
				}
				w.WriteHeader(tt.alertsStatus)
				fmt.Fprint(w, tt.alertsBody)
			})
			mux.HandleFunc("DELETE /notifications/threads/1", func(w http.ResponseWriter, _ *http.Request) {
				done = true
				w.WriteHeader(http.StatusNoContent)
			})

			baseURL, err := url.Parse("https://api.github.test/")
			if err != nil {
				t.Fatal(err)
			}
			client := github.NewClient(&http.Client{Transport: handlerTransport{mux}})
			client.BaseURL = baseURL

			listCond := "*"
			if tt.wantState != "" {
				listCond = fmt.Sprintf("is_dependabot_alert && state == %q && html_url == %q", tt.wantState, htmlURL)
			}

			var out bytes.Buffer
			c := &Client{
				config: &profile.Profile{
					Done: profile.Action{Max: 10, Conditions: []string{"closed"}},
					List: profile.Action{Max: 10, Conditions: []string{listCond}},
				},
				client: client,
				w:      &out,
			}

			err = c.Triage(context.Background())
			if (err != nil) != tt.wantErr {
				t.Fatalf("Triage() error = %v, wantErr %v", err, tt.wantErr)
			}

			if done != tt.wantDone {
				t.Errorf("done = %v, want %v", done, tt.wantDone)
			}
			if got := strings.Contains(out.String(), "octo/app"); got != tt.wantListed {
				t.Errorf("listed = %v, want %v: %q", got, tt.wantListed, out.String())
			}
			if strings.Contains(out.String(), "#") {
				t.Errorf("listed a number for a repository-wide thread: %q", out.String())
			}
		})
	}
}
