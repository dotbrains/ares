# ares

`ares` is a modular VPS hardening runner for fresh Linux servers.

It is designed for the first SSH session on a rented VPS from providers such as
DigitalOcean, Hostinger, Hetzner, Vultr, Linode, OVH, or Lightsail. The core
runner detects the host, builds a hardening plan, preserves SSH access, and runs
small plugins for distro, SSH, firewall, updates, fail2ban, and sysctl behavior.

Apply mode is guarded by `--yes` and root privileges. Use `--dry-run` first to
inspect the exact plan.

## Quick Start

Inspect the hardening plan:

```sh
go install github.com/dotbrains/ares@latest
sudo ares --dry-run
sudo ares --yes
```

Future release bootstrap:

```sh
curl -fsSL https://raw.githubusercontent.com/dotbrains/ares/main/install.sh | sudo sh
sudo ares --dry-run
```

From source:

```sh
git clone https://github.com/dotbrains/ares.git
cd ares
make build
sudo ./ares --dry-run
sudo ./ares --yes
```

## Commands

| Command | Description |
| --- | --- |
| `ares --dry-run` | Detect the host and show the default hardening plan |
| `ares --yes` | Apply the default hardening plan after review |
| `ares plan` | Show the hardening plan |
| `ares detect` | Print detected distro, package manager, SSH, and firewall details |
| `ares status` | Summarize host support status |
| `ares plugins list` | List built-in plugins |
| `ares plugins show <id>` | Show plugin metadata |
| `ares plugins snippet <id>` | Print a config snippet for a plugin |
| `ares config init` | Write default config to `~/.config/ares/config.yaml` |

## Profiles

The default profile is `basic`.

- `basic`: SSH hardening, distro-selected firewall, fail2ban, security updates, sysctl baseline
- `web`: `basic` plus HTTP/HTTPS firewall allowances
- `strict`: `basic` plus stricter fail2ban defaults and root-lock guidance

## Plugin Model

`ares` follows a plugin model similar to `dotbrains/hab`:

- built-in plugin catalog
- explicit custom plugin config
- categories and capabilities
- probe/plan/apply/rollback lifecycle
- no automatic remote plugin execution by default

See [docs/plugins.md](docs/plugins.md).

## Supported Distros

First-class targets:

- Ubuntu 22.04 and 24.04
- Debian 12
- AlmaLinux 9
- Rocky Linux 9
- Fedora Server

See [docs/supported-distros.md](docs/supported-distros.md).

## Safety Rules

Apply mode must:

- detect active SSH sessions
- preserve the active SSH port
- back up modified files
- validate `sshd` config before reload
- generate logs and undo plans under `/var/log/ares`
- fail safely on unsupported hosts

See [docs/recovery.md](docs/recovery.md) and [docs/threat-model.md](docs/threat-model.md).

## Development

```sh
make build
make test
make vet
```

## License

PolyForm Shield License 1.0.0. See [LICENSE](LICENSE).
