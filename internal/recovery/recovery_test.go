package recovery

import (
	"testing"

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
