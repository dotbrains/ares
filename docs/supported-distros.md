# Supported Distros

`ares` targets common systemd-based Linux VPS images. Host detection reads local
OS release data and service state, then the plugin catalog selects distro,
firewall, update, and provider plugins from metadata.

## First-Class Targets

- Ubuntu 22.04 and 24.04
- Debian 12
- AlmaLinux 9
- Rocky Linux 9
- Fedora Server

## Distro Selection

Distro support is modular. A first-class distro is represented by a built-in
plugin with:

- `categories = ["distro"]`
- one or more `distros = [...]` entries matching `/etc/os-release` `ID` or
  `ID_LIKE`
- package, service, and SSH capabilities declared in `capabilities`
- lifecycle handlers declared through `probe`, `plan`, `apply`, and `rollback`

Exact `ID` matches win before `ID_LIKE` family matches. For example, Ubuntu
reports `ID_LIKE=debian`, but `distro-ubuntu` is selected before the generic
Debian adapter. Family adapters can still cover compatible derivatives such as
RHEL-like systems by listing the family ID in `distros`.

Adding a new first-class distro should not require editing the planner. Add a
new distro plugin TOML file under `marketplace/plugins/builtin/`, declare the
matching distro IDs, and add tests/fixtures that prove the adapter, firewall,
and update plugins resolve correctly.

## Distro Behavior

| Family | Package manager | Firewall default | Security updates | SSH service |
| --- | --- | --- | --- | --- |
| Ubuntu | `apt-get` | `ufw` | `unattended-upgrades` | detected service, usually `ssh` |
| Debian | `apt-get` | `ufw` | `unattended-upgrades` | detected service, usually `ssh` |
| AlmaLinux/Rocky/RHEL | `dnf` or `yum` | `firewalld` | `dnf-automatic` | detected service, usually `sshd` |
| Fedora | `dnf` | `firewalld` | `dnf-automatic` | detected service, usually `sshd` |

If the host reports nftables as the active firewall backend, `firewall-auto`
can resolve to `firewall-nftables`.

Unsupported hosts produce warnings in the plan. They should not receive blind
SSH or firewall changes without a matching distro adapter in the plugin catalog.

## Later Targets

- Debian 11
- Arch Linux
- openSUSE Leap
- Alpine Linux
- Oracle Linux
- Amazon Linux

## Providers

`ares` detects common VPS providers from DMI metadata when available, or from
`ARES_PROVIDER` during tests. Provider plugins are advisory only:

- DigitalOcean
- Hostinger
- Hetzner
- Vultr
- Linode/Akamai
- OVH
- AWS Lightsail

Provider plugins are advisory. They record provider-specific recovery reminders
for cloud firewalls, snapshots, rescue consoles, and out-of-band access. They do
not configure provider accounts or call cloud APIs.
