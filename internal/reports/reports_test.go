package reports

import (
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
