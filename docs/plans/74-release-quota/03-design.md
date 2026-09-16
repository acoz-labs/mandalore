# Technical Design

## Component And Behavior Flow

Inspection returns a private verified result internally; public `Inspect`
projects its existing `ReleaseView` and discards private bytes. Standalone
`DownloadAsset` still refreshes the pinned release ID and compares the complete
view before fetching. It must not accept a caller-minted private result.

Apply parses the incoming plan, retains existing pending/replay handling, then
obtains a fresh plan **and private inspection result in the same call**. Compare
the fresh plan with the reviewed input. Stage the original manifest bytes and
download the selected binary directly against the verified asset identity.
Check staged bytes, execute the existing probe, recheck bytes, and perform the
existing fresh full plan comparison before touching the destination. The two
four-request inspections plus one binary download cost nine apply requests.
Initial planning costs four. No special cache survives return/cancellation.

Full verifier obtains its own private inspection result, checks the publication
ledger and exact asset map, then fetches the other six assets. It may skip only
manifest/sums whose exact IDs, sizes and digests were verified by this same
inspection. Final metadata/tag checks stay unchanged: 4 + 1 + 6 + 1 + 1 = 13.

## State And Data Model

Use a private inspection value containing the existing view and bounded original
manifest/checksum byte slices (each existing manifest limit). No exported JSON
fields, durable cache, receipt/plan/schema change or mutable client-wide state.
The planning helper returns private data separately from the serialized plan;
normal public planning discards it. Local/retained sources need no such data.
Reject missing/mismatched verified state before published staging. Rehash/parse
the retained manifest when staging; do not reconstruct JSON from parsed fields.
All owned-temporary cleanup and pending receipts retain existing behavior.

## Interfaces And Contracts

Add a typed rate-limit error with safe numeric HTTP status and optional quota
advice. API errors use `release.rate_limited` only for supported evidence, with
an optional `release_retry` object; other non-404 refusals remain `release.failed`.
Cancellation keeps `operation.cancelled` priority; 404 stays `release.unavailable`.
Keep `release_result`, `write_may_have_occurred` and `inspect_before_retry` derived
from the enclosing operation, not the HTTP status. Advice contains no URLs,
request headers, response body or credential fragments.

Proposed advice fields: `http_status` (403/429), `kind` (`primary` or
`unspecified`), optional `retry_after_seconds` integer and `reset_at` UTC RFC3339.
No automatic-retry flag. Classification: 429 is rate-limited; 403 is rate-limited
only with a single valid remaining-zero header or valid Retry-After. A valid
remaining-zero marks primary; otherwise do not assert secondary without proof.
Reset alone on ordinary 403 is insufficient. Do not read the body for diagnosis.

Accept only one header value per field, bounded to 64 ASCII bytes. Numeric fields
use nonnegative decimal syntax with overflow rejection; reject control, signed,
fractional, duplicate and comma-combined inputs. For GitHub's documented
Retry-After seconds accept 1..86400. HTTP-date Retry-After is deliberately not
interpreted. Reset is a positive epoch in the future and at most 24 hours from
the local observed time. Omit stale/excessive/malformed timing, never clamp.
If both valid timing fields exist, present both as advisory lower bounds and
instruct not to retry before either; do not promise quota is then available.
Unknown timing says wait before an explicit retry, with no invented deadline.
Use an internal clock seam for deterministic boundary tests, not a user option.

## Authorization And Data Exposure

Anonymous public HTTPS GET only; retain existing allowlisted hosts, time/size/
redirect bounds and no auth/cookie behavior. No gh invocation, native credential
read, signet access or broader write permission. Public input cannot supply
verified byte state. Installation still requires the exact reviewed plan and
owned destination. Synthetic tests use inert fixtures and disposable prefixes.

## Failure, Recovery, And Observability

The transport reports request failure, not whole-install effects. Remove the
universal no-installation phrase from transport failures. Existing menu outcome
and receipt blocks show the actual phase and pending state. Safe quota messages
explain advisory timing and explicit retry; ordinary refusal stays ordinary.
No body/header dump, recursive retry, network quota probe or sleep occurs.
Retry uses the existing reviewed plan and revalidation; changed source requires
a new preview. Owned staging may contain partial bytes and is handled by the
existing cleanup; never delete foreign/replaced paths or active runtimes.

## Design Traceability

Count acceptance maps to private inspection/staging/full verifier. Trust and
race acceptance maps to fresh plan comparisons, standalone downloads, byte
checks and existing activation state machine. Guidance maps to HTTP parsing,
typed API error and menu rendering. Partial-effect acceptance maps to preserved
InstallResult/error fields; native scenarios cover the human recovery contract.
