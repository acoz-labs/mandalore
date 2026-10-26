# Cutover from 1.x to the 2.0 server

In 1.x every device holds its own Git clone of the signet. In 2.0 one
[server](server.md) owns the signet and is its only writer. The cutover happens
once, with `mandalore server migrate`. Old clones are retired, never deleted.
This follows section 6 of the [2.0 design](design/2.0.md).

## Commands

### `preflight --clone PATH [--format json|text]`

Run it on each device, against that device's clone. It is read-only: it lists
the remote ref but never fetches, commits or writes state. It checks:

- the signet passes full validation;
- no uncommitted files and no unpushed commits (ignored files count when they
  are memory; `.mandalore/`, including its canon cache, is never listed);
- no pending write intents in `.mandalore/`: an `operation-*.json` left by an
  interrupted razor-crest write, or a prepared format upgrade that isn't active
  (see [Pending intents](#pending-intents));
- the clone hasn't diverged from its `origin`, and `origin` is reachable;
- the signet is format 2 (see [Format 1 signets](#format-1-signets)).

It reports the device ID from the local binding (`MANDALORE_BINDING`, else the
platform default) when that binding names this clone. If no binding names it,
preflight blocks, because `init` can't use a receipt without a device: point
`MANDALORE_BINDING` at the clone's binding, or mark the device lost at init. A clone that is only
behind `origin` is clean: it has nothing the remote lacks. If `origin` has
commits the clone has never fetched, preflight can't tell behind from diverged
without fetching, so it blocks; sync once more and preflight again.

Exit codes: `0` clean (safe to retire), `1` blocked, `2` the clone could not be
inspected; the error names its cause, such as a Git command exceeding its
output limit. The JSON output is the receipt `init` takes; save it to a file:

```sh
mandalore server migrate preflight --clone ~/signet > device-a.json
```

### `init --config FILE --remote URL --slug S --root DIR`

Run it on the server, with the server stopped. `--config` is the server
configuration; it may start with `"signets": []`. `init`:

1. clones `main` from `--remote` fresh into a working directory beside `--root`
   (which must not exist), and runs full validation;
2. refuses unless every device enrolled in `provenance/devices/` has either a
   clean receipt passed with `--preflight FILE` whose head is on the remote, or
   is marked with `--device-lost ID --reason TEXT`;
3. enrolls the server's own device, commits it and pushes it;
4. records each device's evidence (receipt head and time, or the lost reason)
   in `state_dir/signets/<slug>/migration.json`. Reasons stay in server state,
   never in the signet;
5. moves the checkout to `--root` and adds the signet's entry to the
   configuration.

Repeat `--preflight` and `--device-lost ID --reason TEXT` as needed; each
`--reason` belongs to the `--device-lost` just before it. A device enrolled by
`signet create` without a clone of its own is still enrolled: mark it lost with
a reason that says so. Init checks that each receipt's head is on the remote,
not how old the receipt is: pass the receipts from runbook step 5, taken after
1.x sessions stopped, not earlier ones.

A refusal (`migrate.refused`) changes no signet, remote or configuration and
leaves no checkout. Init first checks that it can write the configuration and
its state, so an unwritable one is refused before anything is pushed. A failure
after the enrollment (`migrate.incomplete`, with `write_may_have_occurred`)
names the checkout at root, the server device and whether it was pushed; see
[Recovery](#recovery).

### `retire --clone PATH --server URL [--native HARNESS=HOME,BINARY]...`

Run it on each device after preflight passed. For each `--native` profile
(`claude-code`, `codex`, `pi` or `hermes`, with the profile's home and the
harness's executable) it finds enabled Mandalore connections with the 1.x
inventory code and stops each with the harness's own command:

| Harness | Command |
| --- | --- |
| Claude Code | `claude plugin disable mandalore@mandalore --scope user` |
| Codex | `codex plugin marketplace remove <marketplace> --json`, as 1.x does when it replaces a connection |
| Pi | `pi remove <package path>` (Pi has no disable; the package files stay) |
| Hermes | `hermes plugins disable mandalore` |

It checks each profile again afterwards. If any disable fails or a connection is
still enabled, it stops before touching the clone. If an earlier disable in the
same run already succeeded, the failure is `migrate.incomplete` and lists it:
keep that output, since a rerun reports only its own disables
(`disabled_this_run`) and the profiles it found clear (`profiles_clear`). Two
Mandalore plugins from one Codex marketplace are removed with one command. Then it writes `RETIRED.md`
(date, signet ID and server URL) and removes write permission from the clone,
as `chmod -R a-w` would, leaving `.git` as it was. It deletes nothing. A 1.x
write to the clone then fails with a permission error, and 1.x validation
reports the `RETIRED.md` entry. Rerunning `retire` finishes an interrupted one.

Name every profile that has Mandalore: a profile retire isn't told about keeps
its connection.

### `verify --config FILE --slug S`

Run it on the server after `init`, with the server still stopped: it writes as
the server, so it refuses while the `listen` address answers. It checks that the
checkout's head equals the remote's `main`, reads, writes a canary memory in
the `task` scope `migration-verify`, checks recall shows it, withdraws it,
checks recall no longer shows it, and pushes. It passes only when the remote's
`main` is then the pushed head. The withdrawn canary stays in history.

## Runbook

1. Upgrade a format 1 signet to format 2. Skip this step when the signet is
   already format 2; otherwise preflight blocks it. Use the 1.x binary
   (`mandalore call ... < input.json`):
   1. Stop every 1.x writer on every device, including any razor-crest
      replica.
   2. On one device, run `signet_upgrade_preview` against that clone's binding
      and review the plan.
   3. Run `signet_upgrade_apply` with that exact plan and
      `stopped_writers: true`.
   4. Sync that clone, so the remote has the format 2 signet.

   Each clone opts in separately: another device's format 1 clone doesn't adopt
   the upgrade from the remote, so run 1.2–1.4 on each clone before it syncs
   again. Clients too old to read format 2 refuse the upgraded signet; update
   them before they use it.
2. Preflight every device. Fix what each one reports.
3. Stop all 1.x sessions on every device, including any razor-crest replica.
4. Sync each device a final time.
5. Preflight every device again and keep each receipt. A device that reports
   unfetched remote commits needs one more sync.
6. Retire each device, naming each of its native profiles.
7. On the server, `init`, then `verify`.
8. Start the server (`mandalore serve --config FILE`) and check
   `mandalore server status`.
9. Install the 2.0 plugin.

A device you can't reach is marked lost at step 7. If it comes back, retire it
before it syncs; the server refuses to serve a signet whose remote moved.

## Pending intents

A 1.x clone serving razor-crest writes an `operation-*.json` intent in
`.mandalore/` before it publishes each write. If the publication never
happened, the intent holds the only copy of that entry, and only the client
retrying with the same `request_id` completes it; no operator command does.
Either retry it from that client before stopping 1.x sessions, or keep the
entry (the `revision`, `source` or `journal` in the file), move the file out of
`.mandalore/`, and save the entry again through the 2.0 server after the
cutover. Then sync and preflight again.

## Recovery

A command that stops after writing reports `migrate.incomplete` with
`write_may_have_occurred` and says what it wrote.

- **Init, before the push.** The checkout is at root with the server device
  enrolled but not pushed. Remove root and rerun init. If the message says the
  push failed, check the remote first: if its `main` has moved past the head
  init cloned, treat it as after the push.
- **Init, after the push.** The server device is enrolled on the remote, so a
  plain rerun refuses with that device among those missing a preflight. Remove
  root and rerun init adding `--device-lost <that server device> --reason
  "Server enrollment from an init that stopped"`. A new server device is
  enrolled; the old record stays in provenance, unused.
- **Verify.** The message gives the canary's record ID and how far it got. If
  it is committed but not pushed, start the server once: it commits and pushes
  pending work when it starts. Stop it and rerun verify. If the canary is still
  visible, withdraw it with `memory_withdraw` from a 2.0 client once the server
  runs.
- **Retire.** The failure output names each profile with the state its
  configuration shows after the disable: disabled, unchanged and still enabled,
  failed, or not reached. Keep it, fix the cause and rerun retire; it skips what
  is already disabled.

## Rollback

Before the first 2.0 write (verify's canary doesn't count; it is withdrawn), you
can go back to 1.x: stop the server, un-retire one clone with `chmod -R u+w`,
delete its `RETIRED.md`, re-enable its native connection and restart 1.x. That
clone becomes the writer again; keep the others retired, or un-retire them the
same way.

After the first 2.0 write, never write from 1.x again. A retired clone may still
be read as an older copy.

## The CLI install

Upgrading the CLI is separate from the signet cutover. A 1.x binary can't
install 2.0, so bootstrap with the 2.0 release's own script. It verifies the
platform binary against `SHA256SUMS` and runs that binary's installer:

```sh
curl -fsSLO https://github.com/acoz-labs/mandalore/releases/download/v2.1.0/install.sh
sh install.sh 2.1.0
```

In a prefix that holds a 1.x install (both default to `~/.local`), 2.0 updates
it in place and keeps the 1.x runtime as the previous one. To go back to 1.6.1,
use the 2.0 launcher's retained rollback:

```sh
mandalore release install --retained 6b9f5107b0d7dfd8bf8d0323b38e52276b9d0ec702ffee2b1b983af0e5824f1d --prefix PREFIX
```

1.6.1 can't read a 2.0 install, so it can't return to 2.0 by itself. Run
`sh install.sh 2.1.0` again, or the retained 2.0 runtime under
`PREFIX/lib/mandalore/releases/` with `release install --retained` and its
identity. 1.6.1's own `sh install.sh 1.6.1` refuses a prefix that holds 2.0.
Where no 1.6.1 runtime is retained, as after a fresh 2.0 install, move both
`PREFIX/lib/mandalore` and `PREFIX/bin/mandalore` aside first, then install
1.6.1. Moving only one of them is refused. Put both back to undo.

Rolling the CLI back doesn't roll the signet back; see [Rollback](#rollback).

## Format 1 signets

Verify withdraws its canary, and withdrawal needs format 2, so preflight blocks
format 1 and `init` refuses it. Upgrade first, as step 1 of the
[runbook](#runbook) describes.
