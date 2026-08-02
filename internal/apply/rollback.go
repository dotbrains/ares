package apply

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/dotbrains/ares/internal/hostfs"
	"github.com/dotbrains/ares/internal/mutation"
	"github.com/dotbrains/ares/internal/plugins"
	"github.com/dotbrains/ares/internal/readiness"
	"github.com/dotbrains/ares/internal/recovery"
	"github.com/dotbrains/ares/internal/reports"
)

type RollbackOptions struct {
	Yes    bool
	DryRun bool
	Root   string
	Now    time.Time
}

type rollbackContext struct {
	Options RollbackOptions
	Result  Result
	fs      hostfs.FS
	base    string
}

func RollbackLast(opts RollbackOptions) (Result, error) {
	if opts.Now.IsZero() {
		opts.Now = time.Now()
	}
	ctx := newRollbackContext(opts)
	if err := ctx.prepareReportPaths(); err != nil {
		return ctx.Result, err
	}

	if opts.DryRun {
		ctx.preview()
		ctx.Result.Skipped = append(ctx.Result.Skipped, "dry-run requested; no rollback changes applied")
		return ctx.finish(nil)
	}

	if err := ctx.refusal(); err != nil {
		return ctx.finish(err)
	}

	ctx.execute()

	if opts.Root == "" {
		ctx.Result.Skipped = append(ctx.Result.Skipped, "service reloads are not automated during rollback; review SSH and firewall access before reloading services")
	}
	return ctx.finish(ctx.err())
}

func newRollbackContext(opts RollbackOptions) *rollbackContext {
	fs := hostfs.FS{Root: opts.Root, Now: opts.Now}
	return &rollbackContext{
		Options: opts,
		fs:      fs,
		base:    fs.Path("/var/log/ares"),
	}
}

func (ctx *rollbackContext) prepareReportPaths() error {
	if err := os.MkdirAll(ctx.base, 0o755); err != nil {
		return fmt.Errorf("creating report directory: %w", err)
	}
	stamp := ctx.Options.Now.Format("20060102-150405")
	ctx.Result.LogPath = ctx.path("rollback-" + stamp + ".log")
	ctx.Result.ReportPath = ctx.path("rollback-latest.json")
	ctx.Result.UndoPlanPath = ctx.path("undo-plan.txt")
	return nil
}

func (ctx *rollbackContext) refusal() error {
	return readiness.Refusal(readiness.Request{
		Mode:   readiness.Rollback,
		Yes:    ctx.Options.Yes,
		Root:   ctx.Options.Root,
		DryRun: ctx.Options.DryRun,
	})
}

func (ctx *rollbackContext) preview() {
	report, err := ctx.latestRunReport()
	if err != nil {
		ctx.Result.Skipped = append(ctx.Result.Skipped, "latest report unavailable for rollback preview: "+err.Error())
		ctx.previewLegacyManagedFiles()
		return
	}
	ctx.previewRecoveryPlan(recovery.FromReport(report))
}

func (ctx *rollbackContext) execute() {
	report, err := ctx.latestRunReport()
	if err != nil {
		ctx.Result.Skipped = append(ctx.Result.Skipped, "latest report unavailable for transaction rollback: "+err.Error())
		ctx.rollbackLegacyManagedFiles()
		return
	}
	ctx.executeRecoveryPlan(recovery.FromReport(report))
}

func (ctx *rollbackContext) latestRunReport() (reports.LatestRunReport, error) {
	return readLatestRunReport(ctx.path("latest.json"))
}

func (ctx *rollbackContext) path(name string) string {
	return ctx.fs.Path("/var/log/ares/" + name)
}

func (ctx *rollbackContext) rollbackCustomPlugins(report reports.LatestRunReport) {
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
		if ctx.Options.Root != "" {
			ctx.Result.Applied = append(ctx.Result.Applied, "would run custom rollback "+plugin.ID+": "+plugin.Rollback)
			continue
		}
		output, err := runCustomCommand(custom, custom.Rollback)
		ctx.appendCustomRollbackOutput(custom.ID, output)
		if err != nil {
			ctx.Result.Failed = append(ctx.Result.Failed, custom.ID+": rollback failed: "+err.Error())
		}
	}
}

func (ctx *rollbackContext) executeRecoveryPlan(plan recovery.Plan) {
	mutationResult, legacy := recovery.Execute(plan, mutation.Operator{Root: ctx.Options.Root})
	if legacy {
		ctx.Result.Skipped = append(ctx.Result.Skipped, "latest report has no transaction summary; using legacy rollback targets")
	}
	ctx.appendMutationResult(mutationResult)
	ctx.rollbackCustomPlugins(reports.LatestRunReport{Plugins: plan.Custom})
}

func (ctx *rollbackContext) rollbackLegacyManagedFiles() {
	ctx.appendMutationResult(recovery.ExecuteLegacy(mutation.Operator{Root: ctx.Options.Root}))
}

func (ctx *rollbackContext) previewLegacyManagedFiles() {
	for _, path := range recovery.LegacyManagedFiles() {
		ctx.Result.Applied = append(ctx.Result.Applied, "would remove "+path)
	}
	for _, path := range recovery.LegacyBackupTargets() {
		ctx.Result.Applied = append(ctx.Result.Applied, "would restore newest backup for "+path)
	}
}

func (ctx *rollbackContext) previewRecoveryPlan(plan recovery.Plan) {
	applied, legacy := recovery.Preview(plan)
	if legacy {
		ctx.Result.Skipped = append(ctx.Result.Skipped, "latest report has no transaction summary; using legacy rollback targets")
		ctx.previewLegacyManagedFiles()
	}
	ctx.Result.Applied = append(ctx.Result.Applied, applied...)
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

func (ctx *rollbackContext) appendCustomRollbackOutput(pluginID string, output string) {
	applyCtx := &Context{}
	applyCtx.appendCustomOutput(pluginID, output)
	ctx.Result.Applied = append(ctx.Result.Applied, applyCtx.Result.Applied...)
	ctx.Result.Verified = append(ctx.Result.Verified, applyCtx.Result.Verified...)
	ctx.Result.Skipped = append(ctx.Result.Skipped, applyCtx.Result.Skipped...)
	ctx.Result.Failed = append(ctx.Result.Failed, applyCtx.Result.Failed...)
}

func (ctx *rollbackContext) appendMutationResult(mutationResult mutation.Result) {
	ctx.Result.Applied = append(ctx.Result.Applied, mutationResult.Applied...)
	ctx.Result.Skipped = append(ctx.Result.Skipped, mutationResult.Skipped...)
	ctx.Result.Failed = append(ctx.Result.Failed, mutationResult.Failed...)
}

func (ctx *rollbackContext) finish(runErr error) (Result, error) {
	if err := ctx.artifacts().FinishRollback(reports.RollbackArtifactInput{
		Report: ctx.report(),
		Log:    ctx.logLines(),
	}); err != nil && runErr == nil {
		runErr = err
	}
	return ctx.Result, runErr
}

func (ctx *rollbackContext) err() error {
	if len(ctx.Result.Failed) == 0 {
		return nil
	}
	return fmt.Errorf("rollback failed: %s", strings.Join(ctx.Result.Failed, "; "))
}

func (ctx *rollbackContext) report() reports.RollbackReport {
	return reports.RollbackReport{
		Applied: ctx.Result.Applied,
		Skipped: ctx.Result.Skipped,
		Failed:  ctx.Result.Failed,
	}
}

func (ctx *rollbackContext) logLines() []string {
	var lines []string
	lines = append(lines, "ares rollback")
	lines = append(lines, "")
	lines = append(lines, "applied:")
	lines = appendList(lines, ctx.Result.Applied)
	lines = append(lines, "skipped:")
	lines = appendList(lines, ctx.Result.Skipped)
	lines = append(lines, "failed:")
	lines = appendList(lines, ctx.Result.Failed)
	return lines
}

func (ctx *rollbackContext) artifacts() reports.Artifacts {
	return reports.Artifacts{
		RollbackReportPath: ctx.Result.ReportPath,
		RollbackLogPath:    ctx.Result.LogPath,
	}
}
