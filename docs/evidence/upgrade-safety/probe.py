"""Native installer plus held hook/MCP consumers; not a model-session test.

Use an absent --root, released --old, candidate --new and actual --codex.
No authentication is copied or provider/model requests made.
"""
import argparse
import hashlib
import json
import os
from pathlib import Path
import queue
import subprocess
import threading


def main():
    parser = argparse.ArgumentParser()
    for name in ("root", "old", "new", "codex"):
        parser.add_argument("--" + name, required=True)
    args = parser.parse_args()
    root = Path(args.root).absolute()
    root.mkdir(mode=0o700)  # Refuse reuse, including ambiguous previous runs.
    profile = root / "profile"
    profile.mkdir(mode=0o700)
    env = {k: v for k, v in os.environ.items() if not k.startswith("MANDALORE_")}
    env["CODEX_HOME"] = str(profile)

    def emit(check, **result):
        print(json.dumps({"check": check, **result}), flush=True)

    def invoke(binary, operation, data, success=True):
        reply = subprocess.run([binary, "call", operation], input=json.dumps(data),
                               env=env, text=True, capture_output=True, timeout=60)
        value = json.loads(reply.stdout)
        assert value["ok"] == success, (operation, "unexpected result state")
        assert (reply.returncode == 0) == success
        return value["result"] if success else value["error"]

    invoke(args.old, "signet_create", {"repository": str(root / "bank"),
           "name": "Upgrade canary", "device_label": "Synthetic device"})
    invoke(args.old, "signet_bind", {"repository": str(root / "bank"),
           "binding": str(root / "binding.json"), "device_label": "Synthetic device",
           "actor": "Synthetic tester"})

    def plan(binary):
        return invoke(binary, "connection_plan", {
            "state_dir": str(root / "state"), "native_home": str(profile),
            "native_binary": args.codex, "source_binary": binary,
            "binding": str(root / "binding.json")})

    def snapshot(directory):
        return {str(p.relative_to(directory)): hashlib.sha256(p.read_bytes()).hexdigest()
                for p in directory.rglob("*") if p.is_file()}

    before_bank = snapshot(root / "bank")
    before_binding = (root / "binding.json").read_bytes()
    old_plan = plan(args.old)
    assert invoke(args.old, "connection_apply", old_plan)["installed"]
    old_cache = profile / "plugins/cache/mandalore/mandalore" / old_plan["plugin_version"]
    old_snapshot = snapshot(old_cache)

    def hook(cache):
        value = subprocess.run(["/bin/sh", str(cache / "scripts/connection.sh"), "hook"],
            input=json.dumps({"hook_event_name": "UserPromptSubmit", "prompt": "Read-only canary"}),
            env=dict(env, PLUGIN_ROOT=str(cache)), text=True, capture_output=True, timeout=15)
        assert value.returncode == 0, "hook failed"
        assert json.loads(value.stdout)["hookSpecificOutput"]["additionalContext"]

    hook(old_cache)
    proc = subprocess.Popen(["/bin/sh", str(old_cache / "scripts/connection.sh"), "mcp"],
        env=dict(env, PLUGIN_ROOT=str(old_cache)), stdin=subprocess.PIPE,
        stdout=subprocess.PIPE, stderr=subprocess.DEVNULL, text=True)
    replies = queue.Queue()

    def reader():
        try:
            for line in proc.stdout:
                replies.put(json.loads(line))
        except Exception as error:
            replies.put(error)

    threading.Thread(target=reader, daemon=True).start()

    def rpc(number, method, params):
        proc.stdin.write(json.dumps({"jsonrpc": "2.0", "id": number,
                                    "method": method, "params": params}) + "\n")
        proc.stdin.flush()
        while True:
            reply = replies.get(timeout=15)
            if isinstance(reply, Exception):
                raise reply
            if reply.get("id") == number:
                assert "error" not in reply
                return reply["result"]

    def inspect(number):
        result = rpc(number, "tools/call", {"name": "memory_inspect", "arguments": {}})
        envelope = result.get("structuredContent") or json.loads(result["content"][0]["text"])
        assert envelope["ok"] and envelope["result"]["healthy"]

    try:
        rpc(1, "initialize", {"protocolVersion": "2024-11-05", "capabilities": {},
                               "clientInfo": {"name": "upgrade-probe", "version": "1"}})
        proc.stdin.write('{"jsonrpc":"2.0","method":"notifications/initialized"}\n')
        proc.stdin.flush()
        inspect(2)
        new_plan = plan(args.new)
        denied = invoke(args.new, "connection_apply", new_plan, success=False)
        assert denied["connection_result"]["phase"] == "deferred"
        assert snapshot(old_cache) == old_snapshot
        hook(old_cache)
        inspect(3)
        assert proc.poll() is None
        emit("deferred_with_live_consumers", old_cache_unchanged=True,
             old_hook_working=True, old_mcp_working=True)
    finally:
        proc.stdin.close()
        try:
            proc.wait(timeout=5)
        except subprocess.TimeoutExpired:
            proc.terminate()
            proc.wait(timeout=5)

    activated = invoke(args.new, "connection_apply", dict(new_plan, sessions_stopped=True))
    assert activated["installed"] and activated["phase"] == "verified"
    new_cache = profile / "plugins/cache/mandalore/mandalore" / new_plan["plugin_version"]
    hook(new_cache)
    cache_snapshot = snapshot(new_cache)
    cache_times = {str(p): p.stat().st_mtime_ns for p in new_cache.rglob("*")}
    replay = invoke(args.new, "connection_apply", new_plan)
    assert replay["installed"] and snapshot(new_cache) == cache_snapshot
    assert cache_times == {str(p): p.stat().st_mtime_ns for p in new_cache.rglob("*")}
    assert snapshot(root / "bank") == before_bank
    assert (root / "binding.json").read_bytes() == before_binding
    emit("acknowledged_activation_and_replay", new_hook_working=True,
         replay_cache_bytes_and_mtimes_unchanged=True, bank_and_binding_unchanged=True)
    emit("COMPLETE", passed=True, scope="native installer and hook/MCP components; no model session")


if __name__ == "__main__":
    main()
