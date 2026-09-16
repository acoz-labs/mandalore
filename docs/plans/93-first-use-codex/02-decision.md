# Decision

Creating the native home in preview would violate read-only assessment. Requiring
the user to launch/login to Codex first would defeat supported first-use setup and
unnecessarily entangle credentials. A new profile-management abstraction adds no
value for one directory prerequisite.

Use `realDirectory` after apply revalidates its plan and acquires its lock, before
the first inventory. It verifies canonical paths before and after MkdirAll, creates
missing components with 0700 and leaves existing modes untouched. Keep the existing
race limitations; do not claim protection from a hostile same-user concurrent
filesystem attacker. Recheck cancellation before preparing the profile.

Wrap native inventory errors with fixed marketplace/plugin-list operation context.
Keep raw stdout/stderr out of diagnostics and preserve error identity for
cancellation. A partial profile remains for inspection, never automatically erased.
