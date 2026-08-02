package scenario

type FixtureExpectation struct {
	Fixture        string
	FirewallPlugin string
	UpdatesPlugin  string
}

func SmokeFixtures() []FixtureExpectation {
	return []FixtureExpectation{
		{Fixture: "ubuntu-24.04", FirewallPlugin: "firewall-ufw", UpdatesPlugin: "unattended-upgrades"},
		{Fixture: "debian-12", FirewallPlugin: "firewall-ufw", UpdatesPlugin: "unattended-upgrades"},
		{Fixture: "debian-11", FirewallPlugin: "firewall-ufw", UpdatesPlugin: "unattended-upgrades"},
		{Fixture: "rocky-9", FirewallPlugin: "firewall-firewalld", UpdatesPlugin: "dnf-automatic"},
		{Fixture: "fedora", FirewallPlugin: "firewall-firewalld", UpdatesPlugin: "dnf-automatic"},
		{Fixture: "arch", FirewallPlugin: "firewall-nftables", UpdatesPlugin: "pacman-upgrade"},
		{Fixture: "opensuse-leap", FirewallPlugin: "firewall-firewalld", UpdatesPlugin: "zypper-patches"},
		{Fixture: "alpine", FirewallPlugin: "firewall-nftables", UpdatesPlugin: "apk-upgrade"},
		{Fixture: "oracle-9", FirewallPlugin: "firewall-firewalld", UpdatesPlugin: "dnf-automatic"},
		{Fixture: "amazon-2023", FirewallPlugin: "firewall-firewalld", UpdatesPlugin: "dnf-automatic"},
	}
}

func ContainerExpectPlugin(id string) string {
	switch id {
	case "ubuntu", "debian":
		return "firewall-ufw"
	case "arch", "alpine":
		return "firewall-nftables"
	default:
		return "firewall-firewalld"
	}
}
