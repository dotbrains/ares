# Threat Model

`ares` targets fresh VPS instances exposed to the public internet.

Primary risks:

- password SSH brute force
- root SSH login
- open unnecessary inbound ports
- stale security packages
- missing intrusion throttling
- unsafe manual hardening that locks out the operator

## Controls

The default `basic` profile applies:

- `PermitRootLogin no`
- `PasswordAuthentication no`
- keyboard-interactive password auth disabled
- SSH keepalive and login-attempt limits
- active SSH port preservation in host firewall rules
- fail2ban SSH jail
- distro-native automatic security updates
- conservative network sysctl settings

The `web` profile opens inbound HTTP and HTTPS. The `strict` profile lowers
fail2ban retries and increases ban duration, then records root-lock guidance
instead of locking root automatically.

Non-goals for the first release:

- full compliance hardening
- workstation hardening
- Kubernetes cluster hardening
- malware cleanup on already-compromised hosts
- aggressive package removal

- provider account hardening
- provider firewall mutation through cloud APIs
- user or team SSH key management
- CIS or DISA STIG compliance
