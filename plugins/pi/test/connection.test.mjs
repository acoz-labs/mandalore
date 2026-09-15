import assert from 'node:assert/strict';
import { createHash } from 'node:crypto';
import { execFileSync } from 'node:child_process';
import { appendFileSync, chmodSync, copyFileSync, cpSync, mkdirSync, mkdtempSync, readFileSync, readdirSync, rmSync, writeFileSync } from 'node:fs';
import { tmpdir } from 'node:os';
import { join } from 'node:path';
import { fileURLToPath } from 'node:url';
import test, {after, before} from 'node:test';
import { openConnection, packageDigest, readConnection } from '../package/connection.js';

const sourceRoot = fileURLToPath(new URL('../../../', import.meta.url));
const packageRoot = fileURLToPath(new URL('../package/', import.meta.url));
const buildRoot = mkdtempSync(join(tmpdir(), 'mandalore-pi-compiled-'));
const compiled = join(buildRoot, 'mandalore');
const digest = data => createHash('sha256').update(data).digest('hex');
before(() => execFileSync('go', ['build', '-o', compiled, './cmd/mandalore'], {cwd: sourceRoot, timeout: 120000, maxBuffer: 1048576}));
after(() => rmSync(buildRoot, {recursive: true, force: true}));

function fixture(t, readOnly = false) {
  const root = mkdtempSync(join(tmpdir(), 'mandalore-pi-connection-'));
  t.after(() => rmSync(root, {recursive: true, force: true}));
  const runtime = join(root, 'runtime');
  copyFileSync(compiled, runtime);
  chmodSync(runtime, 0o700);
  const invoke = (args, input) => JSON.parse(execFileSync(runtime, args, {input: input === undefined ? '' : JSON.stringify(input), encoding: 'utf8', timeout: 10000}));
  const bank = join(root, 'signet');
  const binding = join(root, 'binding.json');
  invoke(['signet', 'create', '--repository', bank, '--name', 'Example', '--device-label', 'Test device']);
  invoke(['signet', 'bind', '--repository', bank, '--binding', binding, '--device-label', 'Bound device', '--actor', 'Example actor']);
  const bound = JSON.parse(readFileSync(binding, 'utf8'));
  const info = invoke(['call', 'pi_package_inspect'], {}).result;
  const connectionRoot = join(root, 'connection');
  const pkg = join(connectionRoot, 'package');
  mkdirSync(connectionRoot);
  cpSync(packageRoot, pkg, {recursive: true});
  const config = {
    schema_version: 1, harness: 'pi', runtime, runtime_sha256: digest(readFileSync(runtime)),
    binding, binding_sha256: digest(readFileSync(binding)), signet_id: bound.signet_id,
    package_sha256: info.sha256, package_version: info.version, read_only: readOnly,
    native_home: join(root, 'native'), native_binary: join(root, 'pi'),
    state_dir: join(root, 'state'), connection_root: connectionRoot,
  };
  writeFileSync(join(pkg, 'connection.json'), JSON.stringify(config));
  return {root, pkg, bank, binding, bound, config, invoke};
}

function snapshot(root) {
  const result = {};
  function walk(path, prefix = '') {
    for (const entry of readdirSync(path, {withFileTypes: true})) {
      const name = join(prefix, entry.name);
      if (entry.isDirectory()) walk(join(path, entry.name), name);
      else result[name] = digest(readFileSync(join(path, entry.name)));
    }
  }
  walk(root);
  return result;
}

test('actual runtime/package identities agree; native calls share memory and provenance', async t => {
  const f = fixture(t);
  assert.equal(packageDigest(f.pkg), f.config.package_sha256);
  assert.deepEqual(readConnection(f.pkg).config, f.config);
  const connection = await openConnection(f.pkg);
  t.after(() => connection.close());
  assert.equal(connection.operations.length, 18);
  assert.ok(connection.operations.every(op => op.requires_binding && !op.cli_only));
  assert.equal(connection.operations.find(op => op.name === 'memory_context'), undefined);
  const saved = await connection.call('memory_remember', {kind: 'preference', summary: 'Answer style', body: 'Prefer concise answers.', basis: 'user-direction', reason: 'Confirmed'});
  assert.equal(saved.result.durable_locally, true);
  const history = await connection.call('memory_history', {record_id: saved.result.record_id});
  assert.equal(history.result.items[0].authorship.harness, 'pi');
  assert.equal(history.result.items[0].authorship.device_id, f.bound.device_id);
  const before = snapshot(f.bank);
  const context = await connection.context('concise answers');
  assert.match(context.context, /Prefer concise answers/);
  await connection.context('界'.repeat(30000));
  assert.deepEqual(snapshot(f.bank), before);
  await assert.rejects(connection.call('signet_create', {}), error => error.envelope.error.code === 'operation.unknown');
});

test('read-only is enforced by the real CLI; context and denied tools preserve the bank', async t => {
  const f = fixture(t, true);
  const before = snapshot(f.bank);
  const connection = await openConnection(f.pkg);
  t.after(() => connection.close());
  await connection.context('this is the way');
  for (const name of ['memory_remember', 'memory_journal_append', 'memory_sync']) {
    const response = await connection.call(name, {});
    assert.equal(response.error.code, 'operation.read_only');
    assert.notEqual(response.error.write_may_have_occurred, true);
  }
  assert.deepEqual(snapshot(f.bank), before);
});

test('binding replacement and changed executable cannot redirect an existing connection', async t => {
  const first = fixture(t);
  const other = fixture(t);
  const connection = await openConnection(first.pkg);
  t.after(() => connection.close());
  const beforeFirst = snapshot(first.bank);
  const beforeOther = snapshot(other.bank);
  writeFileSync(first.binding, readFileSync(other.binding));
  const refused = await connection.call('memory_remember', {kind: 'fact', summary: 'Wrong bank', body: 'Do not save', basis: 'observation', reason: 'Probe'});
  assert.equal(refused.error.code, 'binding.invalid');
  assert.deepEqual(snapshot(first.bank), beforeFirst);
  assert.deepEqual(snapshot(other.bank), beforeOther);
  appendFileSync(first.config.runtime, '\n');
  await assert.rejects(connection.call('memory_remember', {}), error => error.envelope.error.code === 'runtime.invalid' && !error.envelope.error.write_may_have_occurred);
});

test('changed package or malformed local metadata is refused before native attachment', async t => {
  const f = fixture(t);
  const configPath = join(f.pkg, 'connection.json');
  for (const patch of [{read_only: 'false'}, {binding: 'relative.json'}, {binding_sha256: [f.config.binding_sha256]}, {extra: true}]) {
    writeFileSync(configPath, JSON.stringify({...f.config, ...patch}));
    assert.throws(() => readConnection(f.pkg), error => error.envelope.error.code === 'connection.invalid');
  }
  writeFileSync(configPath, JSON.stringify(f.config));
  appendFileSync(join(f.pkg, 'index.js'), '\n// Changed bytes\n');
  await assert.rejects(openConnection(f.pkg), error => error.envelope.error.code === 'connection.invalid');
});

test('close disables new requests without dispatching another child', async t => {
  const f = fixture(t);
  const connection = await openConnection(f.pkg);
  await connection.close();
  await connection.close();
  await assert.rejects(connection.call('memory_remember', {}), error => error.envelope.error.code === 'binding.invalid' && !error.envelope.error.write_may_have_occurred);
});
