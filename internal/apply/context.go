package apply

import (
	stdctx "context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	iexec "github.com/dotbrains/ares/internal/exec"
	"github.com/dotbrains/ares/internal/plan"
	"github.com/dotbrains/ares/internal/plugins"
	"github.com/dotbrains/ares/internal/safety"
)

const defaultCustomPluginTimeout = 2 * time.Minute

type Options struct {
	DryRun               bool
	Yes                  bool
	Root                 string
	Now                  time.Time
	Runner               iexec.CommandExecutor
	AllowPasswordLockout bool
}

type Result struct {
	LogPath          string             `json:"-"`
	ReportPath       string             `json:"-"`
	UndoPlanPath     string             `json:"-"`
	Transaction      TransactionSummary `json:"transaction"`
	SSHLockoutPolicy string             `json:"ssh_lockout_policy,omitempty"`
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

func (ctx *Context) verifyPlugin(plugin plugins.Plugin) {
	if plugin.Kind == "custom" {
		ctx.verifyCustomPlugin(plugin)
		return
	}

	switch plugin.ID {
	case "ssh-hardening":
		ctx.verifyPath(plugin.ID, "/etc/ssh/sshd_config.d/99-ares.conf")
	case "fail2ban", "strict-profile":
		ctx.verifyPath(plugin.ID, "/etc/fail2ban/jail.d/ares-sshd.conf")
	case "unattended-upgrades":
		ctx.verifyPath(plugin.ID, "/etc/apt/apt.conf.d/20auto-upgrades")
	case "dnf-automatic":
		ctx.verifyPath(plugin.ID, "/etc/dnf/automatic.conf")
	case "pacman-upgrade", "zypper-patches", "apk-upgrade":
		ctx.Result.Verified = append(ctx.Result.Verified, plugin.ID+": package upgrade command completed")
	case "sysctl-baseline":
		ctx.verifyPath(plugin.ID, "/etc/sysctl.d/99-ares.conf")
	case "firewall-ufw":
		ctx.verifyUFW(plugin.ID)
	case "firewall-firewalld":
		ctx.verifyFirewalld(plugin.ID)
	case "firewall-nftables":
		ctx.verifyNftables(plugin.ID)
	case "web-profile":
		ctx.verifyWebProfile(plugin.ID)
	default:
		if strings.HasPrefix(plugin.ID, "provider-") {
			ctx.Result.Verified = append(ctx.Result.Verified, plugin.ID+": advisory recorded")
			return
		}
		ctx.Result.Verified = append(ctx.Result.Verified, plugin.ID+": no verifier declared")
	}
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
	ctx.Result.LogPath = filepath.Join(base, "ares-"+stamp+".log")
	ctx.Result.ReportPath = filepath.Join(base, "latest.json")
	ctx.Result.UndoPlanPath = filepath.Join(base, "undo-plan.txt")
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
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		switch {
		case strings.HasPrefix(line, "applied:"):
			ctx.Result.Applied = append(ctx.Result.Applied, pluginID+": "+strings.TrimSpace(strings.TrimPrefix(line, "applied:")))
		case strings.HasPrefix(line, "verified:"):
			ctx.Result.Verified = append(ctx.Result.Verified, pluginID+": "+strings.TrimSpace(strings.TrimPrefix(line, "verified:")))
		case strings.HasPrefix(line, "skipped:"):
			ctx.Result.Skipped = append(ctx.Result.Skipped, pluginID+": "+strings.TrimSpace(strings.TrimPrefix(line, "skipped:")))
		case strings.HasPrefix(line, "failed:"):
			ctx.Result.Failed = append(ctx.Result.Failed, pluginID+": "+strings.TrimSpace(strings.TrimPrefix(line, "failed:")))
		default:
			ctx.Result.Applied = append(ctx.Result.Applied, pluginID+": "+line)
		}
	}
}

func (ctx *Context) applyProviderAdvisory(plugin plugins.Plugin) error {
	provider := strings.TrimPrefix(plugin.ID, "provider-")
	ctx.Result.Applied = append(ctx.Result.Applied, plugin.ID+": recorded provider advisory")
	ctx.Result.Skipped = append(ctx.Result.Skipped, provider+": verify provider-level firewalls, rescue console access, snapshots, and out-of-band recovery before relying on host-only hardening")
	return nil
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
	source := ctx.path(path)
	if _, err := os.Stat(source); err != nil {
		if os.IsNotExist(err) {
			ctx.Result.Skipped = append(ctx.Result.Skipped, "backup skipped; missing "+path)
			return nil
		}
		return err
	}
	backupPath := source + ".ares." + ctx.Options.Now.Format("20060102-150405") + ".bak"
	if _, err := os.Stat(backupPath); err == nil {
		ctx.Result.Skipped = append(ctx.Result.Skipped, "backup skipped; existing "+strings.TrimPrefix(backupPath, ctx.Options.Root))
		return nil
	} else if !os.IsNotExist(err) {
		return err
	}
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
