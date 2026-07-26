# Supported Distros

`ares` targets common systemd-based Linux VPS images. Host detection reads the
local OS release data and service state, then selects distro, firewall, update,
and provider plugins.

## First-Class Targets

- Ubuntu 22.04 and 24.04
- Debian 12
- AlmaLinux 9
- Rocky Linux 9
- Fedora Server

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
SSH or firewall changes without a distro adapter.

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
