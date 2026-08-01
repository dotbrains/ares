# ADR 0002: Avoid automatic service reload during rollback

## Status

Accepted

## Context

Rollback often happens when SSH or firewall access is already fragile. Reloading
services automatically can complete a lockout if provider access is not ready.

## Decision

`ares rollback last --yes` removes managed files and restores backups, but it
does not automatically reload SSH or firewall services on a live host.

## Consequences

- Rollback is conservative and leaves final service reload timing to the
  operator.
- Reports and undo plans must tell operators what to validate before reload.
- Recovery automation can use rollback output, but must still make an explicit
  reload decision.
