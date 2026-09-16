# Verification

Add failing-first Go tests for blocked plain selection, text input, confirmation
and partial approval. Require context.Canceled and no effects before unblocking
the test reader. Preserve existing EOF/incomplete input/control/length tests.
Run race-enabled package tests, vet, full pinned host CI and cross-builds.

Native exact-head evidence: real terminal Ctrl+C while idle and with partial
input, no extra Enter/EOF, exit130 and terminal restoration; both menu and release
confirmation surfaces. Check TUI Escape/Ctrl+C, plain EOF/default-No, color and
NO_COLOR. Use synthetic state only. Retain the original failing recording.

Classification: in-pattern interaction correction. Contributor rendered judgment
is distinct from human acceptance; no pixel baseline or screen-reader claim.

## Production Readiness Preflight

Not applicable to the implementation-only envelope. Existing artifact/release
workflows remain unchanged; publication, secrets and live activation are excluded.
A corrected source needs a newly built immutable artifact, nomination and human
verdict. Do not transplant the old candidate's acceptance exception or receipts.
