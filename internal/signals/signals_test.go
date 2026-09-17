package signals

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/jadidbourbaki/vetkitten/internal/disclosure"
	"github.com/jadidbourbaki/vetkitten/internal/forge"
	"github.com/jadidbourbaki/vetkitten/internal/rules"
)

var now = time.Date(2026, 9, 17, 12, 0, 0, 0, time.UTC)

func humanPR() *forge.PullRequest {
	return &forge.PullRequest{
		Owner: "o", Repo: "r", Number: 1,
		Title: "fix: tighten parser",
		Body:  "Fixes #12. The parser accepted a trailing comma and this change rejects it with a clear error. Tests cover both branches.",
		Author: forge.Account{
			Login: "ada", CreatedAt: now.AddDate(-3, 0, 0),
			PullRequestsInRepo: 6, MergedPullRequestsInRepo: 5, MergedPullRequestsAnywhere: 40,
			RecentEventTimes: spread(now, 20, 6*time.Hour),
		},
		HeadBranch:   "ada/parser",
		Files:        []forge.File{{Path: "parser.go", Additions: 10, Deletions: 2}},
		Commits:      []forge.Commit{{SHA: "a", Message: "fix: tighten parser", AuthorEmail: "ada@example.com"}},
		Checks:       forge.CheckPassing,
		LinkedIssues: []forge.Issue{{Number: 12, Labels: []string{"accepted"}, Open: true}},
	}
}

func agentPR() *forge.PullRequest {
	pr := humanPR()
	pr.Author = forge.Account{Login: "fresh-bot", CreatedAt: now.AddDate(0, 0, -3), RecentEventTimes: spread(now, 40, 2*time.Minute)}
	pr.HeadBranch = "copilot/optimize-agg"
	pr.Body = "Optimized rendering."
	pr.Files = make([]forge.File, 60)
	for i := range pr.Files {
		pr.Files[i] = forge.File{Path: "f", Additions: 50, Deletions: 4}
	}
	pr.Commits = []forge.Commit{{
		SHA:         "b",
		Message:     "perf: optimize\n\nAssisted-by: Copilot\nCo-authored-by: Copilot <198982749+Copilot@users.noreply.github.com>",
		AuthorEmail: "198982749+Copilot@users.noreply.github.com",
	}}
	pr.Checks = forge.CheckFailing
	pr.LinkedIssues = nil
	return pr
}

func spread(end time.Time, n int, step time.Duration) []time.Time {
	out := make([]time.Time, n)
	for i := range out {
		out[i] = end.Add(-time.Duration(i) * step)
	}
	return out
}

func mustPreset(t *testing.T, name string) *rules.Policy {
	t.Helper()
	p, err := rules.Preset(name)
	require.NoError(t, err)
	return p
}

func ids(findings []Finding) []string {
	out := make([]string, 0, len(findings))
	for _, f := range findings {
		out = append(out, f.Signal)
	}
	return out
}

func TestRun_HumanPullRequestUnderDefaultPolicyIsClean(t *testing.T) {
	findings, err := Run(context.Background(), Defaults(nil), Input{PR: humanPR(), Policy: mustPreset(t, "default"), Now: now})
	require.NoError(t, err)
	assert.Empty(t, findings)
}

func TestRun_AgentPullRequestUnderKubernetesPolicyFiresEverySignal(t *testing.T) {
	findings, err := Run(context.Background(), Defaults(&disclosure.FakeJudge{}), Input{PR: agentPR(), Policy: mustPreset(t, "kubernetes"), Now: now})
	require.NoError(t, err)
	assert.Equal(t, []string{
		"large_change",
		"ci_failing",
		"thin_description",
		"agent_branch_prefix",
		"ai_trailer",
		"model_commit_email",
		"activity_burst",
		"ai_disclosure_required",
		"ai_disclosure_required",
	}, ids(findings))
	assert.Equal(t, "Disclose AI assistance.", findings[7].Title)
	assert.Equal(t, "Remove AI trailers.", findings[8].Title)
	assert.Contains(t, findings[8].Body, "`Assisted-by: Copilot`, `Co-authored-by: Copilot <198982749+Copilot@users.noreply.github.com>`")
	for _, f := range findings {
		assert.Equal(t, f.Category != CategoryFact, f.IsRequest(), f.Signal)
	}
}

func TestRun_GhosttyPolicyAsksOnlyForDisclosure(t *testing.T) {
	findings, err := Run(context.Background(), Defaults(&disclosure.FakeJudge{}), Input{PR: agentPR(), Policy: mustPreset(t, "ghostty"), Now: now})
	require.NoError(t, err)
	var policy []string
	for _, f := range findings {
		if f.Category == CategoryPolicy {
			policy = append(policy, f.Title)
		}
	}
	assert.Equal(t, []string{"Disclose AI assistance."}, policy)
}

func TestTrustedBot_ProducesNoFacts(t *testing.T) {
	pr := humanPR()
	pr.Author = forge.Account{Login: "dependabot[bot]", IsBot: true, RecentEventTimes: spread(now, 40, time.Minute)}
	findings, err := Run(context.Background(), Defaults(nil), Input{PR: pr, Policy: mustPreset(t, "default"), Now: now})
	require.NoError(t, err)
	assert.Empty(t, findings)

	pr.Author.Login = "mystery[bot]"
	findings, err = Run(context.Background(), Defaults(nil), Input{PR: pr, Policy: mustPreset(t, "default"), Now: now})
	require.NoError(t, err)
	assert.Equal(t, []string{"declared_bot", "activity_burst"}, ids(findings))
}

func TestActivityBurst_UsesDensestThreeHourWindow(t *testing.T) {
	tests := []struct {
		name  string
		times []time.Time
		fires bool
	}{
		{name: "spread over days", times: spread(now, 40, 6*time.Hour), fires: false},
		{name: "one burst", times: spread(now, 40, 2*time.Minute), fires: true},
		{name: "burst hidden among old events", times: append(spread(now, 32, time.Minute), spread(now.Add(-48*time.Hour), 8, time.Hour)...), fires: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pr := humanPR()
			pr.Author.RecentEventTimes = tt.times
			got, err := activityBurst{}.Evaluate(context.Background(), Input{PR: pr, Policy: mustPreset(t, "default"), Now: now})
			require.NoError(t, err)
			assert.Equal(t, tt.fires, len(got) == 1)
		})
	}
}

func TestAIDisclosureRequired_JudgeDecidesAndNilJudgeSkips(t *testing.T) {
	pr := agentPR()
	pr.Body = "I drafted the first pass with an assistant and rewrote it by hand."
	p := mustPreset(t, "ghostty")

	disclosed := &disclosure.FakeJudge{Verdicts: map[string]disclosure.Verdict{pr.Title: {Disclosed: true, Quote: pr.Body}}}
	got, err := aiDisclosureRequired{judge: disclosed}.Evaluate(context.Background(), Input{PR: pr, Policy: p, Now: now})
	require.NoError(t, err)
	assert.Empty(t, got)

	got, err = aiDisclosureRequired{judge: &disclosure.FakeJudge{}}.Evaluate(context.Background(), Input{PR: pr, Policy: p, Now: now})
	require.NoError(t, err)
	assert.Len(t, got, 1)

	got, err = aiDisclosureRequired{}.Evaluate(context.Background(), Input{PR: pr, Policy: p, Now: now})
	require.NoError(t, err)
	assert.Empty(t, got)
}

func TestAIDisclosureRequired_JudgeNotCalledWithoutAgentEvidence(t *testing.T) {
	judge := &disclosure.FakeJudge{}
	got, err := aiDisclosureRequired{judge: judge}.Evaluate(context.Background(), Input{PR: humanPR(), Policy: mustPreset(t, "kubernetes"), Now: now})
	require.NoError(t, err)
	assert.Empty(t, got)
	assert.Empty(t, judge.Calls)
}

func TestRun_KeepsFindingsFromOtherSignalsWhenTheJudgeFails(t *testing.T) {
	boom := errors.New("boom")
	findings, err := Run(context.Background(), Defaults(&disclosure.FakeJudge{Err: boom}), Input{PR: agentPR(), Policy: mustPreset(t, "kubernetes"), Now: now})
	assert.ErrorIs(t, err, boom)
	assert.NotContains(t, ids(findings), "ai_disclosure_required")
	assert.Contains(t, ids(findings), "agent_branch_prefix")
}
