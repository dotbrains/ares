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
	if err := ctx.run(ctx.Plan.Host.PackageManager, "install", "-y", "ufw"); err != nil {
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
	for _, port := range []string{"80/tcp", "443/tcp"} {
		if err := ctx.run("ufw", "allow", port); err != nil {
			return err
		}
	}
	ctx.Result.Applied = append(ctx.Result.Applied, "allowed HTTP and HTTPS")
	return nil
}

func (ctx *Context) applyFail2ban() error {
	if err := ctx.run(ctx.Plan.Host.PackageManager, "install", "-y", "fail2ban"); err != nil {
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
	if err := ctx.run(ctx.Plan.Host.PackageManager, "install", "-y", "unattended-upgrades"); err != nil {
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
