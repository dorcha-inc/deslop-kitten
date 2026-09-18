package report

import (
	"encoding/json"
	"fmt"
	"net/url"
	"slices"
	"strings"
	"time"

	"github.com/dorcha-inc/deslop-kitten/internal/forge"
	"github.com/dorcha-inc/deslop-kitten/internal/signals"
)

// CommentMarker is the hidden HTML comment that identifies deslop-kitten's
// comment on a pull request so reruns update it in place.
const CommentMarker = "<!-- deslop-kitten -->"

// heading names the tool and links its repository, since the comment is
// posted by the workflow's bot account and would otherwise be anonymous.
const heading = "### <img src=\"https://raw.githubusercontent.com/dorcha-inc/deslop-kitten/main/share/cat-icon.gif\" height=\"24\" alt=\"\"> [deslop-kitten](https://github.com/dorcha-inc/deslop-kitten)\n\n"

// Report is the result for one pull request.
type Report struct {
	Owner       string            `json:"owner"`
	Repo        string            `json:"repo"`
	Number      int               `json:"number"`
	RepoURL     string            `json:"repo_url,omitempty"`
	Title       string            `json:"title"`
	Author      forge.Account     `json:"author"`
	Findings    []signals.Finding `json:"findings"`
	GeneratedAt time.Time         `json:"generated_at"`
}

// Policy names the policy a report was checked against, for the footer
// of the comment. PolicyPath is empty when the repository has no policy
// file and the default preset applied.
type Policy struct {
	Preset     string
	Source     string
	PolicyPath string
}

// ShouldComment reports whether the pull request warrants a comment: a
// request or a fact exists, or the author has no merged pull request in
// this repository yet.
func (r Report) ShouldComment() bool {
	return len(r.Findings) > 0 || r.Author.MergedPullRequestsInRepo == 0
}

// Comment renders the pull request comment.
func (r Report) Comment(p Policy) string {
	return CommentMarker + "\n" + r.body(p)
}

// AllClear is the comment body that replaces an earlier comment once
// the author has resolved its requests.
func AllClear() string {
	return CommentMarker + "\n" + heading + "Every check passes on the latest commits.\n"
}

// Markdown renders the report for a job summary.
func (r Report) Markdown(p Policy) string {
	return fmt.Sprintf("## %s/%s#%d %s\n\n", r.Owner, r.Repo, r.Number, r.Title) + r.body(p)
}

// JSON renders the report as indented JSON.
func (r Report) JSON() ([]byte, error) {
	b, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("encode report: %w", err)
	}
	return b, nil
}

func (r Report) body(p Policy) string {
	var b strings.Builder
	b.WriteString(heading)
	requests := r.requests()
	if len(requests) == 0 {
		b.WriteString("Nothing to do before review.\n\n")
	}
	for i, f := range requests {
		fmt.Fprintf(&b, "%d. **%s** %s\n", i+1, f.Title, f.Body)
	}
	if len(requests) > 0 {
		b.WriteString("\n")
	}
	b.WriteString(r.authorSentence())
	if change := r.changeSentence(); change != "" {
		b.WriteString(" " + change)
	}
	b.WriteString("\n\n<sub>" + footer(p) + "</sub>\n")
	return b.String()
}

func (r Report) requests() []signals.Finding {
	var out []signals.Finding
	for _, f := range r.Findings {
		if f.IsRequest() {
			out = append(out, f)
		}
	}
	slices.SortStableFunc(out, func(a, b signals.Finding) int {
		if a.Category != b.Category {
			if a.Category == signals.CategoryPolicy {
				return -1
			}
			return 1
		}
		return b.Severity.Rank() - a.Severity.Rank()
	})
	return out
}

// authorSentence states the author's standing from the numbers on the
// public profile. Repository and follower counts join the sentence only
// for an account with no merged pull request anywhere, where they are
// the remaining evidence of a person behind the account.
func (r Report) authorSentence() string {
	a := r.Author
	login := "`" + a.Login + "`"
	if a.URL != "" {
		login = link(a.Login, a.URL)
	}
	if slices.ContainsFunc(r.Findings, func(f signals.Finding) bool { return f.Signal == "declared_bot" }) {
		return login + " is a bot account outside this repository's trusted list."
	}
	if a.MergedPullRequestsInRepo > 0 {
		s := login + " has " + link(count(a.MergedPullRequestsInRepo, "pull request", "pull requests")+" merged here", r.mergedHereURL())
		if elsewhere := a.MergedPullRequestsAnywhere - a.MergedPullRequestsInRepo; elsewhere > 0 {
			s += " and " + link(fmt.Sprintf("%d elsewhere on GitHub", elsewhere), r.mergedElsewhereURL())
		}
		return s + "."
	}
	lead := "First pull request here from " + login
	if a.PullRequestsInRepo > 1 {
		lead = fmt.Sprintf("%s has opened %s here, none merged yet", login, link(fmt.Sprintf("%d pull requests", a.PullRequestsInRepo), r.openedHereURL()))
	}
	var details []string
	if a.MergedPullRequestsAnywhere > 0 {
		details = append(details, link(fmt.Sprintf("%d merged elsewhere on GitHub", a.MergedPullRequestsAnywhere), r.mergedElsewhereURL()))
	} else {
		details = append(details, "no merged pull requests on GitHub yet")
	}
	if !a.CreatedAt.IsZero() {
		details = append(details, accountAge(a.CreatedAt, r.GeneratedAt))
		if a.MergedPullRequestsAnywhere == 0 {
			details = append(details, countOrNo(a.PublicRepos, "public repository", "public repositories"), countOrNo(a.Followers, "follower", "followers"))
		}
	}
	return lead + ", with " + joinClauses(details) + "."
}

// link renders text as a markdown link, or as plain text when the forge
// returned no URL.
func link(text, href string) string {
	if href == "" {
		return text
	}
	return fmt.Sprintf("[%s](%s)", text, href)
}

// openedHereURL, mergedHereURL, and mergedElsewhereURL point at the
// searches behind the numbers in the author sentence. They are empty
// without a repository URL, and the sentence then carries the numbers
// alone.
func (r Report) openedHereURL() string {
	if r.RepoURL == "" {
		return ""
	}
	return r.RepoURL + "/pulls?q=" + url.QueryEscape("is:pr author:"+r.Author.Login)
}

func (r Report) mergedHereURL() string {
	if r.RepoURL == "" {
		return ""
	}
	return r.RepoURL + "/pulls?q=" + url.QueryEscape("is:pr is:merged author:"+r.Author.Login)
}

func (r Report) mergedElsewhereURL() string {
	u, err := url.Parse(r.RepoURL)
	if r.RepoURL == "" || err != nil {
		return ""
	}
	q := fmt.Sprintf("is:pr is:merged author:%s -repo:%s/%s", r.Author.Login, r.Owner, r.Repo)
	return u.Scheme + "://" + u.Host + "/search?type=pullrequests&q=" + url.QueryEscape(q)
}

func (r Report) changeSentence() string {
	var clauses []string
	for _, f := range r.Findings {
		if !f.IsRequest() && f.Signal != "declared_bot" {
			clauses = append(clauses, f.Body)
		}
	}
	if len(clauses) == 0 {
		return ""
	}
	s := joinClauses(clauses)
	return strings.ToUpper(s[:1]) + s[1:] + "."
}

func accountAge(created, now time.Time) string {
	days := int(now.Sub(created).Hours() / 24)
	switch {
	case days >= 365:
		return fmt.Sprintf("an account since %d", created.Year())
	case days <= 0:
		return "an account created today"
	default:
		return fmt.Sprintf("an account created %s ago", count(days, "day", "days"))
	}
}

func count(n int, singular, plural string) string {
	if n == 1 {
		return "1 " + singular
	}
	return fmt.Sprintf("%d %s", n, plural)
}

func countOrNo(n int, singular, plural string) string {
	if n == 0 {
		return "no " + plural
	}
	return count(n, singular, plural)
}

func joinClauses(parts []string) string {
	switch len(parts) {
	case 0:
		return ""
	case 1:
		return parts[0]
	case 2:
		return parts[0] + " and " + parts[1]
	default:
		return strings.Join(parts[:len(parts)-1], ", ") + ", and " + parts[len(parts)-1]
	}
}

func footer(p Policy) string {
	switch {
	case p.PolicyPath != "" && p.Source != "":
		return fmt.Sprintf("Policy: `%s`, preset `%s` ([source](%s))", p.PolicyPath, p.Preset, p.Source)
	case p.PolicyPath != "":
		return fmt.Sprintf("Policy: `%s`, preset `%s`", p.PolicyPath, p.Preset)
	default:
		return "Policy: default preset"
	}
}
