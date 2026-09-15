import assert from 'node:assert/strict';
import test from 'node:test';
import { attach } from '../package/extension.js';
import { TransportError } from '../package/transport.js';

function fixture({collision = false, fail = false, readOnly = false} = {}) {
  const handlers = new Map();
  const tools = new Map();
  const warnings = [];
  const calls = [];
  let closes = 0;
  const operations = ['memory_recall', 'memory_remember'].map(name => ({name, description: name, input_schema: {type: 'object', additionalProperties: false}, read_only: name === 'memory_recall'}));
  const connection = {
    operations, readOnly,
    async call(name, input, signal) { calls.push({name, input, signal}); return {protocol_version: 1, ok: true, result: {value: 'ONE_BODY_CANARY'}}; },
    async context(prompt, signal) { calls.push({name: 'memory_context', prompt, signal}); return {context: 'Fresh memory evidence: ' + prompt}; },
    async close() { closes++; },
  };
  const pi = {
    on(name, handler) { assert.equal(handlers.has(name), false); handlers.set(name, handler); },
    getAllTools() { return [{name: 'read'}, {name: 'bash'}, ...(collision ? [{name: 'memory_recall'}] : []), ...tools.values()]; },
    registerTool(tool) { tools.set(tool.name, tool); },
    setActiveTools() { throw new Error('Must preserve native tool selection'); },
    sendMessage() { throw new Error('Must not accumulate persistent attachment messages'); },
  };
  const ctx = {hasUI: true, ui: {notify: message => warnings.push(message)}, signal: new AbortController().signal};
  attach(pi, async () => { if (fail) throw new Error('PRIVATE_CONNECTION_CANARY'); return connection; });
  return {handlers, tools, warnings, calls, connection, ctx, closes: () => closes};
}

test('native lifecycle uses typed sequential tools and appends fresh context without writes', async () => {
  const f = fixture();
  assert.deepEqual([...f.handlers.keys()].sort(), ['before_agent_start', 'session_shutdown', 'session_start']);
  await f.handlers.get('session_start')({reason: 'startup'}, f.ctx);
  assert.equal(f.tools.size, 2);
  for (const tool of f.tools.values()) assert.equal(tool.executionMode, 'sequential');
  for (const prompt of ['ordinary question', 'this is the way', 'Quoted: this is the way', 'Read-only, no saving']) {
    const result = await f.handlers.get('before_agent_start')({prompt, systemPrompt: 'Native and other extension instructions'}, f.ctx);
    assert.equal(result.systemPrompt, 'Native and other extension instructions\n\nFresh memory evidence: ' + prompt);
    assert.deepEqual(Object.keys(result), ['systemPrompt']);
  }
  assert.equal(f.calls.length, 4);
  assert.ok(f.calls.every(call => call.name === 'memory_context'));
  const result = await f.tools.get('memory_recall').execute('example', {}, f.ctx.signal, undefined, f.ctx);
  assert.equal(result.content.length, 1);
  assert.equal((JSON.stringify(result).match(/ONE_BODY_CANARY/g) ?? []).length, 1);
  assert.equal(f.calls.at(-1).signal, f.ctx.signal);
  await f.handlers.get('session_start')({reason: 'new'}, f.ctx);
  assert.equal(f.tools.size, 2);
  await f.handlers.get('session_shutdown')({reason: 'quit'}, f.ctx);
  assert.equal(f.closes(), 2);
  const stopped = await f.tools.get('memory_remember').execute('later', {}, undefined, undefined, f.ctx);
  assert.equal(JSON.parse(stopped.content[0].text).error.write_may_have_occurred, false);
});

test('unavailable or colliding attachment does not replace tools or expose raw errors', async () => {
  for (const options of [{collision: true}, {fail: true}]) {
    const f = fixture(options);
    await f.handlers.get('session_start')({reason: 'startup'}, f.ctx);
    assert.equal(f.tools.size, 0);
    assert.equal(f.warnings.length, 1);
    assert.doesNotMatch(f.warnings[0], /PRIVATE_CONNECTION_CANARY/);
    const result = await f.handlers.get('before_agent_start')({prompt: 'Question', systemPrompt: 'Native'}, f.ctx);
    assert.match(result.systemPrompt, /^Native/);
    assert.match(result.systemPrompt, /unavailable/);
    assert.equal(f.calls.length, 0);
  }
});

test('enforced read-only notice and complete transport errors retain their meaning', async () => {
  const f = fixture({readOnly: true});
  await f.handlers.get('session_start')({reason: 'startup'}, f.ctx);
  const context = await f.handlers.get('before_agent_start')({prompt: 'Remember this', systemPrompt: 'Native'}, f.ctx);
  assert.match(context.systemPrompt, /enforced read-only/);
  f.connection.call = async () => { throw new TransportError('operation.cancelled', 'Interrupted', true); };
  const out = await f.tools.get('memory_remember').execute('example', {}, undefined, undefined, f.ctx);
  assert.equal(JSON.parse(out.content[0].text).error.inspect_before_retry, true);
  assert.equal(out.details.mandalore.ok, false);
  f.connection.call = async () => { throw new Error('PRIVATE_TOOL_ERROR_CANARY'); };
  const unknown = await f.tools.get('memory_remember').execute('example', {}, undefined, undefined, f.ctx);
  assert.doesNotMatch(JSON.stringify(unknown), /PRIVATE_TOOL_ERROR_CANARY/);
  assert.equal(JSON.parse(unknown.content[0].text).error.inspect_before_retry, true);
});

test('session shutdown discards a context packet completed for the old attachment', async () => {
  const f = fixture();
  let resolve;
  f.connection.context = () => new Promise(done => { resolve = done; });
  await f.handlers.get('session_start')({reason: 'startup'}, f.ctx);
  const pending = f.handlers.get('before_agent_start')({prompt: 'Old turn', systemPrompt: 'Native'}, f.ctx);
  await f.handlers.get('session_shutdown')({reason: 'new'}, f.ctx);
  resolve({context: 'OLD_PACKET_CANARY'});
  assert.equal(await pending, undefined);
});

test('shutdown cancels in-flight startup and a late connection cannot register tools', async () => {
  const handlers = new Map();
  let publish;
  let entered;
  let closed = false;
  const ready = new Promise(resolve => { entered = resolve; });
  attach({
    on: (name, handler) => handlers.set(name, handler),
    getAllTools: () => [],
    registerTool: () => assert.fail('Late startup registered a tool'),
  }, signal => {
    entered(signal);
    return new Promise(resolve => { publish = resolve; });
  });
  const ctx = {hasUI: true, ui: {notify: () => assert.fail('Superseded startup notified UI')}};
  const starting = handlers.get('session_start')({reason: 'startup'}, ctx);
  const signal = await ready;
  await handlers.get('session_shutdown')({reason: 'quit'}, ctx);
  assert.equal(signal.aborted, true);
  publish({close: async () => { closed = true; }});
  await starting;
  assert.equal(closed, true);
});
