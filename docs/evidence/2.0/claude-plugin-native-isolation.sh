#!/usr/bin/env bash
# Checks that claude-plugin-native.sh can't reach the operator's Claude Code
# login (#235). It runs the script against a stub `claude` that records its
# environment and fails, so the run stops at the first Claude Code call. The
# caller's HOME and CLAUDE_CONFIG_DIR are scratch directories too, so a broken
# guard would still touch nothing real. bin/ci runs it; it needs no Claude Code.
set -euo pipefail

here=$(cd "$(dirname "$0")" && pwd)
script="$here/claude-plugin-native.sh"
scratch=$(mktemp -d)
trap 'rm -rf "$scratch"' EXIT
mkdir -p "$scratch/bin" "$scratch/caller-home" "$scratch/caller-config" "$scratch/copy/docs/evidence/2.0"
cat > "$scratch/bin/claude" <<EOF
#!/usr/bin/env bash
echo "\${CLAUDE_CONFIG_DIR:-<unset>}|\$HOME" >> "$scratch/calls"
exit 1
EOF
chmod +x "$scratch/bin/claude"

fail() { echo "claude-plugin-native isolation: $*" >&2; exit 1; }

run() { # script [assignments, then optionally env -u NAME]; prints stderr, never fails
  local target=$1
  shift
  rm -f "$scratch/calls"
  env -u CLAUDE_CODE_OAUTH_TOKEN -u ANTHROPIC_API_KEY PATH="$scratch/bin:$PATH" \
    HOME="$scratch/caller-home" CLAUDE_CONFIG_DIR="$scratch/caller-config" "$@" \
    "$target" "$scratch/mandalore" "$scratch/out" 2>&1 > /dev/null || true
}

# No credential in the environment: refuse before any Claude Code call.
err=$(run "$script")
[[ ! -e "$scratch/calls" ]] || fail "ran claude without a credential in the environment"
[[ $err == *CLAUDE_CODE_OAUTH_TOKEN* ]] || fail "unexpected refusal without a credential: $err"

# With a credential, Claude Code sees only the run's own HOME and config dir.
run "$script" CLAUDE_CODE_OAUTH_TOKEN=synthetic > /dev/null
[[ -s "$scratch/calls" ]] || fail "the stub claude was never called"
IFS='|' read -r config home < "$scratch/calls"
[[ $config != "$scratch"/caller-* && $home != "$scratch"/caller-* ]] || fail "claude saw the caller's HOME or CLAUDE_CONFIG_DIR"
[[ $config == */claude-config && $home == */home && ${config%/claude-config} == "${home%/home}" ]] ||
  fail "claude ran outside the run's temporary directory: $config, $home"
[[ ! -e ${config%/claude-config} ]] || fail "the run's temporary directory was left behind"

# The guard: without the script's CLAUDE_CONFIG_DIR, unset or inherited from
# the caller, it refuses to start Claude Code.
grep -q -x 'export CLAUDE_CONFIG_DIR="$work/claude-config"' "$script" || fail "the CLAUDE_CONFIG_DIR export line moved; update this check"
copy="$scratch/copy/docs/evidence/2.0/claude-plugin-native.sh"
grep -v -x 'export CLAUDE_CONFIG_DIR="$work/claude-config"' "$script" > "$copy"
chmod +x "$copy"
for caller in "env -u CLAUDE_CONFIG_DIR" "CLAUDE_CONFIG_DIR=$scratch/caller-config"; do
  err=$(run "$copy" CLAUDE_CODE_OAUTH_TOKEN=synthetic $caller)
  [[ ! -e "$scratch/calls" ]] || fail "ran claude without the run's CLAUDE_CONFIG_DIR ($caller)"
  [[ $err == *"Refusing to run claude"* ]] || fail "unexpected failure without the run's CLAUDE_CONFIG_DIR ($caller): $err"
done

echo "claude-plugin-native.sh isolation checks passed."
