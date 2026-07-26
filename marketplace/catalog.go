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
	PackageManager  string `toml:"package_manager"`
	InitSystem      string `toml:"init_system"`
	FirewallBackend string `toml:"firewall_backend"`
	SSHService      string `toml:"ssh_service"`
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
