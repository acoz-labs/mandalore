import test from 'node:test';
import assert from 'node:assert/strict';
import { readFile, stat } from 'node:fs/promises';

const pkg = new URL('../package/', import.meta.url);
const codex = new URL('../../codex/plugins/mandalore/skills/', import.meta.url);

test('Pi discovers two packaged skills and every local reference resolves', async () => {
  const manifest = JSON.parse(await readFile(new URL('package.json', pkg), 'utf8'));
  assert.deepEqual(manifest.pi.skills, ['./skills']);
  for (const name of ['this-is-the-way', 'the-armorer']) {
    const url = new URL(`skills/${name}/SKILL.md`, pkg);
    const text = await readFile(url, 'utf8');
    assert.ok(text.startsWith(`---\nname: ${name}\n`));
    for (const [, target] of text.matchAll(/\]\(([^)]+)\)/g)) {
      if (!target.includes('://')) assert.ok((await stat(new URL(target, url))).isFile());
    }
  }
});

test('neutral memory references stay byte-identical across harnesses', async () => {
  for (const reference of ['foundlings.md', 'delivery.md', 'visibility.md']) {
    const relative = `this-is-the-way/references/${reference}`;
    assert.deepEqual(await readFile(new URL(`skills/${relative}`, pkg)), await readFile(new URL(relative, codex)));
  }
});
