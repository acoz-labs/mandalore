# Context

Issue #7 turns the working development memory engine and native Codex plugin
into a repeatable setup experience. Today users must create/bind a bank, build a
binary, register a marketplace, export paths and review native hooks themselves.
The #6 native receipt proves these components can work, not a finished installer.

Current ownership: internal/api administration and bound operations;
internal/binding for explicit device enrollment; internal/sync for standalone
Git; plugins/codex for source assets; cmd/mandalore for text/JSON CLI. No menu,
embedded bundle or managed installation exists at the repository basis.

Review the pinned public predecessor f3d337bca419fdaf82bd9a6ce31ebde7f3748eb8:
internal/memoryinstall/{bundle,native}.go, internal/console, local staging in
internal/toolkitupdate, and memory-only portions of cmd/my-friday/portable_bank*
and portable_menu.go. Reuse reviewed behavior/tests, not the portable assistant
framework, release manifest, legacy namespace or implicit host-label defaults.

Journeys: create a local signet and initialize its Git history; connect an
already-cloned signet on another machine with a new device binding; inspect or
sync a selected signet; preview/connect native Codex; inspect health; choose a
new local runtime; repair missing managed files without erasing edited state.
Failures retain completed steps and explain the next safe action.

The MVP does not provision hosting accounts/private remote repositories or own
credential enrollment. Existing Git remotes/authentication remain native; menu
text must explain local-only status and that an existing remote clone can be
connected. Remote discovery/download/update acceptance follows #11. Native
thread portability, live predecessor migration and foundling setup are separate.

Verified: the selected Codex CLI accepts native marketplace/plugin commands and
preserves its own auth/resources. Native inventory schemas and generated cache
behavior must be checked on actual 0.153.4, not inferred from predecessor tests.
Changing registration can partially succeed; retain both old and new packages.
