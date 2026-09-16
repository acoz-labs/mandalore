// Opt-in real-native regression. Never point this at a personal profile.
// node verify.mjs <built-runtime> <native-codex> <new-empty-lab>
import assert from 'node:assert/strict';
import {spawnSync} from 'node:child_process';
import {createHash} from 'node:crypto';
import {existsSync, readFileSync, readdirSync, realpathSync, statSync, writeFileSync} from 'node:fs';
import {join} from 'node:path';

const [runtimeArg, nativeArg, labArg] = process.argv.slice(2);
assert.ok(runtimeArg && nativeArg && labArg, 'runtime, native Codex and empty lab required');
const runtime = realpathSync(runtimeArg), native = realpathSync(nativeArg), lab = realpathSync(labArg);
assert.deepEqual(readdirSync(lab), [], 'lab must be empty; no retries into partial state');
const hash = path => createHash('sha256').update(readFileSync(path)).digest('hex');
const env = {...process.env, XDG_CONFIG_HOME: join(lab, 'config'),
  XDG_DATA_HOME: join(lab, 'data'), XDG_STATE_HOME: join(lab, 'local-state')};
for (const name of ['MANDALORE_BINDING', 'MANDALORE_BIN', 'CODEX_HOME', 'PI_CODING_AGENT_DIR']) delete env[name];
function run(binary, args, input) {
  const result = spawnSync(binary, args, {input, env, cwd: lab, encoding: 'utf8', timeout: 120000, maxBuffer: 2 ** 20});
  assert.ifError(result.error);
  return result;
}
function call(operation, input) {
  const result = run(runtime, ['call', operation], JSON.stringify(input));
  // Private retained receipts may contain selected paths. Never print raw bytes.
  writeFileSync(join(lab, operation + '.json'), result.stdout, {flag: 'wx', mode: 0o600});
  assert.equal(result.status, 0, operation + ' failed; inspect the local receipt');
  const envelope = JSON.parse(result.stdout);
  assert.equal(envelope.ok, true);
  return envelope.result;
}
const version = run(native, ['--version']);
assert.equal(version.status, 0);
const catalog = JSON.parse(run(runtime, ['operations']).stdout).result.operations;
for (const name of ['signet_create', 'signet_bind', 'connection_plan', 'connection_apply', 'connection_doctor']) {
  assert.ok(catalog.some(o => o.name === name), 'missing operation ' + name);
}
call('signet_create', {repository: join(lab, 'signet'), name: 'Synthetic first-use bank', device_label: 'Test device'});
call('signet_bind', {repository: join(lab, 'signet'), binding: join(lab, 'binding.json'), device_label: 'Test device', actor: 'Synthetic tester'});
const selection = {state_dir: join(lab, 'state'), native_home: join(lab, 'native'), native_binary: native};
const bindingHash = hash(join(lab, 'binding.json'));
const plan = call('connection_plan', {...selection, binding: join(lab, 'binding.json'), source_binary: runtime});
assert.equal(existsSync(selection.native_home), false, 'preview created native profile');
assert.equal(existsSync(selection.state_dir), false, 'preview created installation state');
const result = call('connection_apply', plan);
assert.equal(result.installed, true);
assert.equal(statSync(selection.native_home).mode & 0o777, 0o700);
const report = call('connection_doctor', selection);
assert.equal(report.healthy, true);
assert.equal(hash(join(lab, 'binding.json')), bindingHash);
assert.equal(existsSync(join(selection.native_home, 'auth.json')), false, 'unexpected authentication file');
const evidence = {native_version: version.stdout.trim(), native_sha256: hash(native),
  runtime_sha256: hash(runtime), package_sha256: plan.package_sha256,
  preview_left_home_and_state_absent: true, installed: result.installed,
  phase: result.phase, healthy: report.healthy, profile_mode: '0700',
  binding_unchanged: true, no_auth_file: true,
  limits: 'Engineering installation and structural checks only; no provider login, hook trust, live model or owner acceptance.'};
writeFileSync(join(lab, 'evidence.json'), JSON.stringify(evidence, null, 2) + '\n', {flag: 'wx', mode: 0o600});
console.log(JSON.stringify(evidence, null, 2));
