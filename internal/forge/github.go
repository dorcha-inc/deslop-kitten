package forge

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/google/go-github/v80/github"
)

// GitHub reads pull requests through the GitHub REST API.
type GitHub struct {
	client *github.Client
}

// NewGitHub returns a client for github.com. A nil httpClient uses
// http.DefaultClient. An empty token sends unauthenticated requests,
// which GitHub rate limits to sixty per hour.
func NewGitHub(httpClient *http.Client, token string) *GitHub {
	c := github.NewClient(httpClient)
	if token != "" {
		c = c.WithAuthToken(token)
	}
	return &GitHub{client: c}
}

// NewGitHubEnterprise returns a client for a GitHub Enterprise Server or
// a test server at baseURL. The REST API is expected under /api/v3/.
func NewGitHubEnterprise(httpClient *http.Client, token, baseURL string) (*GitHub, error) {
	c := github.NewClient(httpClient)
	if token != "" {
		c = c.WithAuthToken(token)
	}
	c, err := c.WithEnterpriseURLs(baseURL, baseURL)
	if err != nil {
		return nil, fmt.Errorf("configure enterprise urls: %w", err)
	}
	return &GitHub{client: c}, nil
}

// PullRequest fetches the pull request, its files, its commits, the
// check status of its head, its author's account and recent activity,
// and the issues its body references. A full fetch costs about nine
// API requests.
func (g *GitHub) PullRequest(ctx context.Context, owner, repo string, number int) (*PullRequest, error) {
	pr, resp, err := g.client.PullRequests.Get(ctx, owner, repo, number)
	if err != nil {
		if isNotFound(resp) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get pull request: %w", err)
	}
	out := &PullRequest{
		Owner:      owner,
		Repo:       repo,
		Number:     number,
		Title:      pr.GetTitle(),
		Body:       pr.GetBody(),
		HeadBranch: pr.GetHead().GetRef(),
		CreatedAt:  pr.GetCreatedAt().Time,
	}
	if out.Author, err = g.account(ctx, owner, repo, pr.GetUser()); err != nil {
		return nil, err
	}
	if out.Files, err = g.files(ctx, owner, repo, number); err != nil {
		return nil, err
	}
	if out.Commits, err = g.commits(ctx, owner, repo, number); err != nil {
		return nil, err
	}
	if out.Checks, err = g.checks(ctx, owner, repo, pr.GetHead().GetSHA()); err != nil {
		return nil, err
	}
	if out.LinkedIssues, err = g.issues(ctx, owner, repo, IssueRefs(pr.GetBody())); err != nil {
		return nil, err
	}
	return out, nil
}

// PolicyFile reads the policy from the repository through the API, so
// the action needs no checkout and a pull request cannot change the
// policy that applies to it.
func (g *GitHub) PolicyFile(ctx context.Context, owner, repo, ref, path string) ([]byte, bool, error) {
	file, _, resp, err := g.client.Repositories.GetContents(ctx, owner, repo, path, &github.RepositoryContentGetOptions{Ref: ref})
	if err != nil {
		if isNotFound(resp) {
			return nil, false, nil
		}
		return nil, false, fmt.Errorf("get %s: %w", path, err)
	}
	if file == nil {
		return nil, false, fmt.Errorf("%s is a directory", path)
	}
	content, err := file.GetContent()
	if err != nil {
		return nil, false, fmt.Errorf("decode %s: %w", path, err)
	}
	return []byte(content), true, nil
}

// UpsertComment keeps one comment per pull request so reruns update it
// in place instead of adding a new one each time.
func (g *GitHub) UpsertComment(ctx context.Context, owner, repo string, number int, marker, body string, create bool) error {
	opts := &github.IssueListCommentsOptions{ListOptions: github.ListOptions{PerPage: 100}}
	for {
		comments, resp, err := g.client.Issues.ListComments(ctx, owner, repo, number, opts)
		if err != nil {
			return fmt.Errorf("list comments: %w", err)
		}
		for _, c := range comments {
			if strings.Contains(c.GetBody(), marker) {
				if _, _, err := g.client.Issues.EditComment(ctx, owner, repo, c.GetID(), &github.IssueComment{Body: new(body)}); err != nil {
					return fmt.Errorf("edit comment: %w", err)
				}
				return nil
			}
		}
		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}
	if !create {
		return nil
	}
	if _, _, err := g.client.Issues.CreateComment(ctx, owner, repo, number, &github.IssueComment{Body: new(body)}); err != nil {
		return fmt.Errorf("create comment: %w", err)
	}
	return nil
}

// account builds the author from the user record when one exists. The
// Copilot coding agent and other app actors have no user record and
// cannot be searched, so a 404 there yields a bot with the login alone.
func (g *GitHub) account(ctx context.Context, owner, repo string, u *github.User) (Account, error) {
	login := u.GetLogin()
	acct := Account{Login: login, IsBot: u.GetType() == "Bot"}
	full, resp, err := g.client.Users.Get(ctx, login)
	switch {
	case err == nil:
		acct.IsBot = full.GetType() == "Bot"
		acct.CreatedAt = full.GetCreatedAt().Time
		acct.PublicRepos = full.GetPublicRepos()
		acct.Followers = full.GetFollowers()
	case isNotFound(resp):
		acct.IsBot = true
		return acct, nil
	default:
		return Account{}, fmt.Errorf("get user %s: %w", login, err)
	}
	if acct.PullRequestsInRepo, err = g.searchCount(ctx, fmt.Sprintf("repo:%s/%s is:pr author:%s", owner, repo, login)); err != nil {
		return Account{}, fmt.Errorf("count pull requests in repo for %s: %w", login, err)
	}
	if acct.MergedPullRequestsInRepo, err = g.searchCount(ctx, fmt.Sprintf("repo:%s/%s is:pr is:merged author:%s", owner, repo, login)); err != nil {
		return Account{}, fmt.Errorf("count merged pull requests in repo for %s: %w", login, err)
	}
	if acct.MergedPullRequestsAnywhere, err = g.searchCount(ctx, fmt.Sprintf("is:pr is:merged author:%s", login)); err != nil {
		return Account{}, fmt.Errorf("count merged pull requests for %s: %w", login, err)
	}
	if acct.RecentEventTimes, err = g.recentEvents(ctx, login); err != nil {
		return Account{}, err
	}
	return acct, nil
}

func (g *GitHub) recentEvents(ctx context.Context, login string) ([]time.Time, error) {
	events, resp, err := g.client.Activity.ListEventsPerformedByUser(ctx, login, true, &github.ListOptions{PerPage: 100})
	if err != nil {
		if isNotFound(resp) {
			return nil, nil
		}
		return nil, fmt.Errorf("list public events for %s: %w", login, err)
	}
	out := make([]time.Time, 0, len(events))
	for _, e := range events {
		out = append(out, e.GetCreatedAt().Time)
	}
	return out, nil
}

// searchCount returns the number of results for an issue search. GitHub
// answers 422 for a login it cannot search, which counts as zero.
func (g *GitHub) searchCount(ctx context.Context, query string) (int, error) {
	res, resp, err := g.client.Search.Issues(ctx, query, &github.SearchOptions{ListOptions: github.ListOptions{PerPage: 1}})
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusUnprocessableEntity {
			return 0, nil
		}
		return 0, err
	}
	return res.GetTotal(), nil
}

func (g *GitHub) files(ctx context.Context, owner, repo string, number int) ([]File, error) {
	var out []File
	opts := &github.ListOptions{PerPage: 100}
	for {
		page, resp, err := g.client.PullRequests.ListFiles(ctx, owner, repo, number, opts)
		if err != nil {
			return nil, fmt.Errorf("list files: %w", err)
		}
		for _, f := range page {
			out = append(out, File{Path: f.GetFilename(), Additions: f.GetAdditions(), Deletions: f.GetDeletions(), Patch: f.GetPatch()})
		}
		if resp.NextPage == 0 {
			return out, nil
		}
		opts.Page = resp.NextPage
	}
}

func (g *GitHub) commits(ctx context.Context, owner, repo string, number int) ([]Commit, error) {
	var out []Commit
	opts := &github.ListOptions{PerPage: 100}
	for {
		page, resp, err := g.client.PullRequests.ListCommits(ctx, owner, repo, number, opts)
		if err != nil {
			return nil, fmt.Errorf("list commits: %w", err)
		}
		for _, c := range page {
			author := c.GetCommit().GetAuthor()
			out = append(out, Commit{
				SHA:         c.GetSHA(),
				Message:     c.GetCommit().GetMessage(),
				AuthorEmail: author.GetEmail(),
				AuthoredAt:  author.GetDate().Time,
			})
		}
		if resp.NextPage == 0 {
			return out, nil
		}
		opts.Page = resp.NextPage
	}
}

func (g *GitHub) checks(ctx context.Context, owner, repo, sha string) (CheckStatus, error) {
	runs, _, err := g.client.Checks.ListCheckRunsForRef(ctx, owner, repo, sha, &github.ListCheckRunsOptions{ListOptions: github.ListOptions{PerPage: 100}})
	if err != nil {
		return CheckUnknown, fmt.Errorf("list check runs: %w", err)
	}
	if len(runs.CheckRuns) == 0 {
		return CheckUnknown, nil
	}
	status := CheckPassing
	for _, r := range runs.CheckRuns {
		if r.GetStatus() != "completed" {
			status = CheckPending
			continue
		}
		switch r.GetConclusion() {
		case "failure", "timed_out", "cancelled", "action_required":
			return CheckFailing, nil
		}
	}
	return status, nil
}

func (g *GitHub) issues(ctx context.Context, owner, repo string, numbers []int) ([]Issue, error) {
	var out []Issue
	for _, n := range numbers {
		is, resp, err := g.client.Issues.Get(ctx, owner, repo, n)
		if err != nil {
			if isNotFound(resp) {
				continue
			}
			return nil, fmt.Errorf("get issue %d: %w", n, err)
		}
		if is.IsPullRequest() {
			continue
		}
		labels := make([]string, 0, len(is.Labels))
		for _, l := range is.Labels {
			labels = append(labels, l.GetName())
		}
		out = append(out, Issue{Number: n, Labels: labels, Open: is.GetState() == "open"})
	}
	return out, nil
}

func isNotFound(resp *github.Response) bool {
	return resp != nil && resp.StatusCode == http.StatusNotFound
}
