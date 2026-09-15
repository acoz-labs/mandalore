#!/usr/bin/env bash
set -euo pipefail
target_os=${1:?expected OS required}
target_arch=${2:?expected architecture required}
case "$target_os/$target_arch/$(uname -s)/$(uname -m)" in
  linux/arm64/Linux/aarch64|linux/amd64/Linux/x86_64|darwin/amd64/Darwin/x86_64|darwin/arm64/Darwin/arm64) ;;
  *) printf 'Unexpected native platform\n' >&2; exit 1 ;;
esac
if test "$target_os" = darwin; then
  test "$(sysctl -n sysctl.proc_translated 2>/dev/null || true)" != 1
fi
repo_root=$PWD
test ! -e retained-platform-evidence
probe_root=$(mktemp -d "${RUNNER_TEMP:?}/mandalore-platform.XXXXXX")
mkdir "$probe_root/payload" "$probe_root/results"
bin/candidate-transport inspect --source f899cf6a2a255f3b6b35dcd778c672f799c65eb2 --run-id 34886390526 --artifact-id 10365425624 > "$probe_root/transport.json"
gh api repos/acoz-labs/mandalore/actions/artifacts/10365425624/zip > "$probe_root/candidate.zip"
bin/candidate-transport verify --receipt "$probe_root/transport.json" --archive "$probe_root/candidate.zip" > "$probe_root/results/verified.json"
test "$(shasum -a 256 "$probe_root/candidate.zip" | cut -d ' ' -f 1)" = 424c4151887d57cfebd667233aa9cb9fffa3304dd449c6c842b91bd8f83b311d
unzip -q "$probe_root/candidate.zip" -d "$probe_root/payload"
(cd "$probe_root/payload"; shasum -a 256 -c SHA256SUMS)
chmod 700 "$probe_root/payload/mandalore_1.0.0_${target_os}_${target_arch}"
# Authentication is used only by the trusted transport verifier/downloader above.
# No Actions/provider environment or repository checkout credentials reach the CLI.
env -i PATH=/usr/bin:/bin:/usr/local/bin LANG=C bash "$repo_root/verification/retained-candidate/probe.sh" "$probe_root" "$repo_root" "$target_os" "$target_arch"
mkdir retained-platform-evidence
cp "$probe_root/results/"*.json retained-platform-evidence/
if grep -Eq '/Users/|/home/|OP_SERVICE_ACCOUNT_TOKEN|gh[pousr]_[A-Za-z0-9]{30,}' retained-platform-evidence/*.json; then
  printf 'Evidence privacy scan refused publication; matching content is not printed.\n' >&2
  exit 1
else
  test "$?" = 1
fi
printf 'Retained candidate executed natively on %s/%s; contributor model-free evidence only.\n' "$target_os" "$target_arch"
