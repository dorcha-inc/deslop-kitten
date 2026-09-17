package forge

import (
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"strings"
)

var shortRef = regexp.MustCompile(`^([A-Za-z0-9_.-]+)/([A-Za-z0-9_.-]+)#(\d+)$`)

var pullRef = regexp.MustCompile(`^refs/pull/(\d+)/`)

// Ref names one pull request on a forge.
type Ref struct {
	Owner  string
	Repo   string
	Number int
}

// String formats the reference as owner/repo#number.
func (r Ref) String() string {
	return fmt.Sprintf("%s/%s#%d", r.Owner, r.Repo, r.Number)
}

// RefFromActions builds the reference from the GITHUB_REPOSITORY and
// GITHUB_REF values a pull_request workflow run provides, where the ref
// has the form refs/pull/<number>/merge.
func RefFromActions(repository, ref string) (Ref, error) {
	owner, repo, ok := strings.Cut(repository, "/")
	if !ok || owner == "" || repo == "" {
		return Ref{}, fmt.Errorf("GITHUB_REPOSITORY must be owner/repo, got %q", repository)
	}
	m := pullRef.FindStringSubmatch(ref)
	if m == nil {
		return Ref{}, fmt.Errorf("GITHUB_REF must name a pull request as refs/pull/<number>/merge, got %q", ref)
	}
	n, err := strconv.Atoi(m[1])
	if err != nil || n <= 0 {
		return Ref{}, fmt.Errorf("pull request number must be positive, got %q", m[1])
	}
	return Ref{Owner: owner, Repo: repo, Number: n}, nil
}

// ParseRef accepts owner/repo#number or a GitHub pull request URL and
// returns the parsed reference.
func ParseRef(s string) (Ref, error) {
	s = strings.TrimSpace(s)
	if m := shortRef.FindStringSubmatch(s); m != nil {
		n, err := strconv.Atoi(m[3])
		if err != nil || n <= 0 {
			return Ref{}, fmt.Errorf("pull request number must be positive, got %q", m[3])
		}
		return Ref{Owner: m[1], Repo: m[2], Number: n}, nil
	}
	u, err := url.Parse(s)
	if err != nil || u.Host == "" {
		return Ref{}, fmt.Errorf("reference must be owner/repo#number or a pull request URL, got %q", s)
	}
	parts := strings.Split(strings.Trim(u.Path, "/"), "/")
	if len(parts) != 4 || parts[2] != "pull" {
		return Ref{}, fmt.Errorf("URL path must be /owner/repo/pull/number, got %q", u.Path)
	}
	n, err := strconv.Atoi(parts[3])
	if err != nil || n <= 0 {
		return Ref{}, fmt.Errorf("pull request number must be positive, got %q", parts[3])
	}
	return Ref{Owner: parts[0], Repo: parts[1], Number: n}, nil
}
