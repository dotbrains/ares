package system

import (
	"strings"
)

type probeContext struct {
	prober            Prober
	root              string
	osReleaseOverride bool
}

func (ctx probeContext) provider() string {
	if provider := strings.TrimSpace(ctx.prober.Env("ARES_PROVIDER")); provider != "" {
		return normalizeProvider(provider)
	}
	probeFiles := []string{
		"/sys/class/dmi/id/sys_vendor",
		"/sys/class/dmi/id/product_name",
		"/sys/class/dmi/id/board_vendor",
	}
	var values []string
	for _, path := range probeFiles {
		data, err := ctx.prober.ReadFile(rootPath(ctx.root, path))
		if err == nil {
			values = append(values, string(data))
		}
	}
	return providerFromText(strings.Join(values, "\n"))
}

func (ctx probeContext) packageManager(host Host) string {
	if ctx.probeHostCommands() {
		if detected := firstCommandWithProber(ctx.prober, "apt-get", "dnf", "yum", "pacman", "zypper", "apk"); detected != "unknown" {
			return detected
		}
	}
	if plugin, ok := distroPlugin(host); ok && plugin.PackageManager != "" {
		return plugin.PackageManager
	}
	return "unknown"
}

func (ctx probeContext) firewallBackend(host Host) string {
	if ctx.probeHostCommands() && ctx.prober.LookPath("ufw") {
		return "ufw"
	}
	if ctx.probeHostCommands() && ctx.prober.LookPath("firewall-cmd") {
		return "firewalld"
	}
	if ctx.probeHostCommands() && ctx.prober.LookPath("nft") {
		return "nftables"
	}
	if plugin, ok := distroPlugin(host); ok && plugin.FirewallBackend != "" {
		return plugin.FirewallBackend
	}
	return "unknown"
}

func (ctx probeContext) initSystem(host Host) string {
	if err := ctx.prober.Stat(rootPath(ctx.root, "/run/systemd/system")); err == nil {
		return "systemd"
	}
	if err := ctx.prober.Stat(rootPath(ctx.root, "/run/openrc")); err == nil {
		return "openrc"
	}
	if plugin, ok := distroPlugin(host); ok && plugin.InitSystem != "" {
		return plugin.InitSystem
	}
	return "unknown"
}

func (ctx probeContext) probeHostCommands() bool {
	return ctx.root == "" && !ctx.osReleaseOverride
}
