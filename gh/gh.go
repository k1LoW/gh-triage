package gh

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/url"
	"path"
	"slices"
	"strconv"
	"strings"
	"sync/atomic"

	"github.com/expr-lang/expr"
	"github.com/fatih/color"
	"github.com/google/go-github/v71/github"
	"github.com/k1LoW/gh-triage/config"
	"github.com/k1LoW/go-github-client/v71/factory"
	"github.com/pkg/browser"
	"github.com/samber/lo"
	"github.com/savioxavier/termlink"
	"golang.org/x/sync/errgroup"
)

type Client struct {
	config    *config.Config
	client    *github.Client
	w         io.Writer
	readLimit atomic.Int64 // Limit the number of issues/pull requests to read
	openLimit atomic.Int64 // Limit the number of issues/pull requests to open
	listLimit atomic.Int64 // Limit the number of issues/pull requests to list
}

var (
	titleC  = color.New(color.FgWhite, color.Bold)
	numberC = color.RGB(64, 64, 64)
	openC   = color.RGB(31, 136, 61)
	mergedC = color.RGB(130, 80, 223)
	closedC = color.RGB(207, 34, 46)
)

func New(cfg *config.Config, w io.Writer) (*Client, error) {
	client, err := factory.NewGithubClient()
	if err != nil {
		return nil, err
	}
	return &Client{
		config: cfg,
		client: client,
		w:      w,
	}, nil
}

func (c *Client) Triage(ctx context.Context) error {
	c.readLimit.Store(int64(c.config.Read.Max))
	c.openLimit.Store(int64(c.config.Open.Max))
	c.listLimit.Store(int64(c.config.List.Max))
	page := 1
	for {
		notifications, _, err := c.client.Activity.ListNotifications(ctx, &github.NotificationListOptions{
			ListOptions: github.ListOptions{
				Page:    page,
				PerPage: 100,
			},
		})
		if err != nil {
			return err
		}
		if len(notifications) == 0 {
			break
		}
		eg, ctx := errgroup.WithContext(ctx)
		for _, n := range notifications {
			eg.Go(func() error {
				return c.action(ctx, n)
			})
		}
		if err := eg.Wait(); err != nil {
			return fmt.Errorf("failed to process notifications: %w", err)
		}
		page++
	}

	return nil
}

func (c *Client) action(ctx context.Context, n *github.Notification) error {
	if c.readLimit.Load() <= 0 && c.openLimit.Load() <= 0 && c.listLimit.Load() <= 0 {
		return nil // No more actions to perform
	}
	m := map[string]any{}
	title := n.GetSubject().GetTitle()
	u, err := url.Parse(n.GetSubject().GetURL())
	if err != nil {
		return fmt.Errorf("failed to parse URL: %w", err)
	}
	owner := n.GetRepository().GetOwner().GetLogin()
	repo := n.GetRepository().GetName()
	m["title"] = title
	m["owner"] = owner
	m["repo"] = repo

	me, _, err := c.client.Users.Get(ctx, "")
	if err != nil {
		return fmt.Errorf("failed to get authenticated user: %w", err)
	}
	m["me"] = me.GetLogin()

	subjectType := n.GetSubject().GetType()
	var htmlURL string
	var number int

	// Initialize default values
	m["is_issue"] = false
	m["is_pull_request"] = false
	m["is_release"] = false
	m["number"] = -1
	m["approved"] = false
	m["review_states"] = []string{}
	m["state"] = "unknown"
	m["draft"] = false
	m["merged"] = false
	m["mergeable"] = false
	m["mergeable_state"] = "unknown"
	m["closed"] = false
	m["labels"] = []string{}
	m["reviewers"] = []string{}
	m["review_teams"] = []string{}
	m["assignees"] = []string{}
	m["author"] = ""
	m["html_url"] = ""
	m["status_passed"] = false
	m["checks_passed"] = false
	m["passed"] = false

	switch subjectType {
	case "Issue":
		m["is_issue"] = true
		number, err = strconv.Atoi(path.Base(u.Path))
		if err != nil {
			return fmt.Errorf("failed to parse number from URL: %w", err)
		}
		m["number"] = number
		issue, _, err := c.client.Issues.Get(ctx, owner, repo, number)
		if err != nil {
			return fmt.Errorf("failed to get issue: %w", err)
		}
		htmlURL = issue.GetHTMLURL()
		m["state"] = issue.GetState()
		m["closed"] = !issue.GetClosedAt().Equal(github.Timestamp{})
		m["labels"] = lo.Map(issue.Labels, func(l *github.Label, _ int) string {
			return l.GetName()
		})
		m["assignees"] = lo.Map(issue.Assignees, func(a *github.User, _ int) string {
			return a.GetLogin()
		})
		m["author"] = issue.GetUser().GetLogin()
		m["html_url"] = issue.GetHTMLURL()
	case "PullRequest":
		m["is_pull_request"] = true
		number, err = strconv.Atoi(path.Base(u.Path))
		if err != nil {
			return fmt.Errorf("failed to parse number from URL: %w", err)
		}
		m["number"] = number
		pr, _, err := c.client.PullRequests.Get(ctx, owner, repo, number)
		if err != nil {
			return fmt.Errorf("failed to get pull request: %w", err)
		}
		htmlURL = pr.GetHTMLURL()
		m["state"] = pr.GetState()
		m["draft"] = pr.GetDraft()
		m["merged"] = pr.GetMerged()
		m["mergeable"] = pr.GetMergeable()
		m["mergeable_state"] = pr.GetMergeableState()
		m["closed"] = !pr.GetClosedAt().Equal(github.Timestamp{})
		m["labels"] = lo.Map(pr.Labels, func(l *github.Label, _ int) string {
			return l.GetName()
		})
		m["reviewers"] = lo.Map(pr.RequestedReviewers, func(r *github.User, _ int) string {
			return r.GetLogin()
		})
		m["review_teams"] = lo.Map(pr.RequestedTeams, func(t *github.Team, _ int) string {
			return t.GetName()
		})
		m["assignees"] = lo.Map(pr.Assignees, func(a *github.User, _ int) string {
			return a.GetLogin()
		})
		m["author"] = pr.GetUser().GetLogin()
		m["html_url"] = pr.GetHTMLURL()
		reviews, _, err := c.client.PullRequests.ListReviews(ctx, owner, repo, number, &github.ListOptions{})
		if err != nil {
			return fmt.Errorf("failed to list pull request reviews: %w", err)
		}
		slices.SortFunc(reviews, func(a, b *github.PullRequestReview) int {
			return a.GetSubmittedAt().Compare(b.GetSubmittedAt().Time)
		})
		m["approved"] = false
		var reviewStates []string
		for _, review := range reviews {
			state := review.GetState()
			reviewStates = append(reviewStates, state)
			if state == "APPROVED" {
				m["approved"] = true
				break
			}
		}
		m["review_states"] = reviewStates
		commitSHA := pr.GetHead().GetSHA()

		combinedStatus, _, err := c.client.Repositories.GetCombinedStatus(ctx, owner, repo, commitSHA, &github.ListOptions{})
		if err != nil {
			return fmt.Errorf("failed to get combined status: %w", err)
		}
		statusPassed := true
		for _, status := range combinedStatus.Statuses {
			if status.GetState() != "success" {
				statusPassed = false
				break
			}
		}
		checkRuns, _, err := c.client.Checks.ListCheckRunsForRef(ctx, owner, repo, commitSHA, &github.ListCheckRunsOptions{})
		if err != nil {
			return fmt.Errorf("failed to list check runs: %w", err)
		}
		checksPassed := true
		for _, checkRun := range checkRuns.CheckRuns {
			if checkRun.GetStatus() != "completed" || !slices.Contains([]string{"neutral", "skipped", "success"}, checkRun.GetConclusion()) {
				checksPassed = false
				break
			}
		}
		m["status_passed"] = statusPassed
		m["checks_passed"] = checksPassed
		m["passed"] = statusPassed && checksPassed
	case "Release":
		m["is_release"] = true
	default:
		slog.Warn("Unknown subject type", "type", subjectType, "url", n.GetSubject().GetURL())
		return nil // Skip unknown subject types
	}
	open := false
	if c.openLimit.Load() > 0 {
		open = evalCond(c.config.Open.Conditions, m)
		if open {
			if err := browser.OpenURL(htmlURL); err != nil {
				return fmt.Errorf("failed to open URL in browser: %w", err)
			}
			c.openLimit.Add(-1)
		}
	}
	if !open {
		if c.readLimit.Load() > 0 {
			read := evalCond(c.config.Read.Conditions, m)
			if read {
				if _, err := c.client.Activity.MarkThreadRead(ctx, n.GetID()); err != nil {
					return fmt.Errorf("failed to mark notification as read: %w", err)
				}
				c.readLimit.Add(-1)
			}
		}
	}
	if c.listLimit.Load() > 0 {
		list := evalCond(c.config.List.Conditions, m)
		if list {
			mark := "▬"
			switch m["state"] {
			case "open":
				mark = openC.Sprint(mark)
			case "closed":
				mark = closedC.Sprint(mark)
			case "merged":
				mark = mergedC.Sprint(mark)
			}
			number := numberC.Sprintf("%s %s/%s #%d", mark, owner, repo, number)
			if _, err := fmt.Fprintf(c.w, "%s\n", number); err != nil {
				return err
			}
			if termlink.SupportsHyperlinks() {
				if _, err := fmt.Fprintf(c.w, "  %s\n", termlink.Link(titleC.Sprint(title), htmlURL)); err != nil {
					return err
				}
			} else {
				if _, err := fmt.Fprintf(c.w, "  %s ( %s )\n", titleC.Sprint(title), htmlURL); err != nil {
					return err
				}
			}
			c.listLimit.Add(-1)
		}
	}
	return nil
}

func evalCond(cond []string, m map[string]any) bool {
	if len(cond) == 0 {
		return false
	}
	joined := "(" + strings.Join(lo.Map(cond, func(cond string, _ int) string {
		if cond == "*" {
			return "true"
		}
		return cond
	}), ") || (") + ")"
	v, err := expr.Eval(joined, m)
	if err != nil {
		slog.Error("Failed to evaluate read condition", "cond", joined, "error", err)
		return false
	}
	switch tf := v.(type) {
	case bool:
		return tf
	default:
		slog.Error("Condition did not evaluate to boolean", "cond", joined, "value", tf, "type", fmt.Sprintf("%T", tf))
		return false
	}
}
