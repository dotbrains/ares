package apply

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/dotbrains/ares/internal/customcommand"
	iexec "github.com/dotbrains/ares/internal/exec"
	"github.com/dotbrains/ares/internal/hostfs"
	"github.com/dotbrains/ares/internal/intent"
	"github.com/dotbrains/ares/internal/mutation"
	"github.com/dotbrains/ares/internal/plan"
	"github.com/dotbrains/ares/internal/plugins"
	"github.com/dotbrains/ares/internal/reports"
	"github.com/dotbrains/ares/internal/safety"
)

type Options struct {
	DryRun                     bool
	Yes                        bool
	Root                       string
	Now                        time.Time
	Runner                     iexec.CommandExecutor
	AllowPasswordLockout       bool
	AllowPasswordLockoutSource string
	Tailscale                  TailscaleOptions
}

type TailscaleOptions struct {
	SSHEnabled       bool
	AuthKeyEnv       string
	AuthKey          string
	Hostname         string
	AcceptRoutes     bool
	ExtraArgs        []string
	SSHEnabledSource string
}

type Result struct {
	LogPath          string             `json:"-"`
	ReportPath       string             `json:"-"`
	UndoPlanPath     string             `json:"-"`
	Transaction      TransactionSummary `json:"transaction"`
	SSHLockoutPolicy string             `json:"ssh_lockout_policy,omitempty"`
	SafetyEvidence   []reports.Evidence `json:"safety_evidence,omitempty"`
	Probed           []string           `json:"probed"`
	Verified         []string           `json:"verified"`
	Applied          []string           `json:"applied"`
	Skipped          []string           `json:"skipped"`
	Failed           []string           `json:"failed"`
}

type Context struct {
	Options Options
	Plan    plan.Plan
	Result  Result
}

func (ctx *Context) fs() hostfs.FS {
	return hostfs.FS{Root: ctx.Options.Root, Now: ctx.Options.Now}
}

func (ctx *Context) mutation() mutation.Operator {
	return mutation.Operator{Root: ctx.Options.Root, Now: ctx.Options.Now}
}

func (ctx *Context) appendMutation(result mutation.Result) {
	ctx.Result.Applied = append(ctx.Result.Applied, result.Applied...)
	ctx.Result.Skipped = append(ctx.Result.Skipped, result.Skipped...)
	ctx.Result.Failed = append(ctx.Result.Failed, result.Failed...)
}

func Run(hardeningPlan plan.Plan, opts Options) (Result, error) {
	if opts.Now.IsZero() {
		opts.Now = time.Now()
	}
	ctx := &Context{
		Options: opts,
		Plan:    hardeningPlan,
	}
	if ctx.Options.Runner == nil {
		ctx.Options.Runner = iexec.NewRealExecutor()
	}
	readiness := safety.EvaluateApply(safety.ApplyReadinessInput{
		Host:                       hardeningPlan.Host,
		Root:                       opts.Root,
		DryRun:                     opts.DryRun,
		Yes:                        opts.Yes,
		AllowPasswordLockout:       opts.AllowPasswordLockout,
		AllowPasswordLockoutSource: opts.AllowPasswordLockoutSource,
	})
	ctx.Result.SSHLockoutPolicy = readiness.SSHLockoutPolicy
	ctx.Result.SafetyEvidence = readiness.SafetyEvidence
	ctx.Result.SafetyEvidence = append(ctx.Result.SafetyEvidence, ctx.tailscaleEvidence()...)
	ctx.Result.Transaction = BuildTransaction(hardeningPlan)

	if err := ctx.prepareReportPaths(); err != nil {
		return ctx.Result, err
	}

	if opts.DryRun {
		ctx.Result.Skipped = append(ctx.Result.Skipped, "dry-run requested; no changes applied")
		return ctx.finish(nil)
	}
	if err := readiness.Refusal; err != nil {
		return ctx.finish(err)
	}

	executor := PluginExecutor{Context: ctx}
	for _, plugin := range hardeningPlan.Plugins {
		if err := executor.Execute(plugin); err != nil {
			return ctx.finish(err)
		}
	}

	return ctx.finish(nil)
}

func (ctx *Context) tailscaleEvidence() []reports.Evidence {
	return []reports.Evidence{
		{
			Name:       "tailscale.ssh_enabled",
			Value:      fmt.Sprintf("%t", ctx.Options.Tailscale.SSHEnabled),
			Source:     sourceOrDefault(ctx.Options.Tailscale.SSHEnabledSource, "default"),
			Confidence: "high",
		},
		{
			Name:       "tailscale.auth_key_env",
			Value:      ctx.Options.Tailscale.AuthKeyEnv,
			Source:     sourceOrDefault(ctx.Options.Tailscale.SSHEnabledSource, "default"),
			Confidence: confidenceForValue(ctx.Options.Tailscale.AuthKeyEnv),
		},
	}
}

func sourceOrDefault(value string, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func confidenceForValue(value string) string {
	if strings.TrimSpace(value) == "" {
		return "low"
	}
	return "high"
}

func (ctx *Context) probePlugin(plugin plugins.Plugin) bool {
	return PluginExecutor{Context: ctx}.Probe(plugin)
}

func probeFailureMessage(output string, err error) string {
	message := strings.TrimSpace(output)
	if message != "" {
		return message
	}
	return err.Error()
}

func (ctx *Context) verifyPluginOrError(plugin plugins.Plugin) error {
	return PluginExecutor{Context: ctx}.Verify(plugin)
}

func (ctx *Context) verifyPath(pluginID string, path string) {
	if _, err := os.Stat(ctx.path(path)); err != nil {
		ctx.Result.Failed = append(ctx.Result.Failed, pluginID+": expected "+path+" was not present")
		return
	}
	ctx.Result.Verified = append(ctx.Result.Verified, pluginID+": verified "+path)
}

func (ctx *Context) prepareReportPaths() error {
	base := ctx.path("/var/log/ares")
	if err := os.MkdirAll(base, 0o755); err != nil {
		return fmt.Errorf("creating report directory: %w", err)
	}
	stamp := ctx.Options.Now.Format("20060102-150405")
	ctx.Result.LogPath = ctx.path("/var/log/ares/ares-" + stamp + ".log")
	ctx.Result.ReportPath = ctx.path("/var/log/ares/latest.json")
	ctx.Result.UndoPlanPath = ctx.path("/var/log/ares/undo-plan.txt")
	return nil
}

func (ctx *Context) applyCustomPlugin(plugin plugins.Plugin) error {
	if plugin.Apply == "" {
		ctx.Result.Skipped = append(ctx.Result.Skipped, plugin.ID+": custom plugin has no apply command")
		return nil
	}
	if ctx.Options.Root != "" {
		ctx.Result.Applied = append(ctx.Result.Applied, "would run custom plugin "+plugin.ID+": "+plugin.Apply)
		return nil
	}
	output, err := runCustomCommand(plugin, plugin.Apply)
	ctx.appendCustomOutput(plugin.ID, output)
	return err
}

func (ctx *Context) verifyCustomPlugin(plugin plugins.Plugin) {
	if plugin.Verify == "" {
		ctx.Result.Verified = append(ctx.Result.Verified, plugin.ID+": no verifier declared")
		return
	}
	if ctx.Options.Root != "" {
		ctx.Result.Verified = append(ctx.Result.Verified, plugin.ID+": would verify with "+plugin.Verify)
		return
	}
	output, err := runCustomCommand(plugin, plugin.Verify)
	ctx.appendCustomOutput(plugin.ID, output)
	if err != nil {
		ctx.Result.Failed = append(ctx.Result.Failed, plugin.ID+": verify failed: "+err.Error())
		return
	}
	ctx.Result.Verified = append(ctx.Result.Verified, plugin.ID+": custom verify passed")
}

func runCustomCommand(plugin plugins.Plugin, command string) (string, error) {
	result := customcommand.New(plugin, "", command).Run()
	return result.Output, result.Err
}

func (ctx *Context) appendCustomOutput(pluginID string, output string) {
	parsed := customcommand.ParseOutput(pluginID, output)
	ctx.Result.Applied = append(ctx.Result.Applied, parsed.Applied...)
	ctx.Result.Verified = append(ctx.Result.Verified, parsed.Verified...)
	ctx.Result.Skipped = append(ctx.Result.Skipped, parsed.Skipped...)
	ctx.Result.Failed = append(ctx.Result.Failed, parsed.Failed...)
}

func (ctx *Context) applyProviderAdvisory(plugin plugins.Plugin) error {
	provider := strings.TrimPrefix(plugin.ID, "provider-")
	ctx.Result.Applied = append(ctx.Result.Applied, plugin.ID+": recorded provider advisory")
	ctx.Result.Skipped = append(ctx.Result.Skipped, provider+": verify provider-level firewalls, rescue console access, snapshots, and out-of-band recovery before relying on host-only hardening")
	return nil
}

func (ctx *Context) path(path string) string {
	return ctx.fs().Path(path)
}

func (ctx *Context) run(name string, args ...string) error {
	result, err := ctx.mutation().Run(name, args...)
	ctx.appendMutation(result)
	if err != nil {
		return err
	}
	return nil
}

func (ctx *Context) installPackages(packages ...string) error {
	name, args, err := installCommand(ctx.Plan.Host.PackageManager, packages...)
	if err != nil {
		return err
	}
	return ctx.run(name, args...)
}

func installCommand(packageManager string, packages ...string) (string, []string, error) {
	return intent.InstallCommand(packageManager, packages...)
}

func (ctx *Context) backup(path string) error {
	result, err := ctx.mutation().Backup(path)
	ctx.appendMutation(result)
	if err != nil {
		return err
	}
	return nil
}

func (ctx *Context) writeFile(path string, data []byte, perm os.FileMode) error {
	result, err := ctx.mutation().WriteFile(path, data, perm)
	ctx.appendMutation(result)
	if err != nil {
		return err
	}
	return nil
}
