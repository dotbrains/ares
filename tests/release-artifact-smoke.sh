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

if ARES_INSTALL_DIR="$install_root/missing" ARES_ARCHIVE="$archive_root/missing.tar.gz" sh ./install.sh >"$install_root/missing.txt" 2>&1; then
  printf 'release-artifact-smoke: missing archive unexpectedly succeeded\n' >&2
  exit 1
fi
grep -q "archive not found" "$install_root/missing.txt"

bad_archive="$archive_root/bad.tar.gz"
printf 'not-ares\n' >"$archive_root/not-ares"
tar -czf "$bad_archive" -C "$archive_root" not-ares
if ARES_INSTALL_DIR="$install_root/bad" ARES_ARCHIVE="$bad_archive" sh ./install.sh >"$install_root/bad.txt" 2>&1; then
  printf 'release-artifact-smoke: malformed archive unexpectedly succeeded\n' >&2
  exit 1
fi
grep -q "archive did not contain ares" "$install_root/bad.txt"

extra_archive="$archive_root/extra.tar.gz"
printf 'extra\n' >"$archive_root/extra"
tar -czf "$extra_archive" -C "$(dirname "$bin")" "$(basename "$bin")" -C "$archive_root" extra
if ARES_INSTALL_DIR="$install_root/extra" ARES_ARCHIVE="$extra_archive" sh ./install.sh >"$install_root/extra.txt" 2>&1; then
  printf 'release-artifact-smoke: extra archive member unexpectedly succeeded\n' >&2
  exit 1
fi
grep -q "archive contains unexpected files" "$install_root/extra.txt"

mkdir -p "$apply_root/etc/ssh"
printf 'Port 2222\n' > "$apply_root/etc/ssh/sshd_config"
ARES_ROOT="$apply_root" ARES_OS_RELEASE=tests/fixtures/os-release/ubuntu-24.04 "$installed" preflight --json >"$apply_root/preflight.json" 2>&1
go run tests/schema-check.go preflight "$apply_root/preflight.json"

ARES_ROOT="$apply_root" ARES_OS_RELEASE=tests/fixtures/os-release/ubuntu-24.04 "$installed" --dry-run --json >"$apply_root/dry-run.json" 2>&1
go run tests/schema-check.go run "$apply_root/dry-run.json"

ARES_ROOT="$apply_root" ARES_OS_RELEASE=tests/fixtures/os-release/ubuntu-24.04 "$installed" --yes >"$apply_root/apply.txt" 2>&1
test -f "$apply_root/etc/ssh/sshd_config.d/99-ares.conf"
test -f "$apply_root/var/log/ares/latest.json"
go run tests/schema-check.go report "$apply_root/var/log/ares/latest.json"

ARES_ROOT="$apply_root" "$installed" rollback last --dry-run >"$apply_root/rollback.txt" 2>&1
go run tests/schema-check.go rollback "$apply_root/var/log/ares/rollback-latest.json"

printf 'ok release-artifact-smoke\n'
