# Progressive foundling retrieval

- **Status:** Draft
- **Issue:** #57
- **Planning PR:** Pending
- **Repository basis:** b7de9f440a46dcc1e15729e2a1f36633730e73d0
- **Execution envelope:** implementation

## Decision

Keep the verified lexical source reader and provenance shape. Use smaller default
search previews and initial reads, add pinned search pagination and explicit byte
budgets, and guide expansion without duplicating already-read passages. Current
memory and historical evidence remain separate; full relevant sources remain
readable on explicit deeper investigation.

## Decision Spotlight

Search defaults: three documents, 512-byte previews, 8192-byte serialized result
budget. Read default: 1024 content bytes; explicit reads still support 8192 bytes.
Continuation names the exact registration. No opaque cache, index, new backend,
automatic summarization or hard cap on the user's requested review scope.
Payload reduction cannot substitute for correct negation, later passages or
current-versus-historical distinctions. Do not claim lower native token usage
merely because raw payloads are smaller.

## Plan map

[Context](01-context.md), [decision and interaction design](02-decision.md),
[contracts](03-design.md), [verification](04-verification.md),
[handoff](05-handoff.md).

## Review

Exact-head contributor engineering/design self-review under ADR0003 is required
before implementation. Finalize the planning PR identity and recorded review;
new-artifact product acceptance and release remain separate.
