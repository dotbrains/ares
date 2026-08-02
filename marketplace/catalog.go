package marketplace

import (
	"embed"
	"fmt"
	"io/fs"
	"path/filepath"
	"sort"
	"strings"

	"github.com/BurntSushi/toml"
)

//go:embed plugins
var pluginFiles embed.FS

type Plugin struct {
	ID              string
	Aliases         []string
	Name            string
	Kind            string
	Summary         string
	Categories      []string
	Requires        []string
	Capabilities    []string
	Distros         []string
	Providers       []string
	Behavior        string
	BehaviorVariant string `toml:"behavior_variant"`
	Verifier        string
	ManagedFiles    []string `toml:"managed_files"`
	BackupFiles     []string `toml:"backup_files"`
	RollbackSteps   []string `toml:"rollback_steps"`
	PackageManager  string   `toml:"package_manager"`
	InitSystem      string   `toml:"init_system"`
	FirewallBackend string   `toml:"firewall_backend"`
	SSHService      string   `toml:"ssh_service"`
	Probe           string
	Plan            string
	Apply           string
	Verify          string
	Rollback        string
	Config          string
	TimeoutSeconds  int
}

type Catalog struct {
	Plugins []Plugin
}

func CatalogFromFiles() (Catalog, error) {
	paths, err := pluginPaths()
	if err != nil {
		return Catalog{}, err
	}

	catalog := Catalog{Plugins: make([]Plugin, 0, len(paths))}
	for _, path := range paths {
		plugin, err := pluginFromFile(path)
		if err != nil {
			return Catalog{}, err
		}
		catalog.Plugins = append(catalog.Plugins, plugin)
	}
	if err := validateCatalog(catalog); err != nil {
		return Catalog{}, err
	}
	return catalog, nil
}

func pluginPaths() ([]string, error) {
	var paths []string
	if err := fs.WalkDir(pluginFiles, "plugins", func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() || filepath.Ext(path) != ".toml" {
			return nil
		}
		paths = append(paths, path)
		return nil
	}); err != nil {
		return nil, fmt.Errorf("walking embedded plugin marketplace: %w", err)
	}
	sort.Strings(paths)
	return paths, nil
}

func pluginFromFile(path string) (Plugin, error) {
	data, err := pluginFiles.ReadFile(path)
	if err != nil {
		return Plugin{}, fmt.Errorf("reading embedded plugin %s: %w", path, err)
	}

	var plugin Plugin
	if _, err := toml.Decode(string(data), &plugin); err != nil {
		return Plugin{}, fmt.Errorf("parsing embedded plugin %s: %w", path, err)
	}
	if err := validatePluginPath(path, plugin); err != nil {
		return Plugin{}, err
	}
	return plugin, nil
}

func validatePluginPath(path string, plugin Plugin) error {
	if plugin.ID == "" {
		return fmt.Errorf("plugin file %s is missing id", path)
	}
	want := strings.TrimSuffix(filepath.Base(path), ".toml")
	if plugin.ID != want {
		return fmt.Errorf("plugin file %s declares id %q, expected %q", path, plugin.ID, want)
	}
	return nil
}

func validateCatalog(catalog Catalog) error {
	ids := map[string]string{}
	capabilities := map[string]string{}
	for _, plugin := range catalog.Plugins {
		if previous, ok := ids[plugin.ID]; ok {
			return fmt.Errorf("duplicate plugin id %q declared by %s and %s", plugin.ID, previous, plugin.ID)
		}
		ids[plugin.ID] = plugin.ID
		for _, capability := range plugin.Capabilities {
			capabilities[capability] = capability
		}
	}
	for _, plugin := range catalog.Plugins {
		if plugin.Kind == "" {
			return fmt.Errorf("plugin %s is missing kind", plugin.ID)
		}
		if plugin.Summary == "" {
			return fmt.Errorf("plugin %s is missing summary", plugin.ID)
		}
		if plugin.Behavior == "" {
			return fmt.Errorf("plugin %s is missing behavior", plugin.ID)
		}
		if err := validateKnownValues(plugin, ids, capabilities); err != nil {
			return err
		}
		if strings.HasPrefix(plugin.ID, "provider-") && !contains(plugin.Capabilities, "provider-advisory") {
			return fmt.Errorf("provider plugin %s must declare provider-advisory capability", plugin.ID)
		}
	}
	return nil
}

func validateKnownValues(plugin Plugin, ids map[string]string, capabilities map[string]string) error {
	for _, category := range plugin.Categories {
		switch category {
		case "advisory", "distro", "firewall", "hardening", "kernel", "network", "profile", "provider", "ssh", "updates":
		default:
			return fmt.Errorf("plugin %s has unknown category %q", plugin.ID, category)
		}
	}
	for _, capability := range plugin.Capabilities {
		switch capability {
		case "fail2ban", "firewall", "package-manager", "profile-strict", "profile-web", "provider-advisory", "security-updates", "service-manager", "ssh-hardening", "ssh-service", "sysctl":
		default:
			return fmt.Errorf("plugin %s has unknown capability %q", plugin.ID, capability)
		}
	}
	if err := ValidateBehavior(plugin); err != nil {
		return err
	}
	for _, requirement := range plugin.Requires {
		for _, alternative := range strings.Split(requirement, "|") {
			switch alternative {
			case "apt-get", "dnf", "yum", "pacman", "zypper", "apk", "ufw", "firewalld", "nftables":
				continue
			}
			if _, ok := ids[alternative]; !ok {
				if _, ok := capabilities[alternative]; !ok {
					return fmt.Errorf("plugin %s has unknown requirement %q", plugin.ID, alternative)
				}
			}
		}
	}
	return nil
}

func Find(catalog Catalog, id string) (Plugin, bool) {
	for _, plugin := range catalog.Plugins {
		if plugin.ID == id || contains(plugin.Aliases, id) {
			return plugin, true
		}
	}
	return Plugin{}, false
}

func contains(values []string, value string) bool {
	for _, item := range values {
		if item == value {
			return true
		}
	}
	return false
}
