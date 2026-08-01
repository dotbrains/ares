# ADR 0003: Keep custom plugin execution explicit

## Status

Accepted

## Context

Custom plugins can execute shell commands. That is useful for local extensions
but risky if remote or implicit execution is introduced.

## Decision

Custom plugins are configured explicitly by the operator. `ares` validates
custom plugin names and command shape, preflight checks command executables, and
executes only configured local commands.

## Consequences

- No automatic remote plugin execution is allowed.
- Custom command validation should fail early on ambiguous or unsafe input.
- Remote marketplace support would require a separate trust model with pinning,
  checksums, signatures, and explicit confirmation.
