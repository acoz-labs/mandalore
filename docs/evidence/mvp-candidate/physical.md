# Two-physical-machine CLI checkpoint

Contributor verification for #10 using the same retained candidate, not independent
product acceptance. The second physical machine reported macOS (`Darwin`) ARM64;
the first is also macOS ARM64. This does not cover Linux, AMD64 or a second native
Codex installation. No product source changed and the candidate was not rebuilt.

## Identity and isolation

The retained executable was transferred to a newly allocated private temporary
directory on the previously authorized second machine. Its SHA-256 was checked
there before both execution stages:
`eed5c72490079d9ee7bb2d08b7bc7916c5b56f4473f91849dc6e6d5c9d9fb3fc`.
The actual [remote version receipt](physical/version-b.json) identifies source
`5b3c7b275bb4bbb058bebdd7153f4b76bbceb59f`. It belongs to manifest
`c46743709dcd02b106c4e6d35cf58247486537e0212603eb7e369ecdfa1bd237`.

The initial strict SSH probe refused an unknown host key. A separate key scan was
compared with the fingerprint previously supplied out of band by the user, and
matched exactly. Only then was a fixture-local known-hosts file used with strict
checking and existing SSH authentication. No credential was copied, authentication
enrolled, global SSH trust changed, shell configuration edited or real bank touched.

The second machine created a fresh synthetic signet, external binding and local
bare remote. The first cloned that remote over real SSH and enrolled a distinct
explicit writer binding. Device labels and actors were synthetic user selections,
not inferred hostnames/accounts. Machine addresses, local bindings and paths are
not published here. Both disposable fixtures remain retained for inspection.

## Actual bidirectional flow

1. Machine B created `Distant Orbit`, initialized Git and delivered its checkpoint
   to the bare remote on B.
2. Machine A cloned over SSH and recalled that value through Mandalore. It corrected
   the same record to `Shared Orbit`, naming the exact predecessor, and delivered.
3. Machine B synchronized, recalled `Shared Orbit`, then explicitly corrected the
   same record to `Shared Horizon`, naming that predecessor, and delivered.
4. Machine A synchronized and recalled `Shared Horizon`. Actual local and remote
   heads matched `d89ecb45686e180a7a20e4ee1f05a1bb9c38eb9d`.

[History](physical/history.json) retains exactly three revisions, one record and
the stable project scope `two-machine`. Authorship alternates `Example User B`,
`Example User A`, `Example User B`; A and B have distinct device IDs and B's two
revisions retain the same origin. Exact predecessor assertions passed. This is
cross-machine memory continuity, not thread transfer or shared credentials.

## Test transport correction

The initial A-side sync checkpointed locally but reported pending at fetch. The
driver had supplied `GIT_SSH_COMMAND` to select its private known-hosts file;
Mandalore intentionally removes Git environment overrides and supplies strict,
noninteractive SSH options. The direct clone honored the override, but Mandalore's
sync did not, so the host was not trusted in that sync process.

The test retained the [pending receipt](physical/sync-a-initial-untrusted.json).
A test-local executable wrapper on that invocation's PATH selected only the
already-verified known-hosts file, then delegated to the system SSH executable.
Strict host checking and existing authentication stayed intact. The driver resumed
delivery of the existing checkpoint without repeating the correction. Both
directions then passed through the normal compiled sync runner.

This is a fixture accommodation, not a product fix or evidence that arbitrary
`GIT_*` overrides are supported. Ordinary installations need their SSH host trust
configured through the normal SSH setup. The test did not weaken the toolkit's
environment isolation to obtain a pass.

## Evidence and remaining scope

The selected JSON receipts and hash-only head files are actual synthetic outputs.
All files in the unrelated native test bank still matched its post-external-write
baseline after this test. No native session was launched on the second machine,
no model account was inspected and no user needed to complete a login.

This adds actual two-machine compiled CLI and SSH Git transport coverage. Native
integration on the second machine, other OS/architectures, rendered candidate
install/update/repair, independent acceptance and public release remain unproven
by this checkpoint. Cross-build success is not a substitute for those observations.
