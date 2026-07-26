package apply

import (
	"fmt"
	"os"
	"path/filepath"
)

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
    tcp dport %s accept
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

func (ctx *Context) applySSHHardening() error {
	if err := ctx.backup("/etc/ssh/sshd_config"); err != nil {
		return fmt.Errorf("backup sshd_config: %w", err)
	}
	dir := ctx.path("/etc/ssh/sshd_config.d")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	path := filepath.Join(dir, "99-ares.conf")
	if err := os.WriteFile(path, []byte(sshDropIn), 0o644); err != nil {
		return err
	}
	ctx.Result.Applied = append(ctx.Result.Applied, "wrote /etc/ssh/sshd_config.d/99-ares.conf")

	if ctx.Options.Root == "" {
		if err := ctx.run("sshd", "-t"); err != nil {
			return err
		}
		if err := ctx.run("systemctl", "reload", ctx.Plan.Host.SSHService); err != nil {
			return err
		}
	}
	return nil
}

func (ctx *Context) applyUFW() error {
	if err := ctx.run(ctx.Plan.Host.PackageManager, "update"); err != nil {
		return err
	}
	if err := ctx.installPackages("ufw"); err != nil {
		return err
	}
	if err := ctx.run("ufw", "allow", ctx.Plan.Host.SSHPort+"/tcp"); err != nil {
		return err
	}
	if err := ctx.run("ufw", "default", "deny", "incoming"); err != nil {
		return err
	}
	if err := ctx.run("ufw", "default", "allow", "outgoing"); err != nil {
		return err
	}
	if err := ctx.run("ufw", "--force", "enable"); err != nil {
		return err
	}
	ctx.Result.Applied = append(ctx.Result.Applied, "enabled ufw with SSH port "+ctx.Plan.Host.SSHPort+" allowed")
	return nil
}

func (ctx *Context) applyWebProfile() error {
	if ctx.Plan.Host.FirewallBackend == "firewalld" {
		for _, service := range []string{"http", "https"} {
			if err := ctx.run("firewall-cmd", "--permanent", "--add-service="+service); err != nil {
				return err
			}
		}
		if err := ctx.run("firewall-cmd", "--reload"); err != nil {
			return err
		}
		ctx.Result.Applied = append(ctx.Result.Applied, "allowed HTTP and HTTPS")
		return nil
	}
	for _, port := range []string{"80/tcp", "443/tcp"} {
		if err := ctx.run("ufw", "allow", port); err != nil {
			return err
		}
	}
	ctx.Result.Applied = append(ctx.Result.Applied, "allowed HTTP and HTTPS")
	return nil
}

func (ctx *Context) applyFirewalld() error {
	if err := ctx.installPackages("firewalld"); err != nil {
		return err
	}
	if err := ctx.run("systemctl", "enable", "--now", "firewalld"); err != nil {
		return err
	}
	if err := ctx.run("firewall-cmd", "--permanent", "--add-port="+ctx.Plan.Host.SSHPort+"/tcp"); err != nil {
		return err
	}
	if err := ctx.run("firewall-cmd", "--set-default-zone=public"); err != nil {
		return err
	}
	if err := ctx.run("firewall-cmd", "--reload"); err != nil {
		return err
	}
	ctx.Result.Applied = append(ctx.Result.Applied, "enabled firewalld with SSH port "+ctx.Plan.Host.SSHPort+" allowed")
	return nil
}

func (ctx *Context) applyNftables() error {
	if err := ctx.installPackages("nftables"); err != nil {
		return err
	}
	if err := ctx.backup("/etc/nftables.conf"); err != nil {
		return fmt.Errorf("backup nftables.conf: %w", err)
	}
	if err := os.WriteFile(ctx.path("/etc/nftables.conf"), []byte(fmt.Sprintf(nftablesRules, ctx.Plan.Host.SSHPort)), 0o644); err != nil {
		return err
	}
	if err := ctx.run("systemctl", "enable", "--now", "nftables"); err != nil {
		return err
	}
	ctx.Result.Applied = append(ctx.Result.Applied, "enabled nftables with SSH port "+ctx.Plan.Host.SSHPort+" allowed")
	return nil
}

func (ctx *Context) applyFail2ban() error {
	if err := ctx.installPackages("fail2ban"); err != nil {
		return err
	}
	dir := ctx.path("/etc/fail2ban/jail.d")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(dir, "ares-sshd.conf"), []byte(fail2banJail), 0o644); err != nil {
		return err
	}
	ctx.Result.Applied = append(ctx.Result.Applied, "wrote /etc/fail2ban/jail.d/ares-sshd.conf")
	if ctx.Options.Root == "" {
		if err := ctx.run("systemctl", "enable", "--now", "fail2ban"); err != nil {
			return err
		}
	}
	return nil
}

func (ctx *Context) applyUnattendedUpgrades() error {
	if err := ctx.installPackages("unattended-upgrades"); err != nil {
		return err
	}
	dir := ctx.path("/etc/apt/apt.conf.d")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(dir, "20auto-upgrades"), []byte(unattendedAutoUpgrades), 0o644); err != nil {
		return err
	}
	ctx.Result.Applied = append(ctx.Result.Applied, "enabled unattended-upgrades")
	return nil
}

func (ctx *Context) applyDNFAutomatic() error {
	if err := ctx.installPackages("dnf-automatic"); err != nil {
		return err
	}
	dir := ctx.path("/etc/dnf")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	if err := ctx.backup("/etc/dnf/automatic.conf"); err != nil {
		return fmt.Errorf("backup dnf automatic.conf: %w", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "automatic.conf"), []byte(dnfAutomaticConf), 0o644); err != nil {
		return err
	}
	if err := ctx.run("systemctl", "enable", "--now", "dnf-automatic.timer"); err != nil {
		return err
	}
	ctx.Result.Applied = append(ctx.Result.Applied, "enabled dnf-automatic security updates")
	return nil
}

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

func (ctx *Context) applySysctlBaseline() error {
	dir := ctx.path("/etc/sysctl.d")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(dir, "99-ares.conf"), []byte(sysctlBaseline), 0o644); err != nil {
		return err
	}
	ctx.Result.Applied = append(ctx.Result.Applied, "wrote /etc/sysctl.d/99-ares.conf")
	if ctx.Options.Root == "" {
		return ctx.run("sysctl", "--system")
	}
	return nil
}

func (ctx *Context) applyStrictProfile() error {
	dir := ctx.path("/etc/fail2ban/jail.d")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(dir, "ares-sshd.conf"), []byte(strictFail2banJail), 0o644); err != nil {
		return err
	}
	ctx.Result.Applied = append(ctx.Result.Applied, "applied strict fail2ban SSH jail defaults")
	ctx.Result.Skipped = append(ctx.Result.Skipped, "strict root account lock is advisory; review provider console access before locking root")
	return nil
}
