package forge

import (
	"regexp"
	"slices"
	"strconv"
	"strings"
)

var trailerLine = regexp.MustCompile(`^([A-Za-z0-9][A-Za-z0-9-]*):\s*(.+?)\s*$`)

var issueRef = regexp.MustCompile(`(?:^|[^A-Za-z0-9/&])#(\d+)\b`)

// ParseTrailers returns the git trailers of a commit message keyed by
// lowercased trailer name. The trailer block is the final paragraph of
// a message with at least two paragraphs, and every line in the block
// has the form "Key: value". A message with no such block yields an
// empty map.
func ParseTrailers(message string) map[string][]string {
	out := map[string][]string{}
	paragraphs := strings.Split(strings.TrimSpace(message), "\n\n")
	if len(paragraphs) < 2 {
		return out
	}
	last := strings.TrimSpace(paragraphs[len(paragraphs)-1])
	for line := range strings.SplitSeq(last, "\n") {
		m := trailerLine.FindStringSubmatch(strings.TrimSpace(line))
		if m == nil {
			return map[string][]string{}
		}
		key := strings.ToLower(m[1])
		out[key] = append(out[key], m[2])
	}
	return out
}

// IssueRefs returns the distinct issue numbers referenced as #N in text,
// in ascending order.
func IssueRefs(text string) []int {
	seen := map[int]bool{}
	for _, m := range issueRef.FindAllStringSubmatch(text, -1) {
		n, err := strconv.Atoi(m[1])
		if err != nil || n <= 0 {
			continue
		}
		seen[n] = true
	}
	refs := make([]int, 0, len(seen))
	for n := range seen {
		refs = append(refs, n)
	}
	slices.Sort(refs)
	return refs
}
