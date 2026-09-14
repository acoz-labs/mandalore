# Context

The owner approved conversational setup/maintenance and the name The Armorer before the first release (#45). Current menus already delegate to typed CLI operations. The native memory skill only covers recall/learning.

Affected components: cmd/mandalore/connection.go and menu.go; internal/install/stage.go; plugins/codex/plugins/mandalore/skills; setup/interface docs. A managed native session does not inherit the bridge process's MANDALORE_BIN or MANDALORE_BINDING exports, so the skill must not assume those variables exist.

Journeys: inspect a configured bank; diagnose missing setup without creating it; preview requested setup/repair/update with explicit scope; handle partial failure and fresh-session handoff. Fresh setup can only use this skill if the plugin has already been installed or an agent has access to its instructions; it is not a bootstrap installer.

Product-design review: retain existing keyboard/plain/no-color patterns. Use The Armorer labels for inspection and repair as separate existing menu actions; do not label inspection as automatic repair. Conversational output leads with outcome, necessary next choices and untested boundaries.

