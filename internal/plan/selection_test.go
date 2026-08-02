package plan

import (
	"testing"

	"github.com/dotbrains/ares/internal/config"
	"github.com/dotbrains/ares/internal/plugins"
	"github.com/dotbrains/ares/internal/scenario"
)

func TestSelectionExpandsProfileAndCustomPlugins(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Profile = "strict"
	cfg.Plugins.Custom = []config.CustomPlugin{{Name: "tailscale-ssh", Apply: "ares-plugin apply"}}
	selected := Selection{Host: scenario.UbuntuHost(), Config: cfg}.Plugins()
	if !hasID(selected, "strict-profile") || !hasID(selected, "tailscale-ssh") {
		t.Fatalf("selected = %+v", selected)
	}
}

func hasID(selected []plugins.Plugin, id string) bool {
	for _, plugin := range selected {
		if plugin.ID == id {
			return true
		}
	}
	return false
}
