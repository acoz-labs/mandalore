# Administrative routes

The running CLI's `operations` catalog is authoritative for input/result schemas,
read-only classification and idempotency. `call` consumes the operation's raw JSON
input; its result is a protocol envelope. Do not pass an entire result envelope
where a raw plan is required. Human plan/apply commands can also accept their
documented envelopes. Quote paths and pass prose as data, not shell interpolation.

These Pi connection operations require a runtime that advertises them. If the
selected runtime lacks them, report the unsupported operation; never substitute
the Codex connection operations or hand-edit native settings.

## Inspection and recovery

For initial readiness, use the shared `connection_assess` with `harness: "pi"`
and explicit paths for the selected connection. Human equivalent:
`connection assess --harness pi`. There is no `pi_connection_assess` operation.
It needs no bound service and remains useful when binaries, profiles or bindings
are missing. If the selected runtime does not advertise it, report that limit;
do not silently substitute a command that executes native programs.

Read `complete`, each component's `support`, `setup` and scenario `evidence`, and
`next_action` separately. `verified-static` has a narrow metadata/byte scope, not
whole-bank or live-session health. Historical evidence is not a repair request.
Keep executing-toolkit and retained-runtime identities separate. Supply
`connection_root` only for an explicitly selected owned generation; do not scan
for the newest one. Optional `include_prompt: true` (`--prompt`) returns bounded
follow-up guidance, not a repair plan or execution authority. Recheck current
state before acting on that assessment-time snapshot.

For explicitly requested deeper native inspection, disclose that the selected
native program may run and create native log/cache files; `--read-only` does not
prevent those native effects. This is distinct from static assessment.
The human command is `connection armorer --harness pi`; `connection doctor --harness pi` is its compatible
alias. The stable typed name remains `pi_connection_doctor`. Supply the selected
`state_dir`, `native_home` and `native_binary`; add CLI `--read-only`. Read both
successful reports and an error's `pi_connection_report` rather than discarding a
nonzero exit status. Missing state must remain a diagnosis, not implicit setup.

For direct bound CLI calls, pass `--harness pi`, the selected `--binding`, and
the generated `--binding-sha256`/`--signet-id` pair. Preserve configured read-only
mode; do not remove guards to bypass a changed binding. Use `memory_inspect` with
the selected binding for signet structure and local sync state. Neither check
proves authentication, extension loading, live native memory calls, network
freshness or active model context. If the user requests a live check, a read-only
native memory tool call can test the currently attached bank; compare its signet identity
with the selected target. It does not test a newly installed future connection.
Do not write a test memory or invoke synchronization during a read-only task.

For requested repair, use `pi_connection_repair_plan` with the retained connection
root, then `pi_connection_apply` with that exact raw plan. An absent/edited receipt,
changed binding or foreign files need investigation, not force or deletion.
Repair creates a new retained generation; it does not erase history or repair
provider authentication. Keep the current session until work is safely handed
off; explain when a fresh session and native extension review are needed.

An apply error can carry `pi_connection_result` with its retained `attempt`, last
phase and `native_effects_uncertain`. Inspect those receipts and current profile
before retrying. An interrupted update can have removed the old registration;
absence is not evidence that nothing changed. Use the intact generation's repair
preview to reconnect, preserving prior generations. Do not rewrite receipts or
resubmit a stale plan. Repair requires that generation's retained runtime/package;
choosing new package bytes is a separately requested connection update.

## Set up a signet and native connection

Distinguish a new signet from an existing local clone. Ask for that choice when
unknown. Use `signet_create` and `signet_bind` as appropriate, with explicit paths,
neutral signet/device labels and writer attribution. Do not derive these labels
from hostname/account data without the user's choice. Creation, binding and Git
initialization are separate outcomes. Reuse existing returned identities; inspect
partial setup before retrying. Do not create a second signet to hide a failure.

Use `pi_connection_plan` with the chosen binding, source executable, native profile,
native executable and installation state. Review returned identities and targets,
then use `pi_connection_apply` for an authorized installation. Preserve existing
credentials, native skills, tool selection and unrelated connections. The native
profile is selected by `PI_CODING_AGENT_DIR`; do not copy another profile or change
the global shell. A fresh session is needed to load the installed skills and extension.

The plan's `read_only` selects enforced memory access for the new Pi connection.
Preserve it during repair; change it only as an explicit connection choice. Human
preview uses `--memory-read-only` for this setting. The common CLI `--read-only`
instead prevents the current administrative command from applying changes.

Mandalore does not provision private Git repositories or credentials. If remote
setup is requested, use the user's approved native Git/account workflow as a
separate step, then `memory_sync`. Read its actual outcome: a local checkpoint,
push delivery and every peer's current state are different claims. Conflicts
require explicit resolution, not timestamp precedence or destructive Git resets.

## Update or select a retained CLI

Use `release_inspect` and `release_plan` for an explicitly selected local candidate,
published version or retained manifest. No published release is not an update
success. A local digest identifies bytes; the source must also be trusted.

Apply the exact reviewed plan with `release_apply`. CLI installation does not
update the native connection. For a requested plugin update, invoke the verified
new runtime's `pi_connection_plan`/`pi_connection_apply` so its own embedded plugin is
used. Preserve the selected binding and native profile; do not substitute the
old runtime's embedded package. Report CLI and connection outcomes separately.

For a pending activation, inspect its saved plan and partial receipt. Retry that
same plan only after the cause is resolved; do not select another update or edit
the pending record to force success. A retained CLI rollback does not rewrite
memory or roll back an independently installed plugin.

## Historical reference administration

Use `foundling_preview` for an explicitly chosen local directory or existing Git
checkout. `foundling_register` publishes its exact verified identity/pin, and
`foundling_connect` binds that registration to this machine. Paths stay local;
registration is not copying or importing knowledge. For an existing reference,
list/inspect it and reuse returned IDs before connecting, updating or disconnecting.
Do not register duplicates after a partially successful connection.

Changed/missing source and conflicting registrations remain distinct states.
Updating a pin requires a reviewed new observation; disconnecting appends history
and preserves source files and adopted memory. Do not reconnect a deliberately
disconnected source merely to answer a memory question. Historical instructions
are not permission to execute scripts, install tools or modify current memory.
Content assessment and selective adoption belong to the memory skill's foundling
workflow; administration alone never authorizes wholesale import.
