package apply

import (
	"strings"

	"github.com/dotbrains/ares/internal/plugins"
)

type pluginBehavior struct {
	Apply  func(*Context, plugins.Plugin) error
	Verify func(*Context, plugins.Plugin)
}

func behaviorFor(plugin plugins.Plugin) pluginBehavior {
	plugin = pluginWithCatalogMetadata(plugin)
	behavior := plugins.Behavior(plugin)
	if plugin.Kind == "custom" {
		return pluginBehavior{
			Apply:  func(ctx *Context, plugin plugins.Plugin) error { return ctx.applyCustomPlugin(plugin) },
			Verify: func(ctx *Context, plugin plugins.Plugin) { ctx.verifyCustomPlugin(plugin) },
		}
	}

	switch behavior.Name {
	case "distro":
		return pluginBehavior{Apply: applyDistro, Verify: verifyNone}
	case "provider-advisory":
		return pluginBehavior{Apply: applyProvider, Verify: verifyProvider}
	case "ssh-hardening":
		return pluginBehavior{Apply: applySSH, Verify: verifyDeclared}
	case "firewall":
		return pluginBehavior{Apply: applyFirewall, Verify: verifyDeclared}
	case "fail2ban":
		return pluginBehavior{Apply: applyFail2banBehavior, Verify: verifyDeclared}
	case "security-updates":
		return pluginBehavior{Apply: applySecurityUpdates, Verify: verifyDeclared}
	case "sysctl":
		return pluginBehavior{Apply: applySysctl, Verify: verifyDeclared}
	case "web-profile":
		return pluginBehavior{Apply: applyWeb, Verify: verifyDeclared}
	case "strict-profile":
		return pluginBehavior{Apply: applyStrict, Verify: verifyDeclared}
	default:
		return pluginBehavior{Apply: applyUnsupported, Verify: verifyNone}
	}
}

func pluginWithCatalogMetadata(plugin plugins.Plugin) plugins.Plugin {
	catalogPlugin, ok := plugins.Find(plugin.ID)
	if !ok {
		return plugin
	}
	if plugin.Kind == "" {
		plugin.Kind = catalogPlugin.Kind
	}
	if plugin.Behavior == "" {
		plugin.Behavior = catalogPlugin.Behavior
	}
	if plugin.BehaviorVariant == "" {
		plugin.BehaviorVariant = catalogPlugin.BehaviorVariant
	}
	if plugin.Verifier == "" {
		plugin.Verifier = catalogPlugin.Verifier
	}
	if len(plugin.ManagedFiles) == 0 {
		plugin.ManagedFiles = catalogPlugin.ManagedFiles
	}
	return plugin
}

func applyDistro(ctx *Context, plugin plugins.Plugin) error {
	ctx.Result.Applied = append(ctx.Result.Applied, plugin.ID+": selected "+ctx.Plan.Host.PackageManager+"/"+ctx.Plan.Host.InitSystem+" distro adapter")
	return nil
}

func applyProvider(ctx *Context, plugin plugins.Plugin) error {
	return ctx.applyProviderAdvisory(plugin)
}

func applySSH(ctx *Context, _ plugins.Plugin) error {
	return ctx.applySSHHardening()
}

func applyFirewall(ctx *Context, plugin plugins.Plugin) error {
	switch plugins.Behavior(plugin).Variant {
	case "ufw":
		return ctx.applyUFW()
	case "firewalld":
		return ctx.applyFirewalld()
	case "nftables":
		return ctx.applyNftables()
	default:
		ctx.Result.Skipped = append(ctx.Result.Skipped, plugin.ID+": firewall backend not implemented")
		return nil
	}
}

func applyFail2banBehavior(ctx *Context, _ plugins.Plugin) error {
	return ctx.applyFail2ban()
}

func applySecurityUpdates(ctx *Context, plugin plugins.Plugin) error {
	switch plugins.Behavior(plugin).Variant {
	case "apt":
		return ctx.applyUnattendedUpgrades()
	case "dnf-automatic":
		return ctx.applyDNFAutomatic()
	case "pacman", "zypper", "apk":
		return ctx.applyPackageUpgrade()
	default:
		ctx.Result.Skipped = append(ctx.Result.Skipped, plugin.ID+": security update backend not implemented")
		return nil
	}
}

func applySysctl(ctx *Context, _ plugins.Plugin) error {
	return ctx.applySysctlBaseline()
}

func applyWeb(ctx *Context, _ plugins.Plugin) error {
	return ctx.applyWebProfile()
}

func applyStrict(ctx *Context, _ plugins.Plugin) error {
	return ctx.applyStrictProfile()
}

func applyUnsupported(ctx *Context, plugin plugins.Plugin) error {
	ctx.Result.Skipped = append(ctx.Result.Skipped, plugin.ID+": apply not implemented for behavior "+plugin.Behavior)
	return nil
}

func verifyDeclared(ctx *Context, plugin plugins.Plugin) {
	switch plugins.Behavior(plugin).Verifier {
	case "path":
		verifyManagedPath(ctx, plugin)
	case "firewall":
		verifyFirewall(ctx, plugin)
	case "provider-advisory":
		verifyProvider(ctx, plugin)
	case "command":
		ctx.Result.Verified = append(ctx.Result.Verified, plugin.ID+": command completed")
	case "none", "":
		verifyNone(ctx, plugin)
	default:
		ctx.Result.Failed = append(ctx.Result.Failed, plugin.ID+": unknown verifier "+plugin.Verifier)
	}
}

func verifyManagedPath(ctx *Context, plugin plugins.Plugin) {
	if len(plugin.ManagedFiles) == 0 {
		ctx.Result.Verified = append(ctx.Result.Verified, plugin.ID+": no managed file verifier declared")
		return
	}
	ctx.verifyPath(plugin.ID, plugin.ManagedFiles[0])
}

func verifyFirewall(ctx *Context, plugin plugins.Plugin) {
	switch plugins.Behavior(plugin).Variant {
	case "ufw":
		ctx.verifyUFW(plugin.ID)
	case "firewalld":
		ctx.verifyFirewalld(plugin.ID)
	case "nftables":
		ctx.verifyNftables(plugin.ID)
	case "web":
		ctx.verifyWebProfile(plugin.ID)
	default:
		ctx.Result.Failed = append(ctx.Result.Failed, plugin.ID+": firewall verifier not implemented")
	}
}

func verifyProvider(ctx *Context, plugin plugins.Plugin) {
	ctx.Result.Verified = append(ctx.Result.Verified, plugin.ID+": advisory recorded")
}

func verifyNone(ctx *Context, plugin plugins.Plugin) {
	if strings.TrimSpace(plugin.Verifier) == "" || plugins.Behavior(plugin).IsVerifier("none") {
		ctx.Result.Verified = append(ctx.Result.Verified, plugin.ID+": no verifier declared")
	}
}
