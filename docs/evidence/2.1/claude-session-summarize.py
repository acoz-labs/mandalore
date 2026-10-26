"""Summarize one native Claude Code run for claude-session-native.sh.

Reads the run's stream-json output and the proxy's memory_context log from the
given line on, and prints the session ID, what the hook sent and got, what
reached the model's context and the answer. Temporary paths become <tmp>.

    python3 claude-session-summarize.py NAME STREAM CALLS FIRST_CALL WORK
"""
import json
import os
import sys

name, stream, calls, first, work = sys.argv[1:6]

events = [json.loads(line) for line in open(stream) if line.startswith("{")]
init = next((e for e in events if e.get("subtype") == "init"), {})
result = next((e for e in events if e.get("type") == "result"), {})
lines = open(calls).read().splitlines() if os.path.exists(calls) else []
hook_calls = [json.loads(line) for line in lines[int(first):]]


def context(reply):
    try:
        return json.loads(reply)["hookSpecificOutput"]["additionalContext"]
    except (ValueError, KeyError, TypeError):
        return ""


summary = {
    "run": name,
    "session_id": init.get("session_id"),
    "mcp_servers": [{"name": s["name"], "status": s["status"]} for s in init.get("mcp_servers", [])],
    "hook_calls": [
        {
            "input": c["arguments"],
            "session_is_this_session": (c["arguments"] or {}).get("session") == init.get("session_id"),
            "is_error": c["is_error"],
            "reply": c["reply"] if c["is_error"] else None,
            "injected": context(c["reply"]),
            "injected_bytes": len(context(c["reply"]).encode()),
        }
        for c in hook_calls
    ],
    "hook_responses": [
        {"hook": e["hook_name"], "outcome": e.get("outcome"), "output": e.get("output") or e.get("stderr")}
        for e in events
        if e.get("subtype") == "hook_response"
    ],
    "answer": result.get("result"),
    "is_error": result.get("is_error"),
}
print(json.dumps(summary, indent=2).replace(work, "<tmp>").replace(os.path.realpath(work), "<tmp>"))
