# Recovery

`ares` hardening must be designed around avoiding lockout.

Before applying SSH or firewall changes, the runner should:

- detect whether it is running over SSH
- detect the active SSH port
- keep the active SSH port open in the firewall
- back up modified files with timestamps
- validate `sshd` config before reload
- prefer service reloads over restarts
- write an undo plan under `/var/log/ares`

Apply mode writes a report and undo plan under `/var/log/ares`. The undo plan is
manual by design: SSH and firewall recovery steps should be reviewed before
being run on a remote VPS.
