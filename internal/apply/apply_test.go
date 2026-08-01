package apply

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/dotbrains/ares/internal/config"
	"github.com/dotbrains/ares/internal/plan"
	"github.com/dotbrains/ares/internal/system"
)

func TestRunDryRunWritesReportAndDoesNotApply(t *testing.T) {
	root := t.TempDir()
	result, err := Run(testPlan(), Options{
		DryRun: true,
		Root:   root,
		Now:    time.Date(2026, 7, 25, 17, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Applied) != 0 {
		t.Fatalf("applied = %v, want none", result.Applied)
	}
	if len(result.Probed) != 0 || len(result.Verified) != 0 {
		t.Fatalf("dry-run probed/verified = %v/%v, want none", result.Probed, result.Verified)
	}
	if result.ReportPath == "" || result.UndoPlanPath == "" || result.LogPath == "" {
		t.Fatalf("missing report paths: %+v", result)
	}
	if _, err := os.Stat(result.ReportPath); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, "etc", "ssh", "sshd_config.d", "99-ares.conf")); !os.IsNotExist(err) {
		t.Fatalf("dry-run wrote SSH drop-in: %v", err)
	}
}

func TestRunApplyWritesManagedFilesAndReports(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "etc", "ssh"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "etc", "ssh", "sshd_config"), []byte("Port 22\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	result, err := Run(testPlan(), Options{
		Yes:  true,
		Root: root,
		Now:  time.Date(2026, 7, 25, 17, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatal(err)
	}

	for _, path := range []string{
		"etc/ssh/sshd_config.d/99-ares.conf",
		"etc/fail2ban/jail.d/ares-sshd.conf",
		"etc/apt/apt.conf.d/20auto-upgrades",
		"etc/sysctl.d/99-ares.conf",
		"var/log/ares/latest.json",
		"var/log/ares/undo-plan.txt",
	} {
		if _, err := os.Stat(filepath.Join(root, path)); err != nil {
			t.Fatalf("expected %s: %v", path, err)
		}
	}

	undo, err := os.ReadFile(result.UndoPlanPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(undo), "Ensure SSH port 22/tcp remains allowed") {
		t.Fatalf("undo plan missing SSH port: %s", undo)
	}
	if len(result.Probed) == 0 || len(result.Verified) == 0 {
		t.Fatalf("expected probe and verify results: %+v", result)
	}
}

func TestRunApplyWritesRHELManagedFilesAndReports(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "etc", "ssh"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "etc", "ssh", "sshd_config"), []byte("Port 22\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	result, err := Run(rhelPlan(), Options{
		Yes:  true,
		Root: root,
		Now:  time.Date(2026, 7, 25, 17, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatal(err)
	}

	for _, path := range []string{
		"etc/ssh/sshd_config.d/99-ares.conf",
		"etc/fail2ban/jail.d/ares-sshd.conf",
		"etc/dnf/automatic.conf",
		"etc/sysctl.d/99-ares.conf",
		"var/log/ares/latest.json",
		"var/log/ares/undo-plan.txt",
	} {
		if _, err := os.Stat(filepath.Join(root, path)); err != nil {
			t.Fatalf("expected %s: %v", path, err)
		}
	}
	if len(result.Verified) == 0 {
		t.Fatalf("expected verification results")
	}
}

func TestRunApplyStrictProfileOverwritesFail2banDefaults(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "etc", "ssh"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "etc", "ssh", "sshd_config"), []byte("Port 22\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := config.DefaultConfig()
	cfg.Profile = "strict"
	result, err := Run(plan.Build(testHost(), cfg), Options{
		Yes:  true,
		Root: root,
		Now:  time.Date(2026, 7, 25, 17, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(root, "etc", "fail2ban", "jail.d", "ares-sshd.conf"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "maxretry = 3") {
		t.Fatalf("strict jail not applied: %s", data)
	}
	if len(result.Skipped) == 0 {
		t.Fatalf("expected advisory skipped item")
	}
}

func TestRunApplyWebProfileWritesNftablesWebRules(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "etc", "ssh"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "etc", "ssh", "sshd_config"), []byte("Port 2222\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := config.DefaultConfig()
	cfg.Profile = "web"
	host := system.Host{
		OSID:            "arch",
		OSName:          "Arch Linux",
		PackageManager:  "pacman",
		InitSystem:      "systemd",
		FirewallBackend: "nftables",
		SSHService:      "sshd",
		SSHPort:         "2222",
		RunningOverSSH:  true,
	}
	result, err := Run(plan.Build(host, cfg), Options{
		Yes:  true,
		Root: root,
		Now:  time.Date(2026, 7, 25, 17, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(filepath.Join(root, "etc", "nftables.conf"))
	if err != nil {
		t.Fatal(err)
	}
	rules := string(data)
	for _, want := range []string{"tcp dport 2222 accept", "tcp dport 80 accept", "tcp dport 443 accept"} {
		if !strings.Contains(rules, want) {
			t.Fatalf("nftables rules missing %q:\n%s", want, rules)
		}
	}
	if !contains(result.Applied, "allowed HTTP and HTTPS") {
		t.Fatalf("missing web profile applied item: %+v", result)
	}
	if !contains(result.Applied, "would run: nft -c -f /etc/nftables.conf") {
		t.Fatalf("missing nftables validation command: %+v", result.Applied)
	}
	if !contains(result.Applied, "would run: nft -f /etc/nftables.conf") {
		t.Fatalf("missing nftables load command: %+v", result.Applied)
	}
}

func TestRunApplyProviderAdvisoryIsReported(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "etc", "ssh"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "etc", "ssh", "sshd_config"), []byte("Port 22\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	host := testHost()
	host.Provider = "digitalocean"
	result, err := Run(plan.Build(host, config.DefaultConfig()), Options{
		Yes:  true,
		Root: root,
		Now:  time.Date(2026, 7, 25, 17, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatal(err)
	}
	if !contains(result.Applied, "provider-digitalocean: recorded provider advisory") {
		t.Fatalf("missing provider applied item: %+v", result)
	}
	if !contains(result.Verified, "provider-digitalocean: advisory recorded") {
		t.Fatalf("missing provider verification: %+v", result)
	}
}

func TestInstallCommandUsesPackageManagerSyntax(t *testing.T) {
	cases := []struct {
		packageManager string
		wantName       string
		wantArgs       []string
	}{
		{packageManager: "apt-get", wantName: "apt-get", wantArgs: []string{"install", "-y", "fail2ban"}},
		{packageManager: "dnf", wantName: "dnf", wantArgs: []string{"install", "-y", "fail2ban"}},
		{packageManager: "yum", wantName: "yum", wantArgs: []string{"install", "-y", "fail2ban"}},
		{packageManager: "pacman", wantName: "pacman", wantArgs: []string{"-S", "--needed", "--noconfirm", "fail2ban"}},
		{packageManager: "zypper", wantName: "zypper", wantArgs: []string{"--non-interactive", "install", "fail2ban"}},
		{packageManager: "apk", wantName: "apk", wantArgs: []string{"add", "fail2ban"}},
	}

	for _, tc := range cases {
		t.Run(tc.packageManager, func(t *testing.T) {
			name, args, err := installCommand(tc.packageManager, "fail2ban")
			if err != nil {
				t.Fatal(err)
			}
			if name != tc.wantName {
				t.Fatalf("name = %q, want %q", name, tc.wantName)
			}
			if strings.Join(args, " ") != strings.Join(tc.wantArgs, " ") {
				t.Fatalf("args = %q, want %q", strings.Join(args, " "), strings.Join(tc.wantArgs, " "))
			}
		})
	}
}

func testPlan() plan.Plan {
	return plan.Build(testHost(), config.DefaultConfig())
}

func testHost() system.Host {
	return system.Host{
		OSID:            "ubuntu",
		OSName:          "Ubuntu 24.04 LTS",
		OSVersion:       "24.04",
		PackageManager:  "apt-get",
		InitSystem:      "systemd",
		FirewallBackend: "ufw",
		SSHService:      "ssh",
		SSHPort:         "22",
		RunningOverSSH:  true,
	}
}

func rhelPlan() plan.Plan {
	host := system.Host{
		OSID:            "rocky",
		OSName:          "Rocky Linux 9.4",
		OSVersion:       "9.4",
		PackageManager:  "dnf",
		InitSystem:      "systemd",
		FirewallBackend: "firewalld",
		SSHService:      "sshd",
		SSHPort:         "22",
		RunningOverSSH:  true,
	}
	return plan.Build(host, config.DefaultConfig())
}

func contains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
