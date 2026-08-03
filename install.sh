#!/bin/sh
set -eu

repo="dotbrains/ares"
bin="ares"
install_dir_was_set="${ARES_INSTALL_DIR+x}"
install_dir="${ARES_INSTALL_DIR:-/usr/local/bin}"
release_base_url="${ARES_RELEASE_BASE_URL:-}"
archive_path="${ARES_ARCHIVE:-}"

need() {
  command -v "$1" >/dev/null 2>&1 || {
    printf 'ares: missing required command: %s\n' "$1" >&2
    exit 1
  }
}

if [ "$(id -u)" -ne 0 ] && [ -z "$install_dir_was_set" ]; then
  printf 'ares: install.sh must run as root; use: curl ... | sudo sh\n' >&2
  exit 1
fi

need uname
need mktemp
need tar
need install

os="$(uname -s | tr '[:upper:]' '[:lower:]')"
arch="$(uname -m)"
case "$arch" in
  x86_64|amd64) arch="amd64" ;;
  aarch64|arm64) arch="arm64" ;;
  *) printf 'ares: unsupported architecture: %s\n' "$arch" >&2; exit 1 ;;
esac

tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT

if [ -n "$archive_path" ]; then
  if [ ! -f "$archive_path" ]; then
    printf 'ares: archive not found: %s\n' "$archive_path" >&2
    exit 1
  fi
  cp "$archive_path" "$tmp/${bin}.tar.gz"
elif [ -n "$release_base_url" ]; then
  need curl
  curl -fsSL "${release_base_url%/}/${bin}_${os}_${arch}.tar.gz" -o "$tmp/${bin}.tar.gz"
elif command -v gh >/dev/null 2>&1; then
  gh release download --repo "$repo" --pattern "${bin}_${os}_${arch}.tar.gz" --dir "$tmp"
elif command -v curl >/dev/null 2>&1; then
  url="https://github.com/${repo}/releases/latest/download/${bin}_${os}_${arch}.tar.gz"
  curl -fsSL "$url" -o "$tmp/${bin}.tar.gz"
else
  printf 'ares: install requires gh or curl\n' >&2
  exit 1
fi

archive="$(find "$tmp" -name '*.tar.gz' -print | head -n 1)"
if [ -z "$archive" ]; then
  printf 'ares: release archive was not downloaded\n' >&2
  exit 1
fi
tar -xzf "$archive" -C "$tmp"
if [ ! -f "$tmp/$bin" ]; then
  printf 'ares: archive did not contain %s\n' "$bin" >&2
  exit 1
fi
mkdir -p "$install_dir"
install -m 0755 "$tmp/$bin" "$install_dir/$bin"

printf 'ares installed to %s/%s\n' "$install_dir" "$bin"
printf 'Inspect the first hardening plan with: sudo %s --dry-run\n' "$bin"
