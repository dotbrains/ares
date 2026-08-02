package scenario

import (
	"github.com/dotbrains/ares/internal/config"
	"github.com/dotbrains/ares/internal/system"
)

type Scenario struct {
	Name             string
	Host             system.Host
	Config           *config.Config
	ExpectedPlugins  []string
	ExpectedCommands []string
	ExpectedFiles    []string
	ExpectedBackups  []string
	ExpectedRollback []string
}

func Matrix() []Scenario {
	return []Scenario{
		UbuntuBasic(),
		ArchWeb(),
		RHELBasic(),
	}
}

func (scenario Scenario) HasExpectedPlugin(id string) bool {
	for _, pluginID := range scenario.ExpectedPlugins {
		if pluginID == id {
			return true
		}
	}
	return false
}

func UbuntuBasic() Scenario {
	return Scenario{
		Name:   "ubuntu-basic",
		Host:   UbuntuHost(),
		Config: config.DefaultConfig(),
		ExpectedPlugins: []string{
			"distro-ubuntu",
			"ssh-hardening",
			"firewall-ufw",
			"fail2ban",
			"unattended-upgrades",
			"sysctl-baseline",
		},
		ExpectedCommands: []string{
			"sshd -t",
			"systemctl reload ssh",
			"ufw allow 22/tcp",
			"systemctl enable --now fail2ban",
			"sysctl --system",
		},
		ExpectedFiles: []string{
			"/etc/ssh/sshd_config.d/99-ares.conf",
			"/etc/fail2ban/jail.d/ares-sshd.conf",
			"/etc/sysctl.d/99-ares.conf",
		},
		ExpectedBackups: []string{
			"/etc/ssh/sshd_config",
		},
		ExpectedRollback: []string{
			"would restore newest backup for /etc/ssh/sshd_config",
			"would remove /etc/ssh/sshd_config.d/99-ares.conf",
		},
	}
}

func RHELBasic() Scenario {
	return Scenario{
		Name:   "rhel-basic",
		Host:   DistroHost("rocky", "dnf", "firewalld"),
		Config: config.DefaultConfig(),
		ExpectedPlugins: []string{
			"distro-rhel",
			"ssh-hardening",
			"firewall-firewalld",
			"fail2ban",
			"dnf-automatic",
			"sysctl-baseline",
		},
		ExpectedCommands: []string{
			"dnf install -y dnf-automatic",
			"systemctl enable --now dnf-automatic.timer",
			"firewall-cmd --permanent --add-port=22/tcp",
		},
		ExpectedFiles: []string{
			"/etc/ssh/sshd_config.d/99-ares.conf",
			"/etc/fail2ban/jail.d/ares-sshd.conf",
			"/etc/dnf/automatic.conf",
			"/etc/sysctl.d/99-ares.conf",
		},
		ExpectedBackups: []string{
			"/etc/ssh/sshd_config",
		},
	}
}

func ArchWeb() Scenario {
	cfg := config.DefaultConfig()
	cfg.Profile = "web"
	return Scenario{
		Name:   "arch-web",
		Host:   DistroHost("arch", "pacman", "nftables"),
		Config: cfg,
		ExpectedPlugins: []string{
			"distro-arch",
			"ssh-hardening",
			"firewall-nftables",
			"fail2ban",
			"pacman-upgrade",
			"sysctl-baseline",
			"web-profile",
		},
		ExpectedCommands: []string{
			"pacman -Syu --noconfirm",
			"nft -c -f /etc/nftables.conf",
			"nft -f /etc/nftables.conf",
		},
		ExpectedFiles: []string{
			"/etc/nftables.conf",
		},
		ExpectedBackups: []string{
			"/etc/nftables.conf",
		},
	}
}

func UbuntuHost() system.Host {
	host := system.Host{
		OSID:            "ubuntu",
		OSName:          "Ubuntu 24.04 LTS",
		OSVersion:       "24.04",
		PackageManager:  "apt-get",
		InitSystem:      "systemd",
		FirewallBackend: "ufw",
		SSHService:      "ssh",
		SSHPort:         "22",
		RunningOverSSH:  true,
		Facts: map[string]system.Fact{
			"os":               {Source: "fixture", Confidence: "high"},
			"package_manager":  {Source: "fixture", Confidence: "high"},
			"init_system":      {Source: "fixture", Confidence: "high"},
			"firewall_backend": {Source: "fixture", Confidence: "high"},
			"ssh_service":      {Source: "fixture", Confidence: "high"},
			"ssh_port":         {Source: "fixture", Confidence: "high"},
		},
	}
	host.RefreshObservations()
	return host
}

func DistroHost(osID string, packageManager string, firewallBackend string) system.Host {
	host := system.Host{
		OSID:            osID,
		OSName:          osID,
		PackageManager:  packageManager,
		InitSystem:      "systemd",
		FirewallBackend: firewallBackend,
		SSHService:      "sshd",
		SSHPort:         "22",
		RunningOverSSH:  true,
		Facts: map[string]system.Fact{
			"os":               {Source: "fixture", Confidence: "high"},
			"package_manager":  {Source: "fixture", Confidence: "high"},
			"init_system":      {Source: "fixture", Confidence: "high"},
			"firewall_backend": {Source: "fixture", Confidence: "high"},
			"ssh_service":      {Source: "fixture", Confidence: "high"},
			"ssh_port":         {Source: "fixture", Confidence: "high"},
		},
	}
	host.RefreshObservations()
	return host
}
