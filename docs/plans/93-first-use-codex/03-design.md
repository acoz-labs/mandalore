# Design

Apply continues: cancellation check, Prepare and exact-plan comparison, state
directory and exclusive lock, cancellation recheck, guarded native-home creation,
inventory/collision checks, existing staging and activation path. The public plan,
result schema and phases do not change. Preflight may create an empty profile;
the existing incomplete-install notice reports retained partial effects.

An existing non-directory or redirected destination is refused. Preview never
creates the home. No authentication file, shell environment or bank is copied or
modified. The selected path remains explicit and part of the compared plan.

Inventory error context names the failed operation, not raw subprocess output.
All callers benefit, including structural diagnostics. Existing collision errors
retain their more specific explanations. A failed apply requires inspection before
retry, with no automatic rollback or deletion.
