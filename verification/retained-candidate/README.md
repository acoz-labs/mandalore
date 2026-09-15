# Replacement-candidate acceptance collector

Verification-only branch for #10/#45, not intended for product merge. Reuses
the reviewed collector from PR #41 with fixed replacement source/artifact/archive
identities. No product binary is built; trusted verifier compilation is separate.

Linux AMD64/ARM64 and Intel macOS jobs use the already configured standard public
runner variables after normal CI. Local macOS ARM64 runs the same probe in the
designated test pane. OS/architecture are asserted and translated macOS execution
is refused. Candidate subprocesses get a minimal environment with no provider
credentials; checkouts do not persist credentials in candidate jobs. Only the
trusted verifier/downloader receives the read-only workflow token.

Checks: exact retained version, same-record rename/history, actual stdio MCP and
read-only refusal/preservation, local Git delivery and fresh-clone recall,
owned CLI install, synthetic migration with intact original and exact JSON
number, unrelated cwd preservation. No model, live memory, remote Git account,
native plugin installation, acceptance submission or release.

Source f899cf6a2a255f3b6b35dcd778c672f799c65eb2; manifest
c2f5a340d0e665c81e01bc26add4c0dfe8ba27faac87b83a3fd81aff254f56ea.
Collector self-review: fixed identities, strict archive verification before
extraction, native guards, no candidate credentials, bounded MCP/job duration,
synthetic receipt privacy scan. Results must be observed before claiming success.
