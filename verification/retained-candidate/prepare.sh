#!/usr/bin/env bash
set -euo pipefail
test "$(uname -s)" = Linux
test "$(uname -m)" = x86_64
repo_root=$PWD
test ! -e retained-platform-evidence
probe_root=$(mktemp -d "${RUNNER_TEMP:?}/mandalore-platform.XXXXXX")
mkdir "$probe_root/payload" "$probe_root/results"
bin/candidate-transport inspect --source 5b3c7b275bb4bbb058bebdd7153f4b76bbceb59f --run-id 34840171935 --artifact-id 10346171185 > "$probe_root/transport.json"
gh api repos/acoz-labs/mandalore/actions/artifacts/10346171185/zip > "$probe_root/candidate.zip"
bin/candidate-transport verify --receipt "$probe_root/transport.json" --archive "$probe_root/candidate.zip" > "$probe_root/results/verified.json"
test "$(sha256sum "$probe_root/candidate.zip" | cut -d ' ' -f 1)" = 3916b8df9270007ff5800c9bb68a53e516301b1f36065293b3844e9a6406616c
unzip -q "$probe_root/candidate.zip" -d "$probe_root/payload"
(cd "$probe_root/payload"; sha256sum -c SHA256SUMS)
chmod 700 "$probe_root/payload/mandalore_1.0.0_linux_amd64"
# Authentication is used only by the trusted transport verifier/downloader above.
# No Actions/provider environment or repository checkout credentials reach the CLI.
env -i PATH=/usr/bin:/bin:/usr/local/bin LANG=C.UTF-8 bash "$repo_root/verification/retained-candidate/probe.sh" "$probe_root" "$repo_root"
mkdir retained-platform-evidence
cp "$probe_root/results/"*.json retained-platform-evidence/
if grep -Eq '/Users/|/home/|OP_SERVICE_ACCOUNT_TOKEN|gh[pousr]_[A-Za-z0-9]{30,}' retained-platform-evidence/*.json; then
  printf 'Evidence privacy scan refused publication; matching content is not printed.\n' >&2
  exit 1
else
  test "$?" = 1
fi
printf 'Retained candidate executed natively on Linux AMD64; contributor model-free evidence only.\n'
