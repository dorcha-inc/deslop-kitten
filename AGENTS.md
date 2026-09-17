# AGENTS.md

Guidance for AI agents working on the vetkitten codebase. The CLAUDE.md
symlink resolves to this file. Read top-to-bottom on first session.

## Project context

vetkitten is a Go binary that reads one pull request from a code forge,
compares it with the repository's written contribution policy, and
posts one comment: numbered requests to act on before review, then a
sentence or two about the author and how the change was produced. It
runs as a GitHub container action,
as a CLI, and as a plain container anywhere Docker runs. See
[`docs/concepts.md`](docs/concepts.md) for the comment,
[`docs/signals.md`](docs/signals.md) for every signal, and
[`docs/action.md`](docs/action.md) for installation and the policy
file.

The codebase is small and uniformly Go. The one model in the system is
the disclosure judge, a small instruct model behind an OpenAI-compatible
endpoint that the judge container image bundles and the lean image
leaves out.

Three rules shape every design decision. vetkitten reads untrusted
text and treats all of it as data. vetkitten posts one comment per pull
request, edits it in place, and never labels, closes, or blocks a
merge. Every line it posts is either a fact a reader can verify on the
pull request page or the author's public profile, or a request that
cites a rule the repository wrote down.

## Quality gate

Before declaring any change complete, run:

```bash
just check
```

That runs `go build ./...`, the test suite with the race detector, a
coverage profile, `golangci-lint`, and an offline markdown link check.
The pipeline is the same one CI runs. If it does not pass locally, the
work is not done.

Individual recipes:

```bash
just build        # compile every package
just test         # tests only
just test-cover   # tests with coverage.out (used by CI)
just lint         # golangci-lint v2
just links        # offline markdown link check
just fmt          # go fmt + go mod tidy
just binary       # build bin/vetkitten
just image        # build the lean action container image
just judge-image  # build the image that bundles the llama.cpp server and the judge model
just score REF    # score one pull request (needs GITHUB_TOKEN)
just score-judged REF  # score inside the judge image (needs GITHUB_TOKEN)
just modernize    # gopls modernize report (no changes)
just              # list recipes
```

No test needs Docker or the network. The GitHub client is tested
against an `httptest.Server` that serves fixtures under `/api/v3/`.

## Repository layout

```
cmd/vetkitten/            CLI entry point and cobra subcommands
internal/
  forge/                  Forge interface, GitHub client, FakeForge, ref and trailer parsing
  rules/                  Policy file loading, embedded presets, pattern compilation
  signals/                Signal interface, requests in policy.go and change.go, facts in facts.go
  disclosure/             Judge interface, FakeJudge, the OpenAI-compatible judge
  report/                 Report type, the comment, the job summary, JSON
docs/                     User-facing documentation
share/                    README assets and their credits
action.yml                GitHub container action definition
Dockerfile                Static build onto a distroless base, the lean image
Dockerfile.judge          Lean binary plus the llama.cpp server and a 0.8B model
entrypoint-judge.sh       Starts the server, then runs the scorer against it
```

There is no `pkg/`. The binary is the public surface.

Dependencies flow one way. `forge`, `rules`, and `disclosure` know
nothing about signals. `signals` reads `forge` and `rules` types, calls
the `disclosure.Judge` interface, and never touches the network itself,
which `.golangci.yml` enforces with depguard. `report` reads `signals`
and `rules`. `cmd` wires everything together.

## Writing prose

The following style rules apply to all prose in the repo: README,
docs, design notes, commit message bodies, code comments. They were
established by the maintainer's stated preference and must be
honored.

### Hard rules

- **No em-dashes.** The character `—` does not appear in prose. If
  you would normally use an em-dash, split into two sentences or use
  a comma. The same goes for en-dashes (`–`) in prose.
- **No semicolons in prose.** Use a period and start a new sentence.
- **No unnecessary parentheses.** Parenthetical asides that pause the
  reader for a thought you could have put in its own sentence should
  go in its own sentence. A one-word gloss such as an abbreviation on
  first use is fine. Anything longer is a sentence you have not
  written yet, and parens are never a substitute for a comma or a
  period.
- **Don't pack settings into parentheses.** A run of
  `NAME (default X), OTHER (default Y)` is unreadable, and a second
  clause tacked on after a semicolon makes it worse. Put environment
  variables, flags, and their defaults in a table with a column for
  the name, the default, and the meaning.
- **No ASCII diagrams.** Describe relationships in prose. ASCII boxes
  and arrows are hard to maintain and rarely earn their space.
- **No emoji** unless the user explicitly asks for them.
- **Don't define a thing by what it is not.** "X, not Y" and its
  variants, "X rather than Y", "X and never Y", tell the reader what
  to erase instead of what to hold. Say what the thing does and stop.
  "The audit is a visibility aid, not an access gate" becomes "The
  audit records every write and lets all of them through." State a
  limit outright when it is load-bearing, as its own sentence, in
  terms of what happens: "The scorer still writes the report."
- **No vague back-references.** Do not open a sentence with "This",
  "That", "These", "Those", "Their", or "It" pointing at a noun from
  an earlier sentence. Name the noun again. "This is the wrong
  baseline" becomes "The best region in hindsight is the wrong
  baseline." The reader should never have to look backward to resolve
  what a pronoun stands for.

### Soft rules

- Write short, direct sentences. If a sentence has more than one
  comma, consider whether it should be two sentences.
- Lead with the noun, not the qualifier. "The signal reads the branch
  name" beats "When a pull request arrives, the signal reads the
  branch name."
- Define jargon on first use, even if you think the reader knows it.
- Do not write in fragments or in a punchy, aphoristic style. Short
  clipped clauses strung together read like a parable, not like
  documentation. "No key, no cost, runs on a laptop" is wrong.
  "Ollama runs models locally, so it needs no API key and costs
  nothing to call" is right.
- Don't write multi-paragraph code docstrings. One short paragraph
  per exported identifier is the ceiling.

### Examples

Wrong:

> The loader applies the default preset (every field, every pattern)
> when the file is absent; the policy flag then decides whether a
> missing file is an error.

Right:

> The loader applies the default preset when the file is absent. A
> missing file at a path the user named is an error.

Wrong:

> A finding carries the signal id, category, and severity, with the
> evidence naming the value that fired it. Evidence is optional, Message
> is required.

Right:

> A finding carries the signal id, category, and severity, with the
> evidence naming the value that fired it. Evidence is optional. Message
> is required.

## Writing comments

The rules under "Writing prose" apply, plus:

- **Default to writing no comment.** A well-named identifier and a
  short function explain themselves. Only comment when the WHY is
  non-obvious: a hidden constraint, a subtle invariant, a workaround
  for a specific upstream bug, behavior that would surprise a reader.
- **Don't describe what the code does.** The code does that.
  "Increments the counter" above `c++` is noise.
- **Describe what the code does do, not what it avoids.** A comment
  that lists rejected alternatives or absent behavior goes stale the
  moment the code changes. State the behavior that exists and the
  reason for it. Name an absence only when it is a load-bearing
  constraint a maintainer would otherwise violate, such as a retry
  that must not happen, and give the why. Write that absence as
  behavior, not as a contrast. "The mapper falls back to the static
  mapping on any error" beats "the mapper is a hint, not a gate."
- **Don't reference the past.** "Renamed from X", "formerly Y",
  "was previously fooBar" all rot. Comments describe the present
  state. If the reader needs migration history they can `git log`.
- **Don't reference callers or PRs.** "Called by X", "added for
  the Y flow", "see issue #123" all rot as the codebase evolves.
  Caller context belongs in the PR description.
- **Don't write multi-line comment banners.** Use one short comment
  per declaration, not a docblock with `@param`/`@returns`/`@example`
  decoration. Go doc tooling reads the comment line directly above
  the symbol.
- **Write comments as prose a human reads top to bottom.** Make them
  correct, clear, and concise. No unnecessary parentheses, semicolons,
  colons, or dashes. When a parenthetical or a clause after a colon
  carries real weight, give it its own sentence instead. The em-dash
  and the semicolon are banned in comments as they are in all prose.
- Package docs go in one file per package and explain the package's
  purpose in a paragraph or two. They are an exception to the
  "default to no comment" rule.

### Doc comment form

When a comment is warranted, for an exported identifier, a package
doc, or a non-obvious why, write it the way `gofmt` and `go doc`
expect. The full guide is [Go Doc Comments](https://go.dev/doc/comment).
The conventions that matter here:

- Put the comment on the line directly above the declaration with no
  blank line between them.
- Begin a doc comment with the name of the thing it documents so it
  reads as a sentence. `// Acquire blocks until a slot frees.` reads
  better than `// blocks until a slot frees.`
- Use complete sentences and end them with a period. A package comment
  starts with `Package <name>`.
- Say what a function returns or does for the caller, not how it works
  inside. Use "reports whether" for a boolean result, never "returns
  true if ... or not".
- Name parameters and results directly in the text. No backticks.
- State a type's concurrency guarantees and any useful zero value when
  they are not obvious.
- Mark a removal with a `Deprecated:` paragraph so tooling can flag it.

## Writing documentation

The docs under `docs/`, the tutorials on the site, and every example
README are read top to bottom by someone following along. The prose
rules above apply, plus a few that matter for a page with commands in
it.

- **Explain every code block.** Never drop a command or snippet
  without saying what it does and what every meaningful flag means.
  Show output too, and say what its columns or fields mean.
- **Headings name content, and are not narration.** "Dynamic stage
  mapper" and "Cost, latency, and feedback" are good. "Now we register
  the backends" narrates the act instead of naming the subject. "What
  makes a workflow static" poses a question the prose should just
  answer.
- **Do not over-chunk.** A heading breaks the reader's flow. Add one
  only where a genuinely new section begins. Merge two short sections
  that are really one idea and let a sentence carry the transition.
- **Cut filler.** Remove words that earn nothing. If "static" already
  carries the meaning, do not also write "fixed". Do not lean on one
  adjective across a passage.

## Writing tests

### Structure

Use table-driven tests with subtests when there are multiple cases of
the same shape. Use standalone `Test<X>_<Scenario>` functions when
there is one case or each case is shaped differently.

```go
tests := []struct {
    name string
    in   T
    want U
}{
    {name: "descriptive case", in: ..., want: ...},
}
for _, tt := range tests {
    t.Run(tt.name, func(t *testing.T) {
        // ...
    })
}
```

Always write struct literals inside table cases with field names.
Positional literals break silently when the struct grows a field,
and the failure shows up as a test passing on the wrong input
rather than as a compile error. The same rule applies to any
struct literal where the field types do not uniquely identify the
field by position.

### Naming

`Test<Type>_<Scenario>` or `Test<FunctionName>_<Scenario>`. The
scenario reads as a clause: `TestParseTrailers_ReadsFinalParagraph`,
`TestFakeForge_ReturnsIndependentCopy`,
`TestLoad_RejectsBadInput`.

### Assertions

`github.com/stretchr/testify` is the assertion library.

- `require` for preconditions and setup that must succeed before the
  rest of the test makes sense. Failure halts the test.
- `assert` for the actual checks under test. Failure records the
  failure and continues so the test surfaces every problem in one
  run.

### Mocks and fakes

Use hand-written fakes with the suffix `Fake*` (e.g. `FakeForge`).
Do not introduce a mocking code generator.

Share one fake per interface across the package's tests. If a fake
already exists for an interface, extend it rather than hand-rolling a
second.

For HTTP-shaped dependencies, prefer `httptest.NewServer` or a
hand-written fake server function that returns a *httptest.Server.
Both are easier to debug than generated mocks.

Every Fake satisfies the same expectations as the real implementation.
`FakeForge` and the GitHub client both return `ErrNotFound` for a
missing pull request and both return independent copies, and
`internal/forge/forge_test.go` covers both.

### What to test

Write a test when it would catch a regression someone would care
about, and skip it otherwise. A test earns its place when it pins one
of these:

- A contract with an outside system. The GitHub client against fixture
  responses, the judge request against a fake chat server, the comment
  upsert editing instead of duplicating.
- A behavior a maintainer or contributor sees. Which signals fire for a
  human pull request and for an agent pull request under each preset,
  when a comment is posted, the order and grouping inside the comment.
- A safety property. A fabricated judge quote is rejected, a trusted
  bot produces no facts, a failing judge leaves every other finding in
  place.
- A boundary that rejects input. Unknown policy keys, negative
  thresholds, a regular expression that does not compile.

Do not write tests that restate a constant, exercise a getter,
round-trip a struct through the standard library, or check that a fake
stores what it was given. One end-to-end test per command against
`FakeForge` covers the wiring. When a test needs a fixture, extend the
existing one instead of adding a second.

## Commit messages

[Conventional commits](https://www.conventionalcommits.org/en/v1.0.0/),
one sentence each, no body unless absolutely necessary.

```
feat: add a signal for lockfile-only churn
fix: treat a cancelled check run as failing CI
docs: add the curl preset to the policy table
refactor: move trailer parsing into its own file
chore: pin the distroless base image by digest
style: replace em-dashes with commas in Go comments
test: cover the preset chain depth limit
```

Rules:

- One sentence subject. No imperative-vs-past-tense pedantry, just
  pick one and be consistent. The examples above use present-tense
  imperative.
- Lowercase the type and the first word after the colon, unless that
  first word is a proper noun or acronym.
- No commit body unless the change is non-obvious and the reason
  cannot fit in the subject. Don't pad with a "Test plan" or
  "Summary" boilerplate.
- **No `Co-Authored-By: Claude` trailer.** Ever. Even when the user
  has authorized commits in advance.
- Do not amend or rewrite published commits without explicit user
  consent. Force-push only with `--force-with-lease`, only on a
  feature branch, and only after confirming with the user.

## Git practices

- **Never commit without asking. Never push without asking.** Both are
  separate acts and each needs its own approval. Propose the commit,
  name the files and the message, then wait for the user to say yes.
  The user verifies the results themselves before saying yes, so leave
  the work uncommitted until they do.
- **Approval never carries forward.** A yes for one commit is not a
  yes for the next one, and a yes to commit is not a yes to push. Ask
  again every time, however routine the change looks and however many
  times the user has already agreed in the session.
- **Approval to do work is not approval to commit it.** "Sounds good",
  "go ahead", and "yes" in reply to a plan mean write the code. They
  do not mean commit it, and they never mean push it. When the user
  approves an implementation, finish it, run the quality gate, and
  stop with a clean summary and a commit proposal.
- Use whatever git identity the user has configured. Never pass
  `-c user.email` or `-c user.name`.
- For PR merges, prefer `gh pr merge <num> --squash --delete-branch`.
- Before any destructive operation (`git reset --hard`, force-push,
  `git rm -r .`, branch delete), confirm with the user. Use
  `--force-with-lease` not `--force` when force-pushing is
  authorized.


## Go style

### Principles

Follow the priorities of [Google's Go style guide](https://google.github.io/styleguide/go/):
code should be correct first, clear second, and concise third. Never
trade an earlier property for a later one. Judge clarity from the
reader's seat, not the writer's.

Aim for the simplest design that solves the problem. Simple sometimes
means clever. A well-chosen invariant can delete a page of special
cases. Simple never means convoluted. Code that a reader must
simulate in their head before trusting it is not simple yet.

When two designs are otherwise equal, pick the one with less
mechanism. Fewer moving parts, fewer knobs, less code, and less for
an operator to run and monitor.

Avoid grab-bag packages. Names like util, common, and helpers say
nothing about what a package contains, and both the
[Go blog](https://go.dev/blog/package-names) and Google's style guide
reject them. Shared code belongs in the package that owns its
concept.

### Naming

- Exported: PascalCase. Unexported: camelCase. Acronyms stay
  uppercase: `LLMBackend`, `APIKeyEnvVar`, `CostUSD`.
- One main type per file. Mocks/fakes in `fake_*.go` or `mock_*.go`.
  Package-shared types in `types.go` if there is no obvious home.
- Receivers are one-letter abbreviations for the type name (`g` for
  `*GitHub`, `f` for `*FakeForge`, `r` for `Report`).

### Errors

- Wrap with `fmt.Errorf("operation: %w", err)`. Include enough
  context that the caller can tell which operation failed without
  reading the trace.
- Error strings start lowercase and end without punctuation. They
  read as the leaf of a wrapped chain. `fmt.Errorf("decode body:
  %w", err)` chains cleanly into `"forge: decode body: ..."`,
  whereas a capitalized or period-terminated leaf produces ugly
  joins.
- Return sentinel errors (`var ErrNotFound = errors.New(...)`) for
  conditions callers branch on. Compare with `errors.Is`, never with
  string matching. Document the sentinel in the package doc.
- Use `errors.As` for typed-error inspection. For multiple errors
  from concurrent work, `errors.Join` is the stdlib aggregator. Do
  not write hand-rolled multi-error types.
- Validation errors should describe both the constraint and the bad
  input: `"max_concurrency must be >= 1, got 0"`.
- Don't swallow errors. If an operation can fail and you choose to
  proceed anyway, log at warn level with enough context that an
  operator can investigate.
- Don't encode failure as a sentinel value of the result type. A
  function that returns `-1` or `""` or `NaN` to mean "not found"
  forces every caller to remember the rule. Return `(T, bool)` for
  lookups and `(T, error)` for fallible work.

### Context

Pass `context.Context` as the first parameter on every function that
touches the network, calls the judge, or starts a goroutine. Plumb
the command context from cobra all the way down to the GitHub API call
so cancellation and the `--timeout` deadline propagate.

Never store a context on a struct field. A struct that needs
cancellation gets a `Shutdown(ctx context.Context)` method instead.

`context.Background()` is reserved for top-level main and tests.
Anywhere else it indicates a missing plumb.

### Logging

`log/slog` everywhere. Use structured key-value attrs, not
formatted strings.

```go
slog.Default().Warn("score: some signals failed",
    "ref", ref.String(),
    "err", err,
)
```

Logs go to stderr so stdout stays clean for the report, which a
workflow may pipe into another tool.

### Concurrency

- Protect shared state with `sync.RWMutex`. Use `RLock`/`RUnlock` for
  read-only access.
- `defer mu.Unlock()` immediately after `mu.Lock()`. Keep the locked
  region small.
- Never write to a map under `RLock`. If you find yourself wanting
  to, restructure or upgrade to `Lock`.
- Channels for ownership transfer, mutexes for protecting fields.
  Don't use channels as locks.
- When passing maps across goroutine boundaries, copy them at the
  boundary unless the producer documents the no-mutate contract.
  `maps.Clone` is the one-line stdlib helper.
- Do not copy a struct that contains a `sync.Mutex`, a
  `sync.RWMutex`, or an `atomic.*` value. Pass it by pointer through
  arguments and return values. `go vet` catches most cases, and the
  resulting bug under load looks like phantom unlock failures.

### Goroutines

Every goroutine has a documented exit. Either it returns when a
context is cancelled, it reads from a channel the owner closes, or
the caller calls a `Stop` or `Shutdown` method that joins it. Never
start a goroutine from `init`, from a constructor that has no
`Stop`, or from a command without a way to wait on it.

The scorer runs its signals sequentially today and starts no long-lived
goroutine. A future webhook server introduces the first one, and it
follows this rule from its first commit.

### Type assertions

Always use the comma-ok form: `v, ok := x.(T)`. A bare `x.(T)`
panics on mismatch and crashes the run. The same rule applies to
map reads where the absent case is meaningful: `v, ok := m[k]`.

### Interfaces

Define an interface in the package that consumes it, not the package
that implements it. Add an interface only when there is a real
second implementation or a real fake. Producers return concrete
types so callers see the full surface.

`forge.Forge` is the right pattern: defined in `forge` because both
`GitHub` and `FakeForge` live there and serve consumers in `cmd` and in
the `signals` tests. `disclosure.Judge` follows the same shape.

### Optional fields

Use pointer types for optional values: `*int`, `*float64`,
`*string`. nil means "not set".

`rules.File` is the canonical example. Every scalar is a pointer and
every list is a slice, so a policy file distinguishes two states:

- absent field in YAML: the pointer or slice is nil. The preset value
  stands.
- present field, including an empty list or an empty string: the value
  overrides the preset.

`rules.Policy` is the resolved form with plain types. Code past the
loader never sees a pointer.

### Deferred Close

On anything writable, do not write `defer x.Close()` without
inspecting the error. A `*sql.Tx`, a buffered writer, or a flushable
sink can fail at `Close` and lose writes. Use a deferred closure
that captures a named return so the error surfaces:

```go
func write(...) (err error) {
    f, err := os.Create(path)
    if err != nil { return err }
    defer func() {
        if cerr := f.Close(); cerr != nil && err == nil {
            err = cerr
        }
    }()
    ...
}
```

Read-only handles are exempt. A `defer body.Close()` on an HTTP
response body is fine.

### Validation

Validate at four boundaries:

1. The cobra command before any work runs: the reference, the format,
   the numeric flags.
2. The rules loader when reading a policy file: unknown keys, negative
   limits, empty prefixes, patterns that fail to compile.
3. The forge when decoding an API response into the data model.
4. `signals.Run` when accepting an `Input`: a nil pull request or a nil
   policy is an error before any signal runs.

Past these boundaries values are trusted and not re-checked. A signal
reads `in.Policy.MinAccountAgeDays` and never re-validates it.

### Failure handling

The scorer runs inside someone else's CI. Fail loud at startup. Degrade
at evaluation time. These apply at different layers.

At startup a bad flag, an unreadable policy file the user named, or a
judge that never becomes ready is an error before any request leaves
the process. A maintainer sees the failure on the first run and fixes
the configuration.

At evaluation time, degrade. A failing signal keeps whatever findings
it produced, its error is joined and logged at warn level, every other
signal runs, and the report is still written. A judge failure leaves
the disclosure signal on the policy's patterns. The process exits zero,
because the report is advice and a scoring failure must never block a
pull request.

Never split the difference by tolerating a required input at startup
and then failing on the first pull request that needs it. Validate it
at the flag boundary or make it a genuinely optional input that no-ops
when absent, the way the disclosure signal treats a nil judge.

### Don't reinvent the wheel

Before writing any non-trivial logic, look for something already
built. Search the standard library first, then the deps already in
go.mod, then the wider module ecosystem. A mature package is almost
always more correct and better tested than a version written under
deadline, and every line not written is a line nobody has to review,
test, or maintain. Hand-rolling what a library already solves does
not just cost the lines, it adds a design of our own that we now own
forever.

Judge a candidate dependency by its maintenance and adoption. Recent
releases, responsive maintainers, wide use in serious projects, and a
focused scope are the signals that matter. Stars and download counts
are hints, not verdicts. A good dependency is welcome. Reinvent only
when nothing fits or the dependency would weigh far more than the
problem it solves.

Go 1.26's standard library covers most of the helpers a new contributor
would otherwise hand-roll. Reach for these before writing your own:

- `maps.Clone(m)` for shallow map copy. `maps.Keys(m)` returns an
  iterator. `slices.Sorted(maps.Keys(m))` gives a sorted slice.
- `slices.Sort`, `slices.Contains`, `slices.Concat`, `slices.Collect`
  are the modern replacements for hand-rolled loops.
- `cmp.Or(a, b, c)` picks the first non-zero value. The common
  `if x == "" { x = fallback }` pattern collapses to one line.
- `errors.Is`, `errors.As`, `errors.Join` for error introspection
  and aggregation.
- `sync.OnceFunc`, `sync.OnceValue` for one-shot initialization.
- `context.WithTimeout`, `context.AfterFunc` for cancellation.
- `golang.org/x/sync/errgroup` for parallel goroutines with shared
  cancellation. Use it instead of hand-rolling
  done-channels-plus-select-on-context.

Already-imported deps to use rather than re-implementing:

- `github.com/google/go-github/v80` for every GitHub API call. Use its
  `Get*` accessors on response structs so a nil pointer never panics.
- `github.com/goccy/go-yaml` for policy files, decoded in strict mode
  so an unknown key is an error.
- `github.com/spf13/cobra` for commands and flags.
- `github.com/openai/openai-go` for the judge, pointed at any
  OpenAI-compatible endpoint through `option.WithBaseURL`.
- `github.com/stretchr/testify` for assertions in tests.


## Working with the user

### Risk and reversibility

Carefully consider the blast radius of every action. Local, reversible
actions (edit a file, run a test) need no preamble. Hard-to-reverse
actions (force-push, drop database, delete a branch, modify a shared
configuration) need explicit user confirmation each time.

Authorization for a single action does not extend to similar actions.
A `git push` approved once is not blanket approval for every future
push.

### Confirmation patterns

- Lay out a plan before doing destructive multi-step work. Get a
  green light, then execute.
- After every destructive step, summarize the state. The user often
  wants to verify before authorizing the next step.
- When you spot a side-effect the user didn't ask for (cleanup,
  refactor, lint fix), name it and ask before doing it. Do not slip
  it into a commit silently.

### Communication style

- Default to terse. The user reads diffs and can see what changed.
- Lead with the result, then the details if asked. Don't bury the
  headline under a recap of the process.
- One-sentence end-of-turn summary: what shipped and what is next.
  Never longer than two sentences.
- Don't restate the user's request back to them. Don't say "Great
  question" or "Let me help with that". Just answer.
- When you've already done a task, don't describe it in the past
  tense; the diff already documents it.

### Scope

Match the scope of your changes to what the user asked. A bug fix
does not get a free refactor of the surrounding code. A one-shot
script does not need a helper module.

If a side-improvement is genuinely small and obvious (one line,
zero behavior change), do it without ceremony. If it is more than
that, surface it as a separate option for the user to opt into.


## Signals and the comment

A signal is a type in `internal/signals` with an `ID` method that
returns a stable snake_case identifier and an `Evaluate` method that
reads its `Input` and returns findings. `Input` carries the pull
request, the resolved policy, and the clock. A signal reads nothing
else, reads no file, and calls no other signal. The disclosure signal
is the one signal that calls out, and it does so only through the
`disclosure.Judge` interface it was constructed with.

A finding is a request or a fact. A request appears as a numbered item
at the top of the comment. A fact appears as a clause in the sentence
about how the change was produced.

- **Silent when nothing applies.** Return `nil, nil` when the policy
  disables the check or the pull request carries nothing to report.
  Only a genuine evaluation failure returns an error.
- **A request opens with a short imperative** in `Title`, such as
  "Disclose AI assistance." or "Consider splitting.", and its `Body`
  gives the fact, then the rule the repository set, then the action, in
  one or two sentences. A request exists only for a rule the policy
  declares or a property of the change the contributor can fix.
- **A fact is a clause.** `Body` is a lowercase clause such as "the
  head branch `copilot/x` carries a prefix used by AI coding agents",
  and the report joins the clauses with "and" into one sentence. Every
  clause names a number, a date, or a string in a code span that a
  reader can verify on the pull request page or the author's public
  profile. A fact never characterizes the author.
- **Tables are out.** A reader scans a table to reconstruct the
  sentence a reviewer would have said. Say the sentence.
- **No at-signs.** Logins go in code spans. An at-sign would notify the
  person and turn a fact into a summons.
- **Severity orders requests** within their group, policy before change.
  A signal never reads another signal's output.
- **Policy knobs live in the policy.** A threshold a maintainer might
  want to change belongs in `rules.Policy` with a preset default. A
  constant only a developer would change stays in the signal file.

### Adding a signal

1. Add the type to `policy.go` for a request that cites a policy field,
   `change.go` for a request about the change itself, or `facts.go` for
   a clause about how the change was produced.
2. Add it to `Defaults` in `signal.go`, requests before facts.
3. If it needs a knob, add the field to `rules.File` and `rules.Policy`,
   thread it through `overlay`, validate it, and set a value in
   `presets/default.yaml`.
4. Extend `humanPR` and `agentPR` in `signals_test.go` so one of them
   fires the signal and the other stays silent, and update the expected
   list in the agent test.
5. Add a row to the table in `docs/signals.md` and, for a new knob, a
   row to the policy table in `docs/action.md`.

### Adding a preset

A preset is a YAML file under `internal/rules/presets/` that encodes a
project's written policy. Read the policy document itself, set only the
fields it supports, put its URL in `source`, and name the document in
the leading comment. A rule the document does not state does not go in
the preset. Every embedded preset is loaded by
`TestPreset_EveryEmbeddedPresetResolves`, and a named preset without a
`source` fails that test.

## The disclosure judge

`disclosure.Judge` takes an `Overview` and returns a `Verdict` that
says whether the description discloses model assistance and quotes the
sentence that does. The disclosure signal in `internal/signals/policy.go`
is the only caller.

Decisions that hold:

- **The judge decides one question.** Does the description state that
  a model helped produce the change. Reputation, size, and policy fields
  are facts the rules compare, and no model reads them.
- **Every yes carries a verified quote.** `OpenAIJudge` checks that the
  quoted sentence appears in the overview after whitespace and case
  normalization and returns `ErrUnverifiedQuote` otherwise. A yes
  without a quote never reaches a finding.
- **Pluggable at the command line.** `--judge-url` takes any
  OpenAI-compatible base URL and `--judge-model` names the model. An
  empty URL disables the judge and the disclosure signal uses the
  policy's patterns.
- **Permissive license only.** The bundled default is Qwen3.5-0.8B
  under Apache 2.0. `MODEL_URL` on `Dockerfile.judge` swaps it for any
  GGUF the llama.cpp server loads.
- **Structured output at temperature zero.** The request carries a
  strict JSON schema so the answer is two fields and nothing else.
- **Thinking off, answer capped.** The request passes
  `chat_template_kwargs.enable_thinking = false` and a 300 token cap. A
  thinking model left on reasons until the context fills and never
  answers, which cost ten minutes per pull request before this was
  found. An answer that hits the cap is `ErrTruncatedAnswer`, and the
  signal reports it instead of guessing.
- **Untrusted input.** The overview is outside text. An author can only
  make the judge say yes by writing a sentence that is a disclosure, so
  the injection surface is the request the author already wants
  granted.

## Adding a feature

A rough order:

1. **Types first.** Extend `forge.PullRequest` if the feature needs data
   the forge does not return yet, and extend `FakeForge` and the GitHub
   fixture server together.
2. **Policy next** if the feature needs a knob. `rules.File`,
   `rules.Policy`, `overlay`, validation, preset default.
3. **Signal** with a firing test and a silent test.
4. **Report** if the feature changes what the maintainer sees.
5. **Documentation** in `docs/signals.md` and `docs/action.md`.
6. **`just check`** until green. Fix everything it surfaces.

## When in doubt

Re-read this document, then the most recent code changes that touched
the same area. The patterns are intentionally consistent across the
codebase. Match them rather than introducing a new variation.
