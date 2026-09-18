// Synthetic compiled-CLI exercise. Never select a personal binding or remote.
// node native-driver.mjs /absolute/new-binary /absolute/released-old-binary
import assert from 'node:assert/strict';
import {spawnSync} from 'node:child_process';
import {createHash} from 'node:crypto';
import {mkdirSync, mkdtempSync, readFileSync, writeFileSync, readdirSync, lstatSync, realpathSync} from 'node:fs';
import {tmpdir} from 'node:os';
import {join, isAbsolute} from 'node:path';
import {fileURLToPath} from 'node:url';

const [binary, legacy] = process.argv.slice(2);
assert.ok(binary && legacy && isAbsolute(binary) && isAbsolute(legacy), 'two explicit absolute binaries required');
const root = realpathSync(mkdtempSync(join(tmpdir(), 'mandalore-withdrawal-native-')));
console.log('Retained synthetic fixture: ' + root);
mkdirSync(join(root, 'config'), {mode: 0o700});
const bank = join(root, 'bank'), binding = join(root, 'config', 'binding.json');
const clone = join(root, 'clone'), cloneBinding = join(root, 'config', 'clone-binding.json'), remote = join(root, 'remote.git');
const hash = data => createHash('sha256').update(data).digest('hex');
const checks = [];
function invoke(args, input = {}, expected = true, executable = binary) {
  const p = spawnSync(executable, args, {input: JSON.stringify(input), encoding: 'utf8', timeout: 40000, maxBuffer: 1048576});
  assert.equal(p.error, undefined, 'CLI process failed');
  const result = JSON.parse(p.stdout);
  assert.equal(result.protocol_version, 1);
  assert.equal(result.ok, expected, JSON.stringify({args, error: result.error}));
  assert.equal(p.status === 0, expected);
  return result;
}
const call = (name, input = {}, selected = binding, expected = true) => invoke(['call', name, '--binding', selected], input, expected);
function git(cwd, args) {
  const p = spawnSync('git', ['-c', 'core.hooksPath=/dev/null', '-c', 'commit.gpgsign=false', '-C', cwd, ...args], {encoding: 'utf8', timeout: 30000});
  assert.equal(p.status, 0, 'synthetic Git command failed');
  return p.stdout.trim();
}
function inventory(directory) {
  const entries = {};
  function walk(dir, prefix = '') {
    for (const name of readdirSync(dir).sort()) {
      if (name === '.git' || name === '.mandalore') continue;
      const path = join(dir, name), key = join(prefix, name), st = lstatSync(path);
      assert.ok(!st.isSymbolicLink());
      if (st.isDirectory()) walk(path, key);
      else entries[key] = {mode: st.mode & 0o777, sha256: hash(readFileSync(path))};
    }
  }
  walk(directory);
  return entries;
}
const version = invoke(['version']).result;
const oldVersion = invoke(['version'], {}, true, legacy).result;
assert.ok(version.signet_read_versions.includes(2));
assert.ok(!oldVersion.signet_read_versions.includes(2));
invoke(['call', 'signet_create'], {repository: bank, name: 'Synthetic visibility bank', device_label: 'Synthetic device'});
invoke(['call', 'signet_bind'], {repository: bank, binding, device_label: 'Synthetic device', actor: 'Synthetic tester'});
const remembered = call('memory_remember', {kind: 'fact', summary: 'Fictional route', body: 'Orchid Pier', basis: 'user-direction', reason: 'Synthetic initial choice'}).result;
const journal = call('memory_journal_append', {kind: 'test', summary: 'Independent synthetic journal'}).result;
call('memory_git_init');
git(root, ['init', '--bare', '--initial-branch=main', remote]);
git(bank, ['remote', 'add', 'origin', remote]);
assert.equal(call('memory_sync', {timeout_seconds: 30}).result.delivered, true);
git(root, ['clone', remote, clone]);
invoke(['call', 'signet_bind'], {repository: clone, binding: cloneBinding, device_label: 'Second synthetic device', actor: 'Synthetic tester'});
for (const selected of [cloneBinding, binding]) assert.equal(call('memory_sync', {timeout_seconds: 30}, selected).result.delivered, true);
const before = inventory(bank), oldHead = git(bank, ['rev-parse', 'HEAD']);
const plan = invoke(['call', 'signet_upgrade_preview', '--read-only'], {binding_path: binding}).result;
assert.deepEqual(inventory(bank), before);
invoke(['call', 'signet_upgrade_apply'], {plan, stopped_writers: false}, false);
assert.deepEqual(inventory(bank), before);
const upgraded = invoke(['call', 'signet_upgrade_apply'], {plan, stopped_writers: true}).result;
assert.ok(upgraded.activated && upgraded.durable_locally && !upgraded.checkpointed && !upgraded.delivered);
assert.equal(git(bank, ['rev-parse', 'HEAD']), oldHead);
for (const [path, metadata] of Object.entries(before)) if (path !== 'signet.json') assert.deepEqual(inventory(bank)[path], metadata);
invoke(['call', 'signet_upgrade_apply'], {plan, stopped_writers: true}, false);
assert.equal(invoke(['call', 'signet_upgrade_recover'], {plan, stopped_writers: true}).result.evidence_id, upgraded.evidence_id);
checks.push('explicit preview/acknowledgement/activation/recovery; original evidence preserved; no implicit checkpoint');

const oldRefusal = invoke(['memory', 'recall', '--binding', binding], {}, false, legacy);
assert.equal(oldRefusal.error.code, 'binding.invalid');
assert.ok(JSON.stringify(invoke(['memory', 'recall', '--binding', cloneBinding], {}, true, legacy)).includes('Orchid Pier'));
checks.push('released old reader refuses upgraded bank; offline old clone still reads historical content');
function decisionInput(selected = binding) {
  const h = call('memory_visibility_history', {record_id: remembered.record_id}, selected).result;
  return {record_id: remembered.record_id, expected_content_heads: h.visibility.content_heads, expected_visibility_heads: h.visibility.visibility_heads, reason: 'Explicit synthetic visibility decision'};
}
const initialDecision = decisionInput();
const withdrawn = call('memory_withdraw', initialDecision).result;
assert.equal(withdrawn.visibility.state, 'withdrawn');
assert.equal(withdrawn.durable_locally, true);
assert.equal(call('memory_recall').result.current.length, 0);
assert.ok(!JSON.stringify(call('memory_context', {prompt: 'What is the fictional route?'})).includes('Orchid Pier'));
assert.equal(call('memory_scopes').result.items[0].visibility_counts.withheld, 1);
assert.equal(call('memory_restore', initialDecision, binding, false).error.code, 'memory.stale_heads');
const reviewRequest = {binding_path: binding, selection: {record_ids: [remembered.record_id]}, policy: {id: 'policy-review', visibility: 'withheld'}};
const reviewBefore = inventory(bank);
const review = invoke(['call', 'retention_preview', '--read-only'], reviewRequest).result;
assert.equal(review.records[0].matched, true);
assert.equal(review.journals.length, 0);
assert.ok(!JSON.stringify(review).includes('Orchid Pier'));
assert.deepEqual(inventory(bank), reviewBefore);
checks.push('withdrawal omitted from recall/context, counted explicitly; stale restore refuses; retention is metadata-only/read-only');

const reportRequest = {binding_path: binding, destination: join(root, 'report'), selection: {record_ids: [remembered.record_id]}};
const reportPlan = invoke(['call', 'export_preview'], reportRequest).result;
assert.equal(reportPlan.projection.withheld_records.length, 1);
invoke(['call', 'export_apply'], reportPlan);
assert.ok(!readFileSync(join(root, 'report', 'report.json'), 'utf8').includes('Orchid Pier'));
const historical = invoke(['call', 'export_preview'], {...reportRequest, destination: join(root, 'history-report'), selection: {...reportRequest.selection, include_history: true, include_withdrawn: true}}).result;
invoke(['call', 'export_apply'], historical);
assert.ok(readFileSync(join(root, 'history-report', 'report.json'), 'utf8').includes('Orchid Pier'));
assert.equal(call('memory_journal').result.items[0].id, journal.id);
checks.push('default export withholds; double historical opt-in discloses preserved history; journal remains independent');

assert.equal(call('memory_sync', {timeout_seconds: 30}).result.delivered, true);
const cloneHead = git(clone, ['rev-parse', 'HEAD']);
const refused = call('memory_sync', {timeout_seconds: 30}, cloneBinding).result;
assert.equal(refused.state, 'upgrade-required');
assert.equal(refused.delivered, false);
assert.equal(git(clone, ['rev-parse', 'HEAD']), cloneHead);
assert.equal(JSON.parse(readFileSync(join(clone, 'signet.json'))).schema_version, 1);
const secondPlan = invoke(['call', 'signet_upgrade_preview'], {binding_path: cloneBinding}).result;
invoke(['call', 'signet_upgrade_apply'], {plan: secondPlan, stopped_writers: true});
for (const selected of [cloneBinding, binding]) assert.equal(call('memory_sync', {timeout_seconds: 30}, selected).result.delivered, true);
assert.equal(call('memory_recall', {}, cloneBinding).result.current.length, 0);
assert.equal(readdirSync(join(bank, 'provenance', 'upgrades')).filter(x => x.endsWith('.json')).length, 2);
checks.push('format1 clone preserves head and reports upgrade-required; explicit independent upgrade converges with both receipts');

const staleContent = decisionInput();
const correction = call('memory_remember', {kind: 'fact', summary: 'Fictional route', body: 'Maple Quay', basis: 'user-direction', reason: 'Synthetic correction', record_id: remembered.record_id, supersedes: [remembered.id]}).result;
assert.equal(call('memory_recall').result.current.length, 0);
assert.equal(call('memory_restore', staleContent, binding, false).error.code, 'memory.stale_heads');
assert.equal(call('memory_restore', decisionInput()).result.visibility.state, 'visible');
assert.equal(call('memory_recall').result.current[0].id, correction.id);
assert.equal(call('memory_history', {record_id: remembered.record_id}).result.items.length, 2);
for (const selected of [binding, cloneBinding]) assert.equal(call('memory_sync', {timeout_seconds: 30}, selected).result.delivered, true);
assert.equal(call('memory_recall', {}, cloneBinding).result.current[0].id, correction.id);
checks.push('correction remains withdrawn; stale content restore refuses; explicit restore and cross-clone delivery retain history');
const summary = {kind: 'mandalore-synthetic-withdrawal-evidence', source_commit: version.source_commit, binary_sha256: hash(readFileSync(binary)), plugin_sha256: version.plugin_sha256, driver_sha256: hash(readFileSync(fileURLToPath(import.meta.url))), legacy_source_commit: oldVersion.source_commit, legacy_binary_sha256: hash(readFileSync(legacy)), platform: process.platform, arch: process.arch, node: process.version, checks, limitations: ['Compiled CLI engineering evidence, not Codex/Pi model behavior or product acceptance', 'Only synthetic banks and local bare Git remote; no personal memory, account or network host', 'Old offline copies and already-read context remain outside revocation; no erasure claim']};
writeFileSync(join(root, 'evidence.json'), JSON.stringify(summary, null, 2) + '\n', {mode: 0o600});
writeFileSync(join(root, 'fixture.json'), JSON.stringify({binary, legacy, bank, binding, clone, cloneBinding, root, remembered}, null, 2) + '\n', {mode: 0o600});
console.log(JSON.stringify(summary, null, 2));
console.log('WITHDRAWAL_NATIVE_CLI_PASSED');
