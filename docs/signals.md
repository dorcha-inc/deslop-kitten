# Signals

Each signal has a stable identifier that appears in the JSON output. A
request appears as a numbered item at the top of the comment. A fact
appears as a clause in the sentence about how the author produced the
change.
A signal with nothing to report emits nothing. A bot on the policy's
`trusted_bots` list produces no facts.

## Requests from the policy

| Signal | Fires when | Opens with |
|---|---|---|
| linked_issue_required | `require_linked_issue` is set and the description references no issue | Reference an issue. |
| linked_issue_not_accepted | `accepted_issue_labels` is set and no linked issue carries one of them | Get the issue triaged. |
| ai_disclosure_required | `require_ai_disclosure` is set, the branch or commits or account show AI involvement, a judge is configured, and the judge finds no disclosure | Disclose AI assistance. |
| ai_disclosure_required | `forbid_ai_trailers` is set and a commit carries an AI co-author, assisted-by, or co-developed-by trailer | Remove AI trailers. |
| files_without_issue | `max_files_without_issue` is set, no issue is linked, and the file count exceeds it | Reference an issue. |

## Requests from the change

| Signal | Fires when | Opens with |
|---|---|---|
| large_change | Changed files exceed `large_change_files`, high severity above twice it | Consider splitting. |
| ci_failing | A check run on the head commit failed, timed out, or got cancelled | Fix CI. |
| thin_description | The description has under 20 words and the diff exceeds 100 lines | Describe the change. |

## Facts about the change

| Signal | Clause | Fires when |
|---|---|---|
| declared_bot | the author is a bot account outside this repository's trusted list | The forge marks the author as a bot and the login is not in `trusted_bots` |
| agent_branch_prefix | the head branch carries a prefix used by AI coding agents | The head branch starts with a prefix in `agent_branch_prefixes` |
| ai_trailer | the commits carry the named trailers | A commit carries a Co-authored-by trailer matching `ai_coauthor_patterns`, or any assisted-by or co-developed-by trailer |
| model_commit_email | the commits come from a model service address | A commit author email matches `model_email_patterns` |
| activity_burst | the account produced N public events inside three hours | The author's recent public events include `burst_events` or more inside any three hour window |

The report composes the sentence about the author's standing, with
pull requests here, merged elsewhere, account age, public repositories,
and followers, from the profile. That sentence is not a signal.

## Adding a signal

A signal is a type with an `ID` method and an `Evaluate` method in
`internal/signals`. It reads only its `Input` and returns findings.
Requests go in `policy.go` or `change.go`, facts in `facts.go`. Add the
type to `Defaults`, add a row to the table above, and add a test that
covers the firing case and the silent case.
