#!/bin/sh
# Review this script before running: sh install.sh VERSION
# Trusts the official GitHub release and its checksums, not an independent signer.
# This only bootstraps the verified CLI's explicit, default-No installation menu.
set -eu
umask 077

fail() { printf 'Mandalore: %s\n' "$1" >&2; exit 1; }
[ "$#" -eq 1 ] || fail 'Usage: sh install.sh VERSION (for example, 1.0.0).'
release_version=$1
case "$release_version" in ''|*[!0-9A-Za-z.+-]*) fail 'Invalid release version.';; esac
awk -v v="$release_version" 'BEGIN {
  if (length(v)>96) exit 1
  n=split(v,b,/[+]/)
  if (n>2 || (n==2 && b[2]!~/^[0-9A-Za-z-]+([.][0-9A-Za-z-]+)*$/)) exit 1
  core=b[1]; dash=index(core,"-")
  if (dash) {
    pre=substr(core,dash+1); core=substr(core,1,dash-1)
    if (pre!~/^[0-9A-Za-z-]+([.][0-9A-Za-z-]+)*$/) exit 1
    count=split(pre,p,/[.]/)
    for(i=1;i<=count;i++) if(p[i]~/^[0-9]+$/ && length(p[i])>1 && substr(p[i],1,1)=="0") exit 1
  }
  if(split(core,c,/[.]/)!=3) exit 1
  for(i=1;i<=3;i++) if(c[i]!~/^(0|[1-9][0-9]*)$/) exit 1
}' || fail 'Invalid release version.'

case "$(uname -s)" in Darwin) target_os=darwin;; Linux) target_os=linux;; *) fail 'Unsupported operating system.';; esac
case "$(uname -m)" in arm64|aarch64) target_arch=arm64;; x86_64|amd64) target_arch=amd64;; *) fail 'Unsupported architecture.';; esac
command -v curl >/dev/null 2>&1 || fail 'curl 8.4 or newer is required for bounded downloads.'
curl -q --version | awk 'NR==1 { split($2,v,"."); ok=(v[1]>8 || (v[1]==8 && v[2]>=4)) } END { exit !ok }' ||
  fail 'curl 8.4 or newer is required for bounded downloads; alternatively download the platform binary and checksums manually.'
if command -v sha256sum >/dev/null 2>&1; then checksum_tool=sha256sum
elif command -v shasum >/dev/null 2>&1; then checksum_tool=shasum
else fail 'A SHA-256 tool (sha256sum or shasum) is required.'
fi

bootstrap_tmp=$(mktemp -d "${TMPDIR:-/tmp}/mandalore-bootstrap.XXXXXXXX") || fail 'Cannot create a private temporary directory.'
cleanup() {
  # Remove only the two fixed scratch files created by this invocation, then its directory.
  rm -f "$bootstrap_tmp/mandalore" "$bootstrap_tmp/SHA256SUMS"
  rmdir "$bootstrap_tmp"
}
trap cleanup EXIT
trap 'exit 1' HUP INT TERM

release_base="https://github.com/acoz-labs/mandalore/releases/download/v$release_version"
asset_name="mandalore_${release_version}_${target_os}_${target_arch}"
download() {
  # -q disables user curlrc settings; no credentials, netrc or token headers are used.
  effective_url=$(curl -q --fail --silent --show-error --location --max-redirs 5 \
    --proto '=https' --proto-redir '=https' --connect-timeout 15 --max-time 120 \
    --max-filesize "$3" --output "$2" --write-out '%{url_effective}' "$1" 2>/dev/null) ||
    fail 'Download failed, exceeded its limit or the selected release does not exist. Nothing was installed.'
  case "$effective_url" in
    https://github.com/*|https://release-assets.githubusercontent.com/*|https://objects.githubusercontent.com/*) ;;
    *) fail 'Download ended at an untrusted release host. Nothing was installed.';;
  esac
}
printf 'Downloading Mandalore %s for %s/%s from its official GitHub release.\n' "$release_version" "$target_os" "$target_arch"
download "$release_base/SHA256SUMS" "$bootstrap_tmp/SHA256SUMS" 65536
expected=$(awk -v name="$asset_name" '
  NF==2 && $2==name { count++; if(length($1)!=64 || $1~/[^0-9a-f]/) bad=1; hash=$1 }
  END { if(count!=1 || bad) exit 1; print hash }
' "$bootstrap_tmp/SHA256SUMS") || fail 'Missing, duplicate or invalid platform checksum.'
download "$release_base/$asset_name" "$bootstrap_tmp/mandalore" 134217728
if [ "$checksum_tool" = sha256sum ]; then actual=$(sha256sum "$bootstrap_tmp/mandalore" | awk '{print $1}')
else actual=$(shasum -a 256 "$bootstrap_tmp/mandalore" | awk '{print $1}')
fi
[ "$actual" = "$expected" ] || fail 'Release checksum mismatch. The downloaded program was not executed.'
chmod 700 "$bootstrap_tmp/mandalore"
printf 'Checksum verified. Opening the installation preview; memory connections are not changed automatically.\n'
"$bootstrap_tmp/mandalore" release install --version "$release_version"
