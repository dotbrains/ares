package marketplace

import (
	"strings"
	"testing"
)

func TestCatalogValidationRejectsUnknownRequirement(t *testing.T) {
	err := validateCatalog(Catalog{Plugins: []Plugin{{
		ID:           "bad",
		Kind:         "builtin",
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
