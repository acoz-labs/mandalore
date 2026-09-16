# Decision

Selected: move the existing conditional code-mode routing paragraph ahead of
recall and state prepare-before-first-call/reuse explicitly. Clarify the same
sequence in its linked reference without duplicating the helper in startup text.

Rejected: removing a wire field breaks compatibility; a global renderer change
is outside this project; embedding the full helper in every session expands
context even for clients that already select a representation. A new daemon,
tool or production JavaScript dependency is disproportionate to this failure.

This is a bounded testable hypothesis, not a promise of deterministic model
adherence. If fresh native runs still bypass selection, return the approach to
review with the counterexample rather than weakening #56 or hiding failures.
