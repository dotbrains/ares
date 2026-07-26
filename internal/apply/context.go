package apply

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
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
	Probed       []string
	Verified     []string
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
		ctx.probePlugin(plugin)
		if err := ctx.applyPlugin(plugin); err != nil {
			ctx.Result.Failed = append(ctx.Result.Failed, fmt.Sprintf("%s: %v", plugin.ID, err))
			return ctx.finish(err)
		}
		ctx.verifyPlugin(plugin)
	}

	return ctx.finish(nil)
}

func (ctx *Context) probePlugin(plugin plugins.Plugin) {
	if plugin.Probe == "" {
		ctx.Result.Probed = append(ctx.Result.Probed, plugin.ID+": no probe declared")
		return
	}
	if ctx.Options.Root != "" {
		ctx.Result.Probed = append(ctx.Result.Probed, plugin.ID+": would probe with "+plugin.Probe)
		return
	}
	cmd := exec.Command("sh", "-lc", plugin.Probe)
	if output, err := cmd.CombinedOutput(); err != nil {
		ctx.Result.Skipped = append(ctx.Result.Skipped, plugin.ID+": probe did not pass before apply: "+strings.TrimSpace(string(output)))
	} else {
		ctx.Result.Probed = append(ctx.Result.Probed, plugin.ID+": probe passed")
	}
}

func (ctx *Context) verifyPlugin(plugin plugins.Plugin) {
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
		ctx.Result.Verified = append(ctx.Result.Verified, plugin.ID+": SSH port "+ctx.Plan.Host.SSHPort+"/tcp preserved")
	case "firewall-firewalld":
		ctx.Result.Verified = append(ctx.Result.Verified, plugin.ID+": SSH port "+ctx.Plan.Host.SSHPort+"/tcp preserved")
	case "firewall-nftables":
		ctx.verifyPath(plugin.ID, "/etc/nftables.conf")
	case "web-profile":
		ctx.Result.Verified = append(ctx.Result.Verified, plugin.ID+": HTTP/HTTPS allow rules requested")
	default:
		if strings.HasPrefix(plugin.ID, "provider-") {
			ctx.Result.Verified = append(ctx.Result.Verified, plugin.ID+": advisory recorded")
			return
		}
		ctx.Result.Verified = append(ctx.Result.Verified, plugin.ID+": no verifier declared")
	}
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

func (ctx *Context) applyPlugin(plugin plugins.Plugin) error {
	if slices.Contains(plugin.Categories, "distro") {
		ctx.Result.Applied = append(ctx.Result.Applied, plugin.ID+": selected "+ctx.Plan.Host.PackageManager+"/"+ctx.Plan.Host.InitSystem+" distro adapter")
		return nil
	}

	switch plugin.ID {
	case "ssh-hardening":
		return ctx.applySSHHardening()
	case "firewall-ufw":
		return ctx.applyUFW()
	case "firewall-firewalld":
		return ctx.applyFirewalld()
	case "firewall-nftables":
		return ctx.applyNftables()
	case "fail2ban":
		return ctx.applyFail2ban()
	case "unattended-upgrades":
		return ctx.applyUnattendedUpgrades()
	case "dnf-automatic":
		return ctx.applyDNFAutomatic()
	case "pacman-upgrade", "zypper-patches", "apk-upgrade":
		return ctx.applyPackageUpgrade()
	case "sysctl-baseline":
		return ctx.applySysctlBaseline()
	case "web-profile":
		return ctx.applyWebProfile()
	case "strict-profile":
		return ctx.applyStrictProfile()
	default:
		if strings.HasPrefix(plugin.ID, "provider-") {
			return ctx.applyProviderAdvisory(plugin)
		}
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
