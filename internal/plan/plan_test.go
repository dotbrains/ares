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
