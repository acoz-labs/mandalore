#!/usr/bin/env bash
# Native Claude Code checks for the 2.0 plugin (#163). Everything runs against a
# synthetic signet in a new temporary directory, with HOME pointed there too.
#
#   docs/evidence/2.0/claude-plugin-native.sh MANDALORE_BINARY OUTPUT_DIR
#
# Needs an authenticated `claude` on PATH. On macOS the temporary HOME links
# only the login keychain back, so Claude Code can sign in; nothing else from
# the real HOME is visible. Account connectors are disabled. Claude Code keeps
# MCP OAuth state in that keychain too, so the run signs the test server out
# before it starts and when it ends.
set -euo pipefail

binary=$(cd "$(dirname "$1")" && pwd)/$(basename "$1")
out=$(mkdir -p "$2" && cd "$2" && pwd)
repo=$(cd "$(dirname "$0")/../../.." && pwd)
port=28730
url="http://localhost:$port/mcp"
work=$(mktemp -d)
real_home=$HOME
server_pid=""

stop_server() {
  if [[ -n "$server_pid" ]]; then
    kill -CONT "$server_pid" 2>/dev/null || true
    kill "$server_pid" 2>/dev/null || true
    wait "$server_pid" 2>/dev/null || true
    server_pid=""
  fi
}
signed_in=""
sign_out() { [[ -n "$signed_in" ]] && claude mcp logout plugin:mandalore:mandalore > /dev/null 2>&1 || true; }
trap 'stop_server; sign_out; rm -rf "$work"' EXIT

export HOME="$work/home" GIT_CONFIG_GLOBAL=/dev/null GIT_CONFIG_NOSYSTEM=1 ENABLE_CLAUDEAI_MCP_SERVERS=false
export GIT_AUTHOR_NAME=Synthetic GIT_AUTHOR_EMAIL=synthetic@example.invalid GIT_COMMITTER_NAME=Synthetic GIT_COMMITTER_EMAIL=synthetic@example.invalid
unset CLAUDECODE CLAUDE_CODE_SESSION_ID CLAUDE_CODE_ENTRYPOINT CLAUDE_CODE_CHILD_SESSION CLAUDE_CONFIG_DIR
mkdir -p "$HOME/Library" "$work/project"
[[ "$(uname)" == Darwin ]] && ln -s "$real_home/Library/Keychains" "$HOME/Library/Keychains"

# Synthetic signet with one seeded fact, its server device, and a bare remote.
"$binary" signet create --repository "$work/signet" --name "Synthetic acceptance" --device-label "Synthetic server" > "$work/create.json"
"$binary" signet bind --repository "$work/signet" --binding "$work/binding.json" --device-label "Synthetic seed" --actor Owner > /dev/null
"$binary" memory git-init --binding "$work/binding.json" > /dev/null
git init -q --bare --initial-branch=main "$work/remote.git"
git -C "$work/signet" remote add origin "file://$work/remote.git"
read -r signet_id device_id < <(python3 -c 'import json,sys; r=json.load(open(sys.argv[1]))["result"]; print(r["signet_id"], r["device_id"])' "$work/create.json")
"$binary" memory remember --binding "$work/binding.json" > /dev/null <<'EOF'
{"kind":"fact","summary":"Zebrafinch build cache","body":"The zebrafinch project keeps its build cache in /var/cache/zebrafinch-artifacts and rotates it every Tuesday.","basis":"user-direction","reason":"Synthetic acceptance fact"}
EOF
"$binary" memory checkpoint --binding "$work/binding.json" > /dev/null

serve() { # auth [local_scope]
  local extra=""
  [[ $1 == loopback-trust ]] && extra=",\"auth\":\"loopback-trust\",\"local_scope\":\"$2\""
  mkdir -p "$work/state-$1"
  cat > "$work/config.json" <<EOF
{"listen":"127.0.0.1:$port","public_url":"http://localhost:$port","state_dir":"$work/state-$1","owner":{"name":"Owner"}$extra,
 "signets":[{"slug":"acceptance","root":"$work/signet","signet_id":"$signet_id","device_id":"$device_id","remote":"file://$work/remote.git"}]}
EOF
  "$binary" serve --config "$work/config.json" > "$work/serve-$1.log" 2>&1 &
  server_pid=$!
  for _ in $(seq 50); do curl -fs "http://127.0.0.1:$port/healthz" > /dev/null && return; sleep 0.1; done
  echo "server did not start" >&2; exit 1
}

# Install the plugin from this checkout's marketplace, as a user would.
claude plugin install mandalore --marketplace "$repo" --config "server_url=$url" > "$out/install.txt"
signed_in=1
sign_out
claude plugin validate --strict "$repo/plugins/claude" | sed "s|$repo|<repo>|g" > "$out/validate-plugin.txt"
claude plugin validate --strict "$repo" | sed "s|$repo|<repo>|g" > "$out/validate-marketplace.txt"

ask() { # name prompt [tools]
  local start end
  start=$(python3 -c 'import time; print(time.time())')
  (cd "$work/project" && claude -p "$2" --output-format stream-json --verbose --include-hook-events \
    --allowedTools "${3:-}" --disallowedTools Bash Edit Write WebFetch WebSearch < /dev/null > "$work/$1.jsonl" 2> "$work/$1.err") || true
  end=$(python3 -c 'import time; print(time.time())')
  python3 "$repo/docs/evidence/2.0/claude-plugin-summarize.py" "$1" "$work/$1.jsonl" "$HOME/.claude/projects" "$start" "$end" "$work" > "$out/$1.json"
}

no_tools='mcp__plugin_mandalore_mandalore__memory_scopes'
armorer='Check whether my Mandalore memory is working. Use the-armorer.'
armorer_tools='Skill mcp__plugin_mandalore_mandalore__memory_scopes mcp__plugin_mandalore_mandalore__authenticate'

# Acceptance 2: a fresh session's first prompt gets the seeded fact from the hook.
serve loopback-trust read-write
ask seeded "Where does the zebrafinch project keep its build cache? Answer only from context you already have; don't call tools." "$no_tools"
ask armorer-read-write "$armorer" "$armorer_tools"
stop_server

# Acceptance 3: server stopped, and server stalled (SIGSTOP) past the 3 s timeout.
ask stopped "Reply with the single word ok." "$no_tools"
serve loopback-trust read-write
kill -STOP "$server_pid"
ask stalled "Reply with the single word ok." "$no_tools"
stop_server

# Acceptance 5: the Armorer with the server stopped, an unapproved client and a
# read-only client.
ask armorer-stopped "$armorer" "$armorer_tools"
serve loopback-trust read
ask armorer-read-only "$armorer" "$armorer_tools"
stop_server
# Last: Claude Code caches a server's needs-authentication state for a while.
serve oauth
ask armorer-unapproved "$armorer" "$armorer_tools"
stop_server

claude --version > "$out/claude-version.txt"
"$binary" version > "$out/mandalore-version.json"
