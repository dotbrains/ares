package plugins

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
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

func TestManagedFilePluginsDeclareTransactionFacts(t *testing.T) {
	builtins, err := Builtins()
	if err != nil {
		t.Fatal(err)
	}
	for _, plugin := range builtins {
		switch plugin.ID {
		case "ssh-hardening", "firewall-nftables", "fail2ban", "unattended-upgrades", "dnf-automatic", "sysctl-baseline", "strict-profile":
			if len(plugin.ManagedFiles) == 0 {
				t.Fatalf("%s missing managed_files metadata", plugin.ID)
			}
			if len(plugin.RollbackSteps) == 0 {
				t.Fatalf("%s missing rollback_steps metadata", plugin.ID)
			}
		}
	}
}

func TestBuiltinsDeclareBehaviorDescriptors(t *testing.T) {
	builtins, err := Builtins()
	if err != nil {
		t.Fatal(err)
	}
	for _, plugin := range builtins {
		if plugin.Behavior == "" {
			t.Fatalf("%s missing behavior descriptor", plugin.ID)
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

func TestDistroFixturesResolveRequiredCapabilities(t *testing.T) {
	fixtures, err := filepath.Glob(filepath.Join("..", "..", "tests", "fixtures", "os-release", "*"))
	if err != nil {
		t.Fatal(err)
	}
	if len(fixtures) == 0 {
		t.Fatal("expected distro fixtures")
	}
	for _, fixture := range fixtures {
		t.Run(filepath.Base(fixture), func(t *testing.T) {
			values := readFixtureOSRelease(t, fixture)
			host := HostMatcher{
				OSID:   values["ID"],
				IDLike: strings.Fields(values["ID_LIKE"]),
			}
			distro, ok := DistroAdapter(host)
			if !ok {
				t.Fatalf("missing distro adapter for %s", fixture)
			}
			host.PackageManager = distro.PackageManager
			host.FirewallBackend = distro.FirewallBackend
			if _, ok := FirstByCapability(host, "firewall"); !ok {
				t.Fatalf("missing firewall capability for %+v", host)
			}
			if _, ok := FirstByCapability(host, "security-updates"); !ok {
				t.Fatalf("missing security-updates capability for %+v", host)
			}
		})
	}
}

func readFixtureOSRelease(t *testing.T, path string) map[string]string {
	t.Helper()
	file, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()

	values := map[string]string{}
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if ok {
			values[key] = strings.Trim(strings.TrimSpace(value), `"`)
		}
	}
	if err := scanner.Err(); err != nil {
		t.Fatal(err)
	}
	return values
}
