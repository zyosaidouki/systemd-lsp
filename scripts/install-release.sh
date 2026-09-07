#!/bin/sh
# Install only inside this plugin. No Go toolchain or PATH changes are needed.
set -eu
root=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
case "$(uname -s)/$(uname -m)" in
  Linux/x86_64) platform=linux-amd64 ;;
  Darwin/arm64) platform=darwin-arm64 ;;
  *) echo 'Unsupported platform (requires Linux x86-64 or macOS arm64)' >&2; exit 1 ;;
esac
for tool in curl tar; do
  command -v "$tool" >/dev/null || { echo "$tool is required" >&2; exit 1; }
done
mkdir -p "$root/bin"
# Serialize updates from different editor instances.
lock="$root/bin/.install-lock"
if ! mkdir "$lock" 2>/dev/null; then
  echo "Another installation is in progress. If no installer is running, remove $lock" >&2
  exit 1
fi
stage=''
cleanup() {
  if [ -n "$stage" ]; then rm -rf -- "$stage"; fi
  rmdir "$lock"
}
trap cleanup EXIT
trap 'exit 1' HUP INT TERM
stage=$(mktemp -d "$root/bin/.install-XXXXXX")
base=https://github.com/zyosaidouki/systemd-lsp/releases
# Resolve latest once, so checksum and archive are from the same release.
url=$(curl --proto '=https' --proto-redir '=https' -fLsS --retry 2 --max-time 120 \
  -w '%{url_effective}' -o "$stage/release.html" "$base/latest")
case "$url" in
  "$base/tag/"*) tag=${url#"$base/tag/"} ;;
  *) echo 'Unable to resolve latest release' >&2; exit 1 ;;
esac
case "$tag" in ''|*[!a-zA-Z0-9._-]*) echo 'Unexpected release tag' >&2; exit 1 ;; esac
asset="systemd-lsp-$platform.tar.gz"
for name in SHA256SUMS "$asset"; do
  curl --proto '=https' --proto-redir '=https' -fLsS --retry 2 --max-time 120 \
    --max-filesize 33554432 -o "$stage/$name" "$base/download/$tag/$name"
done
cd "$stage"
awk -v name="$asset" '$2 == name { print; found++ } END { if (found != 1) exit 1 }' SHA256SUMS > selected.sum
if command -v sha256sum >/dev/null; then
  sha256sum -c selected.sum
else
  shasum -a 256 -c selected.sum
fi
[ "$(tar -tzf "$asset")" = systemd-lsp ] || { echo 'Unexpected archive contents' >&2; exit 1; }
tar -xzf "$asset"
[ -f systemd-lsp ] && [ ! -L systemd-lsp ] || { echo 'Invalid executable' >&2; exit 1; }
chmod 755 systemd-lsp
# Check that the verified binary starts on this machine before replacing it.
./systemd-lsp --help
mv -f systemd-lsp "$root/bin/systemd-lsp"
printf 'Installed systemd-lsp %s (%s)\n' "$tag" "$platform"
