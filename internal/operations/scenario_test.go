package operations

import (
	"slices"
	"testing"

	"github.com/dotbrains/ares/internal/plan"
	"github.com/dotbrains/ares/internal/scenario"
)

func TestSummaryForScenario(t *testing.T) {
	for _, fixture := range scenario.Matrix() {
		t.Run(fixture.Name, func(t *testing.T) {
			hardeningPlan := plan.Build(fixture.Host, fixture.Config)
			summary := SummaryForPlan(hardeningPlan)

			for _, command := range fixture.ExpectedCommands {
				if !slices.Contains(summary.Commands, command) {
					t.Fatalf("expected command %q in %v", command, summary.Commands)
				}
			}
			for _, path := range fixture.ExpectedFiles {
				if !slices.Contains(summary.Files, path) {
					t.Fatalf("expected file %q in %v", path, summary.Files)
				}
			}
			for _, path := range fixture.ExpectedBackups {
				if !slices.Contains(summary.Backups, path) {
					t.Fatalf("expected backup %q in %v", path, summary.Backups)
				}
			}

			rollback := RollbackPreview(summary)
			for _, step := range fixture.ExpectedRollback {
				if !slices.Contains(rollback, step) {
					t.Fatalf("expected rollback step %q in %v", step, rollback)
				}
			}
		})
	}
}
