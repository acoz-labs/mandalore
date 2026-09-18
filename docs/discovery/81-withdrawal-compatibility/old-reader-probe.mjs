// Characterizes a released reader; does not implement or migrate a signet.
// Usage: node old-reader-probe.mjs /absolute/path/to/released/mandalore
import assert from 'node:assert/strict';
import {spawnSync} from 'node:child_process';
import {createHash} from 'node:crypto';
import {mkdtempSync,realpathSync,readFileSync,writeFileSync,readdirSync,lstatSync} from 'node:fs';
import {join,isAbsolute} from 'node:path';
import {tmpdir} from 'node:os';

const binary=process.argv[2]; assert.ok(binary&&isAbsolute(binary));
const root=realpathSync(mkdtempSync(join(tmpdir(),'mandalore-81-old-reader-')));
const bank=join(root,'bank'),binding=join(root,'config','binding.json');
const sha=b=>createHash('sha256').update(b).digest('hex');
function cli(args,input={},ok=true){
  const p=spawnSync(binary,args,{input:JSON.stringify(input),encoding:'utf8',timeout:30000,maxBuffer:1048576});
  assert.equal(p.error,undefined);const v=JSON.parse(p.stdout);assert.equal(v.ok,ok);assert.equal(p.status===0,ok);return v;
}
function git(cwd,...args){
  const p=spawnSync('git',['-c','commit.gpgsign=false','-c','user.name=Synthetic','-c','user.email=synthetic@example.invalid','-c','gc.auto=0','-c','maintenance.auto=false',...args],{cwd,encoding:'utf8',timeout:30000,env:{...process.env,GIT_CONFIG_GLOBAL:'/dev/null',GIT_CONFIG_NOSYSTEM:'1',GIT_TERMINAL_PROMPT:'0',GIT_OPTIONAL_LOCKS:'0'}});
  assert.equal(p.status,0,p.stderr);return p.stdout.trim();
}
function inventory(path){
  const out=[];function walk(dir,prefix=''){for(const name of readdirSync(dir).sort()){if(name==='.git'||name==='.mandalore')continue;const file=join(dir,name),relative=join(prefix,name),st=lstatSync(file);assert.ok(!st.isSymbolicLink());out.push([relative,st.mode,st.mtimeMs,st.isFile()?sha(readFileSync(file)):null]);if(st.isDirectory())walk(file,relative);}}walk(path);return out;
}
const version=cli(['version']).result;
assert.equal(version.version,'1.1.0','probe must target the known released reader');
cli(['call','signet_create'],{repository:bank,name:'Synthetic old-reader bank',device_label:'Synthetic'});
cli(['call','signet_bind'],{repository:bank,binding,device_label:'Synthetic',actor:'Synthetic'});
const saved=cli(['memory','remember','--binding',binding],{kind:'fact',summary:'Compatibility canary',body:'WITHDRAWAL_COMPATIBILITY_CANARY',basis:'observation',reason:'Synthetic probe'}).result;
const recalled=cli(['memory','recall','--binding',binding]);
assert.ok(JSON.stringify(recalled).includes('WITHDRAWAL_COMPATIBILITY_CANARY'));
const manifest=join(bank,'signet.json'),original=readFileSync(manifest);
const changed=JSON.parse(original);changed.schema_version=2;
writeFileSync(manifest,JSON.stringify(changed,null,2)+'\n',{mode:0o600});
const before=inventory(bank),refused=[];
for(const args of [['memory','recall'],['memory','scopes'],['memory','history','--record-id',saved.record_id],['memory','journal'],['memory','inspect'],['memory','sync']]){
  const v=cli([...args,'--binding',binding],{},false);
  assert.equal(v.error.code,'binding.invalid');assert.ok(!JSON.stringify(v).includes('WITHDRAWAL_COMPATIBILITY_CANARY'));refused.push(args[1]);
}
assert.deepEqual(inventory(bank),before);
// Restore only the disposable fixture to establish an old clone/remote pair.
writeFileSync(manifest,original,{mode:0o600});
cli(['memory','git-init','--binding',binding]);
const remote=join(root,'remote.git'),other=join(root,'remote-writer');
git(root,'init','--bare',remote);git(bank,'remote','add','origin',remote);git(bank,'push','-u','origin','main');
git(root,'clone', '--branch','main',remote,other);
writeFileSync(join(other,'signet.json'),JSON.stringify(changed,null,2)+'\n',{mode:0o600});
git(other,'add','signet.json');git(other,'commit','-m','Synthetic unsupported format transition');git(other,'push','origin','main');
const oldHead=git(bank,'rev-parse','HEAD'),oldTree=inventory(bank);
const sync=cli(['memory','sync','--binding',binding,'--timeout-seconds','10']);
assert.equal(sync.result.state,'conflicted');assert.equal(sync.result.phase,'validate');assert.equal(sync.result.delivered,false);
assert.equal(git(bank,'rev-parse','HEAD'),oldHead);assert.deepEqual(inventory(bank),oldTree);
assert.ok(!JSON.stringify(sync).includes('WITHDRAWAL_COMPATIBILITY_CANARY'));
assert.ok(JSON.stringify(cli(['memory','recall','--binding',binding])).includes('WITHDRAWAL_COMPATIBILITY_CANARY'));
const result={kind:'withdrawal-old-reader-characterization',version,binary_sha256:sha(readFileSync(binary)),direct_format_refusals:refused,refused_sync_state:sync.result.state,refused_sync_phase:sync.result.phase,portable_source_unchanged:true,stale_clone_still_recalls_old_content:true,limitations:['Format-2 fixture is not an implemented migration','Old local evidence persists after refused sync','No personal bank, remote account or network host used']};
writeFileSync(join(root,'evidence.json'),JSON.stringify(result,null,2)+'\n',{mode:0o600});
console.log(JSON.stringify(result,null,2));console.log('OLD_READER_PROBE_PASSED');console.log('Retained fixture: '+root);
