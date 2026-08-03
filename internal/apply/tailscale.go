package apply

func (ctx *Context) applyTailscaleSSH() error {
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
	ctx.Result.Skipped = append(ctx.Result.Skipped, "Tailscale SSH is not enabled automatically; run tailscale up --ssh after reviewing tailnet policy and authentication")
	ctx.Result.Applied = append(ctx.Result.Applied, "prepared tailscaled for explicit Tailscale SSH setup")
	return nil
}

func (ctx *Context) verifyTailscaleSSH(pluginID string) {
	ctx.verifyCommandContains(pluginID, []string{"tailscale", "version"}, "")
}
