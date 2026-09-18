<p align="center">
  <img src="share/cat.gif" alt="deslop-kitten" width="160">
</p>

<p align="center">A free GitHub Action that flags AI slop pull requests before you review them.</p>

<p align="center">
  <a href="https://github.com/dorcha-inc/deslop-kitten/actions/workflows/ci.yml"><img src="https://github.com/dorcha-inc/deslop-kitten/actions/workflows/ci.yml/badge.svg" alt="CI"></a>
  <a href="https://github.com/dorcha-inc/deslop-kitten/releases"><img src="https://img.shields.io/github/v/release/dorcha-inc/deslop-kitten" alt="Release"></a>
  <a href="go.mod"><img src="https://img.shields.io/github/go-mod/go-version/dorcha-inc/deslop-kitten" alt="Go version"></a>
  <a href="LICENSE"><img src="https://img.shields.io/badge/license-MIT-blue.svg" alt="MIT"></a>
</p>

deslop-kitten reads each incoming pull request, compares it with the
contribution policy your repository has written down, and leaves one
comment: what to do before review, and who sent the change. It never
blocks a merge.

## Motivation

<p align="center">
  <img src="share/meme.png" alt="deslop-kitten shielding open source maintainers from AI slop PR spam" width="440">
</p>

Are you an open source maintainer tired of AI slop pull requests? Most
of them ignore the contribution policy you wrote, and you review them
anyway, as a volunteer.

The spam has a second cost. Until early 2025 I regularly sent fixes to
projects I liked and got to know their maintainers. Today agents flood
every popular repository with questionable changes, so maintainers
engage with nobody they do not already know, and sending a fix is
rarely worth the effort. Open source needs a fix for that.

deslop-kitten is my attempt. It checks each pull request against the
policy your project already wrote and posts one comment: what the
contributor must fix before you review, and who opened the pull
request, with links. It is free and MIT licensed and runs inside your
GitHub Actions job. The one question that needs a language model goes
to a 0.8B open model on the runner, so it calls no hosted LLM, needs no
API key, and nothing meters it.

## Install

Paste this into your coding agent in your repository:

```
Read https://raw.githubusercontent.com/dorcha-inc/deslop-kitten/main/SKILL.md
and follow it to add deslop-kitten to this repository.
```

Or add `.github/workflows/deslop-kitten.yml` yourself:

```yaml
name: deslop-kitten
on:
  pull_request:
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

## The comment

<img src="share/comment.png" alt="deslop-kitten's comment on a pull request" width="700">

The comment above is from [this repository's own first pull request](https://github.com/dorcha-inc/deslop-kitten/pull/2).
When a policy asks for something the pull request lacks, a numbered
list comes first, and each item names the fact, the rule your
repository wrote down, and the fix. The sentences after it come from
the pull request and the author's public profile. deslop-kitten posts only
when it has something to say, edits the comment in place on every push,
and turns it into an all-clear once the author resolves the requests. A clean
pull request from a returning contributor gets no comment.

## Policy

Without a policy file deslop-kitten reports the facts and asks only
about the change itself: a large diff, a failing check, or a thin
description. To enforce a written policy, add `.github/deslop-kitten.yaml`:

```yaml
preset: kubernetes
```

The `kubernetes` preset encodes the [Kubernetes AI guidance](https://github.com/kubernetes/community/blob/master/contributors/guide/pull-requests.md#ai-guidance):
disclose AI assistance in the description and use no AI co-author
trailers. The `ghostty` preset encodes the [Ghostty AI usage policy](https://github.com/ghostty-org/ghostty/blob/main/AI_POLICY.md):
disclose all AI usage, naming the tool and the extent.
[docs/action.md](docs/action.md) describes every field,
[docs/concepts.md](docs/concepts.md) the comment and the model, and
[docs/signals.md](docs/signals.md) every check.

## Run it locally

```bash
go install github.com/dorcha-inc/deslop-kitten/cmd/deslop-kitten@latest
GITHUB_TOKEN=... deslop-kitten score octo/repo#42
```

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md).
