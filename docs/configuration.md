# Configuration

`ares` reads YAML config from:

```text
~/.config/ares/config.yaml
```

Loaded config is validated before planning. Unknown profiles, unknown enabled
plugin IDs, blank custom plugin names, and negative custom plugin timeouts fail
early instead of silently producing a weaker plan.

Create the default file:

```sh
ares config init
```

Overwrite an existing file:

```sh
ares config init --force
```

## Default Config

The built-in default is equivalent to:

```yaml
profile: basic
ssh:
  allow_password_lockout: false
plugins:
  enabled:
    - ssh-hardening
    - firewall-auto
    - fail2ban
    - security-updates
    - sysctl-baseline
```

`firewall-auto` resolves to `firewall-ufw`, `firewall-firewalld`, or
`firewall-nftables` based on host detection. `security-updates` resolves to
`unattended-upgrades` on apt-based distros and `dnf-automatic` on dnf/yum-based
distros.

`ssh.allow_password_lockout` persists the same explicit intent as
`--allow-password-lockout`: when true, `ares` may disable SSH password auth even
if it cannot detect an authorized key for a likely login account.

## Profiles

Set the default profile:

```yaml
profile: web
plugins:
  enabled:
    - ssh-hardening
    - firewall-auto
    - fail2ban
    - security-updates
    - sysctl-baseline
```

Profiles can also be selected per run:

```sh
ares plan --profile strict
sudo ares --profile strict --yes
```

## Tailscale SSH

`tailscale-ssh` is available as a built-in plugin when explicitly enabled:

```yaml
profile: basic
plugins:
  enabled:
    - ssh-hardening
    - firewall-auto
    - fail2ban
    - security-updates
    - sysctl-baseline
    - tailscale-ssh
tailscale:
  ssh_enabled: true
  auth_key_env: TAILSCALE_AUTHKEY
  hostname: web-01
  accept_routes: false
  extra_args: []
```

By default, the plugin installs the local `tailscale` package and enables
`tailscaled` on systemd hosts without authenticating the node. Set
`tailscale.ssh_enabled: true` and point `tailscale.auth_key_env` at an
environment variable to opt into `tailscale up --ssh`.

Do not put auth keys in config files. `ares` reads the auth key from the named
environment variable at apply time and redacts it from simulated command output
and errors. Preflight fails before host mutation when `tailscale.ssh_enabled`
is true and the configured environment variable is missing.

## Custom Plugins

Custom plugins are declared explicitly in config. `ares` does not download or
execute remote plugin code automatically.

```yaml
profile: basic
plugins:
  enabled:
    - ssh-hardening
    - firewall-auto
    - fail2ban
    - security-updates
    - sysctl-baseline
  custom:
    - name: local-hardening
      probe: command -v local-hardening
      plan: local-hardening plan
      apply: local-hardening apply
      verify: local-hardening verify
      rollback: local-hardening rollback
      timeout_seconds: 120
```

For custom plugins, `probe` runs before `apply` when declared. `apply` runs via
`sh -lc` in real apply mode, followed by `verify` when declared. Custom command
output can emit structured lines prefixed with `applied:`, `verified:`,
`skipped:`, or `failed:`; `ares` records those lines in the run report. `ares
rollback last --yes` executes custom `rollback` commands recorded in the latest
run report. `plan` is surfaced in plugin metadata and reserved for richer
external plugin orchestration. Custom plugin names must be unique and must not
reuse built-in plugin IDs or reserved selectors such as `firewall-auto`. Custom
commands must be single-line, non-blank strings. If `verify` or `rollback` is
declared, `apply` must also be declared.
`ares preflight` checks that the first executable in each declared custom
command is available without executing the command.

## Environment

| Variable | Purpose |
| --- | --- |
| `ARES_NO_BANNER` | Suppress the randomized ASCII banner for scripted runs. |
| `ARES_PROVIDER` | Override provider detection, primarily for tests. |
| `ARES_ROOT` | Redirect file writes under a test root and avoid running host commands. |
