package forge

import (
	"context"
	"slices"
	"strings"
	"sync"
)

// FakeForge serves pull requests from memory and records comments. It is
// safe for concurrent use. The zero value is empty and ready to use.
type FakeForge struct {
	mu       sync.RWMutex
	prs      map[string]*PullRequest
	Comments map[string][]string
	Policies map[string][]byte
	Err      error
}

// PolicyFile returns the policy stored under path, ignoring ref.
func (f *FakeForge) PolicyFile(_ context.Context, _, _, _, path string) ([]byte, bool, error) {
	if f.Err != nil {
		return nil, false, f.Err
	}
	f.mu.RLock()
	defer f.mu.RUnlock()
	data, ok := f.Policies[path]
	return data, ok, nil
}

// Add stores a pull request under its owner, repo, and number.
func (f *FakeForge) Add(pr *PullRequest) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.prs == nil {
		f.prs = map[string]*PullRequest{}
	}
	f.prs[Ref{Owner: pr.Owner, Repo: pr.Repo, Number: pr.Number}.String()] = clonePullRequest(pr)
}

// PullRequest returns an independent copy of the stored pull request,
// or ErrNotFound. When Err is set every call returns it.
func (f *FakeForge) PullRequest(_ context.Context, owner, repo string, number int) (*PullRequest, error) {
	if f.Err != nil {
		return nil, f.Err
	}
	f.mu.RLock()
	defer f.mu.RUnlock()
	pr, ok := f.prs[Ref{Owner: owner, Repo: repo, Number: number}.String()]
	if !ok {
		return nil, ErrNotFound
	}
	return clonePullRequest(pr), nil
}

// UpsertComment replaces the stored comment containing marker, or
// appends one when create is true.
func (f *FakeForge) UpsertComment(_ context.Context, owner, repo string, number int, marker, body string, create bool) error {
	if f.Err != nil {
		return f.Err
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.Comments == nil {
		f.Comments = map[string][]string{}
	}
	key := Ref{Owner: owner, Repo: repo, Number: number}.String()
	for i, c := range f.Comments[key] {
		if strings.Contains(c, marker) {
			f.Comments[key][i] = body
			return nil
		}
	}
	if create {
		f.Comments[key] = append(f.Comments[key], body)
	}
	return nil
}

func clonePullRequest(pr *PullRequest) *PullRequest {
	out := *pr
	out.Author.RecentEventTimes = slices.Clone(pr.Author.RecentEventTimes)
	out.Files = slices.Clone(pr.Files)
	out.Commits = slices.Clone(pr.Commits)
	out.LinkedIssues = make([]Issue, len(pr.LinkedIssues))
	for i, is := range pr.LinkedIssues {
		out.LinkedIssues[i] = Issue{Number: is.Number, Labels: slices.Clone(is.Labels), Open: is.Open}
	}
	return &out
}
