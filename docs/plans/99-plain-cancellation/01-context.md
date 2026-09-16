# Context

#99 is a bounded regression found during #85/#88 acceptance. On macOS arm64,
the retained binary prints Ctrl+C at a plain menu prompt but remains blocked;
Enter permits exit130. Direct terminal and nested real-PTY observations agree.
TUI cancellation and plain EOF pass. No application effects were observed.

`cmd/mandalore/main.go` already cancels the process context and closes stdin.
`menu.line` nevertheless waits for `bufio.Reader.ReadSlice`; closing a terminal
descriptor does not guarantee a currently blocked read returns. Existing pipe
and already-cancelled-context tests are insufficient.

Users and agents need prompt cancellation at numbered choices, text entry and
default-No confirmation without an additional key. No CLI grammar, memory,
credentials, signal policy, native integration or release-control changes.

This is an in-pattern interaction correction, not a new product-design flow.
Retain native evidence for partial input, EOF, narrow/plain and TUI regression.
