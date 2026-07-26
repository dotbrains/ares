# Agent Instructions for ares

`ares` is a VPS hardening CLI. Treat safety as the primary product feature.

## Rules

- Prefer dry-run, report, and undo-plan behavior before adding host mutation.
- Never add SSH or firewall changes that can silently lock out the active user.
- Keep distro-specific behavior behind adapters or plugins.
- Keep custom plugin execution explicit; do not add automatic remote plugin execution.
- Preserve the `curl | sudo sh` install path through release binaries.

## Verification

Before handing off changes, run:

```sh
make build
make test
make smoke
make integration
```

For local apply smoke tests, use `ARES_ROOT` so no host files are modified:

```sh
tmp="$(mktemp -d)"
mkdir -p "$tmp/etc/ssh"
printf 'Port 22\n' > "$tmp/etc/ssh/sshd_config"
ARES_ROOT="$tmp" ARES_OS_RELEASE=tests/fixtures/os-release/ubuntu-24.04 ./ares --yes
```

For heavier container coverage, opt in explicitly:

```sh
ARES_FULL_INTEGRATION=1 make integration
```
