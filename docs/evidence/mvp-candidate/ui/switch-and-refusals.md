# Retained-runtime switching and installer refusals

Fresh contributor verification against the same retained candidate as
[the core UI checkpoint](README.md). Not independent acceptance, a new build,
public release or live-memory adoption. The menu driver is always source
`5b3c7b275bb4bbb058bebdd7153f4b76bbceb59f`, manifest
`c46743709dcd02b106c4e6d35cf58247486537e0212603eb7e369ecdfa1bd237`, ARM64 executable
`eed5c72490079d9ee7bb2d08b7bc7916c5b56f4473f91849dc6e6d5c9d9fb3fc`.

## Actual additional recordings

These unmodified BSD `script -qr` recordings use the same synthetic fixture,
macOS ARM64, English, 57 columns/30 rows and designated Herdr pane. Refusals use
plain/NO_COLOR; runtime switching uses color; navigation uses NO_COLOR TUI.

| Recording | Observed result |
| --- | --- |
| [corrupt](corrupt.recording) | A same-length comment-byte change in a copied bootstrap asset caused payload digest rejection before confirmation/install. |
| [unsupported](unsupported.recording) | A copied manifest with an unsupported declared target was rejected before confirmation/install. Not native execution on an unsupported platform. |
| [foreign](foreign.recording) | Existing unowned launcher refused; original file bytes preserved and no owned state created. |
| [offline](offline.recording) | Invocation-only unreachable loopback proxy made the real public request fail; no prefix created. No global network settings changed. |
| [no-release](no-release.recording) | Actual normal public discovery reported no published release; no installation. No GitHub credentials borrowed. |
| [eof](eof.recording) | Ctrl-D at the plain confirmation stopped without creating its prefix. |
| [older-install](older-install.recording) | Explicitly install the prior retained candidate into a fresh disposable prefix; leave native connections unchanged. |
| [newer-install](newer-install.recording) | Explicitly install the current candidate into that owned prefix; previous runtime shown and retained. |
| [retained-back](retained-back.recording) | Select the prior manifest digest from retained storage, approve rollback, keep native connection unchanged. |
| [retained-forward](retained-forward.recording) | Select the current retained manifest digest, approve forward switch, keep native connection unchanged. |
| [navigation](navigation.recording) | k/Enter opens the CLI source menu; all four source choices visible; j/k moves, Escape returns, gg/G and Enter exit. No source applied. |

The two negative candidate directories were deliberate copies. The original
current and older candidate trees passed all checksum checks after testing.
Refused destination paths remained absent; the foreign launcher matched its exact
original fixture bytes. Negative cases were not weakened into accepted inputs.

## Distinct retained identities and observed launches

The prior candidate is source `0fe7e0e1eb175943995b0d0d5e54130ff65b074e`, manifest
`829be4285324e00ab6c2fdde482c1471df13c74f8facab0e6fdcdba0ed3c09df`, executable
`b959ccc66d518c151a2efc1111bf2739736b3980308a4ee43889bd99b151b819`.
Its historical regression failure and acceptance limits remain unchanged. It is
used here only to verify switching, not nominated anew or approved for release.

Both artifacts have the intended version label `1.0.0`; they are not two published
semantic versions. Source commits and manifest/binary hashes distinguish them.
Both share plugin identity `f0e4df519fc6dbe70d69fd4191c6013b9017800aa4cfe3f45d2b5413872cfef1`.

Actual launcher `version` output was captured after every stage:

- [Initial older runtime](switch-old.json)
- [Current runtime after update](switch-new.json)
- [Prior runtime after retained rollback](switch-back.json)
- [Current runtime after retained forward switch](switch-forward.json)

Each expected pair matched byte-for-byte. Both retained executables remained
byte-identical to their source candidates; neither was removed. No pending record
remained after the completed sequence. Each native handoff stayed at its default
unchanged choice, so rolling back the CLI did not roll back a memory connection.

The post-switch native doctor result exactly matched the earlier final doctor
report. The UI signet/binding and the unrelated native-test bank retained all
baseline file hashes. No session, real credential or profile mutation was part of
this checkpoint. A local successful switch is not a public promotion or guaranteed
compatibility across arbitrary future schema versions.

## Review and limits

Contributor self-review checked failure wording, before-apply refusals, explicit
identity/previous-runtime display, default unchanged native handoff, keyboard
paths and the four actual launcher responses. Recording privacy and evidence
hash/whitespace checks passed. Replay caveats remain those in [README](README.md).
An actual container CI attempt could not reach the Docker daemon; the documented
pinned Go 1.26.4 host fallback passed full CI with cached test results and target
builds. No product source or candidate bytes changed during this checkpoint.

Together with the preceding checkpoint, these recordings cover the selected
installer success/refusal/recovery and distinct retained-runtime matrix on this
candidate. They do not prove native Linux/Intel Mac behavior, screen-reader or
alternate-locale support, independent product acceptance, a working published
release download, or initial bootstrap from a release that does not exist yet.
Those boundaries must remain explicit in the platform/release acceptance audit.
