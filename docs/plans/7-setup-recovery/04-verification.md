# Verification and release design

Use failing-first tests for each slice, porting relevant synthetic predecessor
cases without treating them as Mandalore native acceptance.

1. Local plans: deterministic identity, path/type/symlink/overlap denial, changed
   inputs, foreign signet pin, bounded malformed JSON, no writes on preview.
2. Apply: staged artifact/hash/protocol/OS/arch validation, no replacement,
   concurrent lock refusal, exact native args, ownership collision, duplicate
   memory integration, add/install/verification failure and retained recovery.
3. Doctor/repair: missing runtime/bundle/cache, edited/extra files, overrides,
   stale native version, missing binding and stale identity; no-write hashes.
4. Console/menu: arrows/j/k/Enter, Escape/:back, EOF/Ctrl-C, default-No, invalid
   input, multiple journeys with one buffered reader, NO_COLOR/plain/non-TTY,
   long paths, terminal restoration, narrow widths and partial-success wording.
5. Compiled CLI: JSON plan/apply/doctor/repair routing and menu plain sequence
   against synthetic inputs; no TUI or implicit permission needed by an agent.

Run pinned Go full CI in the designated Herdr pane. Validate bundled plugin and
skill assets using their native validators when packaging changes. Test actual
Codex registration and fresh-session MCP/hooks with the selected immutable test
binary, not only fake native subprocess responses. Preserve native resources;
explicitly resolve the known development marketplace before managed activation.

UI classification: new-or-materially-changed-experience for the menu. Product
design is the contract in 02-decision.md, self-reviewed under delegated authority.
Capture sanitized, openable exact-head rendered evidence for menu, effects
preview/default-No, success and partial/failure/repair, plus narrow/plain modes.
Use synthetic paths/names and keep private desktop/native transcripts out of
public evidence. Functional key-sequence/format tests protect the reviewed
patterns; a first screenshot is not independent visual approval. Record terminal
dimensions, native version, theme/input mode and omitted accessibility surfaces.

Native scenario sequence: cancel unchanged; create/bind synthetic signet; connect
with no exported runtime/binding; fresh agent saves/recalls; doctor; update to an
explicit test artifact; delete only a known synthetic generated file; repair;
fresh session recall; prove original memory/auth/session ownership preserved.
Repeat relevant paths with another local clone and new device attribution.

No staging or public release is invented for this artifact repository. #11 must
provide immutable release distribution before published-update acceptance;
#10 repeats owner/independent acceptance against that exact artifact. Local
engineering evidence is not relabeled as candidate or physical-machine proof.
