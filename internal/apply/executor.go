package apply

import (
	"fmt"
	"os/exec"
	"slices"
	"strings"

	"github.com/dotbrains/ares/internal/plugins"
)

type PluginExecutor struct {
	Context *Context
}

func (executor PluginExecutor) Execute(plugin plugins.Plugin) error {
	ctx := executor.Context
	if !executor.Probe(plugin) {
		return nil
	}
	if err := executor.Apply(plugin); err != nil {
		ctx.Result.Failed = append(ctx.Result.Failed, fmt.Sprintf("%s: %v", plugin.ID, err))
		return err
	}
	return executor.Verify(plugin)
}

func (executor PluginExecutor) Probe(plugin plugins.Plugin) bool {
	ctx := executor.Context
	if plugin.Probe == "" {
		ctx.Result.Probed = append(ctx.Result.Probed, plugin.ID+": no probe declared")
		return true
	}
	if ctx.Options.Root != "" {
		ctx.Result.Probed = append(ctx.Result.Probed, plugin.ID+": would probe with "+plugin.Probe)
		return true
	}
	if plugin.Kind == "custom" {
		output, err := runCustomCommand(plugin, plugin.Probe)
		if err != nil {
			ctx.Result.Skipped = append(ctx.Result.Skipped, plugin.ID+": probe did not pass before apply: "+probeFailureMessage(output, err))
			return false
		}
		ctx.Result.Probed = append(ctx.Result.Probed, plugin.ID+": probe passed")
		return true
	}
	cmd := exec.Command("sh", "-lc", plugin.Probe)
	if output, err := cmd.CombinedOutput(); err != nil {
		ctx.Result.Skipped = append(ctx.Result.Skipped, plugin.ID+": probe did not pass before apply: "+strings.TrimSpace(string(output)))
		return plugin.Kind != "custom"
	}
	ctx.Result.Probed = append(ctx.Result.Probed, plugin.ID+": probe passed")
	return true
}

func (executor PluginExecutor) Apply(plugin plugins.Plugin) error {
	ctx := executor.Context
	if slices.Contains(plugin.Categories, "distro") {
		ctx.Result.Applied = append(ctx.Result.Applied, plugin.ID+": selected "+ctx.Plan.Host.PackageManager+"/"+ctx.Plan.Host.InitSystem+" distro adapter")
		return nil
	}

	switch plugin.ID {
	case "ssh-hardening":
		return ctx.applySSHHardening()
	case "firewall-ufw":
		return ctx.applyUFW()
	case "firewall-firewalld":
		return ctx.applyFirewalld()
	case "firewall-nftables":
		return ctx.applyNftables()
	case "fail2ban":
		return ctx.applyFail2ban()
	case "unattended-upgrades":
		return ctx.applyUnattendedUpgrades()
	case "dnf-automatic":
		return ctx.applyDNFAutomatic()
	case "pacman-upgrade", "zypper-patches", "apk-upgrade":
		return ctx.applyPackageUpgrade()
	case "sysctl-baseline":
		return ctx.applySysctlBaseline()
	case "web-profile":
		return ctx.applyWebProfile()
	case "strict-profile":
		return ctx.applyStrictProfile()
	default:
		if strings.HasPrefix(plugin.ID, "provider-") {
			return ctx.applyProviderAdvisory(plugin)
		}
		if plugin.Kind == "custom" {
			return ctx.applyCustomPlugin(plugin)
		}
		ctx.Result.Skipped = append(ctx.Result.Skipped, plugin.ID+": apply not implemented for this plugin")
	}
	return nil
}

func (executor PluginExecutor) Verify(plugin plugins.Plugin) error {
	ctx := executor.Context
	failuresBeforeVerify := len(ctx.Result.Failed)
	ctx.verifyPlugin(plugin)
	if len(ctx.Result.Failed) > failuresBeforeVerify {
		return fmt.Errorf("%s verification failed", plugin.ID)
	}
	return nil
}
