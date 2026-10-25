#!/usr/bin/env bash
# Smoke-test a server image with a synthetic signet: health check, /healthz and
# MCP tools/list over loopback-trust through docker exec, non-root, read-only
# /config. Uses throwaway named volumes, never a real signet or host path.
# Usage: packaging/server/smoke-test.sh IMAGE
set -euo pipefail
image=${1:?Usage: packaging/server/smoke-test.sh IMAGE}
run=mandalore-smoke-$$
cleanup() { docker rm -f "$run" >/dev/null 2>&1 || true; docker volume rm "$run-state" "$run-config" >/dev/null 2>&1 || true; }
trap cleanup EXIT
fail() { echo "FAIL: $1" >&2; exit 1; }

created=$(docker run --rm -v "$run-state:/state" "$image" signet create --repository /state/signets/primary --name Synthetic --device-label server)
field() { python3 -c 'import json,sys; print(json.loads(sys.argv[1])["result"][sys.argv[2]])' "$created" "$1"; }
signet_id=$(field signet_id)
device_id=$(field device_id)
docker run --rm -v "$run-state:/state" --entrypoint sh "$image" -c '
  cd /state/signets/primary && git init -q -b main && git add -A &&
  git -c user.name=Synthetic -c user.email=synthetic@example.invalid commit -qm "Create synthetic signet" &&
  git init -q --bare -b main /state/remote.git'

docker run --rm -i --user 0 -v "$run-config:/config" --entrypoint sh "$image" -c 'cat > /config/server.json' <<EOF
{"listen": "127.0.0.1:8730", "public_url": "http://127.0.0.1:8730", "state_dir": "/state/server",
 "owner": {"name": "Synthetic Owner"}, "auth": "loopback-trust", "local_scope": "read",
 "signets": [{"slug": "primary", "root": "/state/signets/primary", "signet_id": "$signet_id",
   "device_id": "$device_id", "remote": "file:///state/remote.git"}]}
EOF

docker run -d --name "$run" --read-only --tmpfs /tmp -v "$run-config:/config:ro" -v "$run-state:/state" "$image" >/dev/null
for _ in $(seq 60); do
  health=$(docker inspect -f '{{.State.Health.Status}}' "$run")
  [[ "$health" == healthy ]] && break
  [[ "$(docker inspect -f '{{.State.Running}}' "$run")" == true ]] || { docker logs "$run" >&2; fail "container exited"; }
  sleep 1
done
[[ "$health" == healthy ]] || { docker logs "$run" >&2; fail "health check status $health"; }
echo "ok: HEALTHCHECK healthy"

[[ "$(docker exec "$run" curl -fsS http://127.0.0.1:8730/healthz)" == alive ]] || fail "/healthz"
echo "ok: /healthz alive"

tools=$(docker exec "$run" curl -fsS -X POST http://127.0.0.1:8730/mcp -H 'Content-Type: application/json' \
  -H 'Accept: application/json, text/event-stream' -d '{"jsonrpc":"2.0","id":1,"method":"tools/list"}')
python3 -c '
import json, sys
names = {t["name"] for t in json.loads(sys.argv[1])["result"]["tools"]}
assert "memory_recall" in names and "memory_context" in names, names
assert "memory_remember" not in names, "read scope listed a write tool"
print("ok: tools/list over loopback-trust:", " ".join(sorted(names)))' "$tools"

uid=$(docker exec "$run" id -u)
[[ "$uid" != 0 ]] || fail "runs as root"
echo "ok: runs as uid $uid"
! docker exec "$run" touch /config/probe 2>/dev/null || fail "/config is writable"
docker exec "$run" touch /state/probe || fail "/state is not writable"
echo "ok: /config read-only, /state read-write"
