package scenario

import "testing"

func TestSmokeFixturesDeclareExpectedPlugins(t *testing.T) {
	for _, fixture := range SmokeFixtures() {
		if fixture.Fixture == "" || fixture.FirewallPlugin == "" || fixture.UpdatesPlugin == "" {
			t.Fatalf("incomplete fixture expectation: %+v", fixture)
		}
	}
}

func TestContainerExpectPluginMatchesShellMatrix(t *testing.T) {
	if got := ContainerExpectPlugin("ubuntu"); got != "firewall-ufw" {
		t.Fatalf("ubuntu = %q", got)
	}
	if got := ContainerExpectPlugin("arch"); got != "firewall-nftables" {
		t.Fatalf("arch = %q", got)
	}
	if got := ContainerExpectPlugin("rocky"); got != "firewall-firewalld" {
		t.Fatalf("rocky = %q", got)
	}
}
