package plugins

import (
	_ "embed"
	"fmt"

	"github.com/BurntSushi/toml"
)

//go:embed marketplace.toml
var marketplaceData string

type Plugin struct {
	ID           string
	Aliases      []string
	Name         string
	Kind         string
	Summary      string
	Categories   []string
	Requires     []string
	Capabilities []string
	Distros      []string
	Probe        string
	Plan         string
	Apply        string
	Rollback     string
	Config       string
}

type Catalog struct {
	Plugins []Plugin
}

func CatalogFromMarketplace() (Catalog, error) {
	var catalog Catalog
	if _, err := toml.Decode(marketplaceData, &catalog); err != nil {
		return Catalog{}, fmt.Errorf("parsing embedded plugin marketplace: %w", err)
	}
	return catalog, nil
}

func Builtins() []Plugin {
	catalog, err := CatalogFromMarketplace()
	if err != nil {
		panic(err)
	}
	return catalog.Plugins
}

func Find(id string) (Plugin, bool) {
	catalog, err := CatalogFromMarketplace()
	if err != nil {
		return Plugin{}, false
	}
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
