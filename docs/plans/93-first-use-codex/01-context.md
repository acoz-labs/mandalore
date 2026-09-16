# Context

At the pinned basis, `internal/install/native.go:apply` acquires the state lock
then inventories Codex plugins without creating `Options.NativeHome`. Real Codex
0.154.0 rejects a nonexistent CODEX_HOME before inventory. The fake runner accepts
it; earlier native upgrade tests precreated their profiles. Human first-use apply
therefore failed at preflight despite a valid preview.

The user journey is preview, explicit apply, inventory, staging, registration and
verification. Invalid paths, cancellation and collisions must retain their refusal
behavior. Existing users must retain unrelated profile files and permissions.
The affected bank is not migration input and must remain unchanged.

This is a bounded installer correctness fix using existing UI and error patterns,
not a new rendered interaction. No new authentication, backend, configuration
schema, memory behavior or automatic retry is in scope.
