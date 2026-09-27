# Launch a named signet

Proposed for the next release; verify the installed CLI's help before use.

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
