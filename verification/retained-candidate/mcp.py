"""Bounded actual stdio probe. No model, credentials, transcript or file writes."""
import json
import os
import select
import subprocess
import sys
import time

binary, binding, project = sys.argv[1:]
process = subprocess.Popen(
    [binary, "mcp", "--binding", binding, "--read-only"], cwd=project,
    stdin=subprocess.PIPE, stdout=subprocess.PIPE, stderr=subprocess.DEVNULL,
)
pending = b""
sequence = 0
metrics = []


def request(method, params):
    global pending, sequence
    sequence += 1
    process.stdin.write(json.dumps({"jsonrpc": "2.0", "id": sequence,
                                   "method": method, "params": params}).encode() + b"\n")
    process.stdin.flush()
    start = time.monotonic()
    deadline = start + 10
    while True:
        while b"\n" in pending:
            line, pending = pending.split(b"\n", 1)
            message = json.loads(line)
            if message.get("id") == sequence:
                assert "error" not in message, "MCP protocol error"
                metrics.append({"method": method, "response_bytes": len(line)})
                return message["result"]
        remaining = deadline - time.monotonic()
        assert remaining > 0 and select.select([process.stdout], [], [], remaining)[0], "MCP deadline"
        chunk = os.read(process.stdout.fileno(), 65536)
        assert chunk, "Premature MCP EOF"
        pending += chunk
        assert len(pending) < 1000000, "MCP response bound"


try:
    request("initialize", {"protocolVersion": "2025-03-26", "capabilities": {},
                           "clientInfo": {"name": "platform-verifier", "version": "1"}})
    process.stdin.write(b'{"jsonrpc":"2.0","method":"notifications/initialized"}\n')
    process.stdin.flush()
    tools = request("tools/list", {})["tools"]
    names = {tool["name"] for tool in tools}
    assert len(names) == 16 and "release_apply" not in names
    recalled = request("tools/call", {"name": "memory_recall", "arguments": {
        "scope": {"kind": "project", "id": "portability"}, "query": ""}})
    assert not recalled.get("isError")
    current = recalled["structuredContent"]["result"]["current"]
    assert len(current) == 1 and current[0]["body"] == "Silver Heron"
    refused = request("tools/call", {"name": "memory_journal_append", "arguments": {
        "kind": "note", "summary": "Synthetic request must be refused in read-only mode"}})
    assert refused.get("isError")
    assert refused["structuredContent"]["error"]["code"] == "operation.read_only"
finally:
    process.stdin.close()
    try:
        process.wait(timeout=5)
    except subprocess.TimeoutExpired:
        process.kill()
        process.wait(timeout=5)
        raise AssertionError("MCP failed to exit after EOF")
    assert process.returncode == 0, "MCP nonzero exit"
print(json.dumps({"passed": True, "tool_count": len(names), "requests": metrics}))
