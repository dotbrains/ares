package apply

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/dotbrains/ares/internal/plan"
	"github.com/dotbrains/ares/internal/plugins"
)

type Options struct {
	DryRun bool
	Yes    bool
	Root   string
	Now    time.Time
}

type Result struct {
	LogPath      string
	ReportPath   string
	UndoPlanPath string
	Applied      []string
	Skipped      []string
	Failed       []string
}

type Context struct {
	Options Options
	Plan    plan.Plan
	Result  Result
}

func Run(hardeningPlan plan.Plan, opts Options) (Result, error) {
	if opts.Now.IsZero() {
		opts.Now = time.Now()
	}
	ctx := &Context{
		Options: opts,
		Plan:    hardeningPlan,
	}

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

	for _, plugin := range hardeningPlan.Plugins {
		if err := ctx.applyPlugin(plugin); err != nil {
			ctx.Result.Failed = append(ctx.Result.Failed, fmt.Sprintf("%s: %v", plugin.ID, err))
			return ctx.finish(err)
		}
	}

	return ctx.finish(nil)
}

func (ctx *Context) prepareReportPaths() error {
	base := ctx.path("/var/log/ares")
	if err := os.MkdirAll(base, 0o755); err != nil {
		return fmt.Errorf("creating report directory: %w", err)
	}
	stamp := ctx.Options.Now.Format("20060102-150405")
	ctx.Result.LogPath = filepath.Join(base, "ares-"+stamp+".log")
	ctx.Result.ReportPath = filepath.Join(base, "latest.json")
	ctx.Result.UndoPlanPath = filepath.Join(base, "undo-plan.txt")
	return nil
}

func (ctx *Context) applyPlugin(plugin plugins.Plugin) error {
	switch plugin.ID {
	case "distro-ubuntu", "distro-debian":
		ctx.Result.Applied = append(ctx.Result.Applied, plugin.ID+": selected apt/systemd distro adapter")
	case "ssh-hardening":
		return ctx.applySSHHardening()
	case "firewall-ufw":
		return ctx.applyUFW()
	case "fail2ban":
		return ctx.applyFail2ban()
	case "unattended-upgrades":
		return ctx.applyUnattendedUpgrades()
	case "sysctl-baseline":
		return ctx.applySysctlBaseline()
	case "web-profile":
		return ctx.applyWebProfile()
	default:
		if plugin.Kind == "custom" {
			return ctx.applyCustomPlugin(plugin)
		}
		ctx.Result.Skipped = append(ctx.Result.Skipped, plugin.ID+": apply not implemented for this plugin")
	}
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
	return ctx.run("sh", "-lc", plugin.Apply)
}

func (ctx *Context) path(path string) string {
	if ctx.Options.Root == "" {
		return path
	}
	return filepath.Join(ctx.Options.Root, strings.TrimPrefix(path, "/"))
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

func (ctx *Context) backup(path string) error {
	source := ctx.path(path)
	if _, err := os.Stat(source); err != nil {
		if os.IsNotExist(err) {
			ctx.Result.Skipped = append(ctx.Result.Skipped, "backup skipped; missing "+path)
			return nil
		}
		return err
	}
	backupPath := source + ".ares." + ctx.Options.Now.Format("20060102-150405") + ".bak"
	data, err := os.ReadFile(source)
	if err != nil {
		return err
	}
	if err := os.WriteFile(backupPath, data, 0o600); err != nil {
		return err
	}
	ctx.Result.Applied = append(ctx.Result.Applied, "backed up "+path+" to "+strings.TrimPrefix(backupPath, ctx.Options.Root))
	return nil
}
