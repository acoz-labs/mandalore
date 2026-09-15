# Pi integration

Development integration for issue #13, after accepted Codex MVP #10. The native
package is not yet a released or accepted connection workflow. Installation,
recovery, skills and native model verification remain part of that issue.

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
