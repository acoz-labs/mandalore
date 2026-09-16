import {appendFileSync} from 'node:fs';
export default function(pi) {
  function record(event) { appendFileSync(process.env.PROBE_EVENTS, JSON.stringify(event)+'\n'); }
  pi.on('session_start',()=>record({event:'session_start'}));
  pi.on('input',(event)=>{
    record({event:'input',source:event.source,streaming:event.streamingBehavior??null});
    return {action:'handled'}; // No model, credentials or tools are requested.
  });
  pi.on('session_shutdown',()=>record({event:'session_shutdown'}));
}
