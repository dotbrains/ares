package marketplace

import (
	"strings"
	"testing"
)

func TestCatalogValidationRejectsUnknownRequirement(t *testing.T) {
	err := validateCatalog(Catalog{Plugins: []Plugin{{
		ID:           "bad",
		Kind:         "builtin",
		Behavior:     "sysctl",
		Summary:      "Bad plugin",
		Categories:   []string{"hardening"},
		Capabilities: []string{"sysctl"},
		Requires:     []string{"missing-capability"},
	}}})
	if err == nil {
		t.Fatal("expected validation error")
	}
	if !strings.Contains(err.Error(), `unknown requirement "missing-capability"`) {
		t.Fatalf("error = %v", err)
	}
}

func TestCatalogValidationAcceptsLoadedMarketplace(t *testing.T) {
	if _, err := CatalogFromFiles(); err != nil {
		t.Fatal(err)
	}
}

func TestValidateBehaviorRejectsUnknownVariant(t *testing.T) {
	err := ValidateBehavior(Plugin{ID: "bad", Behavior: "firewall", BehaviorVariant: "iptables"})
	if err == nil {
		t.Fatal("expected validation error")
	}
	if !strings.Contains(err.Error(), `unknown behavior variant "iptables"`) {
		t.Fatalf("error = %v", err)
	}
}
