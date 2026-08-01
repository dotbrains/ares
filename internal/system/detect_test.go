package system

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestReadOSRelease(t *testing.T) {
	path := filepath.Join(t.TempDir(), "os-release")
	data := []byte("ID=ubuntu\nPRETTY_NAME=\"Ubuntu 24.04 LTS\"\nVERSION_ID=\"24.04\"\nID_LIKE=debian\n")
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}

	values, err := readOSRelease(path)
	if err != nil {
		t.Fatal(err)
	}
	if values["ID"] != "ubuntu" {
		t.Fatalf("ID = %q", values["ID"])
	}
	if values["PRETTY_NAME"] != "Ubuntu 24.04 LTS" {
		t.Fatalf("PRETTY_NAME = %q", values["PRETTY_NAME"])
	}
}

func TestDetectSSHPort(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sshd_config")
	if err := os.WriteFile(path, []byte("# Port 22\nPort 2222\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	if got := detectSSHPort(path); got != "2222" {
		t.Fatalf("detectSSHPort() = %q, want 2222", got)
	}
}

func TestDetectSSHPortDefault(t *testing.T) {
	if got := detectSSHPort(filepath.Join(t.TempDir(), "missing")); got != "22" {
		t.Fatalf("detectSSHPort() = %q, want 22", got)
	}
}

func TestDetectSSHPortIgnoresInvalidPort(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sshd_config")
	if err := os.WriteFile(path, []byte("Port not-a-port\nPort 70000\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	if got := detectSSHPort(path); got != "22" {
		t.Fatalf("detectSSHPort() = %q, want 22", got)
	}
}

func TestRootPath(t *testing.T) {
	got := rootPath("/tmp/ares-root", "/etc/ssh/sshd_config")
	want := filepath.Join("/tmp/ares-root", "etc", "ssh", "sshd_config")
	if got != want {
		t.Fatalf("rootPath() = %q, want %q", got, want)
	}
}

func TestProviderFromText(t *testing.T) {
	cases := map[string]string{
		"DigitalOcean Droplet": "digitalocean",
		"Hetzner":              "hetzner",
		"Vultr":                "vultr",
		"Akamai Linode":        "linode",
		"Amazon Lightsail":     "lightsail",
		"unknown board":        "unknown",
	}
	for input, want := range cases {
		if got := providerFromText(input); got != want {
			t.Fatalf("providerFromText(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestNormalizeProvider(t *testing.T) {
	if got := normalizeProvider("do"); got != "digitalocean" {
		t.Fatalf("normalizeProvider(do) = %q", got)
	}
	if got := normalizeProvider("AWS-Lightsail"); got != "lightsail" {
		t.Fatalf("normalizeProvider(AWS-Lightsail) = %q", got)
	}
}

func TestPackageAndFirewallDefaultsIgnoreHostCommandsInFixtureMode(t *testing.T) {
	host := Host{OSID: "rocky"}
	host.PackageManager = packageManager(host, false)
	host.FirewallBackend = firewallBackend(host, false)

	if host.PackageManager != "dnf" {
		t.Fatalf("PackageManager = %q, want dnf", host.PackageManager)
	}
	if host.FirewallBackend != "firewalld" {
		t.Fatalf("FirewallBackend = %q, want firewalld", host.FirewallBackend)
	}
}

func TestPackageAndFirewallDefaultsForNewDistros(t *testing.T) {
	cases := []struct {
		osID            string
		packageManager  string
		initSystem      string
		firewallBackend string
		sshService      string
	}{
		{osID: "arch", packageManager: "pacman", initSystem: "systemd", firewallBackend: "nftables", sshService: "sshd"},
		{osID: "opensuse-leap", packageManager: "zypper", initSystem: "systemd", firewallBackend: "firewalld", sshService: "sshd"},
		{osID: "alpine", packageManager: "apk", initSystem: "openrc", firewallBackend: "nftables", sshService: "sshd"},
		{osID: "ol", packageManager: "dnf", initSystem: "systemd", firewallBackend: "firewalld", sshService: "sshd"},
		{osID: "amzn", packageManager: "dnf", initSystem: "systemd", firewallBackend: "firewalld", sshService: "sshd"},
	}

	for _, tc := range cases {
		t.Run(tc.osID, func(t *testing.T) {
			host := Host{OSID: tc.osID}
			host.PackageManager = packageManager(host, false)
			host.InitSystem = initSystem(host, t.TempDir())
			host.FirewallBackend = firewallBackend(host, false)
			host.SSHService = sshServiceName(host)

			if host.PackageManager != tc.packageManager {
				t.Fatalf("PackageManager = %q, want %q", host.PackageManager, tc.packageManager)
			}
			if host.InitSystem != tc.initSystem {
				t.Fatalf("InitSystem = %q, want %q", host.InitSystem, tc.initSystem)
			}
			if host.FirewallBackend != tc.firewallBackend {
				t.Fatalf("FirewallBackend = %q, want %q", host.FirewallBackend, tc.firewallBackend)
			}
			if host.SSHService != tc.sshService {
				t.Fatalf("SSHService = %q, want %q", host.SSHService, tc.sshService)
			}
		})
	}
}

func TestDetectWithProberUsesInjectedHostFacts(t *testing.T) {
	prober := fakeProber{
		env: map[string]string{
			"ARES_ROOT": "/fixture",
		},
		files: map[string]string{
			"/fixture/etc/os-release":              "ID=ubuntu\nPRETTY_NAME=\"Ubuntu 24.04 LTS\"\nVERSION_ID=\"24.04\"\nID_LIKE=debian\n",
			"/fixture/etc/ssh/sshd_config":         "Port 2200\n",
			"/fixture/sys/class/dmi/id/sys_vendor": "DigitalOcean\n",
		},
		stats: map[string]bool{
			"/fixture/run/systemd/system": true,
		},
		arch: "arm64",
	}

	host, err := DetectWithProber(prober)
	if err != nil {
		t.Fatal(err)
	}
	if host.Provider != "digitalocean" || host.SSHPort != "2200" || host.Architecture != "arm64" {
		t.Fatalf("unexpected host: %+v", host)
	}
	if host.PackageManager != "apt-get" || host.InitSystem != "systemd" || host.FirewallBackend != "ufw" {
		t.Fatalf("unexpected distro defaults: %+v", host)
	}
	if host.Facts["package_manager"].Source != "catalog default" || host.Facts["ssh_port"].Confidence != "medium" {
		t.Fatalf("unexpected fact metadata: %+v", host.Facts)
	}
}

type fakeProber struct {
	env   map[string]string
	files map[string]string
	stats map[string]bool
	paths map[string]bool
	arch  string
}

func (prober fakeProber) Env(name string) string {
	return prober.env[name]
}

func (prober fakeProber) ReadFile(path string) ([]byte, error) {
	if value, ok := prober.files[path]; ok {
		return []byte(value), nil
	}
	return nil, errors.New("missing")
}

func (prober fakeProber) Stat(path string) error {
	if prober.stats[path] {
		return nil
	}
	return errors.New("missing")
}

func (prober fakeProber) LookPath(name string) bool {
	return prober.paths[name]
}

func (prober fakeProber) GOARCH() string {
	return prober.arch
}
