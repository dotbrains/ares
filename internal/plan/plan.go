package plan

import (
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
	selection := Selection{Host: host, Config: cfg}
	selected := selection.Plugins()
	result := Plan{
		Profile:  cfg.Profile,
		Host:     host,
		Plugins:  selected,
		Warnings: selection.Warnings(),
	}

	for _, plugin := range selected {
		result.Actions = append(result.Actions, intent.ForPlugin(host, cfg.Profile, plugin).Actions()...)
	}

	return result
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
