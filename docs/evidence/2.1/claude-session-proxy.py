"""Logging MCP proxy for claude-session-native.sh (#228).

Listens on 127.0.0.1:PORT, forwards every request to UPSTREAM unchanged, and
appends one JSON line per memory_context call to LOG: the arguments Claude Code
sent and the text the server answered. It shows what the hook substitutes,
which the server itself doesn't log.

    python3 claude-session-proxy.py PORT UPSTREAM LOG
"""
import http.client
import json
import sys
import time
import urllib.error
import urllib.request
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer

PORT, UPSTREAM, LOG = int(sys.argv[1]), sys.argv[2], sys.argv[3]
SKIP = {"host", "content-length", "connection", "transfer-encoding"}


class Proxy(BaseHTTPRequestHandler):
    protocol_version = "HTTP/1.1"

    def log_message(self, *args):
        pass

    def forward(self):
        body = self.rfile.read(int(self.headers.get("Content-Length") or 0)) or None
        request = urllib.request.Request(UPSTREAM + self.path, data=body, method=self.command)
        for key, value in self.headers.items():
            if key.lower() not in SKIP:
                request.add_header(key, value)
        try:
            response = urllib.request.urlopen(request, timeout=30)
        except urllib.error.HTTPError as error:
            response = error
        stream = response.headers.get("Content-Type", "").startswith("text/event-stream")
        reply = b"" if stream else response.read()
        self.send_response(response.status)
        for key, value in response.headers.items():
            if key.lower() not in SKIP | {"date", "server"}:
                self.send_header(key, value)
        if stream:  # a listen stream: pass events through as they come
            self.send_header("Connection", "close")
            self.close_connection = True
        else:
            self.send_header("Content-Length", str(len(reply)))
            self.record(body, reply)
        self.end_headers()
        self.wfile.write(reply)
        try:
            while stream and (chunk := response.read1(65536)):
                self.wfile.write(chunk)
                self.wfile.flush()
        except (OSError, http.client.HTTPException):
            pass  # Claude Code exited, or the stream went idle
        response.close()

    def record(self, body, reply):
        try:
            call = json.loads(body or b"{}")
        except ValueError:
            return
        params = call.get("params") or {}
        if call.get("method") != "tools/call" or params.get("name") != "memory_context":
            return
        try:
            result = json.loads(reply).get("result") or {}
        except ValueError:
            result = {}
        text = "".join(c.get("text", "") for c in result.get("content") or [])
        with open(LOG, "a") as log:
            log.write(json.dumps({"at": round(time.time(), 3), "arguments": params.get("arguments"),
                                  "is_error": bool(result.get("isError")), "reply": text}) + "\n")

    do_GET = do_POST = do_DELETE = forward


ThreadingHTTPServer(("127.0.0.1", PORT), Proxy).serve_forever()
