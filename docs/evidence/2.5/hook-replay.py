#!/usr/bin/env python3
"""Replay the Claude plugin's prompt hook against a loopback candidate (#232).

The hook is the UserPromptSubmit mcp_tool entry in plugins/claude/hooks/hooks.json,
read as shipped: its tool, input template and timeout. The replay fills
${prompt} and ${session_id} as Claude Code does and sends the tools/call over the
server's real MCP endpoint, sessionless MCP 2026-07-28 as Claude Code speaks it.
The measured bytes are the reply's hookSpecificOutput.additionalContext, which is
what Claude Code adds to the turn.

The signet is the synthetic evaluation set (internal/server/testdata/context-eval.json),
written by TestWriteEvaluationSignet. The sessions are fixed by
docs/changes/232-hook-selection/plan.md: S0 is the 20 scripted session prompts; S1
to S8 are one per scope, each with that scope's case questions and paraphrases,
its short questions, and every eighth tuning no-memory prompt. Each session runs
with its session ID and with the session input empty, at each budget; budgets
other than the default run on a build that differs only in contextDefaultBytes.

Usage: hook-replay.py OUTPUT_DIR [BUDGET ...]   (default budgets: 2048 1536 1024)
Needs go and git on PATH. Uses only synthetic data, and stops every server it
starts by PID.
"""

import json
import os
import re
import subprocess
import sys
import tempfile
import time
import urllib.request
import uuid

HERE = os.path.dirname(os.path.abspath(__file__))
REPO = os.path.abspath(os.path.join(HERE, "..", "..", ".."))
EVAL = os.path.join(REPO, "internal", "server", "testdata", "context-eval.json")
HOOKS = os.path.join(REPO, "plugins", "claude", "hooks", "hooks.json")
CONTEXT_GO = os.path.join(REPO, "internal", "server", "context.go")
PORT = 28741
RECORD = re.compile(r", (record-[a-z0-9-]+), device ")


def hook_entry():
    """The plugin's UserPromptSubmit mcp_tool hook, as shipped."""
    hooks = json.load(open(HOOKS))["hooks"]["UserPromptSubmit"]
    entries = [h for group in hooks for h in group["hooks"] if h.get("type") == "mcp_tool"]
    if len(entries) != 1 or entries[0]["tool"] != "memory_context":
        sys.exit("expected one mcp_tool hook calling memory_context in hooks.json")
    return entries[0]


def fill(template, values):
    """Substitute ${name} in every string of the hook's input, as Claude Code does."""
    if isinstance(template, str):
        for name, value in values.items():
            template = template.replace("${%s}" % name, value)
        return template
    if isinstance(template, dict):
        return {k: fill(v, values) for k, v in template.items()}
    if isinstance(template, list):
        return [fill(v, values) for v in template]
    return template


def sessions(e):
    """The plan's fixed prompt set: S0, then one session per scope."""
    scope_of = {r["record"]: (r["scope"]["kind"], r["scope"].get("id", "")) for r in e["records"]}
    scopes = []
    for r in e["records"]:
        if scope_of[r["record"]] not in scopes:
            scopes.append(scope_of[r["record"]])
    out = [("S0", [{"prompt": p} for p in e["session"]])]
    for k, scope in enumerate(scopes, 1):
        prompts = []
        for c in e["cases"]:
            if scope_of[c["expect"]] == scope:
                prompts += [{"prompt": c["question"], "expect": c["expect"]}, {"prompt": c["paraphrase"], "expect": c["expect"]}]
        prompts += [{"prompt": s["prompt"], "expect": s["expect"]} for s in e["short"]["tune"] if scope_of[s["expect"]] == scope]
        prompts += [{"prompt": p} for i, p in enumerate(e["no_memory"]["tune"]) if i % len(scopes) == k - 1]
        out.append(("S%d" % k, prompts))
    return out


def build(budget, work):
    """The candidate at this budget: the default build, or one whose only
    difference is contextDefaultBytes."""
    binary = os.path.join(work, "mandalore-%d" % budget)
    args = ["go", "build", "-trimpath", "-o", binary]
    source = open(CONTEXT_GO).read()
    default = int(re.search(r"contextDefaultBytes\s+=\s+(\d+)", source).group(1))
    if budget != default:
        patched = os.path.join(work, "context-%d.go" % budget)
        open(patched, "w").write(re.sub(r"(contextDefaultBytes\s+=\s+)\d+", r"\g<1>%d" % budget, source, count=1))
        overlay = os.path.join(work, "overlay-%d.json" % budget)
        json.dump({"Replace": {CONTEXT_GO: patched}}, open(overlay, "w"))
        args.append("-overlay=" + overlay)
    subprocess.run(args + ["./cmd/mandalore"], cwd=REPO, check=True)
    return binary, default


def write_signet(work):
    env = dict(os.environ, MANDALORE_EVAL_SIGNET_DIR=work, GOFLAGS="-count=1")
    subprocess.run(["go", "test", "./internal/server", "-run", "^TestWriteEvaluationSignet$"], cwd=REPO, env=env, check=True, stdout=subprocess.DEVNULL)
    s = json.load(open(os.path.join(work, "signet.json")))
    config = os.path.join(work, "config.json")
    json.dump({"listen": "127.0.0.1:%d" % PORT, "public_url": "http://127.0.0.1:%d" % PORT, "state_dir": os.path.join(work, "state"),
               "owner": {"name": "Owner"}, "auth": "loopback-trust", "local_scope": "read",
               "signets": [{"slug": "eval", "root": s["root"], "signet_id": s["signet_id"], "device_id": s["device_id"], "remote": s["remote"]}]},
              open(config, "w"))
    os.makedirs(os.path.join(work, "state"), exist_ok=True)
    return config


def call(hook, prompt, session):
    """One prompt through the hook: (bytes injected, injected records, text, error)."""
    arguments = fill(hook["input"], {"prompt": prompt, "session_id": session})
    body = {"jsonrpc": "2.0", "id": 1, "method": "tools/call", "params": {
        "_meta": {"io.modelcontextprotocol/protocolVersion": "2026-07-28", "io.modelcontextprotocol/clientCapabilities": {},
                  "io.modelcontextprotocol/clientInfo": {"name": "hook-replay", "version": "1"}},
        "name": hook["tool"], "arguments": arguments}}
    req = urllib.request.Request("http://127.0.0.1:%d/mcp" % PORT, data=json.dumps(body).encode(), headers={
        "Content-Type": "application/json", "Accept": "application/json, text/event-stream",
        "Mcp-Protocol-Version": "2026-07-28", "Mcp-Method": "tools/call", "Mcp-Name": hook["tool"]})
    try:
        reply = json.load(urllib.request.urlopen(req, timeout=hook.get("timeout", 3)))
    except Exception as err:  # the hook fails open: no memory for the prompt
        return 0, [], "", str(err)
    if "error" in reply or reply["result"].get("isError"):
        return 0, [], "", json.dumps(reply.get("error") or reply["result"])
    text = "".join(c.get("text", "") for c in reply["result"]["content"])
    context = json.loads(text)["hookSpecificOutput"]["additionalContext"] if text else ""
    return len(context.encode()), RECORD.findall(context), context, ""


def answer_line(context, record):
    """Whether the record's line was injected whole or clipped, or not at all."""
    for line in context.splitlines():
        if line.startswith("- ") and (", %s, device " % record) in line:
            return "clipped" if "…" in line.split(" [")[0] else "whole"
    return "absent"


def serve(binary, config):
    log = open(config + ".log", "a")
    proc = subprocess.Popen([binary, "serve", "--config", config], stdout=log, stderr=log)
    for _ in range(100):
        try:
            urllib.request.urlopen("http://127.0.0.1:%d/healthz" % PORT, timeout=1)
            return proc
        except Exception:
            time.sleep(0.1)
    proc.kill()
    proc.wait()
    sys.exit("server did not start:\n" + open(config + ".log").read())


def main():
    out = os.path.abspath(sys.argv[1])
    budgets = [int(b) for b in sys.argv[2:]] or [2048, 1536, 1024]
    os.makedirs(out, exist_ok=True)
    e = json.load(open(EVAL))
    hook = hook_entry()
    plan = sessions(e)
    summary = {"hook": hook, "prompts": sum(len(p) for _, p in plan), "runs": []}
    with tempfile.TemporaryDirectory() as work:
        config = write_signet(work)
        for budget in budgets:
            binary, default = build(budget, work)
            for mode in ("session", "no-session"):
                proc = serve(binary, config)  # a fresh server: no session state carried over
                try:
                    rows = []
                    for name, prompts in plan:
                        sid = str(uuid.uuid5(uuid.NAMESPACE_URL, "https://example.invalid/hook-replay/" + name)) if mode == "session" else ""
                        given = set()  # records this session already injected
                        for p in prompts:
                            n, records, context, err = call(hook, p["prompt"], sid)
                            row = {"session": name, "prompt": p["prompt"], "bytes": n, "records": records, "error": err}
                            if "expect" in p:
                                row["expect"], row["answer"] = p["expect"], answer_line(context, p["expect"])
                                if row["answer"] == "absent" and sid and p["expect"] in given:
                                    row["answer"] = "earlier"
                            given.update(records)
                            rows.append(row)
                finally:
                    proc.terminate()
                    proc.wait()
                sizes = [r["bytes"] for r in rows]
                answered = [r for r in rows if "expect" in r]
                run = {"budget": budget, "default_build": budget == default, "mode": mode, "prompts": len(rows),
                       "average": sum(sizes) / len(sizes), "max": max(sizes), "over_800": sum(s > 800 for s in sizes),
                       "errors": sum(1 for r in rows if r["error"]),
                       "answers": {k: sum(r["answer"] == k for r in answered) for k in ("whole", "clipped", "earlier", "absent")},
                       "per_session": {name: sum(r["bytes"] for r in rows if r["session"] == name) / len(ps) for name, ps in plan}}
                json.dump(rows, open(os.path.join(out, "replay-%d-%s.json" % (budget, mode)), "w"), indent=1, ensure_ascii=False)
                summary["runs"].append(run)
                print("budget %d %-10s average %.0f max %d over 800: %d of %d; answers %s; errors %d" % (
                    budget, mode, run["average"], run["max"], run["over_800"], len(rows), run["answers"], run["errors"]))
    json.dump(summary, open(os.path.join(out, "summary.json"), "w"), indent=1, ensure_ascii=False)


if __name__ == "__main__":
    main()
