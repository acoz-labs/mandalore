// Executable design model only. Not a supported memory writer or runtime.
import assert from 'node:assert/strict';

const sorted = values => [...values].sort();
export function inspect(nodes) {
  const byID = new Map();
  for (const n of nodes) {
    assert.ok(n.id && n.record === 'record-test');
    assert.ok(['content','withdraw','restore'].includes(n.kind));
    assert.ok(!byID.has(n.id), 'duplicate ID');
    byID.set(n.id, n);
  }
  const edges = new Map();
  for (const n of nodes) {
    const refs = n.kind === 'content' ? n.visibility : n.observed;
    assert.ok(Array.isArray(n.parents) && Array.isArray(refs));
    assert.equal(new Set(n.parents).size,n.parents.length);
    assert.equal(new Set(refs).size,refs.length);
    if(n.kind !== 'content') assert.ok(refs.length > 0);
    for(const id of n.parents) {
      assert.ok(byID.has(id), 'missing parent');
      assert.equal(byID.get(id).kind === 'content',n.kind === 'content','wrong parent kind');
    }
    for(const id of refs) {
      assert.ok(byID.has(id), 'missing causal reference');
      assert.notEqual(byID.get(id).kind === 'content',n.kind === 'content','wrong causal reference kind');
    }
    edges.set(n.id,[...n.parents,...refs]);
  }
  const visiting = new Set(), done = new Set();
  function visit(id) {
    assert.ok(!visiting.has(id),'causal cycle');
    if(done.has(id)) return;
    visiting.add(id);for(const other of edges.get(id))visit(other);
    visiting.delete(id);done.add(id);
  }
  for(const id of byID.keys())visit(id);
  const heads = content => {
    const group=nodes.filter(n=>(n.kind==='content')===content);
    const parents=new Set(group.flatMap(n=>n.parents));
    return sorted(group.filter(n=>!parents.has(n.id)).map(n=>n.id));
  };
  const contentHeads=heads(true),visibilityHeads=heads(false);
  assert.ok(contentHeads.length > 0);
  if(!visibilityHeads.length) return {state:contentHeads.length===1?'visible':'content-conflict',contentHeads,visibilityHeads};
  if(visibilityHeads.length!==1) return {state:'visibility-conflict',contentHeads,visibilityHeads};
  const head=byID.get(visibilityHeads[0]);
  if(head.kind==='withdraw') return {state:'withdrawn',contentHeads,visibilityHeads};
  const reviewed=new Set();
  function include(id){if(reviewed.has(id))return;reviewed.add(id);for(const p of byID.get(id).parents)include(p);}
  for(const id of head.observed)include(id);
  const covered=contentHeads.every(id=>reviewed.has(id)||byID.get(id).visibility.includes(head.id));
  return {state:!covered?'unreviewed-content':contentHeads.length===1?'visible':'content-conflict',contentHeads,visibilityHeads};
}

export function appendDecision(nodes,event,expectedContent,expectedVisibility) {
  const before=inspect(nodes);
  assert.deepEqual(sorted(expectedContent),before.contentHeads,'stale content heads');
  assert.deepEqual(sorted(expectedVisibility),before.visibilityHeads,'stale visibility heads');
  assert.deepEqual(sorted(event.observed),before.contentHeads);
  assert.deepEqual(sorted(event.parents),before.visibilityHeads);
  const next=[...nodes,event];inspect(next);return next;
}

export const content=(id,parents=[],visibility=[])=>({id,record:'record-test',kind:'content',parents,visibility});
export const decision=(id,kind,parents,observed)=>({id,record:'record-test',kind,parents,observed});
