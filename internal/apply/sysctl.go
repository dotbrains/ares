package apply

func (ctx *Context) applySysctlBaseline() error {
	if err := ctx.writeFile("/etc/sysctl.d/99-ares.conf", []byte(sysctlBaseline), 0o644); err != nil {
		return err
	}
	if ctx.Options.Root == "" {
		return ctx.run("sysctl", "--system")
	}
	return nil
}
