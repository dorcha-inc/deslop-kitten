package disclosure

import (
	"context"
	"sync"
)

// FakeJudge returns the verdict stored for an overview's title. A title
// with no stored verdict yields a verdict of not disclosed. When Err is
// set every call returns it. The zero value is ready to use and safe
// for concurrent use.
type FakeJudge struct {
	mu       sync.Mutex
	Verdicts map[string]Verdict
	Err      error
	Calls    []Overview
}

// Judge records the call and returns the canned verdict.
func (f *FakeJudge) Judge(_ context.Context, o Overview) (Verdict, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.Calls = append(f.Calls, o)
	if f.Err != nil {
		return Verdict{}, f.Err
	}
	if v, ok := f.Verdicts[o.Title]; ok {
		return v, nil
	}
	return Verdict{}, nil
}
