<p align="center">
  <img src="share/cat.gif" alt="vetkitten" width="160">
</p>

<p align="center">Contribution policy checks for pull requests, run by a small open model inside your GitHub Action.</p>

<p align="center">
  <a href="https://github.com/jadidbourbaki/vetkitten/actions/workflows/ci.yml"><img src="https://github.com/jadidbourbaki/vetkitten/actions/workflows/ci.yml/badge.svg" alt="CI"></a>
  <a href="LICENSE"><img src="https://img.shields.io/badge/license-MIT-blue.svg" alt="MIT"></a>
</p>

vetkitten reads each incoming pull request, compares it with the
contribution policy your repository has written down, and leaves one
comment: what to do before review, and who sent the change. Everything
runs inside the Actions job. The one question that needs a language
model is answered by a 0.8B open model on the runner's CPU, so there is
no proprietary LLM, no API key, and nothing to pay. It never blocks a
merge.

## Install

Paste this into your coding agent in your repository:

```
Read https://raw.githubusercontent.com/jadidbourbaki/vetkitten/main/SKILL.md
and follow it to add vetkitten to this repository.
```

Or add `.github/workflows/vetkitten.yml` yourself:

```yaml
name: vetkitten
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
      - uses: jadidbourbaki/vetkitten@main
```

## The comment

> ### <img src="share/cat.gif" width="24" alt=""> [vetkitten](https://github.com/jadidbourbaki/vetkitten)
>
> 1. **Disclose AI assistance.** Commits carry `Co-authored-by: Copilot` and the description does not say how AI assisted the change. This repository asks for a sentence in the description naming the tool and how it was used.
> 2. **Consider splitting.** 60 files changed, +3000 and -240 lines. This repository treats changes above 25 files as large.
>
> First pull request here from `fresh-bot`, with 40 merged elsewhere on GitHub and an account since 2015. The head branch `copilot/optimize-agg` carries a prefix used by AI coding agents.
>
> <sub>Policy: `.github/vetkitten.yaml`, preset `kubernetes` ([source](https://github.com/kubernetes/community/blob/master/contributors/guide/pull-requests.md#ai-guidance))</sub>

Each request names the fact, the rule your repository wrote down, and
the fix. The sentences after the list come from the pull request and
the author's public profile. vetkitten posts only when it has something
to say, edits the comment in place on every push, and turns it into an
all-clear once the requests are resolved. A clean pull request from a
returning contributor gets no comment.

## Costs nothing to run

Every check is arithmetic over the pull request and the author's public
profile. The one judgment that needs a model, whether the description
discloses AI help, runs a 0.8B open model through llama.cpp on the
Actions runner, and the model ships inside the container image. No
tokens are sent anywhere, no key is configured, and on a public
repository the Actions minutes are free as well. Hosted review bots
charge per seat or per pull request, and a project with hundreds of
inbound pull requests a month feels that. vetkitten costs the
maintainer nothing.

## Policy

Without a policy file vetkitten asks for nothing and reports the facts.
To enforce a written policy, add `.github/vetkitten.yaml`:

```yaml
preset: kubernetes
```

The `kubernetes` preset encodes the [Kubernetes AI guidance](https://github.com/kubernetes/community/blob/master/contributors/guide/pull-requests.md#ai-guidance):
disclose AI assistance in the description and use no AI co-author
trailers. The `ghostty` preset encodes the [Ghostty AI usage policy](https://github.com/ghostty-org/ghostty/blob/main/AI_POLICY.md):
disclose all AI usage, naming the tool and the extent. Every field is
described in [docs/action.md](docs/action.md), the comment and the
model in [docs/concepts.md](docs/concepts.md), and every check in
[docs/signals.md](docs/signals.md).

## Run it locally

```bash
go install github.com/jadidbourbaki/vetkitten/cmd/vetkitten@latest
GITHUB_TOKEN=... vetkitten score octo/repo#42
```

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md). The cat is from Zanin's
[1-Bit Platformer Pack](https://opengameart.org/content/1-bit-platformer-pack-0),
CC0.

## License

[MIT](LICENSE)
