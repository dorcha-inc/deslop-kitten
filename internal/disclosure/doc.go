// Package disclosure decides whether a pull request overview states
// that a model helped produce the change. The Judge interface takes an
// Overview and returns a Verdict that carries the sentence the judge
// treated as the disclosure. The OpenAIJudge sends the overview to any
// OpenAI-compatible chat endpoint, such as a local llama.cpp server,
// and verifies that the quoted sentence appears verbatim in the
// overview before trusting the answer. The FakeJudge serves canned
// verdicts for tests.
package disclosure
