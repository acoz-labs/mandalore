# Pi integration

Development integration for issue #13, after accepted Codex MVP #10. The native
package is not yet a released or accepted connection workflow. Typed CLI and
guided menu installation/recovery are implemented. Source-bound native model,
delivery and rendered evidence is recorded in
[engineering reconciliation](../../docs/evidence/pi/reconciliation.md);
final-head verification and immutable-candidate acceptance remain separate gates.

The dependency-free JavaScript extension discovers the retained runtime's typed
catalog and exposes its bound memory operations as native Pi tools. Every call
uses shell-free CLI transport, explicit guarded binding and `pi` provenance.
The Go engine remains authoritative for validation, memory, read-only boundaries
and synchronization. One complete JSON envelope is returned; result metadata
does not repeat memory content. Lost responses preserve possible-write ambiguity.

Only session start, per-turn context and shutdown are subscribed. Context is a
fresh bounded local read appended to Pi's existing system prompt, not a saved
message, transcript scan or automatic write. Startup checks connection identity
without a redundant record scan. Shutdown cancels and reaps owned calls. Native
tools, model access, authentication and session history remain Pi-owned.

`package/connection.json` is machine-local generated context, never embedded in
the public package. The extension verifies package and executable identities,
pins binding bytes and signet identity, and refuses ambiguous configuration.
`pi_package_inspect` is a CLI-only development operation reporting embedded
package identity; the existing generic version response is unchanged.

Run `mise exec -- node --test plugins/pi/test/*.test.mjs` from the repository root.
Tests include a compiled Go backend as well as controlled child processes and a
native-API double. These layers do not establish real Pi loading, model behavior
or candidate acceptance; native evidence must identify its actual versions and
source/artifact identity. Initial inspected contract: Pi 0.85.1 with Node 24.1.0.

The package supplies `this-is-the-way` and `the-armorer` through native skill
discovery and `/skill:NAME`. Names/descriptions are discovered first; bodies and
linked references are read when useful. Foundling and delivery reference bytes
are tested for parity with Codex; Codex-only MCP rendering instructions are not
included. The Armorer uses Pi's own connection operations; an older runtime
without them must not fall back to Codex installation operations.

The release builder stamps Pi's package version in its isolated source export
before compilation. Pi stays embedded in the CLI: no separate npm dependency,
extra release asset or format-1 manifest/version-response change is introduced.

## Explicit development connection

In `mandalore menu`, choose **Connection**, then **Pi**. Review the trusted
runtime, binding, native profile, installation state and memory access mode.
Preview executes that selected runtime's read-only planning operation so a newer
artifact supplies its own embedded Pi package; it does not register the package.
Applying has a separate default-No confirmation. The release installer likewise
offers Pi as an optional, separately confirmed connection after CLI installation;
declining leaves native connections unchanged.

The Armorer's inspection and repair journeys also offer a harness choice.
Repair uses the intact runtime identified by the owned receipt, preserving the
binding and enforced access mode even when the menu executable is newer. It
creates a fresh generation instead of rewriting the old one. Missing ownership,
changed bindings or edited files require inspection, not silent adoption.
Cancellation gives the selected runtime a bounded chance to stop/reap its native
child and return a partial receipt. Inspect that receipt before retrying; a
completed CLI installation is separate from a failed native connection.

Use the trusted selected Mandalore executable to preview and apply its own
embedded package. Pi and its Node runtime must already be installed; this does
not install them or change authentication. The native contract currently accepts
Pi 0.85.1 only; the initial native test baseline is Node 24.1.0 on macOS arm64.

```sh
mandalore connection plan --harness pi --binding /absolute/binding.json \
  --native-home /absolute/pi-profile --native-binary /absolute/pi \
  --state-dir /absolute/mandalore-state > pi-plan.json
# Review the plan before this explicitly mutating step.
mandalore connection apply --harness pi < pi-plan.json
mandalore connection armorer --harness pi --native-home /absolute/pi-profile \
  --native-binary /absolute/pi --state-dir /absolute/mandalore-state
```

Omitting the harness still selects Codex. Pi defaults to `PI_CODING_AGENT_DIR`
when set, otherwise its native `.pi/agent` home. Explicit paths in the typed
`pi_connection_plan/apply/doctor/repair_plan` operations avoid ambient selection.
The plan's `read_only` (`--memory-read-only`) enforces read-only memory in Pi;
the common `--read-only` flag instead prohibits this CLI call's mutations.

Native `pi install`/`remove` manage only an exact proven registration. Package
bytes, runtime and generated local context are retained outside the profile and
signet, with receipts. Other package settings/resource filters remain native.
Plans reject changed inputs, ambiguous/filtered memory registrations, edited
owned bytes and redirected managed paths. Native effects are recorded by phase;
an interrupted remove/install can leave the connection inactive. Inspect first,
then use `connection repair --harness pi --connection-root RETAINED_ROOT` to
preview a fresh generation and `--apply` only when requested. Unknown files and
changed bindings are not silently repaired. Existing sessions need a fresh start
or native reload; Armorer health does not prove live tools or model access.
