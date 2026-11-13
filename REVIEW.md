# Review

How a reviewer judges a PR here. Read the issue first, and the change's
`docs/changes/<issue>-<slug>/plan.md` when it has one. Repository facts live
in `docs/repository.md`.

## Passes

Make each pass over the whole diff.

### Bugs

Wrong behaviour, failure paths the change can hit, edge cases, and data or
state the change could lose or corrupt.

- Force a late failure in multi-step write commands (migrate and similar),
  for example make the config read-only after the push.
- Storage-failure probes cover the read path (corrupt state) as well as the
  write path (read-only directory).
- A fix that changes a probe also checks the real write path; a probe can
  pass while the real write fails.
- When a fix is scoped to one failure path, sweep the sibling error mappings
  in the same handler.
- Check base-versus-head state compatibility in both directions.
- Docs that name CLI commands are checked against a built binary: `--help`
  plus a bogus-subcommand probe.
- Release acceptance installs over an existing install in the same prefix,
  not only into an empty one.

### Security

Secrets in code, logs or fixtures; untrusted input reaching commands, queries
or output; permissions or access widened.

- Test "approved client, no owner session" in a browser that holds no owner
  cookie.
- Sanitizers for echoed text cover Unicode line separators (U+2028, U+2029),
  not just C0 controls and ANSI.
- For OAuth error-shape fixes, probe compiled base and head binaries live,
  with a fresh cookie jar per browser. When an error code changes, revoked
  and never-existed clients must still get identical bytes.
- Review any per-IP server limit as if every peer were `127.0.0.1` (the
  tunnel).
- Over HTTP, `/token`'s client check reads cached state first, so a read
  failure inside redeem or refresh needs a direct-call test or a live probe
  that holds `oauth.lock` and swaps in an unreadable `oauth.json`.
- Run `govulncheck` (see `docs/repository.md`).
- A live probe of a global rate limit waits out real windows (about 10
  minutes each); budget about 30 minutes.

### Security (risky changes)

In addition to Security, required when the change touches any of:

- secrets or their handling;
- authentication or permissions;
- release, deploy or CI scripts; dependency or install scripts;
- money: payments, billing, prices, invoices, refunds, payouts;
- personal information: customer or user records, names, emails,
  addresses, payment details, and anything that logs, exports or shares
  them;
- deleting or migrating stored data;
- anything that sends messages to real people (email, texts,
  notifications);
- anything that makes a service, file or endpoint publicly reachable, or
  changes who can reach it.

Post it as its own section of the review, naming the category that
triggered it and what was checked. A high finding blocks merge. When none
applies, the review says "Security (risky changes): not triggered", in one
line.

### Tests

Each behaviour the change adds or fixes has a test that would fail without
it. A harness that diffs base and head decisions across several
fixtures is a cheap proof of "same decisions". For relevance work, each
false-injection fix needs short specific-question cases and each
relevance-path fix needs collision prompts; reviewer probe batches are fresh
each round and sit where the author can't read them, or are hashed into the
review before the author starts. Held-out evaluation stays opt-in
(`MANDALORE_EVAL_HELDOUT=1`), runs once after code freeze, and every run is
counted and disclosed. On a bug fix, a change to a pre-existing test that the fix makes pass is
a high finding unless the PR explains why.

### Docs and release

Durable docs match what ships; the version, changelog or release notes the
repository's process expects are in the PR.

### Compliance

The PR does what the issue and `plan.md` say, all of it and nothing else.
A departure from the plan updates `plan.md` in the same PR.

## Severity

- **High**: wrong behaviour, data loss, a security hole, or an acceptance
  criterion not met. Blocks merge.
- **Medium**: a real defect or gap with a workaround or a narrow trigger, or
  missing proof for a claim. Fixed in this PR unless the reviewer agrees to
  a follow-up issue.
- **Low**: worth fixing, harmless if not. Doesn't block.

Report at most 5 lows per review and give the rest as a count. An item the
author flags as borderline is judged in this review, not deferred.

## Escapes

A fix for something an earlier merged change should have caught is an
escape, and so is a revert of a merged change. Whoever files its issue or
PR applies the `escape` label and names the original PR:
`Escape from #<n>`. The reviewer checks for both on such a PR and asks for
whichever is missing.

## Do not report

- Style a formatter or linter in `bin/ci` already enforces.
- Preferences with no effect on behaviour, safety or clarity.
- Pre-existing problems the diff doesn't touch; file an issue instead.
- Automated review-bot noise already triaged.
