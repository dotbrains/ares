package recovery

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/dotbrains/ares/internal/mutation"
	"github.com/dotbrains/ares/internal/reports"
)

func TestPreviewIncludesTransactionAndCustomRollback(t *testing.T) {
	plan := FromReport(reports.LatestRunReport{
		Transaction: reports.TransactionSummary{
			Files:   []string{"/etc/managed.conf"},
			Backups: []string{"/etc/state.conf"},
		},
		Plugins: []reports.CustomPluginReport{{
			ID:       "custom-hardening",
			Kind:     "custom",
			Rollback: "custom rollback",
		}},
	})
	applied, legacy := Preview(plan)
	if legacy {
		t.Fatal("unexpected legacy preview")
	}
	if len(applied) != 3 {
		t.Fatalf("applied = %+v", applied)
	}
	if applied[2] != "would run custom rollback custom-hardening: custom rollback" {
		t.Fatalf("applied = %+v", applied)
	}
}

func TestExecuteTransactionRestoresBackupsBeforeRemovingManagedFiles(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "etc", "ares"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "etc", "ares", "managed.conf"), []byte("managed"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "etc", "ares", "state.ares.20260725-170000.bak"), []byte("original"), 0o600); err != nil {
		t.Fatal(err)
	}

	result, legacy := Execute(Plan{Transaction: reports.TransactionSummary{
		Files:   []string{"/etc/ares/managed.conf"},
		Backups: []string{"/etc/ares/state"},
	}}, mutation.Operator{Root: root})

	if legacy {
		t.Fatal("transaction used legacy recovery")
	}
	if len(result.Failed) > 0 {
		t.Fatalf("failed = %v", result.Failed)
	}
	if _, err := os.Stat(filepath.Join(root, "etc", "ares", "managed.conf")); !os.IsNotExist(err) {
		t.Fatalf("managed file still exists: %v", err)
	}
	restored, err := os.ReadFile(filepath.Join(root, "etc", "ares", "state"))
	if err != nil {
		t.Fatal(err)
	}
	if string(restored) != "original" {
		t.Fatalf("restored = %q", restored)
	}
}
