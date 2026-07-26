# Supported Distros

`ares` should work across common VPS Linux distributions through distro
adapters.

## First-Class Targets

- Ubuntu 22.04 and 24.04
- Debian 12
- AlmaLinux 9
- Rocky Linux 9
- Fedora Server

Ubuntu and Debian use `apt-get`, `ufw`, and `unattended-upgrades`. AlmaLinux,
Rocky Linux, and Fedora use `dnf`, `firewalld`, and `dnf-automatic`.

## Later Targets

- Debian 11
- Arch Linux
- openSUSE Leap
- Alpine Linux
- Oracle Linux
- Amazon Linux

Unsupported hosts should fail safely or produce a plan with warnings. They
should not receive blind SSH or firewall changes.

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
