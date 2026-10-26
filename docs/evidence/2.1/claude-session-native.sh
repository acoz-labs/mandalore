#!/usr/bin/env bash
# Native Claude Code checks for the plugin's session de-duplication (#228).
# Everything runs against a synthetic signet in a new temporary directory, with
# HOME pointed there too, as in docs/evidence/2.0/claude-plugin-native.sh.
#
#   docs/evidence/2.1/claude-session-native.sh SERVER_2_1 SERVER_2_0_3 OUTPUT_DIR
#
# SERVER_2_1 is a 2.1.0 or later mandalore binary, SERVER_2_0_3 a 2.0.3 one.
# Claude Code reaches the server through claude-session-proxy.py, which logs
# each memory_context call's input and reply. Needs an authenticated `claude`
# on PATH; the test server is signed out of Claude Code before and after.
set -euo pipefail

abs() { echo "$(cd "$(dirname "$1")" && pwd)/$(basename "$1")"; }
current=$(abs "$1")
older=$(abs "$2")
out=$(mkdir -p "$3" && cd "$3" && pwd)
here=$(cd "$(dirname "$0")" && pwd)
repo=$(cd "$here/../../.." && pwd)
port=28730
upstream=28731
url="http://localhost:$port/mcp"
work=$(mktemp -d)
real_home=$HOME
server_pid=""
proxy_pid=""

stop() { # pid variable name
  local pid=${!1}
  if [[ -n "$pid" ]]; then
    kill "$pid" 2>/dev/null || true
    wait "$pid" 2>/dev/null || true
    printf -v "$1" ''
  fi
}
signed_in=""
sign_out() { [[ -n "$signed_in" ]] && claude mcp logout plugin:mandalore:mandalore > /dev/null 2>&1 || true; }
trap 'stop server_pid; stop proxy_pid; sign_out; rm -rf "$work"' EXIT

export HOME="$work/home" GIT_CONFIG_GLOBAL=/dev/null GIT_CONFIG_NOSYSTEM=1 ENABLE_CLAUDEAI_MCP_SERVERS=false
export GIT_AUTHOR_NAME=Synthetic GIT_AUTHOR_EMAIL=synthetic@example.invalid GIT_COMMITTER_NAME=Synthetic GIT_COMMITTER_EMAIL=synthetic@example.invalid
unset CLAUDECODE CLAUDE_CODE_SESSION_ID CLAUDE_CODE_ENTRYPOINT CLAUDE_CODE_CHILD_SESSION CLAUDE_CONFIG_DIR
mkdir -p "$HOME/Library" "$work/project"
[[ "$(uname)" == Darwin ]] && ln -s "$real_home/Library/Keychains" "$HOME/Library/Keychains"

# Synthetic signet with one seeded fact.
"$current" signet create --repository "$work/signet" --name "Synthetic acceptance" --device-label "Synthetic server" > "$work/create.json"
"$current" signet bind --repository "$work/signet" --binding "$work/binding.json" --device-label "Synthetic seed" --actor Owner > /dev/null
"$current" memory git-init --binding "$work/binding.json" > /dev/null
git init -q --bare --initial-branch=main "$work/remote.git"
git -C "$work/signet" remote add origin "file://$work/remote.git"
read -r signet_id device_id < <(python3 -c 'import json,sys; r=json.load(open(sys.argv[1]))["result"]; print(r["signet_id"], r["device_id"])' "$work/create.json")
"$current" memory remember --binding "$work/binding.json" > /dev/null <<'JSON'
{"kind":"fact","summary":"Zebrafinch build cache","body":"The zebrafinch project keeps its build cache in /var/cache/zebrafinch-artifacts and rotates it every Tuesday.","basis":"user-direction","reason":"Synthetic acceptance fact"}
JSON
"$current" memory checkpoint --binding "$work/binding.json" > /dev/null

serve() { # binary
  mkdir -p "$work/state"
  cat > "$work/config.json" <<JSON
{"listen":"127.0.0.1:$upstream","public_url":"http://localhost:$port","state_dir":"$work/state","owner":{"name":"Owner"},
 "auth":"loopback-trust","local_scope":"read-write",
 "signets":[{"slug":"acceptance","root":"$work/signet","signet_id":"$signet_id","device_id":"$device_id","remote":"file://$work/remote.git"}]}
JSON
  "$1" serve --config "$work/config.json" >> "$work/serve.log" 2>&1 &
  server_pid=$!
  for _ in $(seq 50); do curl -fs "http://127.0.0.1:$upstream/healthz" > /dev/null && return; sleep 0.1; done
  echo "server did not start" >&2; exit 1
}

# MCP call straight to the server, for the revision (not through the hook).
mcp() { # tool arguments-json
  curl -fs "http://127.0.0.1:$upstream/mcp" -H 'Content-Type: application/json' -H 'Accept: application/json, text/event-stream' \
    -d "{\"jsonrpc\":\"2.0\",\"id\":1,\"method\":\"tools/call\",\"params\":{\"name\":\"$1\",\"arguments\":$2}}"
}

touch "$work/calls.jsonl"
python3 "$here/claude-session-proxy.py" "$port" "http://127.0.0.1:$upstream" "$work/calls.jsonl" &
proxy_pid=$!

# Install the plugin from this checkout's marketplace, as a user would.
claude plugin install mandalore --marketplace "$repo" --config "server_url=$url" > "$out/install.txt"
signed_in=1
sign_out

question="Where does the zebrafinch project keep its build cache? Answer only from context you already have; don't call tools."
ask() { # name session-flag-or-empty
  local calls_before
  calls_before=$(wc -l < "$work/calls.jsonl")
  (cd "$work/project" && claude -p "$question" ${2:+--resume "$2"} --output-format stream-json --verbose --include-hook-events \
    --allowedTools mcp__plugin_mandalore_mandalore__memory_scopes --disallowedTools Bash Edit Write WebFetch WebSearch \
    < /dev/null > "$work/$1.jsonl" 2> "$work/$1.err") || true
  python3 "$here/claude-session-summarize.py" "$1" "$work/$1.jsonl" "$work/calls.jsonl" "$calls_before" "$work" > "$out/$1.json"
}
session_of() { python3 -c 'import json,sys; print(json.load(open(sys.argv[1]))["session_id"])' "$out/$1.json"; }

serve "$current"
# 1: a new session gets the fact; the same session, resumed, doesn't again.
ask first ""
s1=$(session_of first)
ask resumed "$s1"
# 1: once the record is revised, the session gets the new revision, once.
record=$(python3 -c 'import json,re,sys; print(re.search(r"record-[0-9a-f]+", open(sys.argv[1]).read()).group(0))' "$out/first.json")
mcp memory_history "{\"record_id\":\"$record\"}" > "$work/history.json"
revision=$(python3 -c 'import json,re,sys; print(sorted(set(re.findall(r"revision-[0-9a-f]+", open(sys.argv[1]).read())))[0])' "$work/history.json")
mcp memory_remember "{\"request_id\":\"synthetic-revision-0001\",\"record\":{\"kind\":\"fact\",\"record_id\":\"$record\",\"supersedes\":[\"$revision\"],\"summary\":\"Zebrafinch build cache\",\"body\":\"The zebrafinch project keeps its build cache in /var/cache/zebrafinch-artifacts and rotates it every Friday.\",\"basis\":\"user-direction\",\"reason\":\"Synthetic revision\"}}" > "$work/revise.json"
ask revised "$s1"
ask revised-again "$s1"
# 2: another session has its own state.
ask second ""
# 2: a restart forgets every session, so the resumed session gets it again.
stop server_pid
serve "$current"
ask after-restart "$s1"
stop server_pid

# 3: a 2.0.3 server refuses `session`; the hook fails open.
serve "$older"
ask older-server ""
stop server_pid

claude --version > "$out/claude-version.txt"
"$current" version > "$out/server-current-version.json"
"$older" version > "$out/server-older-version.json"
