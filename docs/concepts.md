# Concepts

deslop-kitten reads one pull request, compares it with the repository's
written contribution policy, and posts one comment. The comment has two
parts. The first tells the contributor what to do before review. The
second tells the maintainer who sent the change.

## Requests

A request is one numbered item at the top of the comment. It names the
fact, the rule the repository set, and the action, in that order, and
it opens with a short imperative in bold. Requests come from the policy
fields the repository declared, such as a required issue link or a
required AI disclosure, and from the change itself, such as a large
diff or failing CI. Policy requests come before change requests, and
higher severity comes first within each group. When there is nothing to
ask, "Nothing to do before review." replaces the list.

## The author sentences

One sentence states the author's standing from the numbers on the
public profile: a first pull request here or how many have merged here,
how many merged elsewhere on GitHub, and how old the account is. Public
repository and follower counts join the sentence only for an account
with no merged pull request anywhere, where they are the only public
evidence left. A bot outside the trusted list gets a sentence saying so.

A second sentence, when there is one, covers how the author produced
the change: a head branch with an agent prefix, commit trailers that
name a model, a commit author email from a model service, and a burst of
public activity above the policy threshold. Every clause is a number or
a string a reader can verify with one click, and none of them is a
verdict.

## When the comment appears

deslop-kitten posts when there is a request, when a fact row applies, or
when the author has no merged pull request in this repository yet. It
edits its comment in place on every push and never posts a second one.
Once the author resolves the requests it rewrites the comment to "Every check
passes on the latest commits." A clean pull request from a returning
contributor gets no comment. deslop-kitten never mentions anyone with an
at-sign, never closes a pull request, and never blocks a merge.

## The disclosure judge

One question needs a reader: does the description state that a model
helped produce the change. "I drafted the first pass
with an assistant and rewrote it" is a disclosure. "This fixes the
Claude SDK client" is a subject. "I did not use any AI" is a denial.
Patterns cannot tell these apart, so the judge is a small instruct
model behind an OpenAI-compatible endpoint. It answers yes or no and
quotes the sentence it treated as the disclosure, and deslop-kitten checks
that the sentence appears in the description before trusting the
answer. The judge image bundles a 0.8B model and answers in a few
seconds on one CPU core. Without a judge deslop-kitten skips the disclosure
check and runs every other signal.

## Policy files and presets

A repository declares its policy in `.github/deslop-kitten.yaml`. The file
names a preset and overrides any field. The default preset declares no
policy rule and supplies the change thresholds, the author sentences,
and the trusted bot list. The kubernetes and ghostty presets encode those projects' written
AI guidance and link the documents. See [action.md](action.md) for the
fields.
