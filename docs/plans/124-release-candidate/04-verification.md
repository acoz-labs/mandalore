# Verification And Release Design

Run full pinned host/hosted CI and verify main contains all linked implementation
merges. Verify every payload before executing it. Repeat against retained bytes:

1. #77: plain/TUI,32/80 columns,no-color,default-No/Back and exact shell meaning
   for displayed recovery commands. Keep synthetic quota failures separate from
   successful recovery claims; no live quota exhaustion.
2. #112: stopped-session acknowledgement, deferred updates, old/new native
   connection behavior, truthful notices and unrelated/auth/memory preservation.
3. #80: explicit scoped preview/apply, redaction groups, history/journal opt-ins,
   stale source/binding and destination refusal, cancellation/partial output,
   source invariance and no implicit delivery.
4. #81: format1 unaffected, preview/apply/recovery, old-reader refusal, independent
   clone opt-in/convergence, causal correction/withdrawal/restore, current versus
   historical export, bounded retention and fresh-session Codex/Pi discovery.
5. Distribution: safe old-updater refusal and verified-new-executable transition,
   exact source/version/package identities, retained old runtime and unchanged
   format1 bank. Do not claim public bootstrap download before publication.

Preserve each issue's full contract. Cross-builds are not native platform evidence.
Preparation has no rendered impact, but candidate acceptance repeats actual affected
rendered/native matrices. Human acceptance and explicit publication authority are
required. Reverify public assets and delivery ledger after authorized publication.
