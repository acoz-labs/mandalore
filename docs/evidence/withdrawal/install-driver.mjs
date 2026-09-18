// Isolated installation compatibility probe. Downloads published 1.1.0 only.
// node install-driver.mjs /absolute/released-old-cli /absolute/new-candidate
import assert from 'node:assert/strict';
import {spawnSync} from 'node:child_process';
import {createHash} from 'node:crypto';
import {mkdtempSync, mkdirSync, readFileSync, writeFileSync, readdirSync, lstatSync, realpathSync} from 'node:fs';
import {tmpdir} from 'node:os';
import {join, isAbsolute} from 'node:path';
import {fileURLToPath} from 'node:url';

const [legacy, candidate] = process.argv.slice(2);
assert.ok(legacy && candidate && isAbsolute(legacy) && isAbsolute(candidate));
const root = realpathSync(mkdtempSync(join(tmpdir(), 'mandalore-install-compat-')));
console.log('Retained synthetic installation: ' + root);
const prefix = join(root, 'prefix'), bank = join(root, 'bank'), binding = join(root, 'binding.json');
const hash = bytes => createHash('sha256').update(bytes).digest('hex');
const manifest = JSON.parse(readFileSync(join(candidate, 'manifest.json')));
const goarch = process.arch === 'x64' ? 'amd64' : process.arch;
const asset = manifest.assets.find(a => a.kind === 'cli' && a.os === process.platform && a.arch === goarch);
assert.ok(asset, 'native candidate asset required');
const binary = join(candidate, asset.name);
assert.equal(hash(readFileSync(binary)), asset.sha256);
assert.deepEqual(manifest.signet_read_versions, [1, 2]);
function invoke(executable, name, input = {}, expected = true) {
  const p = spawnSync(executable, ['call', name], {input: JSON.stringify(input), encoding:'utf8', timeout:120000, maxBuffer:1048576});
  assert.equal(p.error, undefined, 'CLI process failed; inspect retained fixture before retry');
  const result = JSON.parse(p.stdout);
  assert.equal(result.protocol_version, 1);
  assert.equal(result.ok, expected, JSON.stringify({name, error:result.error}));
  assert.equal(p.status === 0, expected);
  return result;
}
function inventory(directory) {
  const result = {};
  function walk(dir, prefix = '') {
    for (const name of readdirSync(dir).sort()) {
      const path = join(dir, name), rel = join(prefix, name), st = lstatSync(path);
      assert.ok(!st.isSymbolicLink());
      if (st.isDirectory()) walk(path, rel);
      else result[rel] = {sha256:hash(readFileSync(path)), mode:st.mode & 0o777};
    }
  }
  walk(directory);
  return result;
}
mkdirSync(prefix, {mode:0o700});
mkdirSync(join(prefix, 'bin'), {mode:0o700});
writeFileSync(join(prefix, 'bin', 'unrelated'), 'UNRELATED_TOOL', {mode:0o600});
invoke(binary, 'signet_create', {repository:bank, name:'Synthetic format1 bank', device_label:'Synthetic device'});
invoke(binary, 'signet_bind', {repository:bank, binding, device_label:'Synthetic device', actor:'Synthetic tester'});
const before = inventory(bank), bindingHash = hash(readFileSync(binding));
console.log('Installing public 1.1.0 into disposable prefix');
const oldPlan = invoke(legacy, 'release_plan', {prefix, version:'1.1.0'}).result;
assert.equal(oldPlan.source.manifest.manifest.source_commit, '51aee17afec015ba2ad44584f8190b4bb6d901a8');
const oldInstall = invoke(legacy, 'release_apply', oldPlan).result;
assert.equal(oldInstall.installed, true);
const launcher = join(prefix, 'bin', 'mandalore');
const oldTarget = realpathSync(launcher), oldHash = hash(readFileSync(oldTarget));
const oldReceiptHash = hash(readFileSync(join(prefix, 'lib', 'mandalore', 'receipt.json')));
console.log('Checking old updater refusal of format1/2 candidate');
const refusal = invoke(launcher, 'release_plan', {prefix, candidate}, false);
assert.equal(realpathSync(launcher), oldTarget);
assert.equal(hash(readFileSync(join(prefix, 'lib', 'mandalore', 'receipt.json'))), oldReceiptHash);
console.log('Applying exact candidate through its verified new executable');
const newPlan = invoke(binary, 'release_plan', {prefix, candidate}).result;
const installed = invoke(binary, 'release_apply', newPlan).result;
assert.equal(installed.installed, true);
assert.equal(realpathSync(launcher), installed.runtime);
assert.equal(hash(readFileSync(installed.runtime)), asset.sha256);
assert.equal(hash(readFileSync(oldTarget)), oldHash);
assert.equal(readFileSync(join(prefix, 'bin', 'unrelated'), 'utf8'), 'UNRELATED_TOOL');
assert.deepEqual(inventory(bank), before);
assert.equal(hash(readFileSync(binding)), bindingHash);
assert.equal(JSON.parse(readFileSync(join(bank, 'signet.json'))).schema_version, 1);
const evidence = {kind:'mandalore-runtime-format-compatibility', source_commit:manifest.source_commit, manifest_sha256:hash(readFileSync(join(candidate,'manifest.json'))), binary_sha256:asset.sha256, legacy_source_commit:oldPlan.source.manifest.manifest.source_commit, legacy_binary_sha256:oldHash, driver_sha256:hash(readFileSync(fileURLToPath(import.meta.url))), platform:process.platform, arch:process.arch, old_refusal_code:refusal.error.code, checks:['published old runtime installed in disposable prefix','old updater refuses new format declaration without activation','verified new runtime plans/applies candidate preserving old runtime and unrelated files','runtime update does not migrate format1 bank or change binding'], limitations:['engineering evidence, not candidate acceptance or public release','verified-new-executable route; not download of an unpublished release through public bootstrap','no personal CLI, plugin, signet or credentials changed']};
writeFileSync(join(root,'evidence.json'), JSON.stringify(evidence,null,2)+'\n', {mode:0o600});
console.log(JSON.stringify(evidence,null,2));
console.log('RUNTIME_FORMAT_COMPATIBILITY_PASSED');
