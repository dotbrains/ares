package system

import "testing"

func TestProjectObservedFieldsUsesObservationValues(t *testing.T) {
	host := Host{}
	host.Observe("package_manager", "apt-get", Fact{Source: "test", Confidence: "high"})
	host.Observe("firewall_backend", "ufw", Fact{Source: "test", Confidence: "high"})
	host.ProjectObservedFields()
	if host.PackageManager != "apt-get" || host.FirewallBackend != "ufw" {
		t.Fatalf("projected host = %+v", host)
	}
}
