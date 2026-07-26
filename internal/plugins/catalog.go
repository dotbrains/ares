package plugins

import (
	"fmt"
	"slices"
	"strings"
	"sync"

	"github.com/dotbrains/ares/marketplace"
)

type Plugin = marketplace.Plugin
type Catalog = marketplace.Catalog

type HostMatcher struct {
	OSID            string
	IDLike          []string
	Provider        string
	PackageManager  string
	FirewallBackend string
}

var catalogCache = struct {
	once    sync.Once
	catalog Catalog
	err     error
}{}

func CatalogFromMarketplace() (Catalog, error) {
	catalogCache.once.Do(func() {
		catalog, err := marketplace.CatalogFromFiles()
		if err != nil {
			catalogCache.err = fmt.Errorf("loading embedded plugin marketplace: %w", err)
			return
		}
		catalogCache.catalog = Catalog(catalog)
	})
	if catalogCache.err != nil {
		return Catalog{}, catalogCache.err
	}
	return cloneCatalog(catalogCache.catalog), nil
}

func cloneCatalog(catalog Catalog) Catalog {
	plugins := make([]Plugin, len(catalog.Plugins))
	for i, plugin := range catalog.Plugins {
		plugins[i] = clonePlugin(plugin)
	}
	return Catalog{Plugins: plugins}
}

func clonePlugin(plugin Plugin) Plugin {
	plugin.Aliases = slices.Clone(plugin.Aliases)
	plugin.Categories = slices.Clone(plugin.Categories)
	plugin.Requires = slices.Clone(plugin.Requires)
	plugin.Capabilities = slices.Clone(plugin.Capabilities)
	plugin.Distros = slices.Clone(plugin.Distros)
	plugin.Providers = slices.Clone(plugin.Providers)
	return plugin
}

func Builtins() ([]Plugin, error) {
	catalog, err := CatalogFromMarketplace()
	if err != nil {
		return nil, err
	}
	return catalog.Plugins, nil
}

func Find(id string) (Plugin, bool) {
	catalog, err := CatalogFromMarketplace()
	if err != nil {
		return Plugin{}, false
	}
	return marketplace.Find(marketplace.Catalog(catalog), id)
}

func DistroAdapter(host HostMatcher) (Plugin, bool) {
	return bestHostMatch(host, func(plugin Plugin) bool {
		return hasCategory(plugin, "distro")
	})
}

func FirstByCapability(host HostMatcher, capability string) (Plugin, bool) {
	return bestHostMatch(host, func(plugin Plugin) bool {
		return hasCapability(plugin, capability) && matchesRequirements(plugin, host)
	})
}

func ProviderAdvisory(host HostMatcher) (Plugin, bool) {
	catalog, err := CatalogFromMarketplace()
	if err != nil {
		return Plugin{}, false
	}
	for _, plugin := range catalog.Plugins {
		if hasCapability(plugin, "provider-advisory") && contains(plugin.Providers, host.Provider) {
			return plugin, true
		}
	}
	return Plugin{}, false
}

func bestHostMatch(host HostMatcher, accept func(Plugin) bool) (Plugin, bool) {
	catalog, err := CatalogFromMarketplace()
	if err != nil {
		return Plugin{}, false
	}
	if plugin, ok := firstHostMatch(catalog.Plugins, host, exactDistroMatch, accept); ok {
		return plugin, true
	}
	if plugin, ok := firstRelatedDistroMatch(catalog.Plugins, host, accept); ok {
		return plugin, true
	}
	if plugin, ok := firstHostMatch(catalog.Plugins, host, distroAgnosticMatch, accept); ok {
		return plugin, true
	}
	return Plugin{}, false
}

func firstHostMatch(available []Plugin, host HostMatcher, match func(Plugin, HostMatcher) bool, accept func(Plugin) bool) (Plugin, bool) {
	for _, plugin := range available {
		if accept(plugin) && match(plugin, host) {
			return plugin, true
		}
	}
	return Plugin{}, false
}

func exactDistroMatch(plugin Plugin, host HostMatcher) bool {
	return len(plugin.Distros) > 0 && contains(plugin.Distros, host.OSID)
}

func firstRelatedDistroMatch(available []Plugin, host HostMatcher, accept func(Plugin) bool) (Plugin, bool) {
	for _, like := range host.IDLike {
		for _, plugin := range available {
			if accept(plugin) && contains(plugin.Distros, like) {
				return plugin, true
			}
		}
	}
	return Plugin{}, false
}

func distroAgnosticMatch(plugin Plugin, _ HostMatcher) bool {
	return len(plugin.Distros) == 0
}

func matchesRequirements(plugin Plugin, host HostMatcher) bool {
	for _, requirement := range plugin.Requires {
		if !matchesRequirement(requirement, host) {
			return false
		}
	}
	return true
}

func matchesRequirement(requirement string, host HostMatcher) bool {
	for _, alternative := range strings.Split(requirement, "|") {
		switch alternative {
		case host.PackageManager, host.FirewallBackend:
			return true
		}
	}
	return false
}

func hasCategory(plugin Plugin, category string) bool {
	return contains(plugin.Categories, category)
}

func hasCapability(plugin Plugin, capability string) bool {
	return contains(plugin.Capabilities, capability)
}

func contains(values []string, value string) bool {
	for _, item := range values {
		if item == value {
			return true
		}
	}
	return false
}
