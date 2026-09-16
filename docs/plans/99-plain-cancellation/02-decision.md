# Decision

Keep the bounded buffered reader, but race each pending plain line read against
the menu context using one result channel with capacity one. Recheck cancellation
before parsing/returning an answer. The consumer stops on cancellation even when
the OS reader has not yet returned.

Relying only on stdin.Close is disproven by native evidence. Making stdin globally
nonblocking is broader and risks TUI, JSON and MCP semantics. Calling os.Exit from
the line reader bypasses testability and cleanup. The scoped wait is the smallest
correction and retains the caller's normal exit130 path.

Cancellation cannot undo input already consumed by the worker, but that input is
never used to apply a plan. At most one read remains outstanding per cancelled
menu invocation; buffered completion cannot block the worker. Tests explicitly
release their blocking readers, and process termination releases native resources.
