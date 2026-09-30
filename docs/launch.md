# Launch a named signet

Available in Mandalore 1.4.0. Verify the installed CLI's help before use.

A launch entry gives an existing signet connection a machine-local name and
default agent. Start the default or choose another configured harness:

```sh
mandalore launch work
mandalore launch work --agent claude-code
mandalore launch personal --agent pi
mandalore launch personal --preview
```

The session keeps its selected signet. Launching another environment does not
change a global current signet or rewrite an already-running connection. Native
hooks own memory synchronization and canon foundling refresh; launching does not
add another independent fetch. Direct native launch remains supported.

## Configure once on each machine

First create or bind the intended signet and establish its native connections
through [setup](setup.md). Use a dedicated native profile for each environment.
Authenticate through the native harness's supported mechanism. Mandalore does not
copy authentication, native histories, personal notes, or another profile.

Run `mandalore launch schema` for the exact local entry schema.
A local entry contains the binding, default harness and explicit native connection
paths. For example, save this synthetic selection outside any signet:

```json
{
  "binding": "/example/work/binding.json",
  "default_agent": "claude-code",
  "agents": {
    "claude-code": {
      "native_home": "/example/work/claude-profile",
      "native_binary": "/example/bin/claude",
      "state_dir": "/example/work/installation",
      "connection_root": "/example/work/installation/connections/GENERATION"
    }
  }
}
```

Use the actual paths returned by the verified connection receipt. The configure
command checks the connection and pins binding identity; it does not establish a
missing connection. Confirm that any existing native history in this profile
belongs to this environment before claiming it:

```sh
mandalore launch configure work --confirm-profile < entry.json
mandalore launch list
mandalore launch work --preview
```

The default configuration is `mandalore/launch.json` under the platform user
configuration directory. `--config /example/local/launch.json` selects an explicit
file. Both the launch registry and profile ownership are local. They do not
travel with a signet; set them up explicitly on another machine.

To configure multiple harnesses, add `codex`, `pi` or `claude-code` selections to
`agents`, each with its own verified native connection. `default_agent` must name
one of them. `--agent` overrides it for one invocation only. An unknown agent,
missing executable, changed binding or mismatched profile is an error, not a
reason to fall back to personal memory or another agent.

## Upgrade native agents independently

Mandalore 1.5.0 checks native interfaces instead of requiring the exact Claude
Code or Pi version used for its original tests. Compatible agent upgrades can
keep the same named entry, signet and native profile without reconnecting.
Codex registration is checked through its structured native inventory. Claude
and Pi also check the CLI capabilities used by their adapters. An untested
release is not automatically incompatible; a failed capability check identifies
the missing interface.

When connecting, select the installation's stable executable launcher when one
is available. Mandalore retains that locator alongside the resolved executable
and its preview digest. Existing entries with version-specific targets can
follow the same installation's Claude native launcher, Codex standalone
`current` link, or mise Pi `latest` link. This also handles removal of the old
target. Mandalore does not search arbitrary `PATH` entries or select another
installation when that locator is broken or redirected outside its expected
installation. An unrecognized missing executable needs explicit selection of a
trusted launcher.

Each launch validates the current executable and native registration, then
rechecks the connection and executable snapshot before starting the agent.
Retained receipts, binding identity, profile ownership, runtime and plugin
integrity remain guarded. Upgrading the agent does not change memory access,
transport authorization, native history or authentication. A native update that
races a launch is refused; retry after the update finishes.

Preview is a local structural check. Native inspection can run the agent's
inventory/help commands and create native logs or caches; it does not prove
that hooks, tools, login or model context work in a fresh session. Those remain
separate runtime checks. Updating the Mandalore plugin itself still follows the
stopped-session handoff in [setup](setup.md).

## Native arguments and continuation

Place native arguments after `--`. Arguments are passed directly, without shell
evaluation. The launcher preserves the calling working directory, interactive
terminal and native exit behavior. Inspect native help for supported arguments.
Routing overrides that could change the selected profile or connection are
refused. Resume must retain the selected environment; a foreign session requires
a fresh session rather than importing its conversation into another signet.

A separate signet/profile controls memory routing. It is not an operating-system
sandbox: authorized native tools can still access files and services according
to their native permissions. Project-local instructions and native memory remain
native features. Do not claim that selecting a signet erases previously loaded
context or grants a different security identity.

## Optional shell shortcuts

Put convenience functions in your own local shell configuration if useful:

```sh
work() { mandalore launch work "$@"; }
personal() { mandalore launch personal "$@"; }
```

Mandalore does not edit shell startup files. The supported command remains
`mandalore launch`; the shorthand is local to your shell.

## References and freshness

A primary signet may register another signet as a **canon** foundling. Native
session entry/resume attempts a bounded refresh, and later reads use that
session's verified snapshot. Failed transport produces stale or unavailable
reference receipts. A launch banner identifies the chosen memory environment;
it does not prove that a remote fetch or native context assembly succeeded.
See [foundlings](foundlings.md) for provenance, offline behavior and legacy
references, and [synchronization](synchronization.md) for primary-signet delivery.
