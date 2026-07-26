#!/usr/bin/env sh
set -eu

bin="${ARES_BIN:-./ares}"
fixtures="ubuntu-24.04 debian-12 debian-11 rocky-9 fedora arch opensuse-leap alpine oracle-9 amazon-2023"

if [ ! -x "$bin" ]; then
  printf 'smoke: missing executable %s; run make build first\n' "$bin" >&2
  exit 1
fi

for fixture in $fixtures; do
  root="$(mktemp -d)"
  mkdir -p "$root/etc/ssh"
  printf 'Port 2222\n' > "$root/etc/ssh/sshd_config"

  ARES_ROOT="$root" ARES_OS_RELEASE="tests/fixtures/os-release/$fixture" "$bin" --yes >"$root/output.txt" 2>&1

  test -f "$root/etc/ssh/sshd_config.d/99-ares.conf"
  test -f "$root/etc/fail2ban/jail.d/ares-sshd.conf"
  test -f "$root/etc/sysctl.d/99-ares.conf"
  test -f "$root/var/log/ares/latest.json"
  test -f "$root/var/log/ares/undo-plan.txt"
  grep -q 'ssh port: 2222' "$root/output.txt"

  case "$fixture" in
    ubuntu-*|debian-*)
      test -f "$root/etc/apt/apt.conf.d/20auto-upgrades"
      grep -q 'firewall-ufw' "$root/output.txt"
      ;;
    rocky-*|fedora|oracle-*|amazon-*)
      test -f "$root/etc/dnf/automatic.conf"
      grep -q 'firewall-firewalld' "$root/output.txt"
      ;;
    arch)
      grep -q 'firewall-nftables' "$root/output.txt"
      grep -q 'pacman-upgrade' "$root/output.txt"
      ;;
    opensuse-leap)
      grep -q 'firewall-firewalld' "$root/output.txt"
      grep -q 'zypper-patches' "$root/output.txt"
      ;;
    alpine)
      grep -q 'firewall-nftables' "$root/output.txt"
      grep -q 'apk-upgrade' "$root/output.txt"
      ;;
  esac

  printf 'ok %s\n' "$fixture"
done

provider_root="$(mktemp -d)"
mkdir -p "$provider_root/etc/ssh"
printf 'Port 22\n' > "$provider_root/etc/ssh/sshd_config"
ARES_ROOT="$provider_root" ARES_PROVIDER=digitalocean ARES_OS_RELEASE=tests/fixtures/os-release/ubuntu-24.04 "$bin" --yes >"$provider_root/output.txt" 2>&1
grep -q 'provider-digitalocean' "$provider_root/output.txt"
grep -q 'recorded provider advisory' "$provider_root/output.txt"
printf 'ok provider-digitalocean\n'

install_root="$(mktemp -d)"
archive_root="$(mktemp -d)"
tar -czf "$archive_root/ares_$(uname -s | tr '[:upper:]' '[:lower:]')_$(uname -m | sed 's/x86_64/amd64/; s/aarch64/arm64/').tar.gz" -C "$(dirname "$bin")" "$(basename "$bin")"
ARES_INSTALL_DIR="$install_root/bin" ARES_ARCHIVE="$(find "$archive_root" -name '*.tar.gz' -print | head -n 1)" sh ./install.sh >"$install_root/install.txt" 2>&1
test -x "$install_root/bin/ares"
"$install_root/bin/ares" --version >/dev/null
grep -q 'ares installed' "$install_root/install.txt"
printf 'ok install\n'
