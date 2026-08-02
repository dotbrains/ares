package apply

import (
	"fmt"

	"github.com/dotbrains/ares/internal/sshguard"
)

func (ctx *Context) applySSHHardening() error {
	if err := ctx.ensurePublicKeyAccessBeforeSSHHardening(); err != nil {
		return err
	}
	if err := ctx.backup("/etc/ssh/sshd_config"); err != nil {
		return fmt.Errorf("backup sshd_config: %w", err)
	}
	if err := ctx.writeFile("/etc/ssh/sshd_config.d/99-ares.conf", []byte(sshDropIn), 0o644); err != nil {
		return err
	}

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

func (ctx *Context) ensurePublicKeyAccessBeforeSSHHardening() error {
	decision := sshguard.Evaluate(sshguard.Facts{
		Root:                 ctx.Options.Root,
		RunningOverSSH:       ctx.Plan.Host.RunningOverSSH,
		AllowPasswordLockout: ctx.Options.AllowPasswordLockout,
	})
	if decision.Allowed && !decision.Bypassed {
		return nil
	}
	if decision.Bypassed {
		ctx.Result.Skipped = append(ctx.Result.Skipped, "SSH password lockout guard bypassed by explicit operator flag")
		return nil
	}
	return fmt.Errorf("SSH hardening would disable password auth, but no authorized_keys file was found for the current or root account")
}
