package apply

import (
	"os"
	"path/filepath"
)

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
