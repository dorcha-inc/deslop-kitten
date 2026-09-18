package forge

import (
	"context"
	"errors"
	"time"
)

// ErrNotFound reports that the forge has no pull request at the
// requested coordinates.
var ErrNotFound = errors.New("pull request not found")

// CheckStatus summarizes the CI checks on a pull request head.
type CheckStatus string

const (
	CheckUnknown CheckStatus = "unknown"
	CheckPassing CheckStatus = "passing"
	CheckFailing CheckStatus = "failing"
	CheckPending CheckStatus = "pending"
)

// Account describes the author of a pull request with the facts the
// author's public profile shows. CreatedAt is zero for an app actor
// that has no user record, and the profile counts are zero with it.
// RecentEventTimes holds the timestamps of the author's most recent
// public events, newest first, up to one page.
type Account struct {
	Login string
	// URL is the account's profile page, empty when the forge has no
	// user record for the login.
	URL                        string
	IsBot                      bool
	CreatedAt                  time.Time
	PublicRepos                int
	Followers                  int
	PullRequestsInRepo         int
	MergedPullRequestsInRepo   int
	MergedPullRequestsAnywhere int
	RecentEventTimes           []time.Time
}

// Commit is one commit on a pull request.
type Commit struct {
	SHA         string
	Message     string
	AuthorEmail string
	AuthoredAt  time.Time
}

// Trailers returns the git trailers of the commit message keyed by
// lowercased trailer name.
func (c Commit) Trailers() map[string][]string {
	return ParseTrailers(c.Message)
}

// File is one file touched by a pull request.
type File struct {
	Path      string
	Additions int
	Deletions int
	Patch     string
}

// Issue is an issue the pull request body references.
type Issue struct {
	Number int
	Labels []string
	Open   bool
}

// PullRequest is the forge-independent view of a pull request.
type PullRequest struct {
	Owner        string
	Repo         string
	Number       int
	Title        string
	Body         string
	Author       Account
	HeadBranch   string
	// RepoURL is the base repository's web page and HeadURL the head
	// branch's tree on its own repository. HeadURL is empty when the
	// head repository is gone, as after a deleted fork.
	RepoURL      string
	HeadURL      string
	CreatedAt    time.Time
	Files        []File
	Commits      []Commit
	Checks       CheckStatus
	LinkedIssues []Issue
}

// Additions returns the total added lines across all files.
func (p *PullRequest) Additions() int {
	n := 0
	for _, f := range p.Files {
		n += f.Additions
	}
	return n
}

// Deletions returns the total deleted lines across all files.
func (p *PullRequest) Deletions() int {
	n := 0
	for _, f := range p.Files {
		n += f.Deletions
	}
	return n
}

// Forge reads pull requests and policy files from a code host and
// maintains one comment per pull request.
type Forge interface {
	PullRequest(ctx context.Context, owner, repo string, number int) (*PullRequest, error)
	// PolicyFile returns the file at path on ref, or ok false when the
	// file does not exist. An empty ref means the default branch.
	PolicyFile(ctx context.Context, owner, repo, ref, path string) (data []byte, ok bool, err error)
	// UpsertComment edits the comment on the pull request whose body
	// contains marker, or creates one when none exists and create is
	// true.
	UpsertComment(ctx context.Context, owner, repo string, number int, marker, body string, create bool) error
}
