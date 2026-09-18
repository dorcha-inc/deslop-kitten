// Package signals evaluates a pull request against a policy and emits
// findings of two shapes. A request tells the contributor what to do
// before review and cites the repository's policy. A fact is a clause
// about how the author produced the change that the report joins into one
// sentence, and every clause states something a reader can verify on
// the pull request page or the author's public profile. Every signal
// reads only the forge data and the resolved policy. The disclosure
// signal may additionally consult a disclosure.Judge, and it is skipped
// when no judge is configured. Run evaluates a list of signals, keeps
// every finding, and joins the errors.
package signals
