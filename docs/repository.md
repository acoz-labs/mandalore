# Repository-specific instructions

Shared delivery rules are in `docs/operations/sdlc.md`. They supersede older
mandatory planning/owner gates and prospective self-review exceptions. Local verification and `docs/delivery.md` supersede Actions-only and mandatory staging instructions. Historical
records remain evidence. Product behavior, privacy boundaries and actual
validation/release requirements remain in force.

## Product boundary

Mandalore is a memory-only product. Follow docs/product.md and docs/migration.md.
Use synthetic examples; preserve privacy, licenses and upstream attribution.
Never import personal memory, credentials, workstation paths, machine inventory
or raw transcripts into this public repository. Assigned product delivery may
include release; unrelated live signet migration or installation is not implied.
Historical ADR0001–0003 exceptions do not authorize new self-reviewed deliveries.

## Commands

Run:

```sh
bin/container bin/ci
```

If container support is not ready yet, or there's no terminal (the container
needs a TTY), run:

```sh
bin/ci
```

Document any required deviation in `docs/development.md`.

- `bin/ci` (either form). Healthy: exits 0 and ends with `Engine/interface
  checks and target builds passed; cross-builds are not native acceptance.`
  It skips strict plugin validation when `claude` isn't on `PATH`, so review
  plugin changes where `claude` is installed.
- `bin/check-old-reader` builds the released 2.1.2 binary (79b378e) in a
  temporary worktree and runs it against signets this tree writes topic files
  into, on both formats. It needs full history, so `bin/ci` doesn't run it;
  run it for any change to the signet layout, intents or sync paths. Healthy:
  exits 0 with `--- PASS: TestOldReaderOnTopicSignet`.
- `bin/sdlc check`. Healthy: exits 0 and prints
  `{"version": "<version>", "conforms": true}`.
- `go run golang.org/x/vuln/cmd/govulncheck@v1.9.0 ./...` needs no install and
  runs in every 2.x review. Main has no reachable standard-library
  advisories, so any reachable finding on a PR is new and blocks. GO-2026-5932
  (`x/crypto/openpgp`) is not called and has no fix; it is known and accepted.
- Run receipt test commands with `GOFLAGS=-count=1` so Go doesn't serve
  results from its cache.
- Use `git grep -P` for word-boundary sweeps: `git grep -E` has no `\b` and
  silently misses matches.
- List ignored or untracked files with the pathspec `':!.mandalore'`, which
  leaves out the product-owned `.mandalore/`.
- The local release path: `bin/sdlc-release nominate`, the acceptance
  receipt, `bin/promote-local-candidate` (which runs the gate),
  `bin/release-publication verify`, then `bin/sdlc-release record` and
  finalize. `bin/finalize-release artifact` and `--preflight` are
  Actions-only and exit 64 locally by design.
- The publisher needs `RELEASE_POLICY_READ_TOKEN` (`docs/deployment.md`).
  Without it, it refuses before any write: check that no draft or tag
  exists, then rerun with the token.

## Conventions

- Public receipts and evidence say "the owner", never a personal name, and
  cite only evidence posted on GitHub. Results relayed by an operator stay
  out of the repository.
- The reviewer publishes the `bin/sdlc-evidence` validation receipt on the PR
  for the exact head before merge. `sdlc-evidence publish` refuses a merged
  PR, and a merge without a receipt fails `bin/sdlc audit` for good (#167).
  Post-merge verification receipts are posted by hand.
- Evidence gathered after approval goes on the PR as a comment, not a commit,
  so the approved head doesn't move.
- Any merge to main after approval, even docs-only, means a rebase, a
  re-review and a fresh receipt. Fold pending low fixes into the same push.
  Before merging, check `git merge-base --is-ancestor origin/main <head>`:
  GitHub's clean or mergeable state doesn't mean the branch is on current
  main.
- Merged branches are not deleted automatically; delete the branch after
  merge.
- A `VERSION` or `plugin.json` bump means an artifact release
  (`docs/development.md`, `docs/operations/sdlc.md`). Plugin-only changes
  must bump `plugin.json` for installs to update, so they ship as a patch
  release and their PR says `Refs`, not `Closes`.
- Installed plugins track the marketplace on main (#229), so merging a plugin
  change is its rollout. Until #229 is fixed, a plugin-touching PR merges only
  after the owner confirms the live server supports the change.
- `bin/sdlc-release` nominates only commits that contain the PR's merge
  commit, so a pre-merge candidate's binary always differs at release in its
  embedded `source_commit`. Builds are reproducible (`-trimpath`,
  `-buildvcs=false`, `-buildid=`): name a candidate by tree hash and expect
  the binary to differ only in `source_commit`.
- A retained `sdlc-release` nomination can't be overwritten. After a
  mid-release fix PR, the new nomination names only the final PR, and the
  acceptance evidence links the earlier PR's reviews.
- Write a release record's evidence section only from posted evidence, and
  check every claim it copies was actually exercised. Plan the
  post-publication record (doc pointers) as its own contributor PR.
- Any new public OAuth or HTTP path joins the live Cloudflare WAF path
  allowlist at rollout. Put that step in the release notes for the owner.
- Behind cloudflared every peer is `127.0.0.1`, so per-`RemoteAddr` limits
  are shared by everyone. A Cloudflare rate limit on Free or Pro can't
  prevent a lockout; use an `ip.src` allowlist.
- A bare-binary server upgrades by atomic swap plus `launchctl kickstart -k`.
  `release install` and `install.sh` refuse that prefix safely.
- Image publishing can't use GitHub-hosted Actions: `bin/ci` and the SDLC
  standard require trusted self-hosted runners, and `SDLC_ACTIONS_ENABLED`
  is unset (#188).
- Any field added to the state file needs a rollback note: older versions
  drop unknown fields on their first write.
- `docs/operations/local-verification.md` is SDLC-managed and hash-pinned.
  Point repository-specific procedure at `docs/repository.md` instead.
- In 2.x, unknown subcommands under `server`, `release`, `export` and
  `migration` return `input.invalid`, not `operation.unknown`.
- Relevance-threshold work gets a 4-round budget, not 3.

## Known mistakes

- Latency asserts in `bin/ci` assert only the bound the spec sets (the warm
  path); everything else gets a loose tripwire. Post-write p95 runs near
  300 ms under load.
- Timing tests use fetch hooks, not wall-clock sleeps; sleeps caused a
  load-sensitive flake on #162.
- Tests kill the servers they start by exact PID before reporting. Detached
  `mandalore serve` processes from #161 outlived their run.
- Point `HOME` and `CLAUDE_CONFIG_DIR` at a fresh temporary directory, with
  no keychain link and an environment token, before any command that
  inspects or changes Claude Code configuration, including any `claude mcp`
  command. A temporary `HOME` alone isn't enough: Claude Code still uses the
  real login when `CLAUDE_CONFIG_DIR` is unset. Native OAuth runs keep
  `mcpOAuth` state in the login keychain; run `claude mcp logout
  plugin:mandalore:mandalore` before and after, only inside that isolation.
- Claude Code (2.1.310) revokes on `claude mcp logout`: refresh token first,
  then access token, with `client_id` in the form.
- Claude Code `mcp_tool` hooks inject only JSON
  `hookSpecificOutput.additionalContext`, not plain text. `mcp_tool` on
  `SessionStart` errors at every launch, whatever the matcher.
- Claude Code (2.1.310) drops a `plugin:` MCP server when a claude.ai
  connector has the same normalized URL, and `mcp_tool`'s `server` is an
  exact name with no `user_config` substitution, so plugin hooks can't follow
  a renamed or synced server. Fix it with `deniedMcpServers` for the
  connector (#201).
- The claude.ai connector syncs into `claude mcp list`, so guards ignore
  `claude.ai …` lines and match keychain `mcpOAuth` entries by `serverUrl`.
  Headless Macs need `claude mcp login --no-browser` under a pty.
- `GITHUB_TOKEN` isn't in a step's environment by default. Set
  `env: GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}` and grant only
  `permissions: packages: write`.
- A per-installation identity change must reach the MCP handler cache key.
  A cached handler holds its client and would give a second installation the
  first one's authorship.
- After a format break, check every hardcoded asset count or shape in the
  release wrappers against the new manifest, and sweep the whole repository:
  `bin/` shell and jq, `.github/workflows`, `internal/releaseworkflow` and
  every `cmd/*promote*`. Their tests fake the old count; on #193 the issue
  named 1 wrapper and there were 4.
- The publisher's release-history size caps grow with every release: on
  2.0.2 the `/releases?per_page=100` lookup outgrew a 128 KiB cap (#214).
  Check list-size limits before nominating.
- One `release-publication verify` uses about 16 anonymous GitHub API
  requests, and the anonymous quota is 60 an hour per IP, so a day of release
  checks runs out. Run `install.sh` and `--version` probes from a verified
  local release copy, after the reset, or with an authenticated read.
- When a repository-wide grep should find no old version, dated evidence
  records (`docs/evidence/**`, `*-engineering-evidence.md`) are the expected
  exception. Exclude them by explicit path and put the grep in the PR.

## Current delivery

Profile: `artifact`. Follow [the local delivery runbook](delivery.md)
and [portable evidence commands](operations/local-verification.md). All configured
application validation remains required. Persistent staging and Actions are
optional; independent acceptance and exact-artifact verification are mandatory.
`bin/finalize-release artifact` and `bin/promote-candidate` are Actions-only and
are not part of the local release; see [publication](delivery.md#publication-and-recovery).
