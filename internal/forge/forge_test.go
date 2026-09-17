package forge

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseTrailers_ReadsFinalParagraph(t *testing.T) {
	tests := []struct {
		name    string
		message string
		want    map[string][]string
	}{
		{
			name:    "subject only",
			message: "fix: tighten parser",
			want:    map[string][]string{},
		},
		{
			name:    "subject and trailers",
			message: "fix: tighten parser\n\nCo-authored-by: Claude <noreply@anthropic.com>\nSigned-off-by: A Person <a@example.com>",
			want: map[string][]string{
				"co-authored-by": {"Claude <noreply@anthropic.com>"},
				"signed-off-by":  {"A Person <a@example.com>"},
			},
		},
		{
			name:    "final paragraph is prose",
			message: "feat: x\n\nThis paragraph ends with a Note: that is prose.",
			want:    map[string][]string{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, ParseTrailers(tt.message))
		})
	}
}

func TestIssueRefs_SkipsURLFragmentsAndEntities(t *testing.T) {
	assert.Equal(t, []int{3, 12}, IssueRefs("Fixes #12 and closes #3, see #12 again"))
	assert.Equal(t, []int{7}, IssueRefs("see https://example.com/a#5 and a &#39; quote and #7"))
}

func TestParseRef_AcceptsShortFormAndURL(t *testing.T) {
	tests := []struct {
		name    string
		in      string
		want    Ref
		wantErr bool
	}{
		{name: "short form", in: "octo/repo#7", want: Ref{Owner: "octo", Repo: "repo", Number: 7}},
		{name: "url with trailing slash", in: "https://github.com/octo/repo/pull/42/", want: Ref{Owner: "octo", Repo: "repo", Number: 42}},
		{name: "issue url", in: "https://github.com/octo/repo/issues/42", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseRef(tt.in)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestFakeForge_ReturnsIndependentCopy(t *testing.T) {
	f := &FakeForge{}
	f.Add(&PullRequest{
		Owner: "o", Repo: "r", Number: 1,
		Author:       Account{Login: "a", RecentEventTimes: []time.Time{time.Unix(1, 0)}},
		Files:        []File{{Path: "a"}},
		LinkedIssues: []Issue{{Number: 2, Labels: []string{"x"}}},
	})

	got, err := f.PullRequest(context.Background(), "o", "r", 1)
	require.NoError(t, err)
	got.Files[0].Path = "changed"
	got.LinkedIssues[0].Labels[0] = "changed"
	got.Author.RecentEventTimes[0] = time.Unix(99, 0)

	again, err := f.PullRequest(context.Background(), "o", "r", 1)
	require.NoError(t, err)
	assert.Equal(t, "a", again.Files[0].Path)
	assert.Equal(t, "x", again.LinkedIssues[0].Labels[0])
	assert.Equal(t, time.Unix(1, 0), again.Author.RecentEventTimes[0])
}

func TestFakeForge_UpsertKeepsOneComment(t *testing.T) {
	f := &FakeForge{}
	ctx := context.Background()
	require.NoError(t, f.UpsertComment(ctx, "o", "r", 1, "<!-- m -->", "<!-- m --> first", false))
	assert.Empty(t, f.Comments["o/r#1"])
	require.NoError(t, f.UpsertComment(ctx, "o", "r", 1, "<!-- m -->", "<!-- m --> first", true))
	require.NoError(t, f.UpsertComment(ctx, "o", "r", 1, "<!-- m -->", "<!-- m --> second", true))
	assert.Equal(t, []string{"<!-- m --> second"}, f.Comments["o/r#1"])
}

func TestGitHub_PullRequestAssemblesAllParts(t *testing.T) {
	srv := fakeGitHubServer(t)
	g, err := NewGitHubEnterprise(nil, "", srv.URL)
	require.NoError(t, err)

	pr, err := g.PullRequest(context.Background(), "o", "r", 7)
	require.NoError(t, err)

	assert.Equal(t, "alice/fix", pr.HeadBranch)
	assert.Equal(t, Account{
		Login:                      "alice",
		CreatedAt:                  time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
		PublicRepos:                12,
		Followers:                  80,
		PullRequestsInRepo:         6,
		MergedPullRequestsInRepo:   4,
		MergedPullRequestsAnywhere: 31,
		RecentEventTimes: []time.Time{
			time.Date(2026, 9, 16, 10, 0, 0, 0, time.UTC),
			time.Date(2026, 9, 15, 9, 0, 0, 0, time.UTC),
		},
	}, pr.Author)
	require.Len(t, pr.Commits, 1)
	assert.Equal(t, []string{"Claude <noreply@anthropic.com>"}, pr.Commits[0].Trailers()["co-authored-by"])
	assert.Equal(t, CheckPassing, pr.Checks)
	assert.Equal(t, []Issue{{Number: 12, Labels: []string{"accepted"}, Open: true}}, pr.LinkedIssues)
}

func TestGitHub_AppActorWithoutUserRecordIsABot(t *testing.T) {
	srv := fakeGitHubServer(t)
	g, err := NewGitHubEnterprise(nil, "", srv.URL)
	require.NoError(t, err)

	pr, err := g.PullRequest(context.Background(), "o", "r", 8)
	require.NoError(t, err)
	assert.Equal(t, Account{Login: "Copilot", IsBot: true}, pr.Author)
}

func TestGitHub_UnsearchableLoginCountsAsZero(t *testing.T) {
	srv := fakeGitHubServer(t)
	g, err := NewGitHubEnterprise(nil, "", srv.URL)
	require.NoError(t, err)

	n, err := g.searchCount(context.Background(), "is:pr is:merged author:ghost-actor")
	require.NoError(t, err)
	assert.Equal(t, 0, n)
}

func TestGitHub_MissingPullRequestIsNotFound(t *testing.T) {
	srv := fakeGitHubServer(t)
	g, err := NewGitHubEnterprise(nil, "", srv.URL)
	require.NoError(t, err)

	_, err = g.PullRequest(context.Background(), "o", "r", 99)
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestGitHub_UpsertEditsExistingCommentAndCreatesOtherwise(t *testing.T) {
	srv := fakeGitHubServer(t)
	g, err := NewGitHubEnterprise(nil, "", srv.URL)
	require.NoError(t, err)
	ctx := context.Background()

	require.NoError(t, g.UpsertComment(ctx, "o", "r", 8, "<!-- m -->", "<!-- m --> new", true))
	assert.Equal(t, int32(1), edits.Load())
	assert.Equal(t, int32(0), creates.Load())

	require.NoError(t, g.UpsertComment(ctx, "o", "r", 7, "<!-- m -->", "<!-- m --> new", false))
	assert.Equal(t, int32(0), creates.Load())

	require.NoError(t, g.UpsertComment(ctx, "o", "r", 7, "<!-- m -->", "<!-- m --> new", true))
	assert.Equal(t, int32(1), creates.Load())
}

var edits, creates atomic.Int32

func fakeGitHubServer(t *testing.T) *httptest.Server {
	t.Helper()
	edits.Store(0)
	creates.Store(0)
	write := func(w http.ResponseWriter, v any) {
		w.Header().Set("Content-Type", "application/json")
		require.NoError(t, json.NewEncoder(w).Encode(v))
	}
	notFound := func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		write(w, map[string]any{"message": "Not Found"})
	}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v3/repos/o/r/pulls/7", func(w http.ResponseWriter, _ *http.Request) {
		write(w, map[string]any{
			"number": 7, "title": "fix: tighten parser", "body": "Fixes #12",
			"user": map[string]any{"login": "alice", "type": "User"},
			"head": map[string]any{"ref": "alice/fix", "sha": "abc"},
		})
	})
	mux.HandleFunc("GET /api/v3/repos/o/r/pulls/8", func(w http.ResponseWriter, _ *http.Request) {
		write(w, map[string]any{
			"number": 8, "title": "Bump gateway", "body": "",
			"user": map[string]any{"login": "Copilot", "type": "Bot"},
			"head": map[string]any{"ref": "copilot/bump", "sha": "def"},
		})
	})
	for _, n := range []string{"7", "8"} {
		mux.HandleFunc("GET /api/v3/repos/o/r/pulls/"+n+"/files", func(w http.ResponseWriter, _ *http.Request) {
			write(w, []map[string]any{{"filename": "a.go", "additions": 3, "deletions": 1, "patch": "@@ -1 +1 @@"}})
		})
		mux.HandleFunc("GET /api/v3/repos/o/r/pulls/"+n+"/commits", func(w http.ResponseWriter, _ *http.Request) {
			write(w, []map[string]any{{
				"sha": "abc",
				"commit": map[string]any{
					"message": "fix: tighten parser\n\nCo-authored-by: Claude <noreply@anthropic.com>",
					"author":  map[string]any{"email": "alice@example.com", "date": "2026-09-01T00:00:00Z"},
				},
			}})
		})
	}
	mux.HandleFunc("GET /api/v3/users/alice", func(w http.ResponseWriter, _ *http.Request) {
		write(w, map[string]any{"login": "alice", "type": "User", "created_at": "2020-01-01T00:00:00Z", "public_repos": 12, "followers": 80})
	})
	mux.HandleFunc("GET /api/v3/users/alice/events/public", func(w http.ResponseWriter, _ *http.Request) {
		write(w, []map[string]any{
			{"type": "PushEvent", "created_at": "2026-09-16T10:00:00Z"},
			{"type": "IssueCommentEvent", "created_at": "2026-09-15T09:00:00Z"},
		})
	})
	mux.HandleFunc("GET /api/v3/users/Copilot", notFound)
	mux.HandleFunc("GET /api/v3/users/Copilot/events/public", notFound)
	mux.HandleFunc("GET /api/v3/search/issues", func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query().Get("q")
		if strings.Contains(q, "author:ghost-actor") {
			w.WriteHeader(http.StatusUnprocessableEntity)
			write(w, map[string]any{"message": "Validation Failed"})
			return
		}
		total := 31
		switch {
		case strings.Contains(q, "repo:") && strings.Contains(q, "is:merged"):
			total = 4
		case strings.Contains(q, "repo:"):
			total = 6
		}
		write(w, map[string]any{"total_count": total, "incomplete_results": false, "items": []any{}})
	})
	for _, sha := range []string{"abc", "def"} {
		mux.HandleFunc("GET /api/v3/repos/o/r/commits/"+sha+"/check-runs", func(w http.ResponseWriter, _ *http.Request) {
			write(w, map[string]any{"total_count": 1, "check_runs": []map[string]any{{"status": "completed", "conclusion": "success"}}})
		})
	}
	mux.HandleFunc("GET /api/v3/repos/o/r/issues/12", func(w http.ResponseWriter, _ *http.Request) {
		write(w, map[string]any{"number": 12, "state": "open", "labels": []map[string]any{{"name": "accepted"}}})
	})
	mux.HandleFunc("GET /api/v3/repos/o/r/issues/7/comments", func(w http.ResponseWriter, _ *http.Request) {
		write(w, []any{})
	})
	mux.HandleFunc("GET /api/v3/repos/o/r/issues/8/comments", func(w http.ResponseWriter, _ *http.Request) {
		write(w, []map[string]any{{"id": 55, "body": "<!-- m --> old"}})
	})
	mux.HandleFunc("PATCH /api/v3/repos/o/r/issues/comments/55", func(w http.ResponseWriter, _ *http.Request) {
		edits.Add(1)
		write(w, map[string]any{"id": 55})
	})
	mux.HandleFunc("POST /api/v3/repos/o/r/issues/7/comments", func(w http.ResponseWriter, _ *http.Request) {
		creates.Add(1)
		w.WriteHeader(http.StatusCreated)
		write(w, map[string]any{"id": 56})
	})
	mux.HandleFunc("/", notFound)
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}
