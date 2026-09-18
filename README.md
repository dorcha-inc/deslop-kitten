<p align="center">
  <img src="share/cat.gif" alt="vetkitten" width="160">
</p>

<p align="center">Contribution policy checks for pull requests, run by a small open model inside your GitHub Action.</p>

<p align="center">
  <a href="https://github.com/dorcha-inc/vetkitten/actions/workflows/ci.yml"><img src="https://github.com/dorcha-inc/vetkitten/actions/workflows/ci.yml/badge.svg" alt="CI"></a>
  <a href="LICENSE"><img src="https://img.shields.io/badge/license-MIT-blue.svg" alt="MIT"></a>
</p>

vetkitten reads each incoming pull request, compares it with the
contribution policy your repository has written down, and leaves one
comment: what to do before review, and who sent the change. Everything
runs inside the Actions job. The one question that needs a language
model is answered by a 0.8B open model on the runner's CPU, so there is
no proprietary LLM, no API key, and nothing to pay. It never blocks a
merge.

## Motivation

<p align="center">
  <img src="share/meme.png" alt="vetkitten shielding open source maintainers from AI slop PR spam" width="440">
</p>

Writing a pull request now takes minutes and reviewing one still takes
an afternoon. Agents opened an estimated seventeen million pull requests
a month in 2026. curl closed its bug bounty after a month of fabricated
security reports. GitHub began letting maintainers cap pull requests
from outside their projects. The people carrying that load are
volunteers, and most of them answered by writing a contribution policy
and then enforcing it by hand, one comment at a time, on every pull
request that had not read it.

vetkitten writes that comment. It reads the policy the project already
wrote, tells the contributor what the project asks before a person
spends an afternoon on the review, and tells the maintainer who is on
the other side. A newcomer learns the rules from a list rather than
from a closed pull request. A maintainer opens the thread already
knowing whether the disclosure is there and whether the account has a
history. The afternoons that come back go to the reviews that deserve
them.

## Install

Paste this into your coding agent in your repository:

```
Read https://raw.githubusercontent.com/dorcha-inc/vetkitten/main/SKILL.md
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
      - uses: dorcha-inc/vetkitten@main
```

## The comment

<img src="share/comment.png" alt="vetkitten's comment on a pull request" width="700">

The comment above is from [this repository's own first pull request](https://github.com/dorcha-inc/vetkitten/pull/2).
When a policy asks for something the pull request lacks, a numbered
list comes first, and each item names the fact, the rule your
repository wrote down, and the fix. The sentences after it come from
the pull request and the author's public profile. vetkitten posts only
when it has something to say, edits the comment in place on every push,
and turns it into an all-clear once the requests are resolved. A clean
pull request from a returning contributor gets no comment.

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
go install github.com/dorcha-inc/vetkitten/cmd/vetkitten@latest
GITHUB_TOKEN=... vetkitten score octo/repo#42
```

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md).
