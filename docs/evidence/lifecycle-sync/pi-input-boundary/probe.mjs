import {spawn} from 'node:child_process';
import {mkdirSync,readFileSync,writeFileSync} from 'node:fs';
import {createInterface} from 'node:readline';
import assert from 'node:assert/strict';
const lab=process.argv[2], binary=process.argv[3];
mkdirSync(lab+'/profile',{mode:0o700});
mkdirSync(lab+'/project',{mode:0o700});
const env={...process.env,PI_CODING_AGENT_DIR:lab+'/profile',PI_OFFLINE:'1',PROBE_EVENTS:lab+'/events.jsonl'};
delete env.MANDALORE_BINDING; delete env.MANDALORE_BIN;
const child=spawn(binary,['--offline','--mode','rpc','--no-session','--no-context-files','--no-extensions','--no-skills','--no-prompt-templates','--no-themes','-e',lab+'/observer.ts'],{cwd:lab+'/project',env,stdio:['pipe','pipe','pipe']});
let seq=0,stderrBytes=0;
const pending=new Map();
child.stderr.on('data',v=>stderrBytes+=v.length);
createInterface({input:child.stdout}).on('line',line=>{
  const value=JSON.parse(line);
  if(value.type==='response')pending.get(value.id)?.(value);
});
const exit=new Promise(resolve=>child.on('exit',(code,signal)=>resolve({code,signal})));
function deadline(p,ms){let timer;return Promise.race([p,new Promise((_,reject)=>{timer=setTimeout(()=>reject(Error('timeout')),ms);})]).finally(()=>clearTimeout(timer));}
async function rpc(type,data={}){
  const id=String(++seq),response=new Promise(resolve=>pending.set(id,resolve));
  child.stdin.write(JSON.stringify({id,type,...data})+'\n');
  const value=await deadline(response,15000);pending.delete(id);
  assert.equal(value.success,true);return value.data;
}
function events(){return readFileSync(lab+'/events.jsonl','utf8').trim().split('\n').map(JSON.parse);}
try {
  await rpc('get_state');
  await rpc('prompt',{message:'Synthetic ordinary prompt intercepted before model execution.'});
  assert.equal(events().filter(e=>e.event==='input').length,1);
  await rpc('steer',{message:'Synthetic queued restriction: no synchronization.'});
  const afterSteer=await rpc('get_state');
  await rpc('follow_up',{message:'Synthetic follow-up restriction: no writes.'});
  const afterFollowUp=await rpc('get_state');
  assert.equal(events().filter(e=>e.event==='input').length,1);
  assert.equal(afterSteer.pendingMessageCount,1);
  assert.equal(afterFollowUp.pendingMessageCount,2);
  const cleared=await rpc('clear_queue');
  const afterClear=await rpc('get_state');
  assert.equal(afterClear.pendingMessageCount,0);
  child.stdin.end();
  const ended=await deadline(exit,10000);
  assert.equal(ended.code,0);
  const receipt={native_version:'0.85.1',classification:'discovery-counterexample-not-product-acceptance',events:events(),ordinary_prompt_input_count:1,direct_steer_additional_input_events:0,direct_follow_up_additional_input_events:0,pending_after_steer:afterSteer.pendingMessageCount,pending_after_follow_up:afterFollowUp.pendingMessageCount,pending_after_clear:afterClear.pendingMessageCount,cleared_steering:cleared.steering.length,cleared_follow_up:cleared.followUp.length,stderr_bytes:stderrBytes,exit:ended,limitations:['Idle native RPC queue acceptance, not model-active steering order or interactive UI evidence.','No Mandalore extension, signet or provider prompt; ordinary prompt was handled by observer.','No conclusion that all lifecycle authorization approaches are impossible.']};
  writeFileSync(lab+'/receipt.json',JSON.stringify(receipt,null,2)+'\n',{mode:0o600});
  console.log(JSON.stringify(receipt));
} finally {
  if(child.exitCode===null&&!child.killed)child.kill('SIGTERM');
}
