package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"regexp"
	"testing"
)

func TestDetectSnapshot(t *testing.T) {
	output := runSnapshotCommand(t, "detect")
	output = normalizeSnapshot(output)
	want := `os: Ubuntu 24.04 LTS (ubuntu 24.04)
provider: unknown
arch: <arch>
package manager: apt-get
init system: unknown
firewall backend: ufw
ssh service: ssh
ssh port: 2222
running over ssh: false
`
	if output != want {
		t.Fatalf("detect snapshot mismatch\nwant:\n%s\ngot:\n%s", want, output)
	}
}

func TestStatusSnapshot(t *testing.T) {
	output := runSnapshotCommand(t, "status")
	output = normalizeSnapshot(output)
	want := `os: Ubuntu 24.04 LTS (ubuntu 24.04)
provider: unknown
arch: <arch>
package manager: apt-get
init system: unknown
firewall backend: ufw
ssh service: ssh
ssh port: 2222
running over ssh: false

profile: basic
plugins: 6 selected
warnings: 1
`
	if output != want {
		t.Fatalf("status snapshot mismatch\nwant:\n%s\ngot:\n%s", want, output)
	}
}

func TestPluginsListSnapshot(t *testing.T) {
	output := runSnapshotCommand(t, "plugins", "list")
	if !regexp.MustCompile(`(?m)^ssh-hardening\s+builtin\s+Writes a managed sshd drop-in`).MatchString(output) {
		t.Fatalf("plugins list snapshot missing ssh-hardening:\n%s", output)
	}
	if !regexp.MustCompile(`(?m)^distro-ubuntu\s+builtin\s+Provides apt`).MatchString(output) {
		t.Fatalf("plugins list snapshot missing distro-ubuntu:\n%s", output)
	}
}

func runSnapshotCommand(t *testing.T, args ...string) string {
	t.Helper()
	rootDir := t.TempDir()
	sshDir := filepath.Join(rootDir, "etc", "ssh")
	if err := os.MkdirAll(sshDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sshDir, "sshd_config"), []byte("Port 2222\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("ARES_ROOT", rootDir)
	t.Setenv("ARES_OS_RELEASE", filepath.Join("..", "tests", "fixtures", "os-release", "ubuntu-24.04"))
	t.Setenv("HOME", t.TempDir())

	cmd := newRootCmd("snapshot")
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs(args)
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	return out.String()
}

func normalizeSnapshot(output string) string {
	return regexp.MustCompile(`(?m)^arch: .*$`).ReplaceAllString(output, "arch: <arch>")
}
