package apply

import (
	"strings"

	"github.com/dotbrains/ares/internal/plugins"
)

type pluginBehavior struct {
	Apply  func(*Context, plugins.Plugin) error
	Verify func(*Context, plugins.Plugin)
}

var builtinBehaviors = map[string]pluginBehavior{
	"distro":            {Apply: applyDistro, Verify: verifyNone},
	"provider-advisory": {Apply: applyProvider, Verify: verifyProvider},
	"ssh-hardening":     {Apply: applySSH, Verify: verifyDeclared},
	"firewall":          {Apply: applyFirewall, Verify: verifyDeclared},
	"fail2ban":          {Apply: applyFail2banBehavior, Verify: verifyDeclared},
	"security-updates":  {Apply: applySecurityUpdates, Verify: verifyDeclared},
	"sysctl":            {Apply: applySysctl, Verify: verifyDeclared},
	"tailscale-ssh":     {Apply: applyTailscaleSSH, Verify: verifyTailscaleSSH},
	"web-profile":       {Apply: applyWeb, Verify: verifyDeclared},
	"strict-profile":    {Apply: applyStrict, Verify: verifyDeclared},
}

var verifierHandlers = map[string]func(*Context, plugins.Plugin){
	"path":              verifyManagedPath,
	"firewall":          verifyFirewall,
	"provider-advisory": verifyProvider,
	"command":           verifyCommand,
	"none":              verifyNone,
	"":                  verifyNone,
}

var firewallAppliers = map[string]func(*Context) error{
	"ufw":       (*Context).applyUFW,
	"firewalld": (*Context).applyFirewalld,
	"nftables":  (*Context).applyNftables,
}

var securityUpdateAppliers = map[string]func(*Context) error{
	"apt":           (*Context).applyUnattendedUpgrades,
	"dnf-automatic": (*Context).applyDNFAutomatic,
	"pacman":        (*Context).applyPackageUpgrade,
	"zypper":        (*Context).applyPackageUpgrade,
	"apk":           (*Context).applyPackageUpgrade,
}

var firewallVerifiers = map[string]func(*Context, string){
	"ufw":       (*Context).verifyUFW,
	"firewalld": (*Context).verifyFirewalld,
	"nftables":  (*Context).verifyNftables,
	"web":       (*Context).verifyWebProfile,
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

	if behavior, ok := builtinBehaviors[behavior.Name]; ok {
		return behavior
	}
	return pluginBehavior{Apply: applyUnsupported, Verify: verifyNone}
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
	if apply, ok := firewallAppliers[plugins.Behavior(plugin).Variant]; ok {
		return apply(ctx)
	}
	ctx.Result.Skipped = append(ctx.Result.Skipped, plugin.ID+": firewall backend not implemented")
	return nil
}

func applyFail2banBehavior(ctx *Context, _ plugins.Plugin) error {
	return ctx.applyFail2ban()
}

func applySecurityUpdates(ctx *Context, plugin plugins.Plugin) error {
	if apply, ok := securityUpdateAppliers[plugins.Behavior(plugin).Variant]; ok {
		return apply(ctx)
	}
	ctx.Result.Skipped = append(ctx.Result.Skipped, plugin.ID+": security update backend not implemented")
	return nil
}

func applySysctl(ctx *Context, _ plugins.Plugin) error {
	return ctx.applySysctlBaseline()
}

func applyTailscaleSSH(ctx *Context, _ plugins.Plugin) error {
	return ctx.applyTailscaleSSH()
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
	if verify, ok := verifierHandlers[plugins.Behavior(plugin).Verifier]; ok {
		verify(ctx, plugin)
		return
	}
	if plugin.Verifier != "" {
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
	if verify, ok := firewallVerifiers[plugins.Behavior(plugin).Variant]; ok {
		verify(ctx, plugin.ID)
		return
	}
	ctx.Result.Failed = append(ctx.Result.Failed, plugin.ID+": firewall verifier not implemented")
}

func verifyProvider(ctx *Context, plugin plugins.Plugin) {
	ctx.Result.Verified = append(ctx.Result.Verified, plugin.ID+": advisory recorded")
}

func verifyTailscaleSSH(ctx *Context, plugin plugins.Plugin) {
	ctx.verifyTailscaleSSH(plugin.ID)
}

func verifyCommand(ctx *Context, plugin plugins.Plugin) {
	ctx.Result.Verified = append(ctx.Result.Verified, plugin.ID+": command completed")
}

func verifyNone(ctx *Context, plugin plugins.Plugin) {
	if strings.TrimSpace(plugin.Verifier) == "" || plugins.Behavior(plugin).IsVerifier("none") {
		ctx.Result.Verified = append(ctx.Result.Verified, plugin.ID+": no verifier declared")
	}
}
