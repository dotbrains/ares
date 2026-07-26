package plugins

import (
	"os"
	"testing"

	"github.com/BurntSushi/toml"
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

func TestPublicMarketplaceMatchesEmbeddedCatalog(t *testing.T) {
	data, err := os.ReadFile("../../marketplace/plugins.toml")
	if err != nil {
		t.Fatal(err)
	}
	var public Catalog
	if _, err := toml.Decode(string(data), &public); err != nil {
		t.Fatal(err)
	}

	embedded, err := CatalogFromMarketplace()
	if err != nil {
		t.Fatal(err)
	}
	if len(public.Plugins) != len(embedded.Plugins) {
		t.Fatalf("public plugins = %d, embedded plugins = %d", len(public.Plugins), len(embedded.Plugins))
	}
	for i := range public.Plugins {
		if public.Plugins[i].ID != embedded.Plugins[i].ID {
			t.Fatalf("plugin %d public ID = %q, embedded ID = %q", i, public.Plugins[i].ID, embedded.Plugins[i].ID)
		}
	}
}
