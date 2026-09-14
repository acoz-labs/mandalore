# Actual native inventory and writer-lock observations

Fresh retained-candidate contributor verification on macOS ARM64. Source
`5b3c7b275bb4bbb058bebdd7153f4b76bbceb59f`, executable SHA-256
`eed5c72490079d9ee7bb2d08b7bc7916c5b56f4473f91849dc6e6d5c9d9fb3fc`.
The native binary is the isolated UI test fixture's Codex 0.153.4, SHA-256
`b973d440acac501fd2594a43e7ca9ce41e0a65b9dfb28d0d7a7837c99e1261e3`.

The official [plugin-listing command reference](https://learn.chatgpt.com/docs/developer-commands?surface=cli)
was consulted before operating the existing test profile. No plugin installation,
removal, native authentication or agent conversation was performed. No API key was
needed or requested. The OpenAI Docs skill guided this read-only inventory check.

The source was a fresh copy of the committed synthetic legacy fixture. Explicit
`migration preflight --native-home ... --native-binary ...` invoked the selected
native listing commands and reported exactly one enabled `mandalore@mandalore`
integration. [The observation](native-observation.json) retains its installation
generation and **potential** writer count, without local paths. Native config
bytes and all portable source files matched before/after; output remained absent.
The native CLI may maintain its own logs/caches, so this is not a claim that every
native-home file remained unchanged.

A separate lock probe created only the fixture's legacy `.my-friday/write.lock`
and held an actual exclusive `flock` while executing the retained CLI:

1. Preflight refused the busy lock with `migration.failed`, explicitly reporting
   no write may have occurred.
2. After release, fresh preflight reported the existing lock available.
3. Reacquiring the lock before `migration apply --writers-stopped` made apply
   refuse in preflight, without publishing or creating its output.
4. Releasing again restored successful preflight with the same source digest.

The [summary](lock-summary.json) and refusal receipts preserve these outcomes.
The apply refusal is a sanitized projection: ephemeral output/signet/receipt path
fields are omitted; status, code, phase and write/publication flags are unchanged.
This check did not publish a conversion or activate a writer.

Installed/enabled plugins are not proof of active sessions, same-bank targeting,
or other-machine quiescence. The test profile actually targets another synthetic
signet, which is why the inventory correctly says only *potential*. This does not
prove simultaneous old/new native installations, arbitrary noncooperating-writer
safety or remote writer shutdown. Those limits remain distinct from the successful
single-plugin inventory and cooperating-lock checks.
