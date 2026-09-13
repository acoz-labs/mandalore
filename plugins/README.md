# Native integrations

Each harness gets a thin adapter over the same Mandalore memory engine.
Do not duplicate storage, retrieval or supersession semantics.

- `codex/`: first integration, issue #6.
- `pi/`: follows Codex acceptance, issue #13.
- `claude-code/`: follows Pi acceptance, issue #14.

These directories currently reserve integration boundaries, not installable
plugins. All plugins use the display name Mandalore; native identifiers use
`mandalore`, and the principal memory skill is `this-is-the-way`.
