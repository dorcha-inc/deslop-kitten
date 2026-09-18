package report

import (
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/dorcha-inc/deslop-kitten/internal/forge"
	"github.com/dorcha-inc/deslop-kitten/internal/signals"
)

var now = time.Date(2026, 9, 17, 12, 0, 0, 0, time.UTC)

func TestReport_CommentOrdersRequestsAndWritesTheAuthorInProse(t *testing.T) {
	r := Report{
		Owner:       "o",
		Repo:        "r",
		RepoURL:     "https://github.com/o/r",
		Author:      forge.Account{Login: "fresh-bot", URL: "https://github.com/fresh-bot", CreatedAt: time.Date(2015, 3, 2, 0, 0, 0, 0, time.UTC), PublicRepos: 12, Followers: 80, PullRequestsInRepo: 1, MergedPullRequestsAnywhere: 40},
		GeneratedAt: now,
		Findings: []signals.Finding{
			{Signal: "thin_description", Category: signals.CategoryChange, Severity: signals.SeverityLow, Title: "Describe the change.", Body: "Thin."},
			{Signal: "large_change", Category: signals.CategoryChange, Severity: signals.SeverityHigh, Title: "Consider splitting.", Body: "Big."},
			{Signal: "ai_disclosure_required", Category: signals.CategoryPolicy, Severity: signals.SeverityMedium, Title: "Disclose AI assistance.", Body: "No disclosure."},
			{Signal: "agent_branch_prefix", Category: signals.CategoryFact, Severity: signals.SeverityHigh, Body: "the head branch `copilot/x` carries a prefix used by AI coding agents"},
			{Signal: "model_commit_email", Category: signals.CategoryFact, Severity: signals.SeverityHigh, Body: "the commits come from the model service address `a@b`"},
		},
	}
	md := r.Comment(Policy{Preset: "kubernetes", Source: "https://example.com/policy", PolicyPath: ".github/deslop-kitten.yaml"})

	assert.True(t, strings.HasPrefix(md, CommentMarker))
	assert.Contains(t, md, "1. **Disclose AI assistance.** No disclosure.\n2. **Consider splitting.** Big.\n3. **Describe the change.** Thin.\n")
	assert.Contains(t, md, "First pull request here from [fresh-bot](https://github.com/fresh-bot), with [40 merged elsewhere on GitHub](https://github.com/search?type=pullrequests&q=is%3Apr+is%3Amerged+author%3Afresh-bot+-repo%3Ao%2Fr) and an account since 2015. The head branch `copilot/x` carries a prefix used by AI coding agents and the commits come from the model service address `a@b`.")
	assert.Contains(t, md, "preset `kubernetes` ([source](https://example.com/policy))")
	assert.NotContains(t, md, "|")
	assert.False(t, regexp.MustCompile(`(^|\s)@\w`).MatchString(md), "a mention would notify the author")
}

func TestReport_AuthorSentenceCoversEachStanding(t *testing.T) {
	tests := []struct {
		name     string
		author   forge.Account
		findings []signals.Finding
		want     string
	}{
		{
			name:   "throwaway account",
			author: forge.Account{Login: "x", CreatedAt: now.AddDate(0, 0, -9), PullRequestsInRepo: 3},
			want:   "`x` has opened 3 pull requests here, none merged yet, with no merged pull requests on GitHub yet, an account created 9 days ago, no public repositories, and no followers.",
		},
		{
			name:   "returning contributor",
			author: forge.Account{Login: "ada", PullRequestsInRepo: 158, MergedPullRequestsInRepo: 124, MergedPullRequestsAnywhere: 136},
			want:   "`ada` has 124 pull requests merged here and 12 elsewhere on GitHub.",
		},
		{
			name:     "untrusted bot without a user record",
			author:   forge.Account{Login: "Copilot", IsBot: true},
			findings: []signals.Finding{{Signal: "declared_bot", Category: signals.CategoryFact}},
			want:     "`Copilot` is a bot account outside this repository's trusted list.",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := Report{Author: tt.author, Findings: tt.findings, GeneratedAt: now}
			assert.Equal(t, tt.want, r.authorSentence())
		})
	}
}

func TestReport_ReturningAuthorWithNothingToSayGetsNoComment(t *testing.T) {
	r := Report{Author: forge.Account{Login: "ada", MergedPullRequestsInRepo: 5}}
	assert.False(t, r.ShouldComment())
	assert.Contains(t, AllClear(), "Every check passes")
}
