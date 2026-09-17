package signals

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/jadidbourbaki/vetkitten/internal/disclosure"
)

type linkedIssueRequired struct{}

func (linkedIssueRequired) ID() string { return "linked_issue_required" }

func (linkedIssueRequired) Evaluate(_ context.Context, in Input) ([]Finding, error) {
	if !in.Policy.RequireLinkedIssue || len(in.PR.LinkedIssues) > 0 {
		return nil, nil
	}
	return []Finding{{
		Signal:   "linked_issue_required",
		Category: CategoryPolicy,
		Severity: SeverityHigh,
		Title:    "Reference an issue.",
		Body:     "The description references no issue. This repository asks every pull request to reference one. Add `Fixes #<number>` to the description.",
	}}, nil
}

type linkedIssueNotAccepted struct{}

func (linkedIssueNotAccepted) ID() string { return "linked_issue_not_accepted" }

func (linkedIssueNotAccepted) Evaluate(_ context.Context, in Input) ([]Finding, error) {
	if len(in.Policy.AcceptedIssueLabels) == 0 || len(in.PR.LinkedIssues) == 0 {
		return nil, nil
	}
	var numbers []string
	for _, is := range in.PR.LinkedIssues {
		if slices.ContainsFunc(is.Labels, func(l string) bool { return slices.Contains(in.Policy.AcceptedIssueLabels, l) }) {
			return nil, nil
		}
		numbers = append(numbers, fmt.Sprintf("#%d", is.Number))
	}
	return []Finding{{
		Signal:   "linked_issue_not_accepted",
		Category: CategoryPolicy,
		Severity: SeverityMedium,
		Title:    "Get the issue triaged.",
		Body:     fmt.Sprintf("The linked issue %s carries none of the labels this repository accepts (%s). Ask a maintainer to triage it before review.", strings.Join(numbers, ", "), strings.Join(in.Policy.AcceptedIssueLabels, ", ")),
	}}, nil
}

// aiDisclosureRequired asks the judge whether the description discloses
// AI assistance when the policy requires it and the branch, commits, or
// account type show AI involvement. Without a judge the signal is
// skipped, because a pattern match cannot tell a disclosure from a
// mention or a denial.
type aiDisclosureRequired struct {
	judge disclosure.Judge
}

func (aiDisclosureRequired) ID() string { return "ai_disclosure_required" }

func (s aiDisclosureRequired) Evaluate(ctx context.Context, in Input) ([]Finding, error) {
	if s.judge == nil || !in.Policy.RequireAIDisclosure || !agentEvidence(in.PR, in.Policy) {
		return nil, nil
	}
	v, err := s.judge.Judge(ctx, disclosure.Overview{Title: in.PR.Title, Body: in.PR.Body})
	if err != nil {
		return nil, fmt.Errorf("judge: %w", err)
	}
	var findings []Finding
	trailers := aiTrailers(in.PR, in.Policy)
	if !v.Disclosed {
		evidence := "The branch or commits show AI involvement"
		if len(trailers) > 0 {
			evidence = "Commits carry `" + strings.Join(trailers, "`, `") + "`"
		}
		findings = append(findings, Finding{
			Signal:   "ai_disclosure_required",
			Category: CategoryPolicy,
			Severity: SeverityMedium,
			Title:    "Disclose AI assistance.",
			Body:     evidence + " and the description does not say how AI assisted the change. This repository asks for a sentence in the description naming the tool and how it was used.",
		})
	}
	if in.Policy.ForbidAITrailers && len(trailers) > 0 {
		findings = append(findings, Finding{
			Signal:   "ai_disclosure_required",
			Category: CategoryPolicy,
			Severity: SeverityMedium,
			Title:    "Remove AI trailers.",
			Body:     "This repository does not accept AI co-author or assisted-by trailers on commits. Remove `" + strings.Join(trailers, "`, `") + "`.",
		})
	}
	return findings, nil
}

type filesWithoutIssue struct{}

func (filesWithoutIssue) ID() string { return "files_without_issue" }

func (filesWithoutIssue) Evaluate(_ context.Context, in Input) ([]Finding, error) {
	limit := in.Policy.MaxFilesWithoutIssue
	if limit <= 0 || len(in.PR.LinkedIssues) > 0 || len(in.PR.Files) <= limit {
		return nil, nil
	}
	return []Finding{{
		Signal:   "files_without_issue",
		Category: CategoryPolicy,
		Severity: SeverityMedium,
		Title:    "Reference an issue.",
		Body:     fmt.Sprintf("%d files changed with no linked issue. This repository asks changes above %d files to reference an issue first.", len(in.PR.Files), limit),
	}}, nil
}
