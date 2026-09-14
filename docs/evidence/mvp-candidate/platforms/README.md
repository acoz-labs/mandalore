# Remaining native CLI platforms: Linux ARM64 and Intel macOS

Both native retained-candidate probe jobs passed. This completes actual CLI/MCP
execution evidence across the four declared target combinations, together with
the earlier macOS ARM64 and [Linux AMD64](../linux/README.md) checkpoints. It does
not establish native Codex conversations, terminal UI acceptance or independent
product acceptance on all four platforms.

## Immutable identities

Candidate source is `5b3c7b275bb4bbb058bebdd7153f4b76bbceb59f`; manifest is
`c46743709dcd02b106c4e6d35cf58247486537e0212603eb7e369ecdfa1bd237`.
Both executables came from the original retained artifact `10346171185`, whose
complete archive and live provenance were checked before extraction. Neither
candidate nor plugin was rebuilt or replaced.

| Target | Actual public runner selection | Retained executable SHA-256 |
| --- | --- | --- |
| Linux ARM64 | `ubuntu-24.04-arm`; host guard `Linux/aarch64` | `ae3e6c37522170bd869b983aae4177623676f73294b9bece32102d6d12f81002` |
| macOS AMD64 | `macos-15-intel`; host guard `Darwin/x86_64`, translation refused | `c9a250e884a2fc9c3d69dbeb351ad8ee6bbdb490e92632f841afb8544a4adbfa` |

Collector [head `640df7265c6e94be7e3ac8a4358f90ba30a2f88b`](https://github.com/acoz-labs/mandalore/tree/640df7265c6e94be7e3ac8a4358f90ba30a2f88b/verification/retained-candidate)
and verification-only [PR #41](https://github.com/acoz-labs/mandalore/pull/41)
extend the prior Linux collector. [Run `34851082821`](https://github.com/acoz-labs/mandalore/actions/runs/34851082821),
attempt 1, passed standard CI and both platform jobs. PR merge checkout
`9b8fea2e47a05f7f3d426b70186d39ff212930d4` is collector context, not a new candidate.

| Evidence | Artifact | ZIP bytes | ZIP SHA-256 |
| --- | --- | --- | --- |
| [Linux ARM64 receipts](linux-arm64/summary.json) | `10351321913` | 5,601 | `543810b481b95282563c74aacb987eb136499bbc6294be32288cb83dc236d484` |
| [macOS AMD64 receipts](darwin-amd64/summary.json) | `10351182394` | 5,592 | `401876f13188264886c20cb42c4ae20f5614f15836a683534f4f47d3d409b701` |

Artifacts expire 2026-12-13. Both archives were downloaded in the designated test
pane, matched to live successful-run metadata and their exact SHA-256, checked
for the ten-member regular JSON inventory, and extracted into new directories.
Native identity, clone/installed-version equality and exact migration numeric
value were checked again. Published JSON is unchanged artifact content; the
additional `receipt-hashes.json` files record each original member's digest.

## Results and isolation

Each actual platform binary passed same-record Copper Finch to Silver Heron
correction/history, current recall, real stdio MCP initialization/tool discovery,
read-only journal refusal and unchanged signet hashes, local bare Git delivery
and fresh-clone recall, CLI activation in an owned temporary prefix, explicit
synthetic legacy migration with unchanged original snapshot and exact historical
JSON number. The unrelated project cwd stayed empty. Both MCP tools/list
responses were 44,087 bytes; serialized bytes are not model tokens.

The candidate process received a minimal PATH/locale environment, without
Actions/provider credentials. Trusted transport verification alone used the
read-only workflow token. Candidate-execution job checkouts did not persist Git
credentials. No model, native-agent login, shell setup, real memory, private
runner, local VM, repository acceptance setting or release operation was used.

The standard `CI_RUNNER=ubuntu-latest` was not changed. Two explicit non-secret
test variables selected the documented [standard public runners](https://docs.github.com/en/actions/reference/runners/github-hosted-runners):
`CI_RUNNER_LINUX_ARM64=ubuntu-24.04-arm` and
`CI_RUNNER_MACOS_AMD64=macos-15-intel`. This temporary deviation is documented on
the verification branch, which is not intended for product merge. No missing
check, emulation or cross-compilation is substituted for native execution.

The native-session, multi-plugin coexistence and independent-acceptance limits
remain in the [requirement audit](../acceptance-audit.md). This does not prove
every OS version, architecture-specific fault mode, screen-reader behavior or
public-release download path. Those claims require their own evidence.
