package plugins

import (
	"os"
	"path/filepath"
	"testing"
)

func TestBuiltinsHaveRequiredMetadata(t *testing.T) {
	builtins, err := Builtins()
	if err != nil {
		t.Fatal(err)
	}
	seen := map[string]bool{}
	for _, plugin := range builtins {
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
		if contains(plugin.Categories, "distro") {
			if plugin.PackageManager == "" {
				t.Fatalf("%s missing package manager default", plugin.ID)
			}
			if plugin.InitSystem == "" {
				t.Fatalf("%s missing init system default", plugin.ID)
			}
			if plugin.FirewallBackend == "" {
				t.Fatalf("%s missing firewall backend default", plugin.ID)
			}
			if plugin.SSHService == "" {
				t.Fatalf("%s missing SSH service default", plugin.ID)
			}
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

func TestCatalogFromMarketplaceReturnsIndependentCatalog(t *testing.T) {
	catalog, err := CatalogFromMarketplace()
	if err != nil {
		t.Fatal(err)
	}
	if len(catalog.Plugins) == 0 {
		t.Fatal("expected plugins")
	}
	catalog.Plugins[0].ID = "mutated"
	catalog.Plugins[0].Aliases = append(catalog.Plugins[0].Aliases, "mutated")

	fresh, err := CatalogFromMarketplace()
	if err != nil {
		t.Fatal(err)
	}
	if fresh.Plugins[0].ID == "mutated" {
		t.Fatal("catalog cache was mutated through returned plugin slice")
	}
	if contains(fresh.Plugins[0].Aliases, "mutated") {
		t.Fatal("catalog cache was mutated through returned metadata slice")
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

func TestProviderAdvisoryMatchesProviderMetadata(t *testing.T) {
	plugin, ok := ProviderAdvisory(HostMatcher{Provider: "digitalocean"})
	if !ok {
		t.Fatal("expected provider advisory")
	}
	if plugin.ID != "provider-digitalocean" {
		t.Fatalf("plugin ID = %q, want provider-digitalocean", plugin.ID)
	}
}

func TestProviderAdvisoriesDeclareProviders(t *testing.T) {
	builtins, err := Builtins()
	if err != nil {
		t.Fatal(err)
	}
	for _, plugin := range builtins {
		if !contains(plugin.Capabilities, "provider-advisory") {
			continue
		}
		if len(plugin.Providers) == 0 {
			t.Fatalf("%s missing providers metadata", plugin.ID)
		}
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
