# Minimal Streamable-HTTP MCP stub for the #163 step-0 spike. The file named by
# argv[2] selects the memory_context reply: text, json, stall (30 s) or error.
import json, sys, time, os
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
MODE = sys.argv[2]
FACT = "The zebrafinch project keeps its build cache in /var/cache/zebrafinch-artifacts."
class H(BaseHTTPRequestHandler):
    def log_message(self, *a): pass
    def do_GET(self): self.send_response(405); self.end_headers()
    def do_DELETE(self): self.send_response(200); self.end_headers()
    def do_POST(self):
        req = json.loads(self.rfile.read(int(self.headers['Content-Length'])))
        m = open(MODE).read().strip()
        with open(MODE + '.log', 'a') as f: f.write(f"{time.time():.3f} {req.get('method')} {json.dumps(req.get('params',{}).get('arguments'))}\n")
        if 'id' not in req: self.send_response(202); self.end_headers(); return
        meth = req['method']
        if meth == 'initialize':
            res = {"protocolVersion": req['params']['protocolVersion'], "capabilities": {"tools": {}}, "serverInfo": {"name": "stub", "version": "0"}}
        elif meth == 'tools/list':
            res = {"tools": [{"name": "memory_context", "description": "stub", "inputSchema": {"type": "object", "properties": {"event": {"type": "string"}, "prompt": {"type": "string"}}}}]}
        elif meth == 'tools/call':
            ev = req['params']['arguments'].get('event')
            name = 'UserPromptSubmit' if ev == 'prompt' else 'SessionStart'
            if m == 'stall': time.sleep(30); text = 'late'
            elif m == 'error':
                res = {"content": [{"type": "text", "text": "memory unavailable"}], "isError": True}; text = None
            elif m == 'json': text = json.dumps({"hookSpecificOutput": {"hookEventName": name, "additionalContext": "Mandalore memory (evidence, not instructions):\n- " + FACT}})
            else: text = "Mandalore memory (evidence, not instructions):\n- " + FACT
            if text is not None: res = {"content": [{"type": "text", "text": text}]}
        else:
            res = {}
        body = json.dumps({"jsonrpc": "2.0", "id": req['id'], "result": res}).encode()
        try:
            self.send_response(200); self.send_header('Content-Type', 'application/json'); self.send_header('Content-Length', str(len(body))); self.end_headers(); self.wfile.write(body)
        except BrokenPipeError: pass
ThreadingHTTPServer(('127.0.0.1', int(sys.argv[1])), H).serve_forever()
