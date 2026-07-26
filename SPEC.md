# ares Specification

## Purpose

`ares` hardens fresh Linux VPS instances quickly, repeatably, and safely. It is
provider-agnostic and distro-extensible through plugins.

## Execution Model

The runner lifecycle is:

```text
detect -> probe -> plan -> confirm -> backup -> apply -> verify -> report
```

The current implementation supports detection and planning. Mutating apply mode
must remain disabled until backup, verification, and undo-plan behavior exists.

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
    - intrusion-protection
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
- `firewall-auto`
- `intrusion-protection`
- `security-updates`
- `sysctl-baseline`

## Default Plan

The default `basic` plan should:

1. Select the appropriate distro adapter.
2. Back up SSH configuration.
3. Preserve the active SSH port.
4. Plan SSH hardening through a managed drop-in.
5. Select a firewall backend and keep SSH reachable.
6. Plan fail2ban installation.
7. Plan automatic security updates.
8. Plan conservative sysctl hardening.

## Apply-Mode Requirements

Apply mode cannot ship until these are implemented and tested:

- timestamped backups
- `/var/log/ares` report output
- undo plan generation
- SSH config validation
- firewall SSH-port preservation
- distro-specific package/service execution
- dry-run proving no mutation
- integration tests on supported distros

## Architecture

```text
cmd/                  Cobra commands
internal/config/      YAML config defaults and persistence
internal/system/      host detection
internal/plugins/     built-in plugin catalog
internal/plan/        plan generation
docs/                 architecture, recovery, distro, threat-model docs
examples/             sample config
```
