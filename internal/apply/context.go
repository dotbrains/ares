package apply

import (
	stdctx "context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/dotbrains/ares/internal/customoutput"
	iexec "github.com/dotbrains/ares/internal/exec"
	"github.com/dotbrains/ares/internal/hostfs"
	"github.com/dotbrains/ares/internal/plan"
	"github.com/dotbrains/ares/internal/plugins"
	"github.com/dotbrains/ares/internal/reports"
	"github.com/dotbrains/ares/internal/safety"
)

const defaultCustomPluginTimeout = 2 * time.Minute

type Options struct {
	DryRun                     bool
	Yes                        bool
	Root                       string
	Now                        time.Time
	Runner                     iexec.CommandExecutor
	AllowPasswordLockout       bool
	AllowPasswordLockoutSource string
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
	ctx.Result.SSHLockoutPolicy = safety.SSHLockoutPolicy(opts.Root, opts.AllowPasswordLockout)
	ctx.Result.SafetyEvidence = safety.EvidenceFor(hardeningPlan.Host, opts.Root, opts.AllowPasswordLockout, opts.AllowPasswordLockoutSource)
	ctx.Result.Transaction = BuildTransaction(hardeningPlan)

	if err := ctx.prepareReportPaths(); err != nil {
		return ctx.Result, err
	}

	if opts.DryRun {
		ctx.Result.Skipped = append(ctx.Result.Skipped, "dry-run requested; no changes applied")
		return ctx.finish(nil)
	}
	if os.Geteuid() != 0 && opts.Root == "" {
		return ctx.finish(fmt.Errorf("apply mode must run as root"))
	}
	if !opts.Yes {
		return ctx.finish(fmt.Errorf("apply mode requires --yes after reviewing the plan"))
	}

	executor := PluginExecutor{Context: ctx}
	for _, plugin := range hardeningPlan.Plugins {
		if err := executor.Execute(plugin); err != nil {
			return ctx.finish(err)
		}
	}

	return ctx.finish(nil)
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
	timeout := defaultCustomPluginTimeout
	if plugin.TimeoutSeconds > 0 {
		timeout = time.Duration(plugin.TimeoutSeconds) * time.Second
	}
	commandContext, cancel := stdctx.WithTimeout(stdctx.Background(), timeout)
	defer cancel()

	cmd := exec.CommandContext(commandContext, "sh", "-lc", command)
	output, err := cmd.CombinedOutput()
	if commandContext.Err() == stdctx.DeadlineExceeded {
		return string(output), fmt.Errorf("command timed out after %s", timeout)
	}
	if err != nil {
		return string(output), fmt.Errorf("%s: %w: %s", command, err, strings.TrimSpace(string(output)))
	}
	return string(output), nil
}

func (ctx *Context) appendCustomOutput(pluginID string, output string) {
	parsed := customoutput.Parse(pluginID, output)
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
	if ctx.Options.Root != "" {
		ctx.Result.Applied = append(ctx.Result.Applied, "would run: "+name+" "+strings.Join(args, " "))
		return nil
	}
	cmd := exec.Command(name, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s %s: %w: %s", name, strings.Join(args, " "), err, strings.TrimSpace(string(output)))
	}
	if len(output) > 0 {
		ctx.Result.Applied = append(ctx.Result.Applied, strings.TrimSpace(string(output)))
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
	switch packageManager {
	case "apt-get":
		args := append([]string{"install", "-y"}, packages...)
		return packageManager, args, nil
	case "dnf", "yum":
		args := append([]string{"install", "-y"}, packages...)
		return packageManager, args, nil
	case "pacman":
		args := append([]string{"-S", "--needed", "--noconfirm"}, packages...)
		return packageManager, args, nil
	case "zypper":
		args := append([]string{"--non-interactive", "install"}, packages...)
		return packageManager, args, nil
	case "apk":
		args := append([]string{"add"}, packages...)
		return packageManager, args, nil
	default:
		return "", nil, fmt.Errorf("unsupported package manager %q", packageManager)
	}
}

func (ctx *Context) backup(path string) error {
	backupPath, created, err := ctx.fs().Backup(path)
	if err != nil {
		return err
	}
	if backupPath == "" {
		ctx.Result.Skipped = append(ctx.Result.Skipped, "backup skipped; missing "+path)
		return nil
	}
	if !created {
		ctx.Result.Skipped = append(ctx.Result.Skipped, "backup skipped; existing "+backupPath)
		return nil
	}
	ctx.Result.Applied = append(ctx.Result.Applied, "backed up "+path+" to "+backupPath)
	return nil
}
