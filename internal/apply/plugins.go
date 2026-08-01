package apply

import "fmt"

const sshDropIn = `# Managed by ares.
PermitRootLogin no
PasswordAuthentication no
KbdInteractiveAuthentication no
ChallengeResponseAuthentication no
X11Forwarding no
ClientAliveInterval 300
ClientAliveCountMax 2
MaxAuthTries 3
LoginGraceTime 30
PermitEmptyPasswords no
`

const fail2banJail = `[sshd]
enabled = true
bantime = 1h
findtime = 10m
maxretry = 5
`

const unattendedAutoUpgrades = `APT::Periodic::Update-Package-Lists "1";
APT::Periodic::Unattended-Upgrade "1";
APT::Periodic::AutocleanInterval "7";
`

const dnfAutomaticConf = `[commands]
upgrade_type = security
apply_updates = yes

[emitters]
emit_via = stdio
`

const nftablesRules = `#!/usr/sbin/nft -f
# Managed by ares.
flush ruleset

table inet ares {
  chain input {
    type filter hook input priority 0; policy drop;
    ct state established,related accept
    iif lo accept
%s
    ip protocol icmp accept
    ip6 nexthdr icmpv6 accept
  }

  chain forward {
    type filter hook forward priority 0; policy drop;
  }

  chain output {
    type filter hook output priority 0; policy accept;
  }
}
`

const strictFail2banJail = `[sshd]
enabled = true
bantime = 4h
findtime = 10m
maxretry = 3
`

const sysctlBaseline = `# Managed by ares.
net.ipv4.conf.all.rp_filter=1
net.ipv4.conf.default.rp_filter=1
net.ipv4.tcp_syncookies=1
net.ipv4.conf.all.accept_redirects=0
net.ipv4.conf.default.accept_redirects=0
net.ipv6.conf.all.accept_redirects=0
net.ipv6.conf.default.accept_redirects=0
net.ipv4.conf.all.send_redirects=0
net.ipv4.conf.default.send_redirects=0
`

func (ctx *Context) applyPackageUpgrade() error {
	switch ctx.Plan.Host.PackageManager {
	case "pacman":
		return ctx.run("pacman", "-Syu", "--noconfirm")
	case "zypper":
		return ctx.run("zypper", "--non-interactive", "patch")
	case "apk":
		if err := ctx.run("apk", "update"); err != nil {
			return err
		}
		return ctx.run("apk", "upgrade")
	default:
		return fmt.Errorf("unsupported package manager %q for package upgrade", ctx.Plan.Host.PackageManager)
	}
}
