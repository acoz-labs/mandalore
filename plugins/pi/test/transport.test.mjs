import assert from 'node:assert/strict';
import { mkdtemp, readFile, rm } from 'node:fs/promises';
import { tmpdir } from 'node:os';
import { join } from 'node:path';
import { fileURLToPath } from 'node:url';
import { setTimeout as delay } from 'node:timers/promises';
import test from 'node:test';
import { runJSON, TransportError } from '../package/transport.js';

const fixture = fileURLToPath(new URL('./process-fixture.mjs', import.meta.url));
const run = (mode, input = {}, options = {}, args = []) => runJSON(process.execPath, [fixture, mode, ...args], input, options);

test('shell-free arguments and JSON input are literal; valid envelope is unchanged', async () => {
  const args = ['a b', '$(false)', '; false', '"quoted"'];
  const input = {value: '雪; $(false)'};
  assert.deepEqual(await run('arguments', input, {}, args), {protocol_version: 1, ok: true, result: {args, input}});
});

test('nonzero complete envelopes retain cancellation and partial-write evidence', async () => {
  const result = await run('partial', {}, {mutating: true});
  assert.equal(result.ok, false);
  assert.equal(result.error.code, 'operation.cancelled');
  assert.equal(result.error.write_may_have_occurred, true);
  assert.equal(result.error.sync_status.phase, 'push');
});

test('invalid framing, protocol, exit, UTF-8 and output limits are bounded failures', async () => {
  for (const mode of ['double', 'protocol', 'exit', 'false-success', 'utf8', 'output-limit', 'stderr-limit']) {
    await assert.rejects(run(mode, {}, {mutating: true, timeoutMs: 3000}), error => {
      assert.ok(error instanceof TransportError);
      assert.equal(error.envelope.ok, false);
      assert.equal(error.envelope.error.write_may_have_occurred, true);
      assert.equal(error.envelope.error.inspect_before_retry, true);
      assert.doesNotMatch(JSON.stringify(error.envelope), /PRIVATE_.*CANARY/);
      return true;
    }, mode);
  }
});

test('catalog allows bounded discovery overhead without increasing memory response limits', async () => {
  await assert.rejects(run('catalog'), TransportError);
  const result = await run('catalog', undefined, {maxOutputBytes: 1048576});
  assert.equal(result.result.value.length, 70000);
});

test('rejection before dispatch never claims a write; no raw executable error is exposed', async () => {
  const controller = new AbortController();
  controller.abort();
  for (const invoke of [
    () => run('normal', {}, {signal: controller.signal, mutating: true}),
    () => run('normal', {large: '雪'.repeat(20000)}, {mutating: true}),
    () => runJSON('/absent/PRIVATE_EXECUTABLE_CANARY', [], {}, {mutating: true}),
  ]) {
    await assert.rejects(invoke(), error => {
      assert.ok(error instanceof TransportError);
      assert.equal(error.envelope.error.write_may_have_occurred, false);
      assert.doesNotMatch(error.message, /PRIVATE_EXECUTABLE_CANARY/);
      return true;
    });
  }
});

async function ready(path) {
  const end = Date.now() + 5000;
  while (Date.now() < end) {
    try {
      const value = await readFile(path, 'utf8');
      if (/^[1-9][0-9]*\n$/.test(value)) return Number(value.trim());
    } catch (error) { if (error.code !== 'ENOENT') throw error; }
    await delay(10);
  }
  throw new Error('Synthetic process did not publish readiness');
}

test('abort waits for reaping and preserves a complete receipt when available', async t => {
  for (const mode of ['ignore', 'group', 'cancel-receipt']) {
    const root = await mkdtemp(join(tmpdir(), 'mandalore-pi-transport-'));
    t.after(() => rm(root, {recursive: true, force: true}));
    const controller = new AbortController();
    t.after(() => controller.abort());
    const marker = join(root, 'ready');
    const result = run(mode, {}, {mutating: true, signal: controller.signal, graceMs: 100, timeoutMs: 10000}, [marker]).then(value => ({value}), error => ({error}));
    const pid = await ready(marker);
    controller.abort();
    const out = await result;
    if (mode !== 'cancel-receipt') {
      assert.equal(out.error.envelope.error.code, 'operation.cancelled');
      assert.equal(out.error.envelope.error.write_may_have_occurred, true);
    } else {
      assert.equal(out.value.result.saved.durable_locally, true);
      assert.equal(out.value.result.delivery.ok, false);
    }
    assert.throws(() => process.kill(pid, 0), {code: 'ESRCH'});
  }
});

test('deadline terminates an uncooperative child without an automatic retry', async t => {
  const root = await mkdtemp(join(tmpdir(), 'mandalore-pi-deadline-'));
  t.after(() => rm(root, {recursive: true, force: true}));
  const marker = join(root, 'ready');
  const result = run('ignore', {}, {mutating: true, timeoutMs: 5000, graceMs: 100}, [marker]).then(value => ({value}), error => ({error}));
  const pid = await ready(marker);
  const {error} = await result;
  assert.equal(error.envelope.error.code, 'operation.cancelled');
  assert.equal(error.envelope.error.inspect_before_retry, true);
  assert.throws(() => process.kill(pid, 0), {code: 'ESRCH'});
});
