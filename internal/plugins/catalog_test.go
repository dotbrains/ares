package plugins

import (
	"os"
	"path/filepath"
	"testing"
)

func TestBuiltinsHaveRequiredMetadata(t *testing.T) {
	seen := map[string]bool{}
	for _, plugin := range Builtins() {
		if plugin.ID == "" {
			t.Fatal("plugin with empty ID")
		}
		if seen[plugin.ID] {
			t.Fatalf("duplicate plugin ID %q", plugin.ID)
		}
		seen[plugin.ID] = true
		if plugin.Name == "" {
			t.Fatalf("%s missing name", plugin.ID)
		}
		if plugin.Kind != "builtin" {
			t.Fatalf("%s has kind %q", plugin.ID, plugin.Kind)
		}
		if plugin.Summary == "" {
			t.Fatalf("%s missing summary", plugin.ID)
		}
		if len(plugin.Categories) == 0 {
			t.Fatalf("%s missing categories", plugin.ID)
		}
		if len(plugin.Capabilities) == 0 {
			t.Fatalf("%s missing capabilities", plugin.ID)
		}
	}
}

func TestFind(t *testing.T) {
	plugin, ok := Find("ssh-hardening")
	if !ok {
		t.Fatal("expected ssh-hardening plugin")
	}
	if plugin.Name != "SSH hardening" {
		t.Fatalf("unexpected plugin name %q", plugin.Name)
	}

	if _, ok := Find("missing"); ok {
		t.Fatal("unexpected missing plugin")
	}
}

func TestDistroAdapterPrefersExactOSIDOverIDLike(t *testing.T) {
	plugin, ok := DistroAdapter(HostMatcher{
		OSID:           "ubuntu",
		IDLike:         []string{"debian"},
		PackageManager: "apt-get",
	})
	if !ok {
		t.Fatal("expected distro adapter")
	}
	if plugin.ID != "distro-ubuntu" {
		t.Fatalf("plugin ID = %q, want distro-ubuntu", plugin.ID)
	}
}

func TestDistroAdapterUsesIDLikeForFamilies(t *testing.T) {
	plugin, ok := DistroAdapter(HostMatcher{
		OSID:           "centos",
		IDLike:         []string{"rhel", "fedora"},
		PackageManager: "dnf",
	})
	if !ok {
		t.Fatal("expected distro adapter")
	}
	if plugin.ID != "distro-rhel" {
		t.Fatalf("plugin ID = %q, want distro-rhel", plugin.ID)
	}
}

func TestFirstByCapabilityMatchesRequirementsAndDistro(t *testing.T) {
	plugin, ok := FirstByCapability(HostMatcher{
		OSID:           "rocky",
		IDLike:         []string{"rhel", "fedora"},
		PackageManager: "dnf",
	}, "security-updates")
	if !ok {
		t.Fatal("expected security update plugin")
	}
	if plugin.ID != "dnf-automatic" {
		t.Fatalf("plugin ID = %q, want dnf-automatic", plugin.ID)
	}
}

func TestFirstByCapabilitySupportsRequirementAlternatives(t *testing.T) {
	plugin, ok := FirstByCapability(HostMatcher{
		OSID:           "opensuse-leap",
		PackageManager: "zypper",
	}, "firewall")
	if !ok {
		t.Fatal("expected firewall plugin")
	}
	if plugin.ID != "firewall-firewalld" {
		t.Fatalf("plugin ID = %q, want firewall-firewalld", plugin.ID)
	}
}

func TestMarketplaceFilesMatchPluginIDs(t *testing.T) {
	embedded, err := CatalogFromMarketplace()
	if err != nil {
		t.Fatal(err)
	}

	for _, plugin := range embedded.Plugins {
		path := filepath.Join("..", "..", "marketplace", "plugins", "builtin", plugin.ID+".toml")
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("%s missing source file %s: %v", plugin.ID, path, err)
		}
	}
}
