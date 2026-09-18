import assert from 'node:assert/strict';
import test from 'node:test';
import {inspect,appendDecision,content,decision} from './model.mjs';

const base=()=>[content('c0')];
const withdrawn=()=>[...base(),decision('w0','withdraw',[],['c0'])];
const restored=()=>[...withdrawn(),decision('r0','restore',['w0'],['c0'])];
const state=nodes=>inspect(nodes).state;

test('withdrawal and ordinary correction are distinct, including concurrent correction',()=>{
  assert.equal(state(base()),'visible');
  assert.equal(state(withdrawn()),'withdrawn');
  for(const refs of [[],['w0']]) assert.equal(state([...withdrawn(),content('c1',['c0'],refs)]),'withdrawn');
});
test('restore covers reviewed content and later aware corrections, not a raced correction',()=>{
  assert.equal(state(restored()),'visible');
  assert.equal(state([...restored(),content('c1',['c0'],['r0'])]),'visible');
  const raced=[...restored(),content('c1',['c0'],['w0'])];
  assert.equal(state(raced),'unreviewed-content');
  // A boolean-only last restore rule would expose c1 here. A new explicit
  // decision observing c1 is needed to make it eligible again.
  assert.equal(state(appendDecision(raced,decision('r1','restore',['r0'],['c1']),['c1'],['r0'])),'visible');
});
test('concurrent withdraw/restore and two independent restores remain withheld',()=>{
  assert.equal(state([...restored(),decision('w1','withdraw',['w0'],['c0'])]),'visibility-conflict');
  const fork=[...restored(),decision('r1','restore',['w0'],['c0'])];
  assert.equal(state(fork),'visibility-conflict');
  assert.equal(state(appendDecision(fork,decision('r2','restore',['r0','r1'],['c0']),['c0'],['r0','r1'])),'visible');
});
test('content conflicts remain conflicts after explicit restore',()=>{
  const nodes=[...withdrawn(),content('c1',['c0'],['w0']),content('c2',['c0'],['w0'])];
  assert.equal(state(appendDecision(nodes,decision('r0','restore',['w0'],['c1','c2']),['c1','c2'],['w0'])),'content-conflict');
});
test('stale requests and blind replay fail instead of implicit retries',()=>{
  const event=decision('w0','withdraw',[],['c0']);
  const next=appendDecision(base(),event,['c0'],[]);
  assert.throws(()=>appendDecision(next,event,['c0'],[]),/stale visibility/);
  assert.throws(()=>appendDecision([...base(),content('c1',['c0'])],event,['c0'],[]),/stale content/);
});
test('incomplete, duplicate, wrong-kind and cross-record graphs are rejected',()=>{
  for(const nodes of [
    [content('c0',['missing'])],
    [content('c0'),content('c0')],
    [content('c0',[],['c0'])],
    [...base(),{...decision('w0','withdraw',[],['c0']),record:'another-record'}],
    [...base(),decision('w0','withdraw',[],[])],
  ]) assert.throws(()=>inspect(nodes));
});
test('combined causal cycles are invalid even if each separate parent graph is acyclic',()=>{
  assert.throws(()=>inspect([content('c0',[],['r0']),decision('r0','restore',[],['c0'])]),/causal cycle/);
});
function* permutations(values){if(values.length===0){yield [];return;}for(let i=0;i<values.length;i++)for(const rest of permutations(values.filter((_,j)=>j!==i)))yield[values[i],...rest];}
test('all 120 delivery orders converge; missing dependency prefixes never become guidance',()=>{
  const nodes=[...restored(),content('c1',['c0'],['w0']),decision('w1','withdraw',['w0'],['c0'])];
  let count=0;
  for(const ordered of permutations(nodes)){
    assert.equal(state(ordered),'visibility-conflict');count++;
    for(let size=1;size<ordered.length;size++){
      const prefix=ordered.slice(0,size),ids=new Set(prefix.map(n=>n.id));
      const missing=prefix.some(n=>[...n.parents,...(n.kind==='content'?n.visibility:n.observed)].some(id=>!ids.has(id)));
      if(missing)assert.throws(()=>inspect(prefix));
    }
  }
  assert.equal(count,120);
});
test('changing wall-clock metadata never changes visibility',()=>{
  const nodes=[...restored(),decision('w1','withdraw',['w0'],['c0'])];
  for(const ordered of [nodes,[...nodes].reverse()])assert.equal(state(ordered.map((n,i)=>({...n,time:i%2?'2099-01-01':'1900-01-01'}))),'visibility-conflict');
});
