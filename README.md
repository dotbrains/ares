# ares

[![CI](https://github.com/dotbrains/ares/actions/workflows/ci.yml/badge.svg)](https://github.com/dotbrains/ares/actions/workflows/ci.yml)
[![Release](https://github.com/dotbrains/ares/actions/workflows/release.yml/badge.svg)](https://github.com/dotbrains/ares/actions/workflows/release.yml)
[![License: PolyForm Shield 1.0.0](https://img.shields.io/badge/license-PolyForm%20Shield%201.0.0-blue.svg)](LICENSE)
[![Platform: Linux VPS](https://img.shields.io/badge/platform-Linux%20VPS-lightgrey.svg)](docs/supported-distros.md)
[![Go: 1.24+](https://img.shields.io/badge/go-1.24%2B-00ADD8.svg)](go.mod)
[![pre-commit](https://img.shields.io/badge/pre--commit-enabled-brightgreen?logo=pre-commit&logoColor=white)](.pre-commit-config.yaml)
[![Website: Bun](https://img.shields.io/badge/website-bun-f472b6.svg)](website/package.json)
[![Distros: Ubuntu Debian RHEL Fedora](https://img.shields.io/badge/distros-Ubuntu%20%7C%20Debian%20%7C%20RHEL%20%7C%20Fedora-64748b.svg)](docs/supported-distros.md)

`ares` is a modular hardening runner for fresh Linux VPS instances. It detects
the host, builds a reviewable plan, preserves SSH access, and applies a small
set of provider-agnostic security defaults through built-in or custom plugins.

```sh
curl -fsSL https://raw.githubusercontent.com/dotbrains/ares/main/install.sh | sudo sh
sudo ares --dry-run
sudo ares --yes
```

Apply mode is intentionally guarded: it requires root privileges and `--yes`.
Use `--dry-run` or `ares plan` first to inspect the exact changes.

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
