# ADR 0004: Require explicit SSH password lockout intent

## Status

Accepted

## Context

SSH hardening disables password authentication. On an active SSH session, doing
that without a usable authorized key can lock out the operator.

## Decision

During real apply over SSH, `ares` refuses SSH hardening unless it detects an
authorized key for a likely login account or the operator explicitly opts in
with `--allow-password-lockout` or `ssh.allow_password_lockout`.

## Consequences

- The default path favors preserving access over stricter hardening.
- The bypass is intentionally visible in CLI/config and run reports.
- Future SSH hardening changes must preserve this explicit-intent model.
