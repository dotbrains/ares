# Threat Model

`ares` targets fresh VPS instances exposed to the public internet.

Primary risks:

- password SSH brute force
- root SSH login
- open unnecessary inbound ports
- stale security packages
- missing intrusion throttling
- unsafe manual hardening that locks out the operator

Non-goals for the first release:

- full compliance hardening
- workstation hardening
- Kubernetes cluster hardening
- malware cleanup on already-compromised hosts
- aggressive package removal
