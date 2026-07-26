# Configuration

`ares` reads YAML config from:

```text
~/.config/ares/config.yaml
```

Create the default file:

```sh
ares config init
```

Overwrite an existing file:

```sh
ares config init --force
```

## Default Config

The built-in default is equivalent to:

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

`firewall-auto` resolves to `firewall-ufw`, `firewall-firewalld`, or
`firewall-nftables` based on host detection. `security-updates` resolves to
`unattended-upgrades` on apt-based distros and `dnf-automatic` on dnf/yum-based
distros.

## Profiles

Set the default profile:

```yaml
profile: web
plugins:
  enabled:
    - ssh-hardening
    - firewall-auto
    - fail2ban
    - security-updates
    - sysctl-baseline
```

Profiles can also be selected per run:

```sh
ares plan --profile strict
sudo ares --profile strict --yes
```

## Custom Plugins

Custom plugins are declared explicitly in config. `ares` does not download or
execute remote plugin code automatically.

```yaml
profile: basic
plugins:
  enabled:
    - ssh-hardening
    - firewall-auto
    - fail2ban
    - security-updates
    - sysctl-baseline
  custom:
    - name: tailscale-ssh
      probe: command -v tailscale
      plan: ares-plugin-tailscale-ssh plan
      apply: ares-plugin-tailscale-ssh apply
      verify: ares-plugin-tailscale-ssh verify
      rollback: ares-plugin-tailscale-ssh rollback
      timeout_seconds: 120
```

For custom plugins, `probe` runs before `apply` when declared. `apply` runs via
`sh -lc` in real apply mode, followed by `verify` when declared. Custom command
output can emit structured lines prefixed with `applied:`, `verified:`,
`skipped:`, or `failed:`; `ares` records those lines in the run report. `ares
rollback last --yes` executes custom `rollback` commands recorded in the latest
run report. `plan` is surfaced in plugin metadata and reserved for richer
external plugin orchestration.

## Environment

| Variable | Purpose |
| --- | --- |
| `ARES_NO_BANNER` | Suppress the randomized ASCII banner for scripted runs. |
| `ARES_PROVIDER` | Override provider detection, primarily for tests. |
| `ARES_ROOT` | Redirect file writes under a test root and avoid running host commands. |
