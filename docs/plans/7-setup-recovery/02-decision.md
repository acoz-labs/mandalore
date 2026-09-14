# Solution decision and product design

## Alternatives

A documentation-only wrapper leaves runtime pinning, recovery and defaults to
shell configuration. A full-screen application with its own agent lifecycle
reintroduces the discarded assistant framework. Select a small terminal menu
over shared deterministic operations: familiar navigation, inspectable JSON and
one implementation of each state change.

## Product-design contract

The entry is `mandalore menu` (with `--plain` for automation/accessibility).
Keep bare `mandalore` help for compatibility. Present a concise title and a
statement that opening the menu changes nothing. Group choices around:
signet create/connect, inspect/sync, Codex connect/update, doctor, repair and exit.
Use explanatory language alongside themed names; users need not know lore.

Each journey collects inputs, shows exact paths and effects, then offers
default-No confirmation. Escape or :back cancels before applying; EOF and Ctrl-C
restore terminal state and exit without treating partial input as consent.
After an action, show outcome, local-versus-remote status, preserved resources
and a concrete next step. Show partial success as partial success, not a generic
all-or-nothing failure. Never launch an agent or accept native trust silently.

Arrows and j/k move selection, Enter selects; text fields retain normal letter
entry. Use visible selection markers plus restrained cyan/green/amber/red; color
never carries the sole meaning. Plain mode uses numbered options, stable labels
and no cursor-control sequences. Honor NO_COLOR/non-TTY/dumb terminals. Wrap
long paths and messages; test narrow terminals without hiding critical effects.

Creation defaults to a neutral signet name and prompts for machine label/writer
attribution. It does not silently publish hostnames or assume a provider account.
Existing binding collisions ask for another path; no default overwrite. Existing
clones preserve signet identity but enroll this device independently.

## Installation decisions

Use the CLI-embedded public plugin so an installed binary needs no source tree.
Pin its own selected runtime copy plus binding in a generated local shell bridge,
keeping explicit per-process overrides available and visible to doctor.
Native Codex remains responsible for plugin registration/cache and hook trust.

Retained immutable packages cost some disk space but make partial replacement
and rollback diagnosable. No automatic garbage collection, arbitrary edits to
native TOML, custom auth store or signature authority is introduced. An artifact
hash proves byte identity, not that an arbitrary executable is trustworthy.
