package signals

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/dorcha-inc/vetkitten/internal/disclosure"
	"github.com/dorcha-inc/vetkitten/internal/forge"
	"github.com/dorcha-inc/vetkitten/internal/rules"
)

// Category says where a finding appears in the comment.
type Category string

const (
	// CategoryPolicy is a request that cites a rule the repository set.
	CategoryPolicy Category = "policy"
	// CategoryChange is a request about the size or state of the change.
	CategoryChange Category = "change"
	// CategoryFact is a clause about how the change was produced.
	CategoryFact Category = "fact"
)

// Severity orders requests within the comment. Higher ranks come first.
type Severity string

const (
	SeverityLow    Severity = "low"
	SeverityMedium Severity = "medium"
	SeverityHigh   Severity = "high"
)

// Rank returns an integer that orders severities from low at 1 to high
// at 3.
func (s Severity) Rank() int {
	switch s {
	case SeverityLow:
		return 1
	case SeverityMedium:
		return 2
	case SeverityHigh:
		return 3
	}
	return 0
}

// Finding is one observation about a pull request. For a request, Title
// is a short imperative such as "Disclose AI assistance." and Body is
// one or two sentences giving the fact, the policy, and the action. For
// a fact, Title is empty and Body is a lowercase clause that the report
// joins into one sentence about how the change was produced.
type Finding struct {
	Signal   string
	Category Category
	Severity Severity
	Title    string
	Body     string
}

// IsRequest reports whether the finding asks the contributor to act.
func (f Finding) IsRequest() bool {
	return f.Category != CategoryFact
}

// Input is everything a signal may read. Now defaults to the current
// time and exists so tests can pin the clock.
type Input struct {
	PR     *forge.PullRequest
	Policy *rules.Policy
	Now    time.Time
}

// Signal evaluates one aspect of a pull request. A signal may return
// findings together with an error when part of its evaluation failed.
type Signal interface {
	ID() string
	Evaluate(ctx context.Context, in Input) ([]Finding, error)
}

// Run evaluates every signal in order and returns every finding they
// produced, including findings a signal returned alongside an error.
// Every error is returned joined after the remaining signals have run.
func Run(ctx context.Context, sigs []Signal, in Input) ([]Finding, error) {
	if in.Now.IsZero() {
		in.Now = time.Now()
	}
	var findings []Finding
	var errs []error
	for _, s := range sigs {
		if err := ctx.Err(); err != nil {
			return findings, fmt.Errorf("run signals: %w", err)
		}
		out, err := s.Evaluate(ctx, in)
		findings = append(findings, out...)
		if err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", s.ID(), err))
		}
	}
	return findings, errors.Join(errs...)
}

// Defaults returns every signal. The disclosure signal runs last so a
// slow judge never costs a deterministic finding. A nil judge skips it.
func Defaults(judge disclosure.Judge) []Signal {
	return []Signal{
		linkedIssueRequired{},
		linkedIssueNotAccepted{},
		filesWithoutIssue{},
		largeChange{},
		ciFailing{},
		thinDescription{},
		declaredBot{},
		agentBranchPrefix{},
		aiTrailer{},
		modelCommitEmail{},
		activityBurst{},
		aiDisclosureRequired{judge: judge},
	}
}
