// Capture the actual plain menu with synthetic targets. Existing lab binding only.
// node menu.mjs <built-runtime> <native-codex> <lab-from-verify>
import assert from 'node:assert/strict';
import {spawnSync} from 'node:child_process';
import {createHash} from 'node:crypto';
import {existsSync, readFileSync, realpathSync, writeFileSync} from 'node:fs';
import {join} from 'node:path';
import {homedir} from 'node:os';
const [runtimeArg, nativeArg, labArg] = process.argv.slice(2);
const runtime = realpathSync(runtimeArg), native = realpathSync(nativeArg), lab = realpathSync(labArg);
const hash = bytes => createHash('sha256').update(bytes).digest('hex');
const env = {...process.env, NO_COLOR: '1', XDG_CONFIG_HOME: join(lab, 'config'),
  XDG_DATA_HOME: join(lab, 'data'), XDG_STATE_HOME: join(lab, 'local-state')};
for (const name of ['MANDALORE_BINDING', 'MANDALORE_BIN', 'CODEX_HOME', 'PI_CODING_AGENT_DIR']) delete env[name];
const failing = join(lab, 'synthetic-native-failure');
writeFileSync(failing, '#!/bin/sh\necho PRIVATE_CHILD_OUTPUT_SENTINEL >&2\nexit 7\n', {flag: 'wx', mode: 0o700});
const evidence = [];
for (const scenario of ['failure', 'success']) {
  const profile = join(lab, 'menu-' + scenario + '-profile');
  assert.equal(existsSync(profile), false);
  const result = spawnSync(runtime, ['menu', '--plain', '--binding', join(lab, 'binding.json'),
    '--binary', runtime, '--native-binary', scenario === 'failure' ? failing : native,
    '--native-home', profile, '--state-dir', join(lab, 'menu-' + scenario + '-state')],
    {input: '5\n1\n\n2\n10\n', env, cwd: lab, encoding: 'utf8', timeout: 120000, maxBuffer: 2 ** 20});
  assert.ifError(result.error);
  const raw = result.stdout + result.stderr;
  writeFileSync(join(lab, 'menu-' + scenario + '.raw.txt'), raw, {flag: 'wx', mode: 0o600});
  assert.equal(result.status, scenario === 'failure' ? 1 : 0);
  assert.equal(raw.includes('PRIVATE_CHILD_OUTPUT_SENTINEL'), false);
  if (scenario === 'failure') {
    assert.ok(raw.includes('selected Codex profile inventory (plugin marketplace list --json)'));
    assert.ok(raw.includes('preflight'));
  } else {
    assert.ok(raw.includes('verified'));
  }
  assert.equal(existsSync(profile), true);
  const sanitized = raw.split(native).join('/synthetic/native-codex')
    .split(runtime).join('/synthetic/mandalore').split(lab).join('/synthetic/lab')
    .split(homedir()).join('/synthetic/home');
  assert.equal(/\/Users\/|\/home\/[^\s]|\/private\/tmp\//.test(sanitized.replaceAll('/synthetic/home/', '/synthetic/profile/')), false,
    'sanitized output still contains a workstation path; do not publish');
  writeFileSync(join(lab, 'menu-' + scenario + '.txt'), sanitized, {flag: 'wx', mode: 0o600});
  evidence.push({scenario, exit: result.status, original_sha256: hash(raw), sanitized_sha256: hash(sanitized)});
}
const report = {runtime_sha256: hash(readFileSync(runtime)), evidence,
  capture: 'Actual plain-mode subprocess output; paths replaced with synthetic labels. No generated screenshots.',
  limits: 'Keyboard plain-mode flow and diagnostic text only; no screen-reader, color, font, locale or pixel-regression claim.'};
writeFileSync(join(lab, 'menu-evidence.json'), JSON.stringify(report, null, 2) + '\n', {flag: 'wx', mode: 0o600});
console.log(JSON.stringify(report, null, 2));
