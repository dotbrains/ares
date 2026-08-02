package apply

import (
	"fmt"
	"strings"
)

var webProfileAppliers = map[string]func(*Context) error{
	"firewalld": (*Context).applyFirewalldWebProfile,
	"nftables":  (*Context).applyNftablesWebProfile,
	"ufw":       (*Context).applyUFWWebProfile,
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
	if err := ctx.writeNftablesRules(ctx.Plan.Host.SSHPort); err != nil {
		return err
	}
	if err := ctx.run("nft", "-c", "-f", "/etc/nftables.conf"); err != nil {
		return err
	}
	if err := ctx.run("systemctl", "enable", "--now", "nftables"); err != nil {
		return err
	}
	ctx.Result.Applied = append(ctx.Result.Applied, "enabled nftables with SSH port "+ctx.Plan.Host.SSHPort+" allowed")
	return nil
}

func (ctx *Context) applyWebProfile() error {
	if apply, ok := webProfileAppliers[ctx.Plan.Host.FirewallBackend]; ok {
		return apply(ctx)
	}
	return fmt.Errorf("unsupported firewall backend %q for web profile", ctx.Plan.Host.FirewallBackend)
}

func (ctx *Context) applyFirewalldWebProfile() error {
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

func (ctx *Context) applyNftablesWebProfile() error {
	if err := ctx.backup("/etc/nftables.conf"); err != nil {
		return fmt.Errorf("backup nftables.conf: %w", err)
	}
	if err := ctx.writeNftablesRules(ctx.Plan.Host.SSHPort, "80", "443"); err != nil {
		return err
	}
	if err := ctx.run("nft", "-c", "-f", "/etc/nftables.conf"); err != nil {
		return err
	}
	if err := ctx.run("nft", "-f", "/etc/nftables.conf"); err != nil {
		return err
	}
	ctx.Result.Applied = append(ctx.Result.Applied, "allowed HTTP and HTTPS")
	return nil
}

func (ctx *Context) applyUFWWebProfile() error {
	for _, port := range []string{"80/tcp", "443/tcp"} {
		if err := ctx.run("ufw", "allow", port); err != nil {
			return err
		}
	}
	ctx.Result.Applied = append(ctx.Result.Applied, "allowed HTTP and HTTPS")
	return nil
}

func (ctx *Context) writeNftablesRules(ports ...string) error {
	lines := make([]string, 0, len(ports))
	for _, port := range ports {
		lines = append(lines, fmt.Sprintf("    tcp dport %s accept", port))
	}
	rules := fmt.Sprintf(nftablesRules, strings.Join(lines, "\n"))
	return ctx.writeFile("/etc/nftables.conf", []byte(rules), 0o644)
}
