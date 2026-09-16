# Corrected 1.1.0 candidate — verification in progress

This is not human acceptance or a published release. It supersedes the failing
`b894586` candidate for further verification, not its immutable history.

- Source: `51aee17afec015ba2ad44584f8190b4bb6d901a8`.
- Manifest SHA256: `7adacb6e7dc990a12df2cc25e72404dde0156835d771474c9745d3b517a0780c`.
- [Build 35142798067](https://github.com/acoz-labs/mandalore/actions/runs/35142798067), attempt 1; artifact `10466086799`.
- Archive SHA256: `6f667b095346d14cae83222dab00b12db98a8f2c4e359675831f27d5a917aff0`.
- Retention expiry: December 15, 2026, 19:48:44 UTC, subject to repository policy.
- [Combined-source CI](https://github.com/acoz-labs/mandalore/actions/runs/35142731978) and main-branch audit passed, as did full local pinned host CI.

[Transport and payload verification](verified.json) refreshed official provenance,
checked the archive digest and all eight files before extraction. The builder
validated the source and built once. No local rebuild substitutes for these bytes.

## Fresh exact-artifact evidence

Nine real macOS arm64 PTY journeys against the downloaded runtime verify the #99
correction and related boundaries: idle/partial choice, text, partial setup and
release approval, default-No, EOF, TUI Ctrl+C and Escape/exit. Recordings and
receipts are under [cancellation](cancellation), with hashes in
[the initial manifest](initial-manifest.json). Plain interruption exits 130
without another key, no setup or release effects occur, and terminal state is
restored. TUI cancellation preserves its existing exit 0. Color/NO_COLOR and
80/32-column cases are included. Replay with macOS `script -p <recording>` in a
disposable terminal. No pixel-baseline, screen-reader or non-host-native claim.

The source-level evidence for #99/#100 remains
[separate](https://github.com/acoz-labs/mandalore/tree/89e197029d1650441d446e41da2c4f86b61e2fe8/docs/evidence/acceptance-corrections).
Its 8 Codex/4 Pi model observations used changed development skills with the old
runtime; they are not relabeled fresh observations of these packaged bytes.

Remaining exact-candidate work includes the original outcome matrices: packaged
Codex/Pi behavior, delivery/restrictions, historical retrieval and context costs,
native connection lifecycle, quota/readiness journeys and isolated upgrade.
Earlier human scenarios remain historical scoped evidence; this document does
not demand their repetition or turn them into acceptance of changed bytes.

## Nomination and authority

[Initial nomination](https://github.com/acoz-labs/mandalore/actions/runs/35143190361)
passed. Its selector found the head-associated #100 and existing outstanding
issues, but not the earlier merged #99. The #99 outstanding state was reconciled
from its verified included implementation, and both correction issues received
the standard machine-readable `Implementation PR` lifecycle links. A
[second nomination](https://github.com/acoz-labs/mandalore/actions/runs/35143418519)
uses the same immutable artifact; it is not a rebuild or an acceptance verdict.
It passed, and fresh bot-authored receipts were verified for all eleven issues:
#13, #56, #57, #61, #65, #74, #85, #88, #93, #99 and #100.
Labels alone are not evidence of acceptance.

The old same-account owner-review exception is pinned to `b894586`; it does not
cover this source/artifact pair or correction issues #99/#100. Extending eligibility
requires explicit owner direction. Actual personally performed review, verdicts,
release authorization and personal activation remain separate. No published
release or installed personal runtime was changed.
