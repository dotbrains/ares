# Plugins

`ares` is modular by default. The core runner owns host detection, plan
selection, root and confirmation guards, report paths, backups, and final
reporting. Hardening behavior is selected through built-in or custom plugins.

The model is built around an embedded catalog, categories, capabilities,
metadata inspection commands, and copyable config snippets. Unlike a package
manager, `ares` does not automatically download or execute remote plugin code.

## Lifecycle Contract

Every plugin is expected to fit this lifecycle:

```text
detect -> probe -> plan -> confirm -> backup -> apply -> verify -> report
```

The implementation currently enforces the host-level `confirm` guard for real
apply mode: root privileges and `--yes` are required. Built-in actions are
planned before mutation, and SSH/firewall actions are marked risky in the plan.

Custom plugins can declare `probe`, `plan`, `apply`, and `rollback` commands in
config. The first release executes custom `probe` and `apply`; `plan` and
`rollback` are preserved as metadata for inspection and future richer lifecycle
support.

## Built-Ins

Inspect the live embedded catalog:

```sh
ares plugins list
ares plugins show ssh-hardening
ares plugins snippet ssh-hardening
```

Current built-ins:

| Plugin | Purpose |
| --- | --- |
| `distro-ubuntu` | Use apt/systemd/Ubuntu service assumptions. |
| `distro-debian` | Use apt/systemd/Debian service assumptions. |
| `distro-rhel` | Use dnf/firewalld/RHEL-family assumptions. |
| `distro-fedora` | Use dnf/firewalld/Fedora assumptions. |
| `ssh-hardening` | Write `/etc/ssh/sshd_config.d/99-ares.conf`, validate sshd config, and reload SSH. |
| `firewall-ufw` | Install and enable UFW with active SSH allowed. |
| `firewall-firewalld` | Install and enable firewalld with active SSH allowed. |
| `firewall-nftables` | Write `/etc/nftables.conf` with active SSH allowed. |
| `fail2ban` | Install and enable a conservative SSH jail. |
| `unattended-upgrades` | Enable apt security updates without automatic reboots. |
| `dnf-automatic` | Enable dnf security updates. |
| `sysctl-baseline` | Write conservative network hardening to `/etc/sysctl.d/99-ares.conf`. |
| `web-profile` | Allow inbound HTTP and HTTPS. |
| `strict-profile` | Apply stricter fail2ban defaults and record root-lock guidance. |
| `provider-*` | Record provider recovery reminders without mutating provider APIs. |

## Custom Plugins

Custom plugins are configured explicitly:

```yaml
plugins:
  custom:
    - name: tailscale-ssh
      probe: command -v tailscale
      plan: ares-plugin-tailscale-ssh plan
      apply: ares-plugin-tailscale-ssh apply
      rollback: ares-plugin-tailscale-ssh rollback
```

Guidelines for custom plugin commands:

- keep commands idempotent
- fail loudly on partial configuration
- preserve active SSH access when touching SSH or firewall settings
- write reversible changes or clear manual rollback steps
- avoid fetching remote scripts inside `apply`

Future remote marketplace support must require explicit confirmation plus
pinning, checksums, or signatures.

## Provider Plugins

Provider plugins are advisory. They record provider-specific recovery reminders
for cloud firewalls, snapshots, rescue consoles, and out-of-band access. They do
not mutate provider APIs or assume cloud credentials are available on the VPS.
