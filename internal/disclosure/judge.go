package disclosure

import (
	"context"
	"errors"
	"strings"
)

// ErrUnverifiedQuote reports that the judge quoted a sentence that does
// not appear in the overview.
var ErrUnverifiedQuote = errors.New("quoted sentence does not appear in the overview")

// Overview is the text the judge reads.
type Overview struct {
	Title string
	Body  string
}

// Verdict is the judge's answer. Quote is the sentence from the
// overview that discloses model assistance, and it is empty when
// Disclosed is false.
type Verdict struct {
	Disclosed bool
	Quote     string
}

// Judge decides whether an overview discloses model assistance.
type Judge interface {
	Judge(ctx context.Context, o Overview) (Verdict, error)
}

func containsNormalized(haystack, needle string) bool {
	needle = strings.ToLower(strings.Join(strings.Fields(needle), " "))
	if needle == "" {
		return false
	}
	haystack = strings.ToLower(strings.Join(strings.Fields(haystack), " "))
	return strings.Contains(haystack, needle)
}
