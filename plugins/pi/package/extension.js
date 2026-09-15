import { TransportError } from './transport.js';

const unavailable = 'Mandalore memory is unavailable. Ask the Armorer to inspect the selected Pi connection. No automatic repair or synchronization occurred.';
const readOnlyNotice = 'This Mandalore connection is enforced read-only. Do not save, journal or synchronize.';

function toolResult(name, envelope) {
  return {
    content: [{type: 'text', text: JSON.stringify(envelope)}],
    details: {mandalore: {operation: name, ok: envelope.ok}},
  };
}

// Pi owns native tools, model access, sessions and user authorization. This
// controller only registers memory tools and adds fresh, read-only context.
export function attach(pi, openConnection) {
  let connection = null;
  let sessionAbort = null;
  let catalogSignature = null;
  let lastWarning = null;
  const registered = new Set();
  const warn = (ctx, message) => {
    if (message === lastWarning) return;
    lastWarning = message;
    if (ctx.hasUI) ctx.ui.notify(message, 'warning');
    else console.error(message);
  };
  const disconnect = async () => {
    const previous = connection;
    connection = null;
    await previous?.close();
  };

  pi.on('session_start', async (_event, ctx) => {
    sessionAbort?.abort();
    const starting = new AbortController();
    sessionAbort = starting;
    let candidate;
    try {
      await disconnect();
      candidate = await openConnection(starting.signal);
      if (starting.signal.aborted || sessionAbort !== starting) { await candidate.close(); return; }
      const signature = JSON.stringify(candidate.operations);
      if (catalogSignature !== null && catalogSignature !== signature) throw new Error('Changed catalog requires reload');
      const occupied = new Set(pi.getAllTools().map(tool => tool.name));
      if (candidate.operations.some(op => occupied.has(op.name) && !registered.has(op.name))) throw new Error('Tool collision');
      if (catalogSignature === null) {
        for (const op of candidate.operations) {
          pi.registerTool({
            name: op.name,
            label: 'Mandalore: ' + op.name.replaceAll('_', ' '),
            description: op.description,
            parameters: op.input_schema,
            executionMode: 'sequential',
            async execute(_id, input, signal) {
              const selected = connection;
              if (!selected) return toolResult(op.name, new TransportError('binding.invalid', unavailable).envelope);
              try { return toolResult(op.name, await selected.call(op.name, input, signal)); }
              catch (error) {
                const failure = error instanceof TransportError ? error : new TransportError('operation.io', 'Mandalore tool transport failed; inspect possible effects before retrying.', !op.read_only);
                return toolResult(op.name, failure.envelope);
              }
            },
          });
          registered.add(op.name);
        }
        catalogSignature = signature;
      }
      connection = candidate;
      lastWarning = null;
    } catch {
      await candidate?.close().catch(() => {});
      if (sessionAbort === starting) warn(ctx, unavailable);
    }
  });

  pi.on('before_agent_start', async (event, ctx) => {
    if (typeof event.systemPrompt !== 'string') { warn(ctx, unavailable); return; }
    const selected = connection;
    const turnSession = sessionAbort;
    if (!selected) return {systemPrompt: event.systemPrompt + '\n\n' + unavailable};
    try {
      if (typeof event.prompt !== 'string') throw new Error('Missing native prompt');
      const packet = await selected.context(event.prompt, ctx.signal);
      if (connection !== selected || sessionAbort !== turnSession) return; // A shutdown/reload superseded this turn.
      if (packet.warning) warn(ctx, packet.warning);
      const parts = [event.systemPrompt];
      if (selected.readOnly) parts.push(readOnlyNotice);
      if (packet.context) parts.push(packet.context);
      if (packet.warning) parts.push(packet.warning);
      return {systemPrompt: parts.join('\n\n')};
    } catch {
      if (connection !== selected || sessionAbort !== turnSession) return;
      warn(ctx, unavailable);
      return {systemPrompt: event.systemPrompt + '\n\n' + unavailable};
    }
  });

  pi.on('session_shutdown', async (_event, ctx) => {
    sessionAbort?.abort();
    sessionAbort = null;
    try { await disconnect(); }
    catch { warn(ctx, 'Mandalore process cleanup needs inspection. No automatic retry was attempted.'); }
  });
}
