package system

import (
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
