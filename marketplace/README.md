# Plugin Marketplace

Marketplace source lives under `marketplace/plugins/**/*.toml`.

Each TOML file declares one plugin, and the filename must match the plugin id:

```text
marketplace/plugins/builtin/ssh-hardening.toml
```

```toml
id = "ssh-hardening"
name = "SSH hardening"
kind = "builtin"
summary = "Writes a managed sshd drop-in after preserving the active SSH port"
categories = ["ssh", "hardening"]
capabilities = ["ssh-hardening"]
probe = "test -d /etc/ssh"
plan = "builtin:ssh-hardening:plan"
apply = "builtin:ssh-hardening:apply"
rollback = "builtin:ssh-hardening:rollback"
config = """
[plugins]
enabled = ["ssh-hardening"]
"""
```

Use `kind = "builtin"` for hardening behavior implemented by the `ares` apply
engine. Future `included` or `custom` examples can live in sibling directories
when they become part of the CLI contract.

Guidelines:

- keep plugin ids stable and lowercase
- prefer one narrowly scoped capability per plugin
- keep provider plugins advisory unless cloud API support is explicit
- preserve active SSH access for anything touching SSH or firewall behavior
- include a copyable config snippet when a plugin can be enabled directly
