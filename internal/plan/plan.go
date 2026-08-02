package plan

import (
	"fmt"
	"slices"

	"github.com/dotbrains/ares/internal/config"
	"github.com/dotbrains/ares/internal/intent"
	"github.com/dotbrains/ares/internal/plugins"
	"github.com/dotbrains/ares/internal/system"
)

type Action = intent.Action

type Plan struct {
	Profile  string
	Host     system.Host
	Plugins  []plugins.Plugin
	Actions  []Action
	Warnings []string
}

func Build(host system.Host, cfg *config.Config) Plan {
	selected := selectPlugins(host, cfg)
	result := Plan{
		Profile: cfg.Profile,
		Host:    host,
		Plugins: selected,
	}

	if _, ok := plugins.DistroAdapter(hostMatcher(host)); !ok {
		result.Warnings = append(result.Warnings, fmt.Sprintf("distro %q is not a first-class target yet", host.OSID))
	}
	if !host.RunningOverSSH {
		result.Warnings = append(result.Warnings, "active SSH session was not detected; SSH lockout checks will be less certain")
	}

	for _, plugin := range selected {
		result.Actions = append(result.Actions, intent.ForPlugin(host, cfg.Profile, plugin).Actions()...)
	}

	return result
}

func selectPlugins(host system.Host, cfg *config.Config) []plugins.Plugin {
	ids := resolvePluginIDs(host, append([]string{}, cfg.Plugins.Enabled...))
	if distroPlugin, ok := plugins.DistroAdapter(hostMatcher(host)); ok && !slices.Contains(ids, distroPlugin.ID) {
		ids = append([]string{distroPlugin.ID}, ids...)
	}
	if providerPlugin, ok := plugins.ProviderAdvisory(hostMatcher(host)); ok {
		ids = append(ids, providerPlugin.ID)
	}
	switch cfg.Profile {
	case "web":
		ids = append(ids, "web-profile")
	case "strict":
		ids = append(ids, "strict-profile")
	}
	ids = unique(ids)

	var selected []plugins.Plugin
	for _, id := range ids {
		if plugin, ok := plugins.Find(id); ok {
			selected = append(selected, plugin)
		}
	}
	for _, custom := range cfg.Plugins.Custom {
		selected = append(selected, plugins.Plugin{
			ID:             custom.Name,
			Name:           custom.Name,
			Kind:           "custom",
			Summary:        "Custom local plugin",
			Probe:          custom.Probe,
			Plan:           custom.Plan,
			Apply:          custom.Apply,
			Verify:         custom.Verify,
			Rollback:       custom.Rollback,
			TimeoutSeconds: custom.TimeoutSeconds,
		})
	}
	return selected
}

func resolvePluginIDs(host system.Host, ids []string) []string {
	resolved := make([]string, 0, len(ids))
	for _, id := range ids {
		switch id {
		case "firewall-auto":
			resolved = append(resolved, firewallPluginID(host))
		case "security-updates":
			resolved = append(resolved, updatesPluginID(host))
		default:
			resolved = append(resolved, id)
		}
	}
	return resolved
}

func firewallPluginID(host system.Host) string {
	if host.FirewallBackend == "nftables" {
		return "firewall-nftables"
	}
	if plugin, ok := plugins.FirstByCapability(hostMatcher(host), "firewall"); ok {
		return plugin.ID
	}
	return "firewall-ufw"
}

func updatesPluginID(host system.Host) string {
	if plugin, ok := plugins.FirstByCapability(hostMatcher(host), "security-updates"); ok {
		return plugin.ID
	}
	return "unattended-upgrades"
}

func unique(ids []string) []string {
	seen := map[string]bool{}
	var result []string
	for _, id := range ids {
		if id == "" || seen[id] {
			continue
		}
		seen[id] = true
		result = append(result, id)
	}
	return result
}

func hostMatcher(host system.Host) plugins.HostMatcher {
	return plugins.HostMatcher{
		OSID:            host.OSID,
		IDLike:          host.IDLike,
		Provider:        host.Provider,
		PackageManager:  host.PackageManager,
		FirewallBackend: host.FirewallBackend,
	}
}
