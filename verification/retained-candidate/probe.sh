#!/usr/bin/env bash
set -euo pipefail
probe_root=$1
repo_root=$2
target_os=${3:?expected OS required}
target_arch=${4:?expected architecture required}
driver="$probe_root/payload/mandalore_1.0.0_${target_os}_${target_arch}"
results="$probe_root/results"
case "$target_os/$target_arch/$(uname -s)/$(uname -m)" in
  linux/arm64/Linux/aarch64|linux/amd64/Linux/x86_64|darwin/amd64/Darwin/x86_64|darwin/arm64/Darwin/arm64) ;;
  *) printf 'Unexpected native probe platform\n' >&2; exit 1 ;;
esac
expected=$(jq -er --arg name "mandalore_1.0.0_${target_os}_${target_arch}" '.assets[]|select(.name==$name)|.sha256' "$probe_root/payload/manifest.json")
test "$(shasum -a 256 "$driver" | cut -d ' ' -f 1)" = "$expected"
mkdir "$probe_root/project"
cd "$probe_root/project"
"$driver" version > "$results/version.json"
jq -e --arg os "$target_os" --arg arch "$target_arch" '.ok and .result.os==$os and .result.arch==$arch and .result.source_commit=="f899cf6a2a255f3b6b35dcd778c672f799c65eb2"' "$results/version.json" >/dev/null
"$driver" signet create --repository "$probe_root/signet" --name 'Native Test Signet' --device-label 'Native Test Device' > "$probe_root/create.json"
"$driver" signet bind --repository "$probe_root/signet" --binding "$probe_root/binding.json" --device-label 'Native Test Device' --actor 'Example User' > "$probe_root/bind.json"
binding="$probe_root/binding.json"
"$driver" memory remember --binding "$binding" < "$repo_root/verification/retained-candidate/baseline.json" > "$probe_root/first.json"
jq --slurpfile previous "$probe_root/first.json" '. + {body:"Silver Heron",record_id:$previous[0].result.record_id,supersedes:[$previous[0].result.id],reason:"Explicit synthetic rename on native platform"}' "$repo_root/verification/retained-candidate/baseline.json" > "$probe_root/rename.json"
"$driver" memory remember --binding "$binding" < "$probe_root/rename.json" > "$probe_root/second.json"
"$driver" memory recall --binding "$binding" --scope-kind project --scope-id portability --query '' > "$results/recall.json"
jq -e '.ok and (.result.current|length)==1 and .result.current[0].body=="Silver Heron"' "$results/recall.json" >/dev/null
record=$(jq -er '.result.record_id' "$probe_root/first.json")
"$driver" memory history --binding "$binding" --record-id "$record" > "$results/history.json"
jq -e '.ok and (.result.items|length)==2 and .result.items[1].supersedes==[.result.items[0].id]' "$results/history.json" >/dev/null
find "$probe_root/signet" -type f -exec shasum -a 256 {} + | sort > "$probe_root/before-mcp.txt"
python3 "$repo_root/verification/retained-candidate/mcp.py" "$driver" "$binding" "$probe_root/project" > "$results/mcp.json"
find "$probe_root/signet" -type f -exec shasum -a 256 {} + | sort > "$probe_root/after-mcp.txt"
cmp "$probe_root/before-mcp.txt" "$probe_root/after-mcp.txt"
"$driver" memory git-init --binding "$binding" > "$probe_root/git-init.json"
git -c core.hooksPath=/dev/null init --bare --initial-branch=main "$probe_root/remote.git"
git -C "$probe_root/signet" remote add origin "$probe_root/remote.git"
"$driver" memory sync --binding "$binding" > "$results/sync.json"
jq -e '.ok and .result.delivered and .result.state=="synchronized"' "$results/sync.json" >/dev/null
git -c core.hooksPath=/dev/null clone --no-hardlinks "$probe_root/remote.git" "$probe_root/clone"
"$driver" signet bind --repository "$probe_root/clone" --binding "$probe_root/clone-binding.json" --device-label 'Native Clone Device' --actor 'Example Peer' > "$probe_root/clone-bind.json"
"$driver" memory recall --binding "$probe_root/clone-binding.json" --scope-kind project --scope-id portability --query '' > "$results/clone-recall.json"
cmp "$results/recall.json" "$results/clone-recall.json"
"$driver" release plan --candidate "$probe_root/payload" --prefix "$probe_root/install" > "$probe_root/install-plan.json"
"$driver" release apply < "$probe_root/install-plan.json" > "$probe_root/install-result.json"
jq -e '.ok and .result.installed' "$probe_root/install-result.json" >/dev/null
"$probe_root/install/bin/mandalore" version > "$results/installed-version.json"
cmp "$results/version.json" "$results/installed-version.json"
cp -R "$repo_root/internal/migration/testdata/bank-v1" "$probe_root/legacy"
"$driver" migration preflight --source "$probe_root/legacy" --output "$probe_root/converted" --device-label 'Native Conversion Device' --actor 'Example Migrator' > "$probe_root/migration-plan.json"
"$driver" migration apply --writers-stopped < "$probe_root/migration-plan.json" > "$probe_root/migration-result.json"
jq -e '.ok and .result.published' "$probe_root/migration-result.json" >/dev/null
diff -r "$probe_root/legacy" "$probe_root/converted/original"
"$driver" signet bind --repository "$probe_root/converted/signet" --binding "$probe_root/converted-binding.json" --device-label 'Native Converted Writer' --actor 'Example Writer' > "$probe_root/converted-bind.json"
"$driver" memory history --binding "$probe_root/converted-binding.json" --record-id record-project > "$results/migrated-history.json"
grep -Fq '9007199254740993123456789.123456789' "$results/migrated-history.json"
test -z "$(find "$probe_root/project" -mindepth 1 -print)"
jq -n --arg os "$target_os" --arg arch "$target_arch" --arg digest "$expected" '{passed:true,os:$os,arch:$arch,binary_sha256:$digest,source_commit:"f899cf6a2a255f3b6b35dcd778c672f799c65eb2",manifest_sha256:"c2f5a340d0e665c81e01bc26add4c0dfe8ba27faac87b83a3fd81aff254f56ea",native_agent:false,model_used:false,checks:["actual retained binary version","same-record correction/history","MCP stdio and read-only hash window","local Git delivery and fresh-clone recall","owned CLI install and exact launcher identity","synthetic migration and original snapshot","exact historical JSON number","unrelated cwd unchanged"]}' > "$results/summary.json"
