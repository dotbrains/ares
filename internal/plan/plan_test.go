package plan

import (
	"testing"

	"github.com/dotbrains/ares/internal/config"
	"github.com/dotbrains/ares/internal/scenario"
	"github.com/dotbrains/ares/internal/system"
)

func TestBuildAddsDistroPlugin(t *testing.T) {
	result := Build(scenario.UbuntuHost(), config.DefaultConfig())
	if len(result.Plugins) == 0 {
		t.Fatal("expected selected plugins")
	}
	if result.Plugins[0].ID != "distro-ubuntu" {
		t.Fatalf("first plugin = %q, want distro-ubuntu", result.Plugins[0].ID)
	}
	if len(result.Actions) == 0 {
		t.Fatal("expected planned actions")
	}
	if len(result.Warnings) != 0 {
		t.Fatalf("unexpected warnings: %v", result.Warnings)
	}
}

func TestBuildResolvesRHELDefaults(t *testing.T) {
	result := Build(system.Host{
		OSID:            "rocky",
		OSName:          "Rocky Linux 9.4",
		OSVersion:       "9.4",
		PackageManager:  "dnf",
		InitSystem:      "systemd",
		FirewallBackend: "firewalld",
		SSHService:      "sshd",
		SSHPort:         "22",
		RunningOverSSH:  true,
	}, config.DefaultConfig())

	for _, id := range []string{"distro-rhel", "firewall-firewalld", "dnf-automatic"} {
		if !hasPlugin(result, id) {
			t.Fatalf("expected %s in %+v", id, result.Plugins)
		}
	}
	if hasPlugin(result, "firewall-ufw") || hasPlugin(result, "unattended-upgrades") {
		t.Fatalf("unexpected Debian defaults in %+v", result.Plugins)
	}
}

func TestBuildResolvesNewDistroDefaults(t *testing.T) {
	cases := []struct {
		name            string
		host            system.Host
		expectedPlugins []string
	}{
		{
			name: "arch",
			host: scenario.DistroHost("arch", "pacman", "nftables"),
			expectedPlugins: []string{
				"distro-arch",
				"firewall-nftables",
				"pacman-upgrade",
			},
		},
		{
			name: "opensuse",
			host: scenario.DistroHost("opensuse-leap", "zypper", "firewalld"),
			expectedPlugins: []string{
				"distro-opensuse",
				"firewall-firewalld",
				"zypper-patches",
			},
		},
		{
			name: "alpine",
			host: scenario.DistroHost("alpine", "apk", "nftables"),
			expectedPlugins: []string{
				"distro-alpine",
				"firewall-nftables",
				"apk-upgrade",
			},
		},
		{
			name: "oracle",
			host: scenario.DistroHost("ol", "dnf", "firewalld"),
			expectedPlugins: []string{
				"distro-oracle",
				"firewall-firewalld",
				"dnf-automatic",
			},
		},
		{
			name: "amazon",
			host: scenario.DistroHost("amzn", "dnf", "firewalld"),
			expectedPlugins: []string{
				"distro-amazon",
				"firewall-firewalld",
				"dnf-automatic",
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			result := Build(tc.host, config.DefaultConfig())
			for _, id := range tc.expectedPlugins {
				if !hasPlugin(result, id) {
					t.Fatalf("expected %s in %+v", id, result.Plugins)
				}
			}
			if hasPlugin(result, "unattended-upgrades") {
				t.Fatalf("unexpected apt update plugin in %+v", result.Plugins)
			}
			if len(result.Warnings) != 0 {
				t.Fatalf("unexpected warnings: %v", result.Warnings)
			}
		})
	}
}

func TestBuildWarnsForUnsupportedDistroAndNoSSH(t *testing.T) {
	host := system.Host{
		OSID:            "void",
		OSName:          "Void Linux",
		PackageManager:  "unknown",
		InitSystem:      "unknown",
		FirewallBackend: "unknown",
		SSHService:      "sshd",
		SSHPort:         "22",
	}

	result := Build(host, config.DefaultConfig())
	if len(result.Warnings) != 2 {
		t.Fatalf("warnings = %v, want unsupported distro and no SSH warnings", result.Warnings)
	}
}

func TestBuildAddsWebProfile(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Profile = "web"

	result := Build(scenario.UbuntuHost(), cfg)
	if !hasPlugin(result, "web-profile") {
		t.Fatalf("expected web-profile in %+v", result.Plugins)
	}
}

func TestBuildAddsCustomPlugins(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Plugins.Custom = []config.CustomPlugin{{
		Name:  "tailscale-ssh",
		Probe: "command -v tailscale",
		Plan:  "ares-plugin-tailscale-ssh plan",
		Apply: "ares-plugin-tailscale-ssh apply",
	}}

	result := Build(scenario.UbuntuHost(), cfg)
	if !hasPlugin(result, "tailscale-ssh") {
		t.Fatalf("expected custom plugin in %+v", result.Plugins)
	}
}

func TestBuildAddsStrictProfile(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Profile = "strict"

	result := Build(scenario.UbuntuHost(), cfg)
	if !hasPlugin(result, "strict-profile") {
		t.Fatalf("expected strict-profile in %+v", result.Plugins)
	}
}

func TestBuildAddsProviderPlugin(t *testing.T) {
	host := scenario.UbuntuHost()
	host.Provider = "digitalocean"

	result := Build(host, config.DefaultConfig())
	if !hasPlugin(result, "provider-digitalocean") {
		t.Fatalf("expected provider plugin in %+v", result.Plugins)
	}
}

func hasPlugin(result Plan, id string) bool {
	for _, plugin := range result.Plugins {
		if plugin.ID == id {
			return true
		}
	}
	return false
}
