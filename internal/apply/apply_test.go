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
}

func testPlan() plan.Plan {
	host := system.Host{
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
	return plan.Build(host, config.DefaultConfig())
}
