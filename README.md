# ares

[![CI](https://github.com/dotbrains/ares/actions/workflows/ci.yml/badge.svg)](https://github.com/dotbrains/ares/actions/workflows/ci.yml)
[![Release](https://github.com/dotbrains/ares/actions/workflows/release.yml/badge.svg)](https://github.com/dotbrains/ares/actions/workflows/release.yml)
[![License: PolyForm Shield 1.0.0](https://img.shields.io/badge/license-PolyForm%20Shield%201.0.0-blue.svg)](LICENSE)
[![Platform: Linux VPS](https://img.shields.io/badge/platform-Linux%20VPS-lightgrey.svg)](docs/supported-distros.md)
[![Go: 1.24+](https://img.shields.io/badge/go-1.24%2B-00ADD8.svg)](go.mod)
[![pre-commit](https://img.shields.io/badge/pre--commit-enabled-brightgreen?logo=pre-commit&logoColor=white)](.pre-commit-config.yaml)
[![Dev env: Flox](https://img.shields.io/badge/dev%20env-flox-7c3aed.svg)](.flox/env/manifest.toml)
[![Website: Bun](https://img.shields.io/badge/website-bun-f472b6.svg)](website/package.json)
[![Distros: Ubuntu Debian RHEL Fedora](https://img.shields.io/badge/distros-Ubuntu%20%7C%20Debian%20%7C%20RHEL%20%7C%20Fedora-64748b.svg)](docs/supported-distros.md)

`ares` is a modular hardening runner for fresh Linux VPS instances. It detects
the host, builds a reviewable plan, preserves SSH access, and applies a small
set of provider-agnostic security defaults through built-in or custom plugins.

It is designed for the first SSH session on a rented VPS from providers such as
DigitalOcean, Hostinger, Hetzner, Vultr, Linode, OVH, or AWS Lightsail. The
default profile covers SSH hardening, a distro-appropriate firewall, fail2ban,
automatic security updates, and a conservative sysctl baseline.

```sh
curl -fsSL https://raw.githubusercontent.com/dotbrains/ares/main/install.sh | sudo sh
sudo ares --dry-run
sudo ares --yes
```

Apply mode is intentionally guarded: it requires root privileges and `--yes`.
Use `--dry-run` or `ares plan` first to inspect the exact changes.

## Install

`ares` ships as a single Go binary. The bootstrap installer pulls the latest
GitHub release for the current platform:

```sh
curl -fsSL https://raw.githubusercontent.com/dotbrains/ares/main/install.sh | sudo sh
```

From Go:

```sh
go install github.com/dotbrains/ares@latest
```

From source:

```sh
git clone https://github.com/dotbrains/ares.git
cd ares
make build
sudo ./ares --dry-run
```

Release artifacts are published for Linux and macOS on `amd64` and `arm64`.

## First Run

Start by inspecting the host and the selected plan:

```sh
ares detect
sudo ares --dry-run
```

Then apply after confirming the active SSH port and provider recovery path:

```sh
sudo ares --yes
```

`ares` writes run output under `/var/log/ares`, including `latest.json` and a
manual `undo-plan.txt`.

## Commands

| Command | What it does |
| --- | --- |
| `ares --dry-run` | Detect the host, print the default plan, and skip mutation. |
| `ares --yes` | Apply the selected hardening plan after review. |
| `ares plan [--profile <name>]` | Print the selected plan without entering apply mode. |
| `ares detect` | Show OS, provider, architecture, package manager, SSH, and firewall detection. |
| `ares status` | Summarize support status, selected profile, plugins, and warnings. |
| `ares config init [--force]` | Write `~/.config/ares/config.yaml`. |
| `ares plugins list` | List embedded plugin catalog entries. |
| `ares plugins show <id>` | Show plugin metadata and lifecycle commands. |
| `ares plugins snippet <id>` | Print a copyable config snippet for a plugin. |

## Profiles

The default profile is `basic`.

- `basic` applies SSH hardening, firewall defaults, fail2ban, security updates,
  and sysctl hardening.
- `web` adds inbound `80/tcp` and `443/tcp` firewall allowances.
- `strict` uses stricter fail2ban SSH jail defaults and records root-lock
  guidance without locking root automatically.

```sh
sudo ares --profile web --dry-run
sudo ares --profile web --yes
```

## Plugin Model

Hardening behavior is selected through an embedded plugin catalog plus explicit
custom plugins in config. The catalog source is split into one TOML file per
plugin under `marketplace/plugins/<kind>/<id>.toml`, which keeps review and
ownership manageable as new distro, provider, profile, and hardening plugins
are added.

Current built-in groups:

- distro adapters for Ubuntu, Debian, RHEL-family images, and Fedora
- SSH hardening
- UFW, firewalld, and nftables firewall adapters
- fail2ban
- apt and dnf security update adapters
- sysctl baseline
- `web` and `strict` profile plugins
- provider advisory plugins for common VPS vendors

Custom plugins are configured explicitly and run local commands only. `ares`
does not automatically download or execute remote plugin code.

```yaml
plugins:
  custom:
    - name: tailscale-ssh
      probe: command -v tailscale
      plan: ares-plugin-tailscale-ssh plan
      apply: ares-plugin-tailscale-ssh apply
      rollback: ares-plugin-tailscale-ssh rollback
```

See [Plugins](docs/plugins.md) for the catalog schema and lifecycle contract.

## Safety

The main product constraint is avoiding SSH lockout. Apply mode:

- requires root privileges and `--yes`
- preserves the detected active SSH port in firewall plans
- backs up key files before replacement where applicable
- validates `sshd -t` before reloading SSH
- prefers service reloads over blind restarts
- records applied, skipped, verified, and failed steps under `/var/log/ares`
- treats provider plugins as advisory reminders, not cloud API mutators

Before hardening a remote VPS, confirm provider console or rescue access and
provider-level firewall rules.

## Supported Targets

First-class distro targets:

- Ubuntu 22.04 and 24.04
- Debian 12
- AlmaLinux 9
- Rocky Linux 9
- Fedora Server

Provider detection is advisory for DigitalOcean, Hostinger, Hetzner, Vultr,
Linode/Akamai, OVH, and AWS Lightsail.

## Development

```sh
make ci
```

`make ci` runs Markdown lint, Go tests, vet, golangci-lint, build, smoke,
GitHub Actions lint, container integration, release config checks, and the
Bun-backed website typecheck/build. The repository also includes a pre-commit
hook that runs the same target before commits.

With [Flox](https://flox.dev) installed, activate the repo toolchain:

```sh
flox activate
make ci
```

## Docs

- [Docs index](docs/README.md)
- [Getting started](docs/getting-started.md)
- [Commands](docs/commands.md)
- [Configuration](docs/configuration.md)
- [Plugins](docs/plugins.md)
- [Supported distros](docs/supported-distros.md)
- [Recovery](docs/recovery.md)
- [Threat model](docs/threat-model.md)
- [Development](docs/development.md)

## License

[PolyForm Shield License 1.0.0](LICENSE).
