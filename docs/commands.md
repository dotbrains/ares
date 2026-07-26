# Commands

`ares` is a Cobra CLI. Commands read `~/.config/ares/config.yaml` when present
and otherwise use the built-in default config.

## Root Command

```sh
ares [--profile basic|web|strict] [--dry-run] [--yes]
```

The root command detects the host, prints a hardening plan, and then runs the
apply engine. `--dry-run` exits after writing dry-run report output. Real apply
mode requires root privileges and `--yes`.

## Planning and Inspection

| Command | Description |
| --- | --- |
| `ares plan [--profile <profile>]` | Detect the host and print the selected plan without applying changes. |
| `ares detect` | Print detected OS, provider, architecture, package manager, init system, SSH service, SSH port, firewall backend, and SSH-session status. |
| `ares status` | Print host detection plus selected profile, plugin count, and warning count. |
| `ares --version` | Print the binary version. |

## Configuration

| Command | Description |
| --- | --- |
| `ares config init` | Write the default config to `~/.config/ares/config.yaml`. |
| `ares config init --force` | Overwrite the config file with defaults. |

## Plugins

| Command | Description |
| --- | --- |
| `ares plugins list` | List embedded built-in plugins. |
| `ares plugins show <id>` | Show plugin metadata, lifecycle commands, categories, requirements, capabilities, and distro hints. |
| `ares plugins snippet <id>` | Print a YAML config snippet when the plugin provides one. |

## Rollback

| Command | Description |
| --- | --- |
| `ares rollback last --yes` | Remove `ares` managed files and restore the newest available `*.ares.*.bak` backups. |

Examples:

```sh
ares plugins list
ares plugins show ssh-hardening
ares plugins snippet strict-profile
```

## Test Roots

The apply engine supports `ARES_ROOT` for tests and fixtures. When set, file
writes are redirected under that root and shell commands are recorded as
`would run` messages instead of being executed.

```sh
ARES_ROOT=/tmp/ares-root ares --yes
```

`ARES_ROOT` is for local test harnesses, not normal VPS hardening.
