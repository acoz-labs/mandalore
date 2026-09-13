# Solution decision

Reuse the predecessor's characterized machine-binding and MCP adapter concepts,
but not its assistant-aware CLI routing or native plugin installation package.
Put machine-local connection logic in internal/binding, shared typed operations
and discovery in internal/api, the stdio adapter in internal/mcp, and process/flag
handling in cmd/mandalore. The memory package remains dependency-independent.

Expose explicit setup commands plus a consistent memory command family and
machine-call path. CLI and MCP call the same operation methods; neither
reimplements default scope, byte budgets, correction semantics or write receipts.
Use the pinned upstream Go MCP SDK after inspecting its actual schema and
transport behavior; do not hand-roll a partial JSON-RPC protocol.

Creating a signet and enrolling/binding a device are separate inspectable actions.
Failure can leave the created signet or an unused device record; preserve it and
report partial outcomes instead of deleting user data or inventing atomicity.
A running MCP server cannot silently switch banks or create arbitrary new banks.

The protocol gets its own explicit version and stable error codes. Runtime
version remains development-only until distribution defines release identity.
Credentials, hooks, native sessions and model extraction are out of scope.
