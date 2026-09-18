// Synthetic compiled-CLI evidence only. Never select a personal binding.
// Usage: node native-driver.mjs /absolute/path/to/mandalore
import assert from 'node:assert/strict';
import {spawnSync} from 'node:child_process';
import {createHash} from 'node:crypto';
import {mkdtempSync, mkdirSync, readFileSync, writeFileSync, readdirSync, lstatSync, realpathSync} from 'node:fs';
import {tmpdir} from 'node:os';
import {join, isAbsolute} from 'node:path';

const binary = process.argv[2];
assert.ok(binary && isAbsolute(binary), 'explicit absolute test binary required');
const root = realpathSync(mkdtempSync(join(tmpdir(), 'mandalore-export-native-')));
const bank = join(root, 'bank'), binding = join(root, 'config', 'binding.json');
const hash = b => createHash('sha256').update(b).digest('hex');
const results = [];
function invoke(args, input, expected = true) {
  const child = spawnSync(binary, args, {input: JSON.stringify(input ?? {}), encoding: 'utf8', timeout: 30000, maxBuffer: 1048576});
  assert.equal(child.error, undefined, 'CLI process failed');
  const envelope = JSON.parse(child.stdout);
  assert.equal(envelope.protocol_version, 1);
  assert.equal(envelope.ok, expected);
  assert.equal(child.status === 0, expected);
  return envelope;
}
function inventory(path) {
  const entries = [];
  function walk(dir, relative = '') {
    for (const name of readdirSync(dir).sort()) {
      const file = join(dir, name), key = join(relative, name), st = lstatSync(file);
      assert.ok(!st.isSymbolicLink());
      entries.push([key, st.mode, st.mtimeMs, st.isFile() ? hash(readFileSync(file)) : null]);
      if (st.isDirectory()) walk(file, key);
    }
  }
  walk(path);
  return entries;
}
invoke(['call', 'signet_create'], {repository: bank, name: 'Synthetic report bank', device_label: 'Synthetic test host'});
invoke(['call', 'signet_bind'], {repository: bank, binding, device_label: 'Synthetic test host', actor: 'Synthetic tester'});
const remember = input => invoke(['memory', 'remember', '--binding', binding], input).result;
const old = remember({kind: 'fact', summary: 'Project', body: 'Copper Finch', basis: 'user-direction', reason: 'Initial synthetic name'});
const current = remember({kind: 'fact', summary: 'Project', body: 'Silver Heron', basis: 'user-direction', reason: 'Synthetic rename', record_id: old.record_id, supersedes: [old.id]});
remember({kind: 'fact', summary: 'Unselected', body: 'UNSELECTED_NATIVE_CANARY', basis: 'observation', reason: 'Synthetic unrelated entry'});
const journal = invoke(['memory', 'journal-append', '--binding', binding], {kind: 'test', summary: 'JOURNAL_NATIVE_CANARY'}).result;
const before = inventory(bank);
const request = {binding_path: binding, destination: join(root, 'report'), selection: {record_ids: [old.record_id]}};
const preview = invoke(['export', 'preview'], request);
const typed = invoke(['call', 'export_preview'], request);
assert.deepEqual(typed, preview);
assert.ok(!JSON.stringify(preview).includes('Silver Heron'));
assert.deepEqual(preview.result.projection.revision_ids, [current.id]);
results.push('preview metadata only; typed/wrapper parity; current supersession selected');
const applied = invoke(['export', 'apply'], preview).result;
assert.equal(applied.published, true); assert.equal(applied.durable, true);
const raw = readFileSync(join(request.destination, 'report.json'));
assert.equal(hash(raw), preview.result.report_sha256);
assert.ok(raw.includes('Silver Heron'));
for (const marker of ['Copper Finch', 'UNSELECTED_NATIVE_CANARY', 'JOURNAL_NATIVE_CANARY']) assert.ok(!raw.includes(marker));
assert.equal(lstatSync(request.destination).mode & 0o777, 0o700);
assert.equal(lstatSync(join(request.destination, 'report.json')).mode & 0o777, 0o600);
assert.equal(JSON.parse(raw).kind, 'mandalore-memory-report');
const replay = invoke(['export', 'apply'], preview, false);
assert.equal(replay.error.write_may_have_occurred, false);
results.push('reviewed apply exact hash; private modes; selected-only output; replay refuses overwrite');
const redacted = invoke(['export', 'preview'], {...request, destination: join(root, 'redacted'), selection: {...request.selection, include_history: true, include_details: true, journal_ids: [journal.id], omit_fields: ['identity','content','classification','timestamps','authorship','citations','change_history','extensions']}});
invoke(['call', 'export_apply'], redacted.result);
const hidden = JSON.parse(readFileSync(join(root, 'redacted', 'report.json')));
for (const item of hidden.items) assert.ok(Object.keys(item).every(k => ['ordinal','type','status'].includes(k)));
assert.equal(hidden.items.length, 3);
results.push('history and exact journal opt-ins; all-field omission leaves fixed structure only');
const denied = invoke(['export', 'apply', '--read-only'], {}, false);
assert.equal(denied.error.code, 'operation.read_only');
invoke(['export','preview'], {...request,destination:join(bank,'unsafe')},false);
const racer = invoke(['export','preview'], {...request,destination:join(root,'racing')});
mkdirSync(join(root,'racing'),{mode:0o700});
invoke(['export','apply'],racer,false);
assert.deepEqual(readdirSync(join(root,'racing')),[]);
assert.deepEqual(inventory(bank), before);
results.push('read-only denied; bank overlap denied; existing destination preserved; complete bank bytes/modes/mtimes unchanged');
const summary = {kind:'mandalore-synthetic-export-evidence', binary_sha256:hash(readFileSync(binary)), platform:process.platform, arch:process.arch, node:process.version, checks:results, limitations:['Synthetic CLI engineering evidence, not product acceptance','No user memory, network, Git initialization or provider authentication','No secure-erasure or safe-publication certification']};
writeFileSync(join(root,'evidence.json'),JSON.stringify(summary,null,2)+'\n',{mode:0o600});
console.log(JSON.stringify(summary,null,2));
console.log('EXPORT_NATIVE_PASSED');
console.log('Retained fixture: '+root);
