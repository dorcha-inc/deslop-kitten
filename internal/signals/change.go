package signals

import (
	"context"
	"fmt"
	"strings"

	"github.com/dorcha-inc/deslop-kitten/internal/forge"
)

const thinDescriptionWords = 20

const thinDescriptionDiffLines = 100

type largeChange struct{}

func (largeChange) ID() string { return "large_change" }

func (largeChange) Evaluate(_ context.Context, in Input) ([]Finding, error) {
	limit := in.Policy.LargeChangeFiles
	files := len(in.PR.Files)
	if limit <= 0 || files <= limit {
		return nil, nil
	}
	severity := SeverityMedium
	if files > 2*limit {
		severity = SeverityHigh
	}
	return []Finding{{
		Signal:   "large_change",
		Category: CategoryChange,
		Severity: severity,
		Title:    "Consider splitting.",
		Body:     fmt.Sprintf("%d files changed, +%d and -%d lines. This repository treats changes above %d files as large.", files, in.PR.Additions(), in.PR.Deletions(), limit),
	}}, nil
}

type ciFailing struct{}

func (ciFailing) ID() string { return "ci_failing" }

func (ciFailing) Evaluate(_ context.Context, in Input) ([]Finding, error) {
	if in.PR.Checks != forge.CheckFailing {
		return nil, nil
	}
	return []Finding{{
		Signal:   "ci_failing",
		Category: CategoryChange,
		Severity: SeverityMedium,
		Title:    "Fix CI.",
		Body:     "A check run on the head commit failed.",
	}}, nil
}

type thinDescription struct{}

func (thinDescription) ID() string { return "thin_description" }

func (thinDescription) Evaluate(_ context.Context, in Input) ([]Finding, error) {
	words := len(strings.Fields(in.PR.Body))
	lines := in.PR.Additions() + in.PR.Deletions()
	if words >= thinDescriptionWords || lines <= thinDescriptionDiffLines {
		return nil, nil
	}
	return []Finding{{
		Signal:   "thin_description",
		Category: CategoryChange,
		Severity: SeverityLow,
		Title:    "Describe the change.",
		Body:     fmt.Sprintf("The description has %d words for a diff of %d lines. Say what changed and why so reviewers can follow the diff.", words, lines),
	}}, nil
}
