import { fileURLToPath } from 'node:url';
import { attach } from './extension.js';
import { openConnection } from './connection.js';

export default function mandalore(pi) {
  const root = fileURLToPath(new URL('.', import.meta.url));
  attach(pi, signal => openConnection(root, signal));
}
