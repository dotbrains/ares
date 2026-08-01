package recovery

import (
	"github.com/dotbrains/ares/internal/operations"
	"github.com/dotbrains/ares/internal/reports"
)

type Plan struct {
	Transaction reports.TransactionSummary
	Custom      []reports.CustomPluginReport
	Legacy      bool
}

func FromReport(report reports.LatestRunReport) Plan {
	return Plan{
		Transaction: report.Transaction,
		Custom:      report.Plugins,
		Legacy:      len(report.Transaction.Files) == 0 && len(report.Transaction.Backups) == 0,
	}
}

func Preview(plan Plan) (applied []string, legacy bool) {
	if !plan.Legacy {
		applied = append(applied, operations.RollbackPreview(plan.Transaction)...)
	}
	for _, plugin := range plan.Custom {
		if plugin.Kind == "custom" && plugin.Rollback != "" {
			applied = append(applied, "would run custom rollback "+plugin.ID+": "+plugin.Rollback)
		}
	}
	return applied, plan.Legacy
}
