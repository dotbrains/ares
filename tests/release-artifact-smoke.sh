#!/usr/bin/env sh
set -eu

bin="${ARES_BIN:-./ares}"
if [ ! -x "$bin" ]; then
  printf 'release-artifact-smoke: missing executable %s; run make build first\n' "$bin" >&2
  exit 1
fi

os="$(uname -s | tr '[:upper:]' '[:lower:]')"
arch="$(uname -m | sed 's/x86_64/amd64/; s/aarch64/arm64/')"
case "$arch" in
  amd64|arm64) ;;
  *) printf 'release-artifact-smoke: unsupported architecture: %s\n' "$arch" >&2; exit 1 ;;
esac

archive_root="$(mktemp -d)"
install_root="$(mktemp -d)"
apply_root="$(mktemp -d)"
trap 'rm -rf "$archive_root" "$install_root" "$apply_root"' EXIT

archive="$archive_root/ares_${os}_${arch}.tar.gz"
tar -czf "$archive" -C "$(dirname "$bin")" "$(basename "$bin")"

ARES_INSTALL_DIR="$install_root/bin" ARES_ARCHIVE="$archive" sh ./install.sh >"$install_root/install.txt" 2>&1
installed="$install_root/bin/ares"
test -x "$installed"
"$installed" --version >/dev/null

mkdir -p "$apply_root/etc/ssh"
printf 'Port 2222\n' > "$apply_root/etc/ssh/sshd_config"
ARES_ROOT="$apply_root" ARES_OS_RELEASE=tests/fixtures/os-release/ubuntu-24.04 "$installed" preflight --json >"$apply_root/preflight.json" 2>&1
grep -q '"profile": "basic"' "$apply_root/preflight.json"
grep -q '"transaction": {' "$apply_root/preflight.json"

ARES_ROOT="$apply_root" ARES_OS_RELEASE=tests/fixtures/os-release/ubuntu-24.04 "$installed" --yes >"$apply_root/apply.txt" 2>&1
test -f "$apply_root/etc/ssh/sshd_config.d/99-ares.conf"
test -f "$apply_root/var/log/ares/latest.json"

printf 'ok release-artifact-smoke\n'
