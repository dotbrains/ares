# CLI Template — AI Agent Instructions

This is a template repository for creating CLI tools under the `dotbrains` org. It supports Go, Python, and Rust via the `generate.sh` scaffold script.

## Choosing a Language

See [LANGUAGES.md](LANGUAGES.md) for the decision guide. The agent should read the project requirements and select the best language based on performance needs, distribution constraints, and ecosystem fit.

## How to Use This Template

### Option A: AI Agent (recommended)

Tell an agent: "Create a CLI project called `mytool` that does X." The agent reads `LANGUAGES.md`, picks the best language, and runs `generate.sh`.

### Option B: Manual

```sh
cd cli-template
./generate.sh --lang go --name mytool --desc "My awesome tool" --desc-long "A longer description of what this tool does."
```

### Step 1: Generate

```sh
./generate.sh --lang <go|python|rust> --name <project-name> --desc "<one-line desc>" --desc-long "<longer desc>"
```

This creates `output/<project-name>/` with all files scaffolded.

### Step 2: Initialize

```sh
cd output/<project-name>
git init && git add -A && git commit -m "chore: scaffold from cli-template"
```

### Step 3: Initialize Go module

```sh
rm go.mod go.sum
go mod init github.com/dotbrains/<project-name>
go mod tidy
```

### Step 3: Verify

```sh
make build
make test
```

## What's Included

### Shared (all languages)

| File/Dir | Purpose |
|---|---|
| `LICENSE` | PolyForm Shield 1.0.0 license |
| `assets/` | Place `og-image.svg` here |
| `website/` | Next.js + Tailwind marketing site |
| `README.md` | OG image, badges, Quick Start, Installation, Commands table |
| `SPEC.md` | Detailed specification: Problem, Configuration, Commands, Architecture, Testing, Release |
| `TEMPLATE.md` | This file — AI agent instructions for using the template |

### Go-specific

| File/Dir | Purpose |
|---|---|
| `main.go` | Entry point with version injection via ldflags |
| `cmd/root.go` | Cobra root command using `newRootCmd(version)` factory pattern |
| `cmd/cmd_test.go` | Baseline tests for Execute, version, subcommands, config init |
| `internal/config/` | YAML config management (Load, Save, DefaultConfig, ConfigDir, ConfigPath) |
| `internal/exec/` | `CommandExecutor` interface + `RealExecutor` for testable shell-outs |
| `go.mod` | Go module definition |
| `Makefile` | Standard targets: build, test, lint, install, clean, vet, cover |
| `.goreleaser.yaml` | GoReleaser config: darwin+linux, amd64+arm64, Homebrew cask tap |
| `.golangci.yml` | golangci-lint v2 config with standard linters |
| `.github/workflows/ci.yml` | CI: test (ubuntu+macos matrix), lint, build |
| `.github/workflows/release.yml` | Release: test+lint+build → GoReleaser on tag push |

### Customizing the website

The website uses generic `accent-primary`, `accent-secondary`, `accent-tertiary` color names in `tailwind.config.js`. Rename these to match your project branding.

Files to customize:
- `website/tailwind.config.js` — accent color names and hex values
- `website/src/styles/globals.css` — `text-gradient` utility references accent colors
- `website/app/layout.tsx` — metadata (title, description, OG tags)
- `website/src/components/sections/*.tsx` — all section content
- `website/public/favicon.svg` and `website/public/og-image.svg` — replace placeholders
- `website/package.json` — update `name`, `description`, dev port

### Creating the OG image

Every project needs an `og-image.svg` (1200×630px) for social sharing and the README banner.

**Place the file in two locations:**
- `assets/og-image.svg` — used by `README.md` (`![name](./assets/og-image.svg)`)
- `website/public/og-image.svg` — served by the Next.js site

## Conventions to Follow

### Code patterns

- **One subcommand per file** in `cmd/` (e.g. `cmd/review.go`, `cmd/history.go`).
- **Factory functions** for commands: `newXxxCmd() *cobra.Command`.
- **`_test.go` files** alongside every source file.
- **`internal/` for all domain logic** — `cmd/` should be thin wrappers.
- **Use `exec.CommandExecutor` interface** for anything that shells out.
- **Use `config.Load()`** to read config — always fall back to `DefaultConfig()`.
- **Hide completion command**: `CompletionOptions: cobra.CompletionOptions{HiddenDefaultCmd: true}`.

### Testing patterns

- Use `t.TempDir()` for temporary directories.
- Use `t.Setenv("HOME", tmp)` to isolate config paths.
- Test via `newRootCmd()` + `root.SetArgs()` + `root.Execute()` for integration-style tests.
- Use `bytes.Buffer` with `root.SetOut(buf)` to capture output.
- Use `MockExecutor` for testing code that shells out.
- Target 80%+ test coverage.

### Documentation patterns

- **README.md** sections in order: title + badges, description, Quick Start, How It Works, Installation (4 methods), Configuration, Commands table, Dependencies, License.
- **SPEC.md** is the detailed spec — everything a developer needs to understand and extend the tool.
- **Badges**: CI, Release, License, then tech stack.

### Release patterns

- Version injected via ldflags: `-X main.version={{.Version}}`.
- GoReleaser builds for darwin+linux × amd64+arm64.
- Homebrew cask published to `dotbrains/homebrew-tap`.
- macOS quarantine removed via post-install hook.
- Changelog excludes `docs:` and `chore:` commits.

## Checklist for New Projects

- [ ] Replaced all `ares` placeholders
- [ ] Replaced all `Modular VPS hardening runner` placeholders
- [ ] Replaced all `ares hardens fresh Linux VPS instances with a safe, modular plugin-based execution model. It detects the host distro, plans changes, preserves SSH access, and applies provider-agnostic security defaults.` placeholders
- [ ] Created `assets/og-image.svg`
- [ ] Initialized `go.mod` with correct module path
- [ ] Added project-specific config fields to `internal/config/config.go`
- [ ] Added project-specific subcommands to `cmd/`
- [ ] Added domain logic packages under `internal/`

- [ ] Tests pass with `make test`
- [ ] Lint passes with `make lint`
- [ ] Updated README.md with real Quick Start examples
- [ ] Updated SPEC.md with real config format and command docs
- [ ] Updated Commands table in README.md
- [ ] Added tech stack badges to README.md
- [ ] Created GitHub repo and pushed
- [ ] Removed this `TEMPLATE.md` file
