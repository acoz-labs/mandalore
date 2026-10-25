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
- no uncommitted files and no unpushed commits;
- no pending write intents in `.mandalore/` (an interrupted 1.x write, or a
  prepared format upgrade that isn't active);
- the clone hasn't diverged from its `origin`, and `origin` is reachable;
- the signet is format 2 (see [Format 1 signets](#format-1-signets)).

It reports the device ID from the local binding (`MANDALORE_BINDING`, else the
platform default) when that binding names this clone. A clone that is only
behind `origin` is clean: it has nothing the remote lacks. If `origin` has
commits the clone has never fetched, preflight can't tell behind from diverged
without fetching, so it blocks; sync once more and preflight again.

Exit codes: `0` clean (safe to retire), `1` blocked, `2` the clone could not be
inspected. The JSON output is the receipt `init` takes; save it to a file:

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
a reason that says so. On refusal nothing is configured and no checkout is left.

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
still enabled, it stops before touching the clone. Then it writes `RETIRED.md`
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

1. Preflight every device. Fix what each one reports.
2. Stop all 1.x sessions on every device, including any razor-crest replica.
3. Sync each device a final time.
4. Preflight every device again and keep each receipt. A device that reports
   unfetched remote commits needs one more sync.
5. Retire each device, naming each of its native profiles.
6. On the server, `init`, then `verify`.
7. Start the server (`mandalore serve --config FILE`) and check
   `mandalore server status`.
8. Install the 2.0 plugin.

A device you can't reach is marked lost at step 6. If it comes back, retire it
before it syncs; the server refuses to serve a signet whose remote moved.

## Rollback

Before the first 2.0 write (verify's canary doesn't count; it is withdrawn), you
can go back to 1.x: stop the server, un-retire one clone with `chmod -R u+w`,
delete its `RETIRED.md`, re-enable its native connection and restart 1.x. That
clone becomes the writer again; keep the others retired, or un-retire them the
same way.

After the first 2.0 write, never write from 1.x again. A retired clone may still
be read as an older copy.

## Format 1 signets

Verify withdraws its canary, and withdrawal needs format 2, so preflight blocks
format 1 and `init` refuses it. Upgrade first with the 1.x
`signet_upgrade_preview` and `signet_upgrade_apply` operations, sync, then
start the cutover.
