package plan

import (
	"fmt"
	"slices"

	"github.com/dotbrains/ares/internal/config"
	"github.com/dotbrains/ares/internal/plugins"
	"github.com/dotbrains/ares/internal/system"
)

type Action struct {
	Plugin string
	Title  string
	Detail string
	Risky  bool
}

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
		result.Actions = append(result.Actions, actionsForPlugin(host, cfg.Profile, plugin)...)
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

func actionsForPlugin(host system.Host, profile string, plugin plugins.Plugin) []Action {
	if slices.Contains(plugin.Categories, "distro") {
		return []Action{{
			Plugin: plugin.ID,
			Title:  "Use distro adapter",
			Detail: fmt.Sprintf("Use %s, %s, and SSH service %s for host operations", host.PackageManager, host.InitSystem, host.SSHService),
		}}
	}

	switch plugin.Behavior {
	case "ssh-hardening":
		return []Action{
			{
				Plugin: plugin.ID,
				Title:  "Back up SSH configuration",
				Detail: "Create timestamped backups before writing a managed sshd drop-in",
			},
			{
				Plugin: plugin.ID,
				Title:  "Preserve active SSH access",
				Detail: fmt.Sprintf("Keep port %s reachable, validate sshd config, then reload %s", host.SSHPort, host.SSHService),
				Risky:  true,
			},
		}
	case "firewall":
		return []Action{{
			Plugin: plugin.ID,
			Title:  "Configure firewall",
			Detail: fmt.Sprintf("Use %s, allow SSH port %s, deny other inbound traffic, allow outbound traffic", host.FirewallBackend, host.SSHPort),
			Risky:  true,
		}}
	case "fail2ban":
		return []Action{{
			Plugin: plugin.ID,
			Title:  "Enable fail2ban",
			Detail: "Install and enable a conservative SSH jail",
		}}
	case "security-updates":
		return []Action{{
			Plugin: plugin.ID,
			Title:  "Configure distro-native updates",
			Detail: "Use the selected distro update mechanism without automatic reboots",
		}}
	case "sysctl":
		return []Action{{
			Plugin: plugin.ID,
			Title:  "Apply sysctl baseline",
			Detail: "Write conservative network hardening to /etc/sysctl.d/99-ares.conf",
		}}
	case "web-profile":
		return []Action{{
			Plugin: plugin.ID,
			Title:  "Allow web traffic",
			Detail: "Allow inbound 80/tcp and 443/tcp through the selected firewall backend",
			Risky:  true,
		}}
	case "strict-profile":
		return []Action{{
			Plugin: plugin.ID,
			Title:  "Apply strict profile",
			Detail: "Use stricter fail2ban defaults and document optional root account lockout steps",
			Risky:  true,
		}}
	default:
		if slices.Contains(plugin.Categories, "provider") {
			return []Action{{
				Plugin: plugin.ID,
				Title:  "Record provider advisory",
				Detail: "Add provider-specific recovery and firewall-console reminders to the run report without mutating provider settings",
			}}
		}
		if plugin.Kind == "custom" {
			return []Action{{
				Plugin: plugin.ID,
				Title:  "Run custom plugin",
				Detail: fmt.Sprintf("Probe with %q, plan with %q, apply with %q", plugin.Probe, plugin.Plan, plugin.Apply),
				Risky:  true,
			}}
		}
		return []Action{{
			Plugin: plugin.ID,
			Title:  "Run plugin",
			Detail: fmt.Sprintf("Execute plugin under the %s profile", profile),
		}}
	}
}
