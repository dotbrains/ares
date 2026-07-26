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

	if !isSupported(host) {
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
	ids := append([]string{}, cfg.Plugins.Enabled...)
	if distroPlugin := distroPluginID(host); distroPlugin != "" && !slices.Contains(ids, distroPlugin) {
		ids = append([]string{distroPlugin}, ids...)
	}
	if cfg.Profile == "web" && !slices.Contains(ids, "web-profile") {
		ids = append(ids, "web-profile")
	}

	var selected []plugins.Plugin
	for _, id := range ids {
		if plugin, ok := plugins.Find(id); ok {
			selected = append(selected, plugin)
		}
	}
	for _, custom := range cfg.Plugins.Custom {
		selected = append(selected, plugins.Plugin{
			ID:       custom.Name,
			Name:     custom.Name,
			Kind:     "custom",
			Summary:  "Custom local plugin",
			Probe:    custom.Probe,
			Plan:     custom.Plan,
			Apply:    custom.Apply,
			Rollback: custom.Rollback,
		})
	}
	return selected
}

func distroPluginID(host system.Host) string {
	switch host.OSID {
	case "ubuntu":
		return "distro-ubuntu"
	case "debian":
		return "distro-debian"
	case "almalinux", "rocky", "rhel":
		return "distro-rhel"
	case "fedora":
		return "distro-fedora"
	default:
		return ""
	}
}

func isSupported(host system.Host) bool {
	return distroPluginID(host) != ""
}

func actionsForPlugin(host system.Host, profile string, plugin plugins.Plugin) []Action {
	switch plugin.ID {
	case "distro-ubuntu", "distro-debian", "distro-rhel", "distro-fedora":
		return []Action{{
			Plugin: plugin.ID,
			Title:  "Use distro adapter",
			Detail: fmt.Sprintf("Use %s, %s, and SSH service %s for host operations", host.PackageManager, host.InitSystem, host.SSHService),
		}}
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
	case "firewall-ufw", "firewall-firewalld", "firewall-nftables":
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
	case "unattended-upgrades", "dnf-automatic":
		return []Action{{
			Plugin: plugin.ID,
			Title:  "Enable automatic security updates",
			Detail: "Use distro-native security updates without automatic reboots",
		}}
	case "sysctl-baseline":
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
	default:
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
