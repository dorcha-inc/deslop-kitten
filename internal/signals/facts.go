package signals

import (
	"context"
	"fmt"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/dorcha-inc/deslop-kitten/internal/forge"
	"github.com/dorcha-inc/deslop-kitten/internal/rules"
)

const burstWindow = 3 * time.Hour

// assistanceTrailers are trailer names that declare AI involvement on
// their own, whatever their value.
var assistanceTrailers = []string{"assisted-by", "co-developed-by"}

type declaredBot struct{}

func (declaredBot) ID() string { return "declared_bot" }

func (declaredBot) Evaluate(_ context.Context, in Input) ([]Finding, error) {
	if !in.PR.Author.IsBot || in.Policy.IsTrustedBot(in.PR.Author.Login) {
		return nil, nil
	}
	return []Finding{{
		Signal:   "declared_bot",
		Category: CategoryFact,
		Severity: SeverityHigh,
		Body:     "the author is a bot account outside this repository's trusted list",
	}}, nil
}

type agentBranchPrefix struct{}

func (agentBranchPrefix) ID() string { return "agent_branch_prefix" }

func (agentBranchPrefix) Evaluate(_ context.Context, in Input) ([]Finding, error) {
	if _, ok := matchedPrefix(in.PR.HeadBranch, in.Policy.AgentBranchPrefixes); !ok {
		return nil, nil
	}
	return []Finding{{
		Signal:   "agent_branch_prefix",
		Category: CategoryFact,
		Severity: SeverityHigh,
		Body:     fmt.Sprintf("the head branch `%s` carries a prefix used by AI coding agents", in.PR.HeadBranch),
	}}, nil
}

type aiTrailer struct{}

func (aiTrailer) ID() string { return "ai_trailer" }

func (aiTrailer) Evaluate(_ context.Context, in Input) ([]Finding, error) {
	trailers := aiTrailers(in.PR, in.Policy)
	if len(trailers) == 0 {
		return nil, nil
	}
	return []Finding{{
		Signal:   "ai_trailer",
		Category: CategoryFact,
		Severity: SeverityMedium,
		Body:     "the commits carry `" + strings.Join(trailers, "`, `") + "`",
	}}, nil
}

type modelCommitEmail struct{}

func (modelCommitEmail) ID() string { return "model_commit_email" }

func (modelCommitEmail) Evaluate(_ context.Context, in Input) ([]Finding, error) {
	var emails []string
	for _, c := range in.PR.Commits {
		if matchesAny(in.Policy.ModelEmailPatterns, c.AuthorEmail) {
			emails = append(emails, c.AuthorEmail)
		}
	}
	emails = unique(emails)
	if len(emails) == 0 {
		return nil, nil
	}
	noun := "the model service address"
	if len(emails) > 1 {
		noun = "the model service addresses"
	}
	return []Finding{{
		Signal:   "model_commit_email",
		Category: CategoryFact,
		Severity: SeverityHigh,
		Body:     "the commits come from " + noun + " `" + strings.Join(emails, "`, `") + "`",
	}}, nil
}

type activityBurst struct{}

func (activityBurst) ID() string { return "activity_burst" }

func (activityBurst) Evaluate(_ context.Context, in Input) ([]Finding, error) {
	limit := in.Policy.BurstEvents
	times := in.PR.Author.RecentEventTimes
	if trustedBot(in) || limit <= 0 || len(times) < limit {
		return nil, nil
	}
	count, start := densestWindow(times, burstWindow)
	if count < limit {
		return nil, nil
	}
	return []Finding{{
		Signal:   "activity_burst",
		Category: CategoryFact,
		Severity: SeverityMedium,
		Body:     fmt.Sprintf("the account produced %d public events inside three hours on %s", count, start.UTC().Format("2006-01-02")),
	}}, nil
}

// densestWindow returns the largest number of timestamps that fall
// inside any window of the given length, and the start of that window.
func densestWindow(times []time.Time, window time.Duration) (int, time.Time) {
	sorted := slices.SortedFunc(slices.Values(times), func(a, b time.Time) int { return a.Compare(b) })
	best, bestStart := 0, time.Time{}
	lo := 0
	for hi := range sorted {
		for sorted[hi].Sub(sorted[lo]) > window {
			lo++
		}
		if n := hi - lo + 1; n > best {
			best, bestStart = n, sorted[lo]
		}
	}
	return best, bestStart
}

func trustedBot(in Input) bool {
	return in.PR.Author.IsBot && in.Policy.IsTrustedBot(in.PR.Author.Login)
}

// aiTrailers returns every commit trailer that declares AI involvement,
// formatted as "Name: value". A co-authored-by trailer counts when its
// value matches a policy pattern. An assistance trailer counts always.
func aiTrailers(pr *forge.PullRequest, p *rules.Policy) []string {
	var out []string
	for _, c := range pr.Commits {
		trailers := c.Trailers()
		for _, v := range trailers["co-authored-by"] {
			if matchesAny(p.AICoauthorPatterns, v) {
				out = append(out, "Co-authored-by: "+v)
			}
		}
		for _, name := range assistanceTrailers {
			for _, v := range trailers[name] {
				out = append(out, strings.ToUpper(name[:1])+name[1:]+": "+v)
			}
		}
	}
	return unique(out)
}

// agentEvidence reports whether the branch, the commits, or the account
// type points at an agent or a model service.
func agentEvidence(pr *forge.PullRequest, p *rules.Policy) bool {
	if _, ok := matchedPrefix(pr.HeadBranch, p.AgentBranchPrefixes); ok {
		return true
	}
	if len(aiTrailers(pr, p)) > 0 {
		return true
	}
	if slices.ContainsFunc(pr.Commits, func(c forge.Commit) bool { return matchesAny(p.ModelEmailPatterns, c.AuthorEmail) }) {
		return true
	}
	return pr.Author.IsBot && !p.IsTrustedBot(pr.Author.Login)
}

func matchedPrefix(branch string, prefixes []string) (string, bool) {
	for _, prefix := range prefixes {
		if strings.HasPrefix(branch, prefix) {
			return prefix, true
		}
	}
	return "", false
}

func matchesAny(res []*regexp.Regexp, s string) bool {
	return slices.ContainsFunc(res, func(re *regexp.Regexp) bool { return re.MatchString(s) })
}

func unique(in []string) []string {
	return slices.Compact(slices.Sorted(slices.Values(in)))
}
