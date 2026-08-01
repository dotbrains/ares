package apply

import (
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"strings"
)

func (ctx *Context) applySSHHardening() error {
	if err := ctx.ensurePublicKeyAccessBeforeSSHHardening(); err != nil {
		return err
	}
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

func (ctx *Context) ensurePublicKeyAccessBeforeSSHHardening() error {
	if ctx.Options.Root != "" || !ctx.Plan.Host.RunningOverSSH {
		return nil
	}
	if ctx.Options.AllowPasswordLockout {
		ctx.Result.Skipped = append(ctx.Result.Skipped, "SSH password lockout guard bypassed by explicit operator flag")
		return nil
	}
	if hasAuthorizedKeys(ctx.Options.Root, authorizedKeyCandidates()) {
		return nil
	}
	return fmt.Errorf("SSH hardening would disable password auth, but no authorized_keys file was found for the current or root account")
}

func sshLockoutPolicy(opts Options) string {
	if opts.AllowPasswordLockout {
		return "password lockout explicitly allowed by config or CLI"
	}
	return "refuse active SSH password lockout without authorized_keys"
}

func authorizedKeyCandidates() []string {
	candidates := []string{"/root/.ssh/authorized_keys"}
	if home := strings.TrimSpace(os.Getenv("HOME")); home != "" {
		candidates = append(candidates, filepath.Join(home, ".ssh", "authorized_keys"))
	}
	if current, err := user.Current(); err == nil && current.HomeDir != "" {
		candidates = append(candidates, filepath.Join(current.HomeDir, ".ssh", "authorized_keys"))
	}
	return candidates
}

func hasAuthorizedKeys(root string, candidates []string) bool {
	for _, path := range candidates {
		data, err := os.ReadFile(rootedApplyPath(root, path))
		if err == nil && strings.TrimSpace(string(data)) != "" {
			return true
		}
	}
	return false
}

func rootedApplyPath(root string, path string) string {
	if root == "" {
		return path
	}
	return filepath.Join(root, strings.TrimPrefix(path, "/"))
}
