package apply

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestRollbackLastRemovesManagedFilesAndRestoresBackups(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "etc", "ssh", "sshd_config.d"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "etc", "ssh", "sshd_config.d", "99-ares.conf"), []byte("managed"), 0o644); err != nil {
		t.Fatal(err)
	}
	backup := filepath.Join(root, "etc", "ssh", "sshd_config.ares.20260725-170000.bak")
	if err := os.WriteFile(backup, []byte("Port 2222\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	result, err := RollbackLast(RollbackOptions{
		Yes:  true,
		Root: root,
		Now:  time.Date(2026, 7, 25, 18, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, "etc", "ssh", "sshd_config.d", "99-ares.conf")); !os.IsNotExist(err) {
		t.Fatalf("managed SSH drop-in still exists: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(root, "etc", "ssh", "sshd_config"))
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "Port 2222\n" {
		t.Fatalf("restored sshd_config = %q", data)
	}
	if len(result.Applied) == 0 {
		t.Fatalf("expected rollback applied items: %+v", result)
	}
	if _, err := os.Stat(result.ReportPath); err != nil {
		t.Fatalf("expected rollback report: %v", err)
	}
}

func TestRollbackLastRunsCustomRollbackFromLatestReport(t *testing.T) {
	root := t.TempDir()
	reportDir := filepath.Join(root, "var", "log", "ares")
	if err := os.MkdirAll(reportDir, 0o755); err != nil {
		t.Fatal(err)
	}
	report := map[string]any{
		"plugins": []map[string]any{{
			"ID":             "custom-hardening",
			"Kind":           "custom",
			"Rollback":       "ares-plugin-custom rollback",
			"TimeoutSeconds": 5,
		}},
	}
	data, err := json.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(reportDir, "latest.json"), data, 0o644); err != nil {
		t.Fatal(err)
	}

	result, err := RollbackLast(RollbackOptions{
		Yes:  true,
		Root: root,
		Now:  time.Date(2026, 7, 25, 18, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatal(err)
	}
	if !contains(result.Applied, "would run custom rollback custom-hardening: ares-plugin-custom rollback") {
		t.Fatalf("missing custom rollback item: %+v", result)
	}
}

func TestRollbackLastReturnsErrorWhenManagedFileRemovalFails(t *testing.T) {
	root := t.TempDir()
	managedPath := filepath.Join(root, "etc", "ssh", "sshd_config.d", "99-ares.conf")
	if err := os.MkdirAll(filepath.Join(managedPath, "child"), 0o755); err != nil {
		t.Fatal(err)
	}

	result, err := RollbackLast(RollbackOptions{
		Yes:  true,
		Root: root,
		Now:  time.Date(2026, 7, 25, 18, 0, 0, 0, time.UTC),
	})
	if err == nil {
		t.Fatal("expected rollback error")
	}
	if len(result.Failed) == 0 {
		t.Fatalf("expected rollback failure result: %+v", result)
	}
	if _, statErr := os.Stat(result.ReportPath); statErr != nil {
		t.Fatalf("expected rollback report despite failure: %v", statErr)
	}
}
