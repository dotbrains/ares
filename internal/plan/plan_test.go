package plan

import (
	"testing"

	"github.com/dotbrains/ares/internal/config"
	"github.com/dotbrains/ares/internal/system"
)

func TestBuildAddsDistroPlugin(t *testing.T) {
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
	}

	result := Build(host, config.DefaultConfig())
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

	result := Build(ubuntuHost(), cfg)
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

	result := Build(ubuntuHost(), cfg)
	if !hasPlugin(result, "tailscale-ssh") {
		t.Fatalf("expected custom plugin in %+v", result.Plugins)
	}
}

func ubuntuHost() system.Host {
	return system.Host{
		OSID:            "ubuntu",
		OSName:          "Ubuntu 24.04 LTS",
		OSVersion:       "24.04",
		PackageManager:  "apt-get",
		InitSystem:      "systemd",
		FirewallBackend: "ufw",
		SSHService:      "ssh",
		SSHPort:         "22",
		RunningOverSSH:  true,
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
