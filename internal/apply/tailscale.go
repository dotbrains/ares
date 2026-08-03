package apply

import (
	"context"
	"fmt"
	"strings"
)

func (ctx *Context) applyTailscaleSSH() error {
	if ctx.Options.Tailscale.SSHEnabled && strings.TrimSpace(ctx.Options.Tailscale.AuthKey) == "" {
		return fmt.Errorf("tailscale SSH requires auth key environment variable %s", ctx.Options.Tailscale.AuthKeyEnv)
	}
	if err := ctx.installPackages("tailscale"); err != nil {
		return err
	}
	if ctx.Plan.Host.InitSystem == "systemd" {
		if err := ctx.run("systemctl", "enable", "--now", "tailscaled"); err != nil {
			return err
		}
	} else {
		ctx.Result.Skipped = append(ctx.Result.Skipped, "tailscaled service enablement is not implemented for init system "+ctx.Plan.Host.InitSystem)
	}
	if !ctx.Options.Tailscale.SSHEnabled {
		ctx.Result.Skipped = append(ctx.Result.Skipped, "Tailscale SSH is not enabled automatically; set tailscale.ssh_enabled and tailscale.auth_key_env to opt in")
		ctx.Result.Applied = append(ctx.Result.Applied, "prepared tailscaled for explicit Tailscale SSH setup")
		return nil
	}
	args, redactedArgs := ctx.tailscaleUpArgs()
	if err := ctx.runRedacted("tailscale", args, redactedArgs); err != nil {
		return err
	}
	ctx.Result.Applied = append(ctx.Result.Applied, "prepared tailscaled for explicit Tailscale SSH setup")
	ctx.Result.Applied = append(ctx.Result.Applied, "enabled Tailscale SSH using auth key from "+ctx.Options.Tailscale.AuthKeyEnv)
	return nil
}

func (ctx *Context) verifyTailscaleSSH(pluginID string) {
	ctx.verifyCommandContains(pluginID, []string{"tailscale", "version"}, "")
}

func (ctx *Context) tailscaleUpArgs() ([]string, []string) {
	args := []string{"up", "--ssh", "--auth-key", ctx.Options.Tailscale.AuthKey}
	redacted := []string{"up", "--ssh", "--auth-key", "REDACTED"}
	if ctx.Options.Tailscale.Hostname != "" {
		args = append(args, "--hostname", ctx.Options.Tailscale.Hostname)
		redacted = append(redacted, "--hostname", ctx.Options.Tailscale.Hostname)
	}
	if ctx.Options.Tailscale.AcceptRoutes {
		args = append(args, "--accept-routes")
		redacted = append(redacted, "--accept-routes")
	}
	if ctx.Options.Tailscale.LoginServer != "" {
		args = append(args, "--login-server", ctx.Options.Tailscale.LoginServer)
		redacted = append(redacted, "--login-server", ctx.Options.Tailscale.LoginServer)
	}
	if len(ctx.Options.Tailscale.Tags) > 0 {
		tags := strings.Join(ctx.Options.Tailscale.Tags, ",")
		args = append(args, "--advertise-tags", tags)
		redacted = append(redacted, "--advertise-tags", tags)
	}
	args = append(args, ctx.Options.Tailscale.ExtraArgs...)
	redacted = append(redacted, ctx.Options.Tailscale.ExtraArgs...)
	return args, redacted
}

func (ctx *Context) runRedacted(name string, args []string, redactedArgs []string) error {
	if ctx.Options.Root != "" {
		ctx.Result.Applied = append(ctx.Result.Applied, "would run: "+name+" "+strings.Join(redactedArgs, " "))
		return nil
	}
	output, err := ctx.Options.Runner.Run(context.Background(), name, args...)
	if err != nil {
		return fmt.Errorf("%s %s: %w: %s", name, strings.Join(redactedArgs, " "), err, strings.TrimSpace(output))
	}
	if len(output) > 0 {
		ctx.Result.Applied = append(ctx.Result.Applied, strings.TrimSpace(output))
	}
	return nil
}
