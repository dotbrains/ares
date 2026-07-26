package apply

import (
	"fmt"
	"os"
	"path/filepath"
)

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
