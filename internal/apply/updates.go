package apply

import (
	"fmt"
	"os"
	"path/filepath"
)

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
