package apply

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/dotbrains/ares/internal/config"
	"github.com/dotbrains/ares/internal/plan"
	"github.com/dotbrains/ares/internal/system"
)

func TestRunPreservesFirstBackupWhenFileChangesTwice(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "etc", "ssh"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "etc", "ssh", "sshd_config"), []byte("Port 2222\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "etc", "nftables.conf"), []byte("original rules\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	now := time.Date(2026, 7, 25, 17, 0, 0, 0, time.UTC)
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

	if _, err := Run(plan.Build(host, cfg), Options{Yes: true, Root: root, Now: now}); err != nil {
		t.Fatal(err)
	}

	backup, err := os.ReadFile(filepath.Join(root, "etc", "nftables.conf.ares."+now.Format("20060102-150405")+".bak"))
	if err != nil {
		t.Fatal(err)
	}
	if string(backup) != "original rules\n" {
		t.Fatalf("backup was overwritten by later nftables write: %q", backup)
	}
}
