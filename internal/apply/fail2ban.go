package apply

import (
	"os"
	"path/filepath"
)

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
