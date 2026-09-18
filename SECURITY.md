# Security Policy

Report a vulnerability through GitHub's private vulnerability reporting
at [Report a vulnerability](https://github.com/dorcha-inc/deslop-kitten/security/advisories/new).
The report stays private between you and the maintainers until a fix
ships. Please include a description, steps to reproduce when
applicable, an impact assessment, and a suggested fix when you have
one. Please do not open a public issue for a vulnerability.

## Response timeline

- Initial response within 2 business days.
- Status update within 7 business days.
- Resolution depends on severity and complexity.

## Disclosure

We acknowledge receipt within 48 hours. Once a fix ships we publish the
advisory through GitHub, request a CVE when the severity warrants one,
and credit the reporter when they wish.

## Security model

deslop-kitten reads untrusted text. Every pull request title, body, commit
message, and diff is content an outside party wrote, and deslop-kitten
treats all of it as data.

- **The policy comes from the base branch.** The action reads
  `.github/deslop-kitten.yaml` through the API at the pull request's base
  ref, never from the pull request's own tree, so a contributor cannot
  loosen the rules that apply to their change.
- **One write, to its own comment.** The action needs `pull-requests:
  write` to post and edit the comment and read access to everything
  else. It never labels, closes, approves, requests changes, or blocks a
  merge, and it never mentions anyone with an at-sign.
- **Comment text comes from templates.** Every sentence in the comment
  is a fixed string with numbers, branch names, trailer values, and
  email addresses substituted in code spans. Every link target is a URL
  the API returned or a search URL built from the login and the
  repository name. deslop-kitten echoes no text from the pull request
  as prose.
- **Signals match structured fields.** Requests and facts come from
  regular expressions and counts over fields the forge returns.
  deslop-kitten executes nothing from the pull request.
- **The judge answers one yes or no question.** The disclosure judge
  reads the description and returns a boolean plus a quoted sentence,
  which deslop-kitten verifies against the description and never posts. An
  author can only make the judge say yes by writing a disclosure.
- **Static images.** The default Dockerfile compiles the binary with a
  pinned Go toolchain and copies it onto a distroless base with no
  shell. The judge image adds the upstream llama.cpp server image and a
  pinned model file.

## Known considerations

1. The GitHub token in `GITHUB_TOKEN` is readable by anything running in
   the same job. Run deslop-kitten in its own job.
2. Presets encode third-party policies as regular expressions and
   prefixes. A contributor can adapt to them. The signal list is public
   by design, and the author sentence reports profile facts that are
   hard to fake.
3. Rate limits. An unauthenticated run costs about nine requests of the
   sixty an hour GitHub allows.
