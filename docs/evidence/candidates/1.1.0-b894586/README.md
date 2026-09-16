# First-use fix candidate — engineering checkpoint

**Latest result: changes required.** [Fresh engineering follow-up](engineering/README.md)
retains passed continuity/delivery/retrieval/recovery checks and two failures
blocking sign-off (#99/#100). No product acceptance or publication occurred.

Source `b89458617b49d2eb0a8686dcd75c26aa7ea6f6ba` is the reviewed #95 merge.
This separate evidence branch does not rebuild or change the retained artifact,
record owner acceptance, authorize publication or update a personal installation.

- [Build](https://github.com/acoz-labs/mandalore/actions/runs/35105578901), attempt 1,
  artifact `10450157997`, archive size 70,381,489 bytes.
- Archive SHA-256: `c495d8a11f56fa87630088c79738ee7b72ff05cb2c09e4aae11851cd2a7c3818`.
- Manifest SHA-256: `7995f820a7182a95b828851afce12bacea7108f82afac9369d307a43d961f45a`.
- [Transport](transport.json) and [verification](verified.json) rechecked provenance
  and every payload before extraction/execution. Expiry observed December 15, 2026;
  availability must be rechecked before promotion.
- [Nomination](https://github.com/acoz-labs/mandalore/actions/runs/35105986297) passed
  with the same candidate identity. Main CI and main-branch audit passed.

## Actual packaged checks

[Native receipt](native.json) verifies a newly absent profile with real Codex
0.154.0 on macOS arm64: preview leaves profile/state absent, explicit apply reaches
verified, structural inspection is healthy, profile mode is 0700, binding remains
unchanged and no auth file is created. No provider/model session was launched.

[Menu receipt](menu.json), [failure output](menu-failure.txt) and
[success output](menu-success.txt) retain actual plain-menu subprocess results.
The failure case is an intentionally failing synthetic native executable to test
bounded operation context and raw-output suppression. The success uses real Codex
and another absent profile. Both preserve default-No confirmation and explicit
partial/success phases. Captures normalize selected paths; they are not screenshots
or screen-reader, color, alternate-font/locale or other-platform acceptance.

The Git text projection trims final prompt whitespace and adds a terminal newline.
`published_text_sha256` binds those committed text files separately from the
helper's original and path-normalized output hashes. Interior prompt spacing is
retained, including intentional trailing blanks visible to whitespace checks.

The first packaged check already passed real installation/inspection. Its menu
capture then stopped at the publication privacy guard because the longer runtime
path wrapped across lines and exact-string masking missed it. Raw output stayed
private. The evidence-only capture helper here additionally normalizes renderer
line breaks inside selected paths; rerun in a new fixture passed both menu cases.
No product failure is hidden, no private path is published and no candidate byte
was changed. Original raw-output hashes are retained in the receipt.

## Acceptance handoff

The old `bb16a57` human failure and Pi successes remain evidence for that older
artifact, not relabeled acceptance of this one. Its records are retained on the
previous candidate evidence branch. The reviewed source change is confined to
Codex profile preparation/diagnostics; the embedded Codex package content hash is
unchanged. Nevertheless, a new candidate is a new acceptance identity.

The owner completed Codex connection from a fresh synthetic profile using the
same disposable acceptance bank, retaining its correction history. Structural
inspection passed; a fresh session automatically recalled the Pi-authored current
project fact and preference. The owner accepted this setup-and-recall experience
in the [recorded scenario verdict](https://github.com/acoz-labs/mandalore/issues/93#issuecomment-5701496652).
New profile, state and capture paths keep the original failed attempt intact.
No whole-candidate human verdict has been supplied. See the
[release reconciliation](release-reconciliation.md) for remaining evidence and
authority gates; no acceptance workflow has been dispatched.
