# Provider Recovery Notes

Provider plugins are advisory. They remind operators to verify console access,
provider firewalls, snapshots, and rescue paths before relying on host-only
hardening.

```mermaid
flowchart TD
  plan[Review ares plan] --> console[Confirm console or rescue access]
  console --> firewall[Confirm provider firewall allows SSH]
  firewall --> snapshot[Create snapshot when practical]
  snapshot --> apply[Run sudo ares --yes]
  apply --> report[Review /var/log/ares/latest.json]
```

- [DigitalOcean](digitalocean.md)
- [Hetzner](hetzner.md)
- [Hostinger](hostinger.md)
- [AWS Lightsail](lightsail.md)
- [Linode/Akamai](linode.md)
- [OVH](ovh.md)
- [Vultr](vultr.md)
