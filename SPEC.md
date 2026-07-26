# ares Specification

## Purpose

`ares` hardens fresh Linux VPS instances quickly, repeatably, and safely. It is
provider-agnostic and distro-extensible through plugins.

## Execution Model

The runner lifecycle is:

```text
detect -> probe -> plan -> confirm -> backup -> apply -> verify -> report
```

The implementation supports detection, planning, guarded apply mode, backups,
report output, and undo-plan generation for the Ubuntu/Debian MVP plugins.

## Configuration

Config path:

```text
~/.config/ares/config.yaml
```

Default config:

```yaml
profile: basic
plugins:
  enabled:
    - ssh-hardening
    - firewall-auto
    - fail2ban
    - security-updates
    - sysctl-baseline
```

Custom plugin example:

```yaml
plugins:
  custom:
    - name: tailscale-ssh
      probe: command -v tailscale
      plan: ares-plugin-tailscale-ssh plan
      apply: ares-plugin-tailscale-ssh apply
      rollback: ares-plugin-tailscale-ssh rollback
```

## Built-In Plugins

- `distro-ubuntu`
- `distro-debian`
- `distro-rhel`
- `distro-fedora`
- `ssh-hardening`
- `firewall-ufw`
- `firewall-firewalld`
- `firewall-nftables`
- `fail2ban`
- `unattended-upgrades`
- `dnf-automatic`
- `sysctl-baseline`
- `web-profile`

## Default Plan

The default `basic` plan should:

1. Select the appropriate distro adapter.
2. Back up SSH configuration.
3. Preserve the active SSH port.
4. Plan SSH hardening through a managed drop-in.
5. Select a distro-appropriate firewall backend and keep SSH reachable.
6. Plan fail2ban installation.
7. Plan automatic security updates.
8. Plan conservative sysctl hardening.

## Apply-Mode Requirements

Apply mode is restricted to root execution unless `ARES_ROOT` is set for tests.
It requires `--yes` and should be reviewed through `--dry-run` first.

Implemented:

- timestamped backups
- `/var/log/ares` report output
- undo plan generation
- SSH config validation
- firewall SSH-port preservation
- distro-specific package/service execution
- probe and verify lifecycle result reporting
- dry-run proving no mutation
- fixture-backed smoke tests for supported distro planning/apply paths
- container-backed integration tests for Ubuntu/Debian detection/apply paths
- opt-in full container integration for Rocky/Fedora images

Still required before a public production release:

- live integration tests on supported VPS distros

## Architecture

```text
cmd/                  Cobra commands
internal/config/      YAML config defaults and persistence
internal/system/      host detection
internal/plugins/     built-in plugin catalog
internal/plan/        plan generation
internal/apply/       guarded apply engine, reports, undo plans
docs/                 architecture, recovery, distro, threat-model docs
examples/             sample config
```
