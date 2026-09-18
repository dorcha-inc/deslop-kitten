# vetkitten

Add vetkitten to a GitHub repository. vetkitten is a GitHub Action that
checks each incoming pull request against the repository's written
contribution policy and leaves one comment: numbered requests to act on
before review, then a sentence about the author and how the change was
produced. It runs a small open model inside the Actions job, needs no
API key, and never labels, closes, or blocks a merge.

Follow these steps in the repository you are working in.

## 1. Add the workflow

Create `.github/workflows/vetkitten.yml` with exactly this content:

```yaml
name: vetkitten
on:
  pull_request:
    types: [opened, synchronize, reopened]
permissions:
  contents: read
  pull-requests: write
  checks: read
jobs:
  check:
    runs-on: ubuntu-latest
    steps:
      - uses: dorcha-inc/vetkitten@main
```

`pull-requests: write` lets the action post and edit its comment. Do
not add any other permission.

## 2. Pick a policy

Read `CONTRIBUTING.md` and any file whose name mentions AI or policy.
Then decide:

- The repository states that contributors must disclose AI assistance
  and must not list AI tools as co-authors. Use the `kubernetes`
  preset.
- The repository states that all AI usage must be disclosed and says
  nothing about co-author trailers. Use the `ghostty` preset.
- The repository has no written rule about AI assistance. Use no
  preset. The comment then carries the author sentences and requests
  about the change itself, and asks for nothing the repository did not
  write down.

For a preset, create `.github/vetkitten.yaml`:

```yaml
preset: kubernetes
```

Do not invent rules the repository has not written. Do not set
`require_ai_disclosure` or `forbid_ai_trailers` unless the repository's
own documents say so. If the repository runs bots other than
Dependabot, Renovate, and GitHub Actions, list their logins:

```yaml
preset: kubernetes
trusted_bots:
  - dependabot[bot]
  - renovate[bot]
  - my-release-bot[bot]
```

## 3. Decide on the disclosure check

The disclosure request needs a small model that reads the description.
The default action skips it. To enable it, replace the step with the
judge image, which bundles the model and runs on the Actions runner:

```yaml
    steps:
      - uses: docker://ghcr.io/dorcha-inc/vetkitten:judge
        with:
          args: score --comment --summary
        env:
          GITHUB_TOKEN: ${{ github.token }}
```

Use the judge image only when the chosen preset requires disclosure.
Otherwise keep the default step.

## 4. Open a pull request

Commit the new files on a branch named `vetkitten-setup` and open a
pull request titled `ci: add vetkitten`. In the description, say which
preset you chose and why in one sentence, and state that an AI agent
prepared the change. That last sentence is the disclosure vetkitten
itself will look for.

Do not enable branch protection, do not create labels, and do not
change any existing workflow.

## Reference

- Every policy field and its default: https://github.com/dorcha-inc/vetkitten/blob/main/docs/action.md
- Every signal and when it fires: https://github.com/dorcha-inc/vetkitten/blob/main/docs/signals.md
