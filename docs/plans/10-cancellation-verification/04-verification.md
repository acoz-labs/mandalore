# Verification

The recorded full-suite failure and controlled delayed-wrapper reproduction are
the failing-first baseline. First add structured diagnostics and characterize the
timing assumption, then implement event-bound cancellation tests. Require the
normal and slow-start fetch cases to reach the marker before cancellation and the
deadline case to stop before fetch, preserving all error/status assertions.

Run focused race-enabled repetitions, the complete API package and full bin/ci
in the designated pane. Attempt the documented container path first; use pinned
Go 1.26.4 host fallback only when container prerequisites are unavailable. Required
hosted checks must pass before merge; never relabel the earlier failure as success.
Test-only overlays used for diagnosis are not shipped product patches.

No rendered UI changes: no new screenshots or visual baseline approval applies.
The existing candidate's native evidence retains its original identity. A changed
merged source needs its own candidate identity before release acceptance; no old
payload is overwritten or rebuilt in place.

## Production Readiness Preflight

Not applicable to this implementation-only test repair. It cannot deploy, activate
a native profile, provision credentials, configure acceptors or publish a release.
Independent acceptance and immutable-publication prerequisites remain unchanged.
