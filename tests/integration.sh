#!/usr/bin/env sh
set -eu

runtime="${ARES_CONTAINER_RUNTIME:-}"
if [ -z "$runtime" ]; then
  if command -v docker >/dev/null 2>&1; then
    runtime="docker"
  elif command -v podman >/dev/null 2>&1; then
    runtime="podman"
  else
    printf 'integration: skipping; docker or podman is required\n'
    exit 0
  fi
fi

if ! "$runtime" info >/dev/null 2>&1; then
  printf 'integration: skipping; %s daemon is not available\n' "$runtime"
  exit 0
fi

arch="$("$runtime" info --format '{{.Architecture}}' 2>/dev/null || uname -m)"
case "$arch" in
  x86_64|amd64) goarch="amd64" ;;
  aarch64|arm64) goarch="arm64" ;;
  *) printf 'integration: unsupported container architecture: %s\n' "$arch" >&2; exit 1 ;;
esac

tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT

CGO_ENABLED=0 GOOS=linux GOARCH="$goarch" go build -o "$tmp/ares" .

images="${ARES_INTEGRATION_IMAGES:-}"
if [ -z "$images" ]; then
  images="
ubuntu:24.04 ubuntu
debian:12 debian
"
  if [ "${ARES_FULL_INTEGRATION:-0}" = "1" ]; then
    images="$images
rockylinux:9 rocky
fedora:latest fedora
"
  fi
fi

printf '%s\n' "$images" | while read -r image id; do
  [ -n "$image" ] || continue
  root="/tmp/ares-root"
  case "$id" in
    ubuntu|debian) expect_plugin="firewall-ufw" ;;
    *) expect_plugin="firewall-firewalld" ;;
  esac
  "$runtime" run --rm \
    -e ARES_ROOT="$root" \
    -e ARES_OS_RELEASE=/etc/os-release \
    -e ARES_EXPECT_PLUGIN="$expect_plugin" \
    -v "$tmp/ares:/usr/local/bin/ares:ro" \
    "$image" \
    sh -ceu '
      mkdir -p "$ARES_ROOT/etc/ssh"
      printf "Port 2222\n" > "$ARES_ROOT/etc/ssh/sshd_config"
      ares --yes > /tmp/ares-output.txt 2>&1
      grep -q "ssh port: 2222" /tmp/ares-output.txt
      grep -q "$ARES_EXPECT_PLUGIN" /tmp/ares-output.txt
      test -f "$ARES_ROOT/etc/ssh/sshd_config.d/99-ares.conf"
      test -f "$ARES_ROOT/etc/fail2ban/jail.d/ares-sshd.conf"
      test -f "$ARES_ROOT/etc/sysctl.d/99-ares.conf"
      test -f "$ARES_ROOT/var/log/ares/latest.json"
    '
  printf 'ok container %s\n' "$image"
done
