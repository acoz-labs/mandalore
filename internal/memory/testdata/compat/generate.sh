#!/usr/bin/env bash
# Rebuilds the synthetic format 1 and format 2 fixtures with a 2.1.2 binary,
# so tests can prove the current code reads data older releases wrote.
#
#   internal/memory/testdata/compat/generate.sh /path/to/mandalore-2.1.2
#
# Build that binary from commit 79b378e (bin/check-old-reader does). Every
# value here is synthetic. IDs and times are random, so a rebuild differs.
set -euo pipefail

old=$(cd "$(dirname "$1")" && pwd)/$(basename "$1")
out=$(cd "$(dirname "$0")" && pwd)
work=$(mktemp -d)
trap 'rm -rf "$work"' EXIT
export GIT_CONFIG_GLOBAL=/dev/null GIT_CONFIG_NOSYSTEM=1
git_() { git -c user.name=Synthetic -c user.email=synthetic@example.invalid -c commit.gpgsign=false "$@"; }

ok() { # Run the 2.1.2 CLI, fail unless it answers ok, print its result.
  local response
  response=$("$old" "$@") || true
  jq -e '.ok' >/dev/null <<<"$response" || { echo "failed: $* -> $response" >&2; exit 1; }
  jq -c '.result' <<<"$response"
}

build() { # build NAME: a format 1 signet with every format 1 document type.
  local root="$work/$1" binding="$work/$1.binding.json"
  ok signet create --repository "$root" --name "Synthetic $1" --device-label "Synthetic device" >/dev/null
  ok signet bind --repository "$root" --binding "$binding" --device-label "Synthetic second device" --actor "Synthetic actor" >/dev/null
  local scope='{"kind":"project","id":"copper-finch"}'
  local first
  first=$(ok memory remember --binding "$binding" <<<"{\"kind\":\"fact\",\"scope\":$scope,\"summary\":\"Project codename\",\"body\":\"Copper Finch\",\"basis\":\"user-direction\",\"reason\":\"Synthetic fixture\"}")
  local record revision
  record=$(jq -r '.record_id' <<<"$first")
  revision=$(jq -r '.id' <<<"$first")
  # Two corrections of the same revision: a correction and a visible conflict.
  for name in "Silver Heron" "Amber Lark"; do
    ok memory remember --binding "$binding" <<<"{\"kind\":\"fact\",\"scope\":$scope,\"summary\":\"Project codename\",\"body\":\"$name\",\"basis\":\"user-direction\",\"reason\":\"Renamed\",\"record_id\":\"$record\",\"supersedes\":[\"$revision\"]}" >/dev/null
  done
  ok memory remember --binding "$binding" <<<'{"kind":"project-state","scope":{"kind":"task","id":"zebra-task"},"summary":"Zebra task: drafting","body":"Half done.","basis":"observation","reason":"Synthetic fixture"}' >/dev/null
  ok memory journal-append --binding "$binding" <<<'{"kind":"outcome","summary":"Synthetic journal entry"}' >/dev/null
  local pin; pin=$(printf '%040d' 1)
  ok foundling register --binding "$binding" <<<"{\"name\":\"Legacy reference\",\"description\":\"Synthetic legacy foundling\",\"source\":{\"kind\":\"git\",\"locator\":\"https://github.com/example/legacy.git\"},\"pin\":{\"algorithm\":\"git-sha1\",\"value\":\"$pin\"},\"reason\":\"Synthetic fixture\"}" >/dev/null
  ok foundling register --binding "$binding" <<<"{\"mode\":\"canon\",\"branch\":\"main\",\"source_signet_id\":\"signet-synthetic-canon\",\"name\":\"Canon reference\",\"description\":\"Synthetic canon foundling\",\"source\":{\"kind\":\"git\",\"locator\":\"https://github.com/example/canon.git\"},\"pin\":{\"algorithm\":\"git-sha1\",\"value\":\"$pin\"},\"reason\":\"Synthetic fixture\"}" >/dev/null
  echo "$binding"
}

copy() { # copy ROOT NAME: the portable tree only.
  rm -rf "${out:?}/$2"
  mkdir -p "$out/$2"
  (cd "$1" && tar -cf - .gitignore signet.json memory provenance foundlings) | (cd "$out/$2" && tar -xf -)
}

binding=$(build format1)
copy "$work/format1" format1

binding=$(build format2)
root="$work/format2"
ok memory git-init --binding "$binding" >/dev/null
plan=$(ok call signet_upgrade_preview <<<"{\"binding_path\":\"$binding\"}")
ok call signet_upgrade_apply <<<"{\"plan\":$plan,\"stopped_writers\":true}" >/dev/null
ok memory checkpoint --binding "$binding" >/dev/null
state=$(ok memory recall --binding "$binding" --scope-kind task --scope-id zebra-task)
record=$(jq -r '.current[0].record_id' <<<"$state")
heads=$(ok memory visibility-history --binding "$binding" --record-id "$record")
content=$(jq -c '.visibility.content_heads' <<<"$heads")
ok memory withdraw --binding "$binding" <<<"{\"record_id\":\"$record\",\"expected_content_heads\":$content,\"expected_visibility_heads\":[],\"reason\":\"Synthetic withdrawal\"}" >/dev/null
heads=$(ok memory visibility-history --binding "$binding" --record-id "$record")
visibility=$(jq -c '.visibility.visibility_heads' <<<"$heads")
ok memory restore --binding "$binding" <<<"{\"record_id\":\"$record\",\"expected_content_heads\":$content,\"expected_visibility_heads\":$visibility,\"reason\":\"Synthetic restoration\"}" >/dev/null
copy "$root" format2
echo "Fixtures written to $out" >&2
