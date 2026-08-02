package plugins

import "testing"

func TestBehaviorIncludesCatalogVariant(t *testing.T) {
	plugin, ok := Find("firewall-ufw")
	if !ok {
		t.Fatal("missing firewall-ufw")
	}
	behavior := Behavior(plugin)
	if behavior.Name != "firewall" || behavior.Variant != "ufw" || behavior.Verifier != "firewall" {
		t.Fatalf("behavior = %+v", behavior)
	}
}
