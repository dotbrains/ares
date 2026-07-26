# Development

`ares` is a Go CLI built with Cobra. The source is intentionally split into
small packages:

- `cmd/` - CLI commands and output formatting
- `internal/system/` - host, distro, provider, SSH, and firewall detection
- `internal/config/` - YAML config loading and default config
- `internal/plugins/` - embedded plugin catalog
- `internal/plan/` - plugin selection and action planning
- `internal/apply/` - guarded apply engine, reports, backups, and verifiers
- `tests/` - smoke and container integration scripts
- `website/` - Next.js marketing site

## Build and Test

With [Flox](https://flox.dev) installed, the repo provides a complete local
toolchain for Go, Bun, golangci-lint, GoReleaser, pre-commit, actionlint, git,
and the Docker CLI:

```sh
flox activate
make ci
```

Without Flox, install the equivalent tools on your host and run:

```sh
make build
make test
make smoke
make integration
make vet
```

`make integration` runs lightweight Ubuntu/Debian container checks by default.
Use `ARES_FULL_INTEGRATION=1 make integration` to also pull and test heavier
Rocky/Fedora images.

Install the pre-commit hook after cloning:

```sh
pre-commit install
```

## Local Website

```sh
cd website
bun install
bun run build
```

## Release Notes

Releases are cut from Git tags. The release workflow runs CI, builds release
artifacts, and publishes platform archives plus `checksums.txt`.

Current release assets:

- `ares_darwin_amd64.tar.gz`
- `ares_darwin_arm64.tar.gz`
- `ares_linux_amd64.tar.gz`
- `ares_linux_arm64.tar.gz`
- `checksums.txt`
