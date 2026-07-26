package plugins

type Plugin struct {
	ID           string
	Name         string
	Kind         string
	Summary      string
	Categories   []string
	Requires     []string
	Capabilities []string
	Distros      []string
}

func Builtins() []Plugin {
	return []Plugin{
		{
			ID:           "distro-debian",
			Name:         "Debian adapter",
			Kind:         "builtin",
			Summary:      "Provides apt, systemd, SSH, and service defaults for Debian VPS hosts",
			Categories:   []string{"distro"},
			Capabilities: []string{"package-manager", "service-manager", "ssh-service"},
			Distros:      []string{"debian"},
		},
		{
			ID:           "distro-ubuntu",
			Name:         "Ubuntu adapter",
			Kind:         "builtin",
			Summary:      "Provides apt, systemd, SSH, and service defaults for Ubuntu VPS hosts",
			Categories:   []string{"distro"},
			Capabilities: []string{"package-manager", "service-manager", "ssh-service"},
			Distros:      []string{"ubuntu"},
		},
		{
			ID:           "distro-rhel",
			Name:         "RHEL-family adapter",
			Kind:         "builtin",
			Summary:      "Provides dnf, systemd, SSH, and service defaults for AlmaLinux and Rocky Linux",
			Categories:   []string{"distro"},
			Capabilities: []string{"package-manager", "service-manager", "ssh-service"},
			Distros:      []string{"almalinux", "rocky", "rhel"},
		},
		{
			ID:           "distro-fedora",
			Name:         "Fedora adapter",
			Kind:         "builtin",
			Summary:      "Provides dnf, systemd, SSH, and service defaults for Fedora Server",
			Categories:   []string{"distro"},
			Capabilities: []string{"package-manager", "service-manager", "ssh-service"},
			Distros:      []string{"fedora"},
		},
		{
			ID:           "ssh-hardening",
			Name:         "SSH hardening",
			Kind:         "builtin",
			Summary:      "Writes a managed sshd drop-in after preserving the active SSH port",
			Categories:   []string{"ssh", "hardening"},
			Requires:     []string{"ssh-service"},
			Capabilities: []string{"ssh-hardening"},
		},
		{
			ID:           "firewall-auto",
			Name:         "Automatic firewall",
			Kind:         "builtin",
			Summary:      "Selects UFW, firewalld, or nftables and keeps the active SSH port reachable",
			Categories:   []string{"firewall", "network"},
			Capabilities: []string{"firewall"},
		},
		{
			ID:           "intrusion-protection",
			Name:         "Intrusion protection",
			Kind:         "builtin",
			Summary:      "Installs and enables fail2ban with conservative SSH jail defaults",
			Categories:   []string{"ssh", "hardening"},
			Capabilities: []string{"fail2ban"},
		},
		{
			ID:           "security-updates",
			Name:         "Security updates",
			Kind:         "builtin",
			Summary:      "Enables distro-native automatic security updates without automatic reboots",
			Categories:   []string{"updates", "hardening"},
			Capabilities: []string{"security-updates"},
		},
		{
			ID:           "sysctl-baseline",
			Name:         "Kernel network baseline",
			Kind:         "builtin",
			Summary:      "Applies conservative VPS-safe network sysctl hardening",
			Categories:   []string{"kernel", "network", "hardening"},
			Capabilities: []string{"sysctl"},
		},
	}
}

func Find(id string) (Plugin, bool) {
	for _, plugin := range Builtins() {
		if plugin.ID == id {
			return plugin, true
		}
	}
	return Plugin{}, false
}
