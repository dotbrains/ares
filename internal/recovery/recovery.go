package recovery

import (
	"github.com/dotbrains/ares/internal/mutation"
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

func Execute(plan Plan, operator mutation.Operator) (mutation.Result, bool) {
	if plan.Legacy {
		return ExecuteLegacy(operator), true
	}
	result := ExecuteTransaction(plan.Transaction, operator)
	if len(plan.Transaction.Files) == 0 && len(plan.Transaction.Backups) == 0 {
		legacy := ExecuteLegacy(operator)
		result.Applied = append(result.Applied, legacy.Applied...)
		result.Skipped = append(result.Skipped, legacy.Skipped...)
		result.Failed = append(result.Failed, legacy.Failed...)
		return result, true
	}
	return result, false
}

func ExecuteLegacy(operator mutation.Operator) mutation.Result {
	var result mutation.Result
	for _, path := range LegacyManagedFiles() {
		appendResult(&result, operator.Remove(path))
	}
	for _, path := range LegacyBackupTargets() {
		appendResult(&result, operator.RestoreNewestBackup(path))
	}
	return result
}

func ExecuteTransaction(transaction reports.TransactionSummary, operator mutation.Operator) mutation.Result {
	var result mutation.Result
	backedUp := map[string]bool{}
	for _, path := range transaction.Backups {
		backedUp[path] = true
		appendResult(&result, operator.RestoreNewestBackup(path))
	}
	for _, path := range transaction.Files {
		if backedUp[path] {
			continue
		}
		appendResult(&result, operator.Remove(path))
	}
	return result
}

func LegacyManagedFiles() []string {
	return []string{
		"/etc/ssh/sshd_config.d/99-ares.conf",
		"/etc/fail2ban/jail.d/ares-sshd.conf",
		"/etc/apt/apt.conf.d/20auto-upgrades",
		"/etc/sysctl.d/99-ares.conf",
	}
}

func LegacyBackupTargets() []string {
	return []string{
		"/etc/ssh/sshd_config",
		"/etc/nftables.conf",
		"/etc/dnf/automatic.conf",
	}
}

func appendResult(result *mutation.Result, next mutation.Result) {
	result.Applied = append(result.Applied, next.Applied...)
	result.Skipped = append(result.Skipped, next.Skipped...)
	result.Failed = append(result.Failed, next.Failed...)
}
