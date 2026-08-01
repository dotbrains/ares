# ADR 0001: Keep provider plugins advisory

## Status

Accepted

## Context

`ares` hardens VPS hosts, but cloud provider firewalls, rescue consoles,
snapshots, and access policies vary by provider and account.

## Decision

Provider plugins remain advisory. They record recovery and firewall-console
reminders in plans and reports, but they do not call provider APIs or mutate
provider-level settings.

## Consequences

- Host hardening remains provider-agnostic and safe for `curl | sudo sh`.
- Operators must still verify provider firewalls and recovery consoles.
- Future provider mutation must require explicit opt-in, credentials, and a
  separate safety review.
