package plugins

import (
	"fmt"

	"github.com/dotbrains/ares/marketplace"
)

type Plugin = marketplace.Plugin
type Catalog = marketplace.Catalog

func CatalogFromMarketplace() (Catalog, error) {
	catalog, err := marketplace.CatalogFromFiles()
	if err != nil {
		return Catalog{}, fmt.Errorf("loading embedded plugin marketplace: %w", err)
	}
	return Catalog(catalog), nil
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
	return marketplace.Find(marketplace.Catalog(catalog), id)
}
