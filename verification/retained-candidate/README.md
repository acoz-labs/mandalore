# Verification-only retained Linux candidate probe

This branch is an evidence collector, not a runtime implementation or candidate
rebuild. Do not merge its temporary CI wiring into the product by implication.
The existing public `CI_RUNNER` remains unchanged. Only this same-repository PR
branch can run the extra steps; permissions add read-only Actions access to fetch
the exact already-retained artifact. No release, acceptance or repository write
permission is requested.

The existing trusted transport verifier checks current successful-run provenance
and the complete archive before extraction. The archive hash and all payload
checksums are checked again. Only the retained Linux AMD64 executable is tested;
building the verifier or normal source regression binaries does not substitute for
that executable. Candidate execution runs with an empty credential environment.

The probe exercises actual CLI memory correction/history, bounded MCP stdio,
read-only file hashes, local Git delivery/fresh-clone recall, owned CLI activation
and explicit synthetic migration. Output artifacts contain only selected synthetic
receipts and summaries, not bindings, credentials or model transcripts. This is
native Linux engineering evidence if the job passes, not a Codex conversation or
independent product acceptance. macOS AMD64 and Linux ARM64 remain separate gaps.

Contributor pre-run self-review: fixed source/run/artifact/archive identities;
no change to product source, runner variable or release policy; no credentials
passed to the candidate; local-only synthetic data and Git endpoints; bounded
transport/process operations; output privacy scan. The fixture is expected to
fail visibly for wrong platform, changed artifact or incorrect actual behavior.
