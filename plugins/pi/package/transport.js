import { spawn } from 'node:child_process';

export const MAX_INPUT_BYTES = 32768;
export const MAX_OUTPUT_BYTES = 65536;
export const MAX_CATALOG_BYTES = 1048576;

export class TransportError extends Error {
  constructor(code, message, mayWrite = false) {
    super(message);
    this.name = 'TransportError';
    this.envelope = {protocol_version: 1, ok: false, error: {
      code, message, retryable: false,
      write_may_have_occurred: mayWrite,
      inspect_before_retry: mayWrite,
    }};
  }
}

function decodeEnvelope(buffer, exitCode) {
  const value = JSON.parse(new TextDecoder('utf-8', {fatal: true}).decode(buffer));
  if (!value || typeof value !== 'object' || Array.isArray(value) || value.protocol_version !== 1 || typeof value.ok !== 'boolean') throw new Error('Invalid envelope');
  if (Object.keys(value).some(key => !['protocol_version', 'ok', 'result', 'error'].includes(key))) throw new Error('Invalid envelope fields');
  if (value.ok) {
    if (exitCode !== 0 || !Object.hasOwn(value, 'result') || Object.hasOwn(value, 'error')) throw new Error('Contradictory success');
  } else if (!Number.isInteger(exitCode) || exitCode === 0 || !value.error || typeof value.error !== 'object' || typeof value.error.code !== 'string' || typeof value.error.message !== 'string' || Object.hasOwn(value, 'result')) {
    throw new Error('Contradictory failure');
  }
  return value;
}

// One shell-free process, one complete envelope, no retry. Error output is never
// echoed. A complete cancellation receipt is more useful than a generic abort.
export async function runJSON(executable, args, input, {
  signal, mutating = false, timeoutMs = 45000, graceMs = 1000,
  maxOutputBytes = MAX_OUTPUT_BYTES,
} = {}) {
  if (signal?.aborted) throw new TransportError('operation.cancelled', 'Mandalore call cancelled before execution.');
  if (!Number.isInteger(timeoutMs) || timeoutMs < 1 || timeoutMs > 45000 || !Number.isInteger(graceMs) || graceMs < 1 || graceMs > 1000 || !Number.isInteger(maxOutputBytes) || maxOutputBytes < 1 || maxOutputBytes > MAX_CATALOG_BYTES) {
    throw new TransportError('input.invalid', 'Mandalore transport limits are invalid.');
  }
  let payload;
  try { payload = input === undefined ? '' : JSON.stringify(input); }
  catch { throw new TransportError('input.invalid', 'Mandalore input must be bounded JSON.'); }
  if (typeof payload !== 'string' || Buffer.byteLength(payload) > MAX_INPUT_BYTES) throw new TransportError('input.invalid', 'Mandalore input exceeds its JSON byte limit.');

  return new Promise((resolve, reject) => {
    let child;
    try { child = spawn(executable, args, {stdio: ['pipe', 'pipe', 'pipe'], detached: true, shell: false}); }
    catch { reject(new TransportError('operation.io', 'Mandalore executable could not be started.')); return; }
    let fault = null;
    let forced = false;
    let escalation;
    let outputSize = 0;
    let errorSize = 0;
    const chunks = [];
    const dispatched = Number.isInteger(child.pid);
    const killGroup = signalName => {
      if (!dispatched) return;
      try { process.kill(-child.pid, signalName); }
      catch (error) { if (error.code !== 'ESRCH') forced = true; }
    };
    const stop = (code, message, discard = false) => {
      if (fault) {
        if (discard) fault = {code, message, discard: true};
        return;
      }
      fault = {code, message, discard};
      killGroup('SIGTERM');
      escalation = setTimeout(() => { forced = true; killGroup('SIGKILL'); }, graceMs);
    };
    const abort = () => stop('operation.cancelled', 'Mandalore call was interrupted; inspect possible effects before retrying.');
    const deadline = setTimeout(() => stop('operation.cancelled', 'Mandalore call exceeded its deadline; inspect possible effects before retrying.'), timeoutMs);
    signal?.addEventListener('abort', abort, {once: true});
    if (signal?.aborted) abort();
    child.on('error', () => stop('operation.io', 'Mandalore executable could not be started.', true));
    child.stdin.on('error', () => {}); // A refusal may close stdin before reading it.
    child.stdout.on('data', chunk => {
      outputSize += chunk.length;
      if (outputSize > maxOutputBytes) stop('output.invalid', 'Mandalore output exceeded its byte limit.', true);
      else if (!fault?.discard) chunks.push(chunk);
    });
    child.stderr.on('data', chunk => {
      errorSize += chunk.length;
      if (errorSize > MAX_OUTPUT_BYTES) stop('output.invalid', 'Mandalore diagnostics exceeded their byte limit.', true);
    });
    child.on('close', exitCode => {
      clearTimeout(deadline);
      clearTimeout(escalation);
      signal?.removeEventListener('abort', abort);
      if (!fault?.discard && !forced && outputSize <= maxOutputBytes) {
        try { resolve(decodeEnvelope(Buffer.concat(chunks), exitCode)); return; }
        catch { /* Invalid or lost response is not evidence that nothing happened. */ }
      }
      reject(new TransportError(fault?.code ?? 'output.invalid', fault?.message ?? 'Mandalore did not return one complete valid envelope; inspect possible effects before retrying.', mutating && dispatched));
    });
    child.stdin.end(payload);
  });
}
