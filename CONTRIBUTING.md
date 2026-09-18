# Contributing

Thank you for considering a contribution. This document covers the
mechanics. [AGENTS.md](AGENTS.md) covers the style of code and prose,
and every contributor reads it once.

## Code of conduct

This project follows the [Contributor Covenant](CODE_OF_CONDUCT.md).

## Getting started

1. Fork the repository on GitHub.
2. Clone your fork and add the upstream remote:

   ```bash
   git clone https://github.com/YOUR_USERNAME/vetkitten.git
   cd vetkitten
   git remote add upstream https://github.com/dorcha-inc/vetkitten.git
   ```

## Development setup

- Go 1.26 or newer. Check with `go version`.
- [just](https://just.systems), the task runner for every local pipeline.
- [golangci-lint](https://golangci-lint.run/) v2.
- [lychee](https://lychee.cli.rs/) for the offline link check.
- Docker, only to build the action image.

Verify the setup by running the full pipeline:

```bash
just check
```

That runs build, race tests, lint, and the link check. It is the same
gate CI runs.

## Making changes

1. Create a branch from `main` named `<your_github_username>/<topic>`.
2. Write or update tests.
3. Run `just check` until it is green.
4. Commit with a [conventional commit](https://www.conventionalcommits.org/en/v1.0.0/)
   message, one sentence:

   ```bash
   git commit -m "feat: add a signal for lockfile-only churn"
   ```

## Adding a signal

Most contributions are signals. Read the "Adding a signal" section in
[AGENTS.md](AGENTS.md) and the table in [docs/signals.md](docs/signals.md).
A signal ships with a firing test, a silent test, and a row in that
table.

## Adding a preset

A preset is a YAML file under `internal/rules/presets/` that encodes a
project's written contribution policy. Link the policy document in the
file's leading comment so a reader can check the encoding against the
source.

## Pull request checklist

- [ ] Tests cover the new behavior
- [ ] `just check` passes locally
- [ ] `docs/` is updated if a signal, field, or preset changed
- [ ] Commit messages follow the conventional commit format
- [ ] The description says whether AI assisted the change

The last item is the same disclosure this tool checks for. vetkitten
scores its own pull requests.
