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
      rollback: ares-plugin-tailscale-ssh rollback
```

For custom plugins, `probe` runs before `apply` when declared. `apply` runs via
`sh -lc` in real apply mode. `plan` and `rollback` are part of the extension
contract and are surfaced in plugin metadata, but the first release's apply
engine only executes custom `probe` and `apply`.

## Environment

| Variable | Purpose |
| --- | --- |
| `ARES_PROVIDER` | Override provider detection, primarily for tests. |
| `ARES_ROOT` | Redirect file writes under a test root and avoid running host commands. |
