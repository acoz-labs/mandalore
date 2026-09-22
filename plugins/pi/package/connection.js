import { createHash } from 'node:crypto';
import { constants, openSync, closeSync, fstatSync, readSync, lstatSync, readdirSync, realpathSync } from 'node:fs';
import { isAbsolute, join, relative } from 'node:path';
import { runJSON, TransportError, MAX_CATALOG_BYTES } from './transport.js';

const shaPattern = /^[a-f0-9]{64}$/;
const idPattern = /^[a-z][a-z0-9-]{2,127}$/;
const invalid = () => new TransportError('connection.invalid', 'Mandalore local connection or retained package is invalid; inspect it with the Armorer.');

function bytes(path, limit) {
  const fd = openSync(path, constants.O_RDONLY | constants.O_NOFOLLOW);
  try {
    const stat = fstatSync(fd);
    if (!stat.isFile() || stat.size > limit) throw invalid();
    const buffer = Buffer.alloc(limit + 1);
    let count = 0;
    while (count <= limit) {
      const size = readSync(fd, buffer, count, buffer.length - count, null);
      if (!size) break;
      count += size;
    }
    if (count > limit) throw invalid();
    return buffer.subarray(0, count);
  } finally { closeSync(fd); }
}

function runtimeStamp(path) {
  const stat = lstatSync(path, {bigint: true});
  if (!stat.isFile() || stat.size < 1n || stat.size > 134217728n || !(stat.mode & 0o111n)) throw invalid();
  return [stat.dev, stat.ino, stat.size, stat.mtimeNs, stat.ctimeNs].join(':');
}

function runtimeDigest(path) {
  const fd = openSync(path, constants.O_RDONLY | constants.O_NOFOLLOW);
  try {
    if (!fstatSync(fd).isFile()) throw invalid();
    const hash = createHash('sha256');
    const buffer = Buffer.alloc(65536);
    let total = 0;
    for (;;) {
      const count = readSync(fd, buffer, 0, buffer.length, null);
      if (!count) break;
      total += count;
      if (total > 134217728) throw invalid();
      hash.update(buffer.subarray(0, count));
    }
    return hash.digest('hex');
  } finally { closeSync(fd); }
}

// Matches Go's JSON encoding of the sorted filename-to-byte-slice map. The one
// generated local context and pinned session policy are outside the public payload hash.
export function packageDigest(root) {
  const content = new Map();
  let total = 0;
  let entries = 0;
  function walk(directory, depth) {
    if (depth > 8) throw invalid();
    for (const entry of readdirSync(directory, {withFileTypes: true})) {
      if (++entries > 256) throw invalid();
      const path = join(directory, entry.name);
      const name = relative(root, path);
      if (name === 'connection.json' || name === 'session-policy.json') continue;
      if (entry.isDirectory()) { walk(path, depth + 1); continue; }
      if (!entry.isFile() || content.size >= 128) throw invalid();
      const data = bytes(path, 1048576 - total);
      total += data.length;
      content.set(name, data.toString('base64'));
    }
  }
  walk(root, 0);
  const names = [...content.keys()].sort((a, b) => Buffer.compare(Buffer.from(a), Buffer.from(b)));
  const encoded = ('{' + names.map(name => JSON.stringify(name) + ':' + JSON.stringify(content.get(name))).join(',') + '}')
    .replace(/[<>&\u2028\u2029]/g, value => '\\u' + value.charCodeAt(0).toString(16).padStart(4, '0'));
  return createHash('sha256').update(encoded).digest('hex');
}

export function readConnection(root) {
  try {
    if (!['darwin', 'linux'].includes(process.platform)) throw invalid();
    const config = JSON.parse(new TextDecoder('utf-8', {fatal: true}).decode(bytes(join(root, 'connection.json'), 16384)));
    const paths = ['runtime', 'binding', 'native_home', 'native_binary', 'state_dir', 'connection_root'];
    const sessionEnabled = config?.session_policy !== undefined || config?.session_policy_sha256 !== undefined;
    const keys = ['schema_version', 'harness', ...paths, 'runtime_sha256', 'binding_sha256', 'signet_id', 'package_sha256', 'package_version', 'read_only', ...(sessionEnabled ? ['session_policy', 'session_policy_sha256'] : [])];
    if (!config || Array.isArray(config) || Object.keys(config).length !== keys.length || Object.keys(config).some(key => !keys.includes(key))) throw invalid();
    if (config.schema_version !== 1 || config.harness !== 'pi' || typeof config.read_only !== 'boolean' || typeof config.signet_id !== 'string' || !idPattern.test(config.signet_id) || typeof config.package_version !== 'string' || !config.package_version || config.package_version.length > 128) throw invalid();
    if (sessionEnabled && (config.read_only || typeof config.session_policy !== 'string' || config.session_policy !== join(config.connection_root, 'package', 'session-policy.json') || !shaPattern.test(config.session_policy_sha256))) throw invalid();
    if (sessionEnabled && createHash('sha256').update(bytes(config.session_policy, 16384)).digest('hex') !== config.session_policy_sha256) throw invalid();
    if (paths.some(key => typeof config[key] !== 'string' || !isAbsolute(config[key]) || /[\x00-\x1f\x7f]/.test(config[key]))) throw invalid();
    if (['runtime_sha256', 'binding_sha256', 'package_sha256'].some(key => typeof config[key] !== 'string' || !shaPattern.test(config[key]))) throw invalid();
    if (realpathSync(root) !== realpathSync(join(config.connection_root, 'package')) || packageDigest(root) !== config.package_sha256) throw invalid();
    const stamp = runtimeStamp(config.runtime);
    if (runtimeDigest(config.runtime) !== config.runtime_sha256 || runtimeStamp(config.runtime) !== stamp) throw invalid();
    return {config: Object.freeze(config), stamp};
  } catch { throw invalid(); }
}

function boundedPrompt(prompt) {
  const raw = Buffer.from(prompt, 'utf8');
  let end = Math.min(raw.length, 2048);
  for (;;) {
    try { return new TextDecoder('utf-8', {fatal: true}).decode(raw.subarray(0, end)); }
    catch { end--; }
  }
}

export async function openConnection(root, sessionSignal) {
  const {config, stamp} = readConnection(root);
  const pending = new Map();
  let closed = false;
  const request = (args, input, options = {}) => {
    if (closed) return Promise.reject(new TransportError('binding.invalid', 'Mandalore connection is closed.'));
    try { if (runtimeStamp(config.runtime) !== stamp) return Promise.reject(new TransportError('runtime.invalid', 'The retained Mandalore runtime changed; inspect the connection before continuing.')); }
    catch { return Promise.reject(invalid()); }
    const controller = new AbortController();
    const signals = [controller.signal, sessionSignal, options.signal].filter(Boolean);
    const result = runJSON(config.runtime, args, input, {...options, signal: AbortSignal.any(signals)});
    pending.set(controller, result);
    result.then(() => pending.delete(controller), () => pending.delete(controller));
    return result;
  };
  const close = async () => {
    closed = true;
    for (const controller of pending.keys()) controller.abort();
    await Promise.allSettled([...pending.values()]);
  };
  const sessionEnabled = !!config.session_policy;
  const guards = ['--binding', config.binding, '--binding-sha256', config.binding_sha256, '--signet-id', config.signet_id, '--harness', 'pi', ...(sessionEnabled ? ['--session-policy', config.session_policy, '--session-policy-sha256', config.session_policy_sha256] : [])];
  const bound = (name, input, options = {}) => request(['call', name, ...guards, ...(config.read_only || options.readOnly && !sessionEnabled ? ['--read-only'] : [])], input, options);
  const requireOK = response => { if (!response.ok) throw invalid(); return response.result; };
  try {
    const version = requireOK(await request(['version'], undefined, {timeoutMs: 5000}));
    if (version.name !== 'mandalore' || version.protocol_version !== 1) throw invalid();
    const identity = requireOK(await request(['call', 'pi_package_inspect'], {}, {timeoutMs: 5000}));
    if (identity.name !== 'mandalore' || identity.harness_protocol_version !== 1 || identity.sha256 !== config.package_sha256 || identity.version !== config.package_version) throw invalid();
    const catalog = requireOK(sessionEnabled ? await bound('memory_session_catalog', {}, {timeoutMs: 5000, maxOutputBytes: MAX_CATALOG_BYTES}) : await request(['operations'], undefined, {timeoutMs: 5000, maxOutputBytes: MAX_CATALOG_BYTES}));
    if (!Array.isArray(catalog.operations) || !sessionEnabled && (catalog.max_input_bytes !== 32768 || catalog.max_output_bytes !== 65536)) throw invalid();
    const operations = catalog.operations.filter(op => op.requires_binding === true && op.cli_only === false);
    if (operations.length < 1 || new Set(operations.map(op => op.name)).size !== operations.length || operations.some(op => !/^[a-z][a-z0-9_]{0,63}$/.test(op.name) || typeof op.description !== 'string' || typeof op.read_only !== 'boolean' || !op.input_schema || typeof op.input_schema !== 'object')) throw invalid();
    // Opening the guarded service validates the connection. Orientation-only
    // avoids a redundant whole-bank validation before the first actual turn.
    if (!sessionEnabled) requireOK(await bound('memory_context', {}, {timeoutMs: 5000, readOnly: true}));
    return {
      operations, readOnly: config.read_only, sessionEnabled, close,
      async start(boundary, signal) {
        if (!sessionEnabled) return;
        return requireOK(await bound('memory_context', {boundary}, {signal, timeoutMs: 5000, mutating: true}));
      },
      async call(name, input, signal) {
        const operation = operations.find(op => op.name === name);
        if (!operation) throw new TransportError('operation.unknown', 'Operation is not a native memory tool.');
        return bound(name, input, {signal, mutating: !operation.read_only});
      },
      async context(prompt, signal, boundary) {
        const packet = requireOK(await bound('memory_context', {prompt: boundedPrompt(prompt), ...(sessionEnabled ? {boundary} : {})}, {signal, timeoutMs: 5000, readOnly: true, mutating: sessionEnabled}));
        if (!packet || typeof packet !== 'object' || Array.isArray(packet) || Object.keys(packet).some(key => !['context', 'warning', 'synchronization'].includes(key)) || (packet.context !== undefined && typeof packet.context !== 'string') || (packet.warning !== undefined && typeof packet.warning !== 'string') || (!packet.context && !packet.warning) || Buffer.byteLength(JSON.stringify(packet)) > 16383) throw invalid();
        return packet;
      },
    };
  } catch (error) { await close(); throw error; }
}
