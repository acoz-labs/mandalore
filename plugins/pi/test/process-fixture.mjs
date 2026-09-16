import { writeFileSync } from 'node:fs';
import { spawn } from 'node:child_process';

const mode = process.argv[2];
const ok = result => ({protocol_version: 1, ok: true, result});
const emit = value => process.stdout.write(JSON.stringify(value) + '\n');
if (mode === 'arguments') {
  let input = '';
  for await (const chunk of process.stdin) input += chunk;
  emit(ok({args: process.argv.slice(3), input: JSON.parse(input)}));
} else if (mode === 'partial') {
  emit({protocol_version: 1, ok: false, error: {code: 'operation.cancelled', message: 'Stopped', write_may_have_occurred: true, inspect_before_retry: true, sync_status: {phase: 'push', checkpointed: true}}});
  process.exitCode = 130;
} else if (mode === 'double') {
  emit(ok({})); emit(ok({}));
} else if (mode === 'protocol') {
  emit({protocol_version: 99, ok: true, result: {}});
} else if (mode === 'exit') {
  emit(ok({})); process.exitCode = 1;
} else if (mode === 'false-success') {
  emit({protocol_version: 1, ok: false, error: {code: 'input.invalid', message: 'Refused'}});
} else if (mode === 'catalog') {
  emit(ok({value: 'x'.repeat(70000)}));
} else if (mode === 'utf8') {
  process.stdout.write(Buffer.from([255]));
} else if (mode === 'output-limit') {
  process.stdout.write('PRIVATE_OUTPUT_CANARY'.repeat(10000));
} else if (mode === 'stderr-limit') {
  process.stderr.write('PRIVATE_STDERR_CANARY'.repeat(10000));
} else if (mode === 'group') {
  const child = spawn(process.execPath, ['-e', 'setInterval(() => {}, 1000)'], {stdio: 'ignore'});
  process.on('SIGTERM', () => {}); // Group cancellation must reach the child too.
  child.on('close', () => process.exit(0));
  writeFileSync(process.argv[3], String(child.pid) + '\n');
  setInterval(() => {}, 1000);
} else if (mode === 'ignore' || mode === 'cancel-receipt') {
  if (mode === 'ignore') process.on('SIGTERM', () => {});
  else process.on('SIGTERM', () => {
    emit(ok({saved: {id: 'event-example', durable_locally: true}, delivery: {ok: false, error: {code: 'operation.cancelled'}}}));
    process.exit(0);
  });
  writeFileSync(process.argv[3], String(process.pid) + '\n');
  setInterval(() => {}, 1000);
} else {
  emit(ok({value: 'example'}));
}
