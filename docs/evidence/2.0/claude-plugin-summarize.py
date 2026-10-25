"""Summarize one native Claude Code run for claude-plugin-native.sh.

Reads the stream-json output and the session transcript, and prints the hook
outcome, what reached the model's context, timings and the answer. Temporary
paths are replaced with <tmp>.
"""
import datetime
import glob
import json
import os
import sys

name, stream, projects, start, end, work = sys.argv[1:7]


def ts(value):
    return datetime.datetime.fromisoformat(value.replace("Z", "+00:00")).timestamp()


events = [json.loads(line) for line in open(stream) if line.startswith("{")]
result = next((e for e in events if e.get("type") == "result"), {})
init = next((e for e in events if e.get("subtype") == "init"), {})
hooks = [
    {"hook": e["hook_name"], "outcome": e.get("outcome"), "output": e.get("output") or e.get("stderr")}
    for e in events
    if e.get("subtype") == "hook_response"
]
tools = [
    c["name"]
    for e in events
    if e.get("type") == "assistant"
    for c in e["message"]["content"]
    if c.get("type") == "tool_use"
]

transcript = {"attachments": []}
session = result.get("session_id")
files = glob.glob(os.path.join(projects, "*", f"{session}.jsonl")) if session else []
if files:
    prompt_at = None
    for line in open(files[0]):
        d = json.loads(line)
        if d.get("type") == "user" and prompt_at is None and not d.get("isMeta"):
            prompt_at = ts(d["timestamp"])
        a = d.get("attachment") or {}
        if d.get("type") == "attachment" and "hook" in a.get("type", ""):
            item = {k: a[k] for k in ("type", "hookName", "hookEvent", "timedOut", "timeoutMs", "stderr") if k in a}
            if a["type"] == "hook_additional_context":
                text = "\n".join(a.get("content") or [])
                item["content"] = text
                item["bytes"] = len(text.encode())
                header = text.split("\n", 1)[0]
                item["policy_bytes"] = len(header.encode())
            if prompt_at is not None:
                item["seconds_after_prompt"] = round(ts(d["timestamp"]) - prompt_at, 3)
            transcript["attachments"].append(item)

summary = {
    "run": name,
    "mcp_servers": [{"name": s["name"], "status": s["status"]} for s in init.get("mcp_servers", [])],
    "hooks": hooks,
    "context": transcript["attachments"],
    "tools_called": tools,
    "answer": result.get("result"),
    "is_error": result.get("is_error"),
    "wall_seconds": round(float(end) - float(start), 2),
    "api_seconds": round(result.get("duration_api_ms", 0) / 1000, 2),
}
print(json.dumps(summary, indent=2).replace(work, "<tmp>").replace(os.path.realpath(work), "<tmp>"))
