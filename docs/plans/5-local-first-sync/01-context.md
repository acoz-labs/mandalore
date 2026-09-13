# Context and interaction

The local interface is merged in #22. Its receipts honestly say no remote sync.
Users need continuity across machines without treating an unavailable network as
failed memory storage, and agents need typed outcomes rather than Git transcripts.

## Intended flows

1. Bind a new signet, explicitly initialize its standalone Git repository and
   checkpoint it. Native Git setup supplies an optional origin remote/credentials;
   #7 will guide this without publishing private configuration.
2. Remember offline and obtain the existing local durability receipt. Checkpoint
   successfully with no origin and report local-only; no credential prompt.
3. Sync two clones of the same signet. Fetch a pinned main head, validate it,
   integrate safe appended records, then push without force. Preserve both devices'
   provenance, corrections, journals and reference registrations.
4. If remote/auth is unavailable or another push races ours, report pending with
   local commit identity. If a candidate is unsafe or files conflict, preserve
   local branch/work and remote fetched objects; explain that reconciliation is needed.
5. Inspect status read-only. Show dirty/pending state and last remote check time;
   a remembered synced receipt is not a claim the remote has not changed since.

CLI stays noninteractive with machine-readable results; no new menu in this slice.
No raw command output, token-bearing URLs, filesystem inventory or memory bodies
are included in operational errors. Unsupported/busy/dirty cases fail explicitly.

## Passive behavior and latency

Do not add network writes to unconditional startup hooks: the next task may be
read-only. The native memory skill may request a short-budget sync before its
first authorized recall and after useful memory writes. It must skip this for
no-save/read-only tasks and keep local memory useful when pending. #6 maps this
policy to real Codex lifecycle surfaces and measures the actual user experience.
Repeated read-only calls never retry the network automatically.
