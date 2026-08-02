package apply

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/dotbrains/ares/internal/hostfs"
	"github.com/dotbrains/ares/internal/mutation"
	"github.com/dotbrains/ares/internal/plugins"
	"github.com/dotbrains/ares/internal/recovery"
	"github.com/dotbrains/ares/internal/reports"
)

type RollbackOptions struct {
	Yes    bool
	DryRun bool
	Root   string
	Now    time.Time
}

func RollbackLast(opts RollbackOptions) (Result, error) {
	if opts.Now.IsZero() {
		opts.Now = time.Now()
	}
	result := Result{}
	fs := hostfs.FS{Root: opts.Root, Now: opts.Now}
	base := fs.Path("/var/log/ares")
	if err := os.MkdirAll(base, 0o755); err != nil {
		return result, fmt.Errorf("creating report directory: %w", err)
	}
	stamp := opts.Now.Format("20060102-150405")
	result.LogPath = filepath.Join(base, "rollback-"+stamp+".log")
	result.ReportPath = filepath.Join(base, "rollback-latest.json")
	result.UndoPlanPath = filepath.Join(base, "undo-plan.txt")

	if opts.DryRun {
		report, reportErr := readLatestRunReport(filepath.Join(base, "latest.json"))
		if reportErr != nil {
			result.Skipped = append(result.Skipped, "latest report unavailable for rollback preview: "+reportErr.Error())
			previewLegacyManagedFiles(&result)
		} else {
			previewRecoveryPlan(&result, recovery.FromReport(report))
		}
		result.Skipped = append(result.Skipped, "dry-run requested; no rollback changes applied")
		return finishRollback(result, nil)
	}

	if !opts.Yes {
		return finishRollback(result, fmt.Errorf("rollback requires --yes after reviewing the undo plan"))
	}
	if os.Geteuid() != 0 && opts.Root == "" {
		return finishRollback(result, fmt.Errorf("rollback must run as root"))
	}

	report, reportErr := readLatestRunReport(filepath.Join(base, "latest.json"))
	if reportErr != nil {
		result.Skipped = append(result.Skipped, "latest report unavailable for transaction rollback: "+reportErr.Error())
		rollbackLegacyManagedFiles(&result, opts.Root)
	} else {
		executeRecoveryPlan(&result, opts.Root, recovery.FromReport(report))
	}

	if opts.Root == "" {
		result.Skipped = append(result.Skipped, "service reloads are not automated during rollback; review SSH and firewall access before reloading services")
	}
	return finishRollback(result, rollbackError(result))
}

func rollbackCustomPlugins(result *Result, root string, report reports.LatestRunReport) {
	for _, plugin := range report.Plugins {
		if plugin.Kind != "custom" || plugin.Rollback == "" {
			continue
		}
		custom := plugins.Plugin{
			ID:             plugin.ID,
			Kind:           plugin.Kind,
			Rollback:       plugin.Rollback,
			TimeoutSeconds: plugin.TimeoutSeconds,
		}
		if root != "" {
			result.Applied = append(result.Applied, "would run custom rollback "+plugin.ID+": "+plugin.Rollback)
			continue
		}
		output, err := runCustomCommand(custom, custom.Rollback)
		appendCustomRollbackOutput(result, custom.ID, output)
		if err != nil {
			result.Failed = append(result.Failed, custom.ID+": rollback failed: "+err.Error())
		}
	}
}

func executeRecoveryPlan(result *Result, root string, plan recovery.Plan) {
	if plan.Legacy {
		result.Skipped = append(result.Skipped, "latest report has no transaction summary; using legacy rollback targets")
		rollbackLegacyManagedFiles(result, root)
		rollbackCustomPlugins(result, root, reports.LatestRunReport{Plugins: plan.Custom})
		return
	}
	rollbackTransaction(result, root, plan.Transaction)
	rollbackCustomPlugins(result, root, reports.LatestRunReport{Plugins: plan.Custom})
}

func rollbackLegacyManagedFiles(result *Result, root string) {
	for _, path := range []string{
		"/etc/ssh/sshd_config.d/99-ares.conf",
		"/etc/fail2ban/jail.d/ares-sshd.conf",
		"/etc/apt/apt.conf.d/20auto-upgrades",
		"/etc/sysctl.d/99-ares.conf",
	} {
		rollbackManagedFile(result, root, path)
	}
	for _, path := range []string{
		"/etc/ssh/sshd_config",
		"/etc/nftables.conf",
		"/etc/dnf/automatic.conf",
	} {
		restoreNewestBackup(result, root, path)
	}
}

func rollbackTransaction(result *Result, root string, transaction TransactionSummary) {
	if len(transaction.Files) == 0 && len(transaction.Backups) == 0 {
		result.Skipped = append(result.Skipped, "latest report has no transaction summary; using legacy rollback targets")
		rollbackLegacyManagedFiles(result, root)
		return
	}
	backedUp := map[string]bool{}
	for _, path := range transaction.Backups {
		backedUp[path] = true
		restoreNewestBackup(result, root, path)
	}
	for _, path := range transaction.Files {
		if backedUp[path] {
			continue
		}
		rollbackManagedFile(result, root, path)
	}
}

func previewLegacyManagedFiles(result *Result) {
	for _, path := range []string{
		"/etc/ssh/sshd_config.d/99-ares.conf",
		"/etc/fail2ban/jail.d/ares-sshd.conf",
		"/etc/apt/apt.conf.d/20auto-upgrades",
		"/etc/sysctl.d/99-ares.conf",
	} {
		result.Applied = append(result.Applied, "would remove "+path)
	}
	for _, path := range []string{
		"/etc/ssh/sshd_config",
		"/etc/nftables.conf",
		"/etc/dnf/automatic.conf",
	} {
		result.Applied = append(result.Applied, "would restore newest backup for "+path)
	}
}

func previewRecoveryPlan(result *Result, plan recovery.Plan) {
	applied, legacy := recovery.Preview(plan)
	if legacy {
		result.Skipped = append(result.Skipped, "latest report has no transaction summary; using legacy rollback targets")
		previewLegacyManagedFiles(result)
	}
	result.Applied = append(result.Applied, applied...)
}

func readLatestRunReport(path string) (reports.LatestRunReport, error) {
	report, err := reports.ReadLatestRun(path)
	if err != nil {
		if os.IsNotExist(err) {
			return report, fmt.Errorf("latest report unavailable")
		}
		return report, fmt.Errorf("latest report is invalid")
	}
	return report, nil
}

func appendCustomRollbackOutput(result *Result, pluginID string, output string) {
	ctx := &Context{}
	ctx.appendCustomOutput(pluginID, output)
	result.Applied = append(result.Applied, ctx.Result.Applied...)
	result.Verified = append(result.Verified, ctx.Result.Verified...)
	result.Skipped = append(result.Skipped, ctx.Result.Skipped...)
	result.Failed = append(result.Failed, ctx.Result.Failed...)
}

func rollbackManagedFile(result *Result, root string, path string) {
	appendMutationResult(result, mutation.Operator{Root: root}.Remove(path))
}

func restoreNewestBackup(result *Result, root string, path string) {
	appendMutationResult(result, mutation.Operator{Root: root}.RestoreNewestBackup(path))
}

func appendMutationResult(result *Result, mutationResult mutation.Result) {
	result.Applied = append(result.Applied, mutationResult.Applied...)
	result.Skipped = append(result.Skipped, mutationResult.Skipped...)
	result.Failed = append(result.Failed, mutationResult.Failed...)
}

func finishRollback(result Result, runErr error) (Result, error) {
	if err := writeRollbackReport(result); err != nil && runErr == nil {
		runErr = err
	}
	if err := writeRollbackLog(result); err != nil && runErr == nil {
		runErr = err
	}
	return result, runErr
}

func rollbackError(result Result) error {
	if len(result.Failed) == 0 {
		return nil
	}
	return fmt.Errorf("rollback failed: %s", strings.Join(result.Failed, "; "))
}

func writeRollbackReport(result Result) error {
	return rollbackArtifacts(result).WriteRollbackReport(reports.RollbackReport{
		Applied: result.Applied,
		Skipped: result.Skipped,
		Failed:  result.Failed,
	})
}

func writeRollbackLog(result Result) error {
	var lines []string
	lines = append(lines, "ares rollback")
	lines = append(lines, "")
	lines = append(lines, "applied:")
	lines = appendList(lines, result.Applied)
	lines = append(lines, "skipped:")
	lines = appendList(lines, result.Skipped)
	lines = append(lines, "failed:")
	lines = appendList(lines, result.Failed)
	return rollbackArtifacts(result).WriteRollbackLog(lines)
}

func rollbackArtifacts(result Result) reports.Artifacts {
	return reports.Artifacts{
		RollbackReportPath: result.ReportPath,
		RollbackLogPath:    result.LogPath,
	}
}
