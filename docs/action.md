# GitHub Action

The action runs as a container on every pull request event, posts or
updates deslop-kitten's comment, and writes the same report to the job
summary.

## Install

Add a workflow at `.github/workflows/deslop-kitten.yml`:

```yaml
name: deslop-kitten
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
      - uses: dorcha-inc/deslop-kitten@v1
```

`pull-requests: write` lets the action post and edit its comment. With
`comment: "false"` the action needs only read permissions and the report
goes to the job summary alone.

The workflow needs no checkout step. The action reads the policy file
from the pull request's base branch through the API, so a pull request
cannot change the policy that applies to it.

## Inputs

| Input | Default | Meaning |
|---|---|---|
| policy | `.github/deslop-kitten.yaml` | Policy file inside the repository. A missing file at the default path applies the default preset |
| comment | `true` | Post the report as deslop-kitten's single comment, edited in place on reruns |
| judge-url | empty | OpenAI-compatible base URL of a disclosure judge. Empty skips the disclosure check |
| judge-model | `local` | Model name to request from the judge |
| github-token | `${{ github.token }}` | Token used to read the pull request and post the comment |

## Policy file

The file names a preset and overrides fields. Every field is optional.

```yaml
preset: kubernetes
large_change_files: 40
trusted_bots:
  - dependabot[bot]
  - renovate[bot]
  - my-release-bot[bot]
```

Bot logins contain square brackets, so list them one per line as above.
Inside a flow list they need quotes, as in `["dependabot[bot]"]`.

| Field | Default preset value | Meaning |
|---|---|---|
| preset | `default` | Preset to inherit from |
| source | none | URL of the written policy the file encodes, shown in the comment footer |
| require_linked_issue | `false` | The description must reference an issue |
| accepted_issue_labels | `[]` | A linked issue must carry one of these labels |
| require_ai_disclosure | `false` | The description must disclose AI assistance when the branch, commits, or account show it. Needs a judge |
| forbid_ai_trailers | `false` | AI co-author, assisted-by, and co-developed-by trailers are a request to remove them |
| max_files_without_issue | `0` | File count above which a pull request with no linked issue gets a request. Zero disables |
| large_change_files | `25` | File count above which the change gets a splitting request. Zero disables |
| burst_events | `30` | Public events inside any three hour window that add an activity clause to the comment. Zero disables |
| trusted_bots | Dependabot, Renovate, GitHub Actions | Bot logins the repository invited. They produce no facts |
| agent_branch_prefixes | eight common agent prefixes | Head branch prefixes that mark an agent |
| ai_coauthor_patterns | nine regular expressions | Co-authored-by values that mark a model |
| model_email_patterns | four regular expressions | Commit author emails that mark a model service |

## Presets

| Preset | Encodes | Source |
|---|---|---|
| default | The change thresholds, the author sentences, and the trusted bot list. Declares no policy rule | none |
| kubernetes | Disclosure required, AI trailers not accepted | [Kubernetes pull request guide, AI Guidance](https://github.com/kubernetes/community/blob/master/contributors/guide/pull-requests.md#ai-guidance) |
| ghostty | Disclosure required, naming the tool and the extent | [Ghostty AI Usage Policy](https://github.com/ghostty-org/ghostty/blob/main/AI_POLICY.md) |

Run `deslop-kitten presets` to list them. Both named presets need a judge
for the disclosure request to fire.

## The judge image

The default image is 4 MB and skips the disclosure check. The judge
image adds the llama.cpp server and a 0.8B instruct model, about 820 MB
in total, and decides disclosure by reading the description. Every
release publishes it to the GitHub Container Registry as
`ghcr.io/dorcha-inc/deslop-kitten:judge` and `judge-<version>`, and
`just judge-image` builds it locally. Reference it in the workflow
step:

```yaml
    steps:
      - uses: docker://ghcr.io/dorcha-inc/deslop-kitten:judge
        with:
          args: score --comment --summary
        env:
          GITHUB_TOKEN: ${{ github.token }}
```

Without an argument, `score` reads the pull request from
`GITHUB_REPOSITORY` and `GITHUB_REF`, which every pull_request workflow
run sets.

The entrypoint starts the server on localhost, and deslop-kitten waits for
the model to load before its first request. Loading the model takes a
few seconds and the judge answers in about three, so a pull request
takes around twenty seconds end to end. `MODEL_URL` is a build
argument, so any GGUF the server can load replaces the default.

A repository that already runs a model can pass `judge-url` to the
default action instead. deslop-kitten reads `JUDGE_API_KEY` from the
environment when the endpoint needs one.

## Running outside GitHub

Both images run anywhere Docker runs:

```bash
docker run --rm -e GITHUB_TOKEN ghcr.io/dorcha-inc/deslop-kitten:judge score octo/repo#42 --format json
```

deslop-kitten reads `GITHUB_TOKEN` from the environment. Without
`--comment` it prints the report and posts nothing. Without a token
GitHub allows sixty requests an hour, and one pull request costs about
nine.
