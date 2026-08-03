package reports

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWriteAndReadLatestRun(t *testing.T) {
	path := filepath.Join(t.TempDir(), "latest.json")
	want := LatestRunReport{
		SchemaVersion: ReportSchemaVersion,
		Transaction: TransactionSummary{
			Files: []string{"/etc/example.conf"},
		},
		Plugins: []CustomPluginReport{{
			ID:       "custom-hardening",
			Kind:     "custom",
			Rollback: "custom rollback",
		}},
	}
	if err := WriteJSON(path, want); err != nil {
		t.Fatal(err)
	}
	got, err := ReadLatestRun(path)
	if err != nil {
		t.Fatal(err)
	}
	if got.SchemaVersion != ReportSchemaVersion || got.Transaction.Files[0] != "/etc/example.conf" || got.Plugins[0].Rollback != "custom rollback" {
		t.Fatalf("unexpected report: %+v", got)
	}
}

func TestWriteJSONReplacesExistingReportAtomically(t *testing.T) {
	path := filepath.Join(t.TempDir(), "latest.json")
	if err := os.WriteFile(path, []byte("old\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := WriteJSON(path, LatestRunReport{SchemaVersion: ReportSchemaVersion}); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) == "old\n" {
		t.Fatal("report was not replaced")
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := info.Mode().Perm(); got != 0o644 {
		t.Fatalf("mode = %v, want 0644", got)
	}
	matches, err := filepath.Glob(filepath.Join(filepath.Dir(path), ".latest.json.tmp-*"))
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 0 {
		t.Fatalf("temporary report files left behind: %v", matches)
	}
}
