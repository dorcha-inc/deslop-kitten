package forge

import (
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"strings"
)

var shortRef = regexp.MustCompile(`^([A-Za-z0-9_.-]+)/([A-Za-z0-9_.-]+)#(\d+)$`)

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
