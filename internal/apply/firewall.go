package apply

import (
	"fmt"
	"os"
)

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
	if err := os.WriteFile(ctx.path("/etc/nftables.conf"), []byte(fmt.Sprintf(nftablesRules, ctx.Plan.Host.SSHPort)), 0o644); err != nil {
		return err
	}
	if err := ctx.run("systemctl", "enable", "--now", "nftables"); err != nil {
		return err
	}
	ctx.Result.Applied = append(ctx.Result.Applied, "enabled nftables with SSH port "+ctx.Plan.Host.SSHPort+" allowed")
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
