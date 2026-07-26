# Recovery

`ares` hardening must be designed around avoiding lockout.

Before applying SSH or firewall changes, review:

- `ares detect` reports the expected SSH port
- your provider firewall allows that SSH port
- you have provider console, rescue mode, snapshot, or another out-of-band path
- `ares plan` shows only the plugins and profile you expect

## Built-In Safety

The first release implements these safety behaviors:

- real apply mode requires root privileges
- real apply mode requires `--yes`
- dry runs skip mutation
- SSH hardening backs up `/etc/ssh/sshd_config`
- SSH hardening writes a drop-in at `/etc/ssh/sshd_config.d/99-ares.conf`
- SSH hardening validates `sshd -t` before reloading the detected SSH service
- firewall plans preserve the detected active SSH port
- nftables and dnf automatic configs are backed up before replacement when
  existing files are present
- run reports are written under `/var/log/ares`

## Reports

Each run prepares:

```text
/var/log/ares/ares-<timestamp>.log
/var/log/ares/latest.json
/var/log/ares/undo-plan.txt
```

`latest.json` records applied, skipped, verified, probed, and failed steps.
`undo-plan.txt` records recovery guidance.

`ares rollback last --yes` performs the conservative automated subset of that
guidance: it removes `ares` managed files and restores the newest matching
`*.ares.*.bak` backup for files that `ares` backed up. It does not blindly reload
SSH or firewall services on a live host; review provider console access first.

## Recovery Pattern

If you lose access after applying changes:

1. use the provider console or rescue mode
2. inspect `/var/log/ares/latest.json`
3. inspect `/var/log/ares/undo-plan.txt`
4. restore backed-up files with `.ares.<timestamp>.bak` suffixes where
   appropriate
5. validate SSH config with `sshd -t`
6. reload the SSH service instead of rebooting when possible
