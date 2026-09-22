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

test('repeated native session-start notifications replace connections without duplicate tools or writes', async () => {
  // Observed in Pi 0.85.1 RPC new/resume/fork, including a control with no
  // Mandalore extension: the host rebinds the replacement session twice.
  const f = fixture();
  const reasons = ['startup', 'reload', 'new', 'new', 'resume', 'resume', 'fork', 'fork'];
  for (const reason of reasons) {
    await f.handlers.get('session_start')({reason}, f.ctx);
    assert.equal(f.tools.size, 2);
    assert.equal(f.warnings.length, 0);
    assert.equal(f.calls.length, 0);
  }
  assert.equal(f.closes(), reasons.length - 1);
  const result = await f.tools.get('memory_recall').execute('after-fork', {}, f.ctx.signal);
  assert.equal(JSON.parse(result.content[0].text).ok, true);
  assert.deepEqual(f.calls.map(call => call.name), ['memory_recall']);
  await f.handlers.get('session_shutdown')({reason: 'quit'}, f.ctx);
  assert.equal(f.closes(), reasons.length);
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

test('enabled session awaits startup and refreshes each distinct top-level turn', async () => {
  const f = fixture();
  const boundaries = [];
  f.ctx.sessionManager = {getSessionId: () => 'native-session'};
  let release;
  const gate = new Promise(resolve => { release = resolve; });
  f.connection.start = async boundary => { boundaries.push(boundary); await gate; return {}; };
  f.connection.context = async (_prompt, _signal, boundary) => { boundaries.push(boundary); return {context: 'refreshed'}; };
  let ready = false;
  const starting = f.handlers.get('session_start')({}, f.ctx).then(() => {ready = true;});
  await new Promise(resolve => setImmediate(resolve));
  assert.equal(ready, false);
  release(); await starting;
  for (let i=0;i<2;i++) await f.handlers.get('before_agent_start')({prompt:'identical text',systemPrompt:'native'},f.ctx);
  assert.deepEqual(boundaries[0],{kind:'startup',session_id:'native-session',event_key:''});
  assert.equal(boundaries[1].session_id,'native-session');
  assert.equal(boundaries[2].session_id,'native-session');
  assert.match(boundaries[1].event_key,/:1$/);
  assert.match(boundaries[2].event_key,/:2$/);
  assert.notEqual(boundaries[1].event_key,boundaries[2].event_key);
});


test('two resumed Pi instances never reuse the same turn key', async () => {
  const keys=[];
  for (let i=0;i<2;i++) {
    const f=fixture();
    f.ctx.sessionManager={getSessionId:()=> 'same-persisted-session'};
    f.connection.context=async (_p,_s,boundary)=>{ keys.push(boundary.event_key);return {context:'fresh'}; };
    await f.handlers.get('session_start')({},f.ctx);
    await f.handlers.get('before_agent_start')({prompt:'identical',systemPrompt:'native'},f.ctx);
  }
  assert.notEqual(keys[0],keys[1]);
});
