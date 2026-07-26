# Plugins

`ares` is modular by default. The core runner owns detection, planning, safety
checks, logging, backup orchestration, confirmation, and reporting. Hardening
behavior lives in built-in or custom plugins.

The model follows the same broad shape as `dotbrains/hab`: a catalog of built-in
plugins, explicit custom plugins, probe commands, categories, capabilities, and
small snippets users can copy into config.

## Lifecycle

Every plugin is expected to fit this lifecycle:

```text
detect -> probe -> plan -> confirm -> backup -> apply -> verify -> report
```

The important rule is that `plan` must explain intended changes before `apply`
can mutate the host. SSH and firewall plugins must preserve the active SSH port
and validate their config before reloading services.

## Built-Ins

Initial built-ins:

- `distro-ubuntu`
- `distro-debian`
- `distro-rhel`
- `distro-fedora`
- `ssh-hardening`
- `firewall-auto`
- `intrusion-protection`
- `security-updates`
- `sysctl-baseline`

Inspect them with:

```sh
ares plugins list
ares plugins show ssh-hardening
```

## Custom Plugins

Custom plugins are configured explicitly:

```yaml
plugins:
  custom:
    - name: tailscale-ssh
      probe: command -v tailscale
      plan: ares-plugin-tailscale-ssh plan
      apply: ares-plugin-tailscale-ssh apply
      rollback: ares-plugin-tailscale-ssh rollback
```

Remote plugin execution should not be automatic. Future remote marketplace
support must require explicit confirmation plus pinning, checksums, or
signatures.
