package apply

import (
	"fmt"

	"github.com/dotbrains/ares/internal/reports"
)

func (ctx *Context) finish(runErr error) (Result, error) {
	if err := ctx.artifacts().FinishRun(reports.RunArtifactInput{
		Report: ctx.runReport(),
		Log:    ctx.logLines(),
		Undo:   ctx.undoPlanLines(),
	}); err != nil && runErr == nil {
		runErr = err
	}
	return ctx.Result, runErr
}

func (ctx *Context) runReport() reports.RunReport {
	return reports.RunReport{
		Profile:          ctx.Plan.Profile,
		Host:             ctx.Plan.Host,
		Plugins:          ctx.Plan.Plugins,
		Warnings:         ctx.Plan.Warnings,
		SSHLockoutPolicy: ctx.Result.SSHLockoutPolicy,
		SafetyEvidence:   ctx.Result.SafetyEvidence,
		Transaction:      ctx.Result.Transaction,
		Probed:           ctx.Result.Probed,
		Verified:         ctx.Result.Verified,
		Applied:          ctx.Result.Applied,
		Skipped:          ctx.Result.Skipped,
		Failed:           ctx.Result.Failed,
	}
}

func (ctx *Context) logLines() []string {
	var lines []string
	lines = append(lines, "ares run "+ctx.Options.Now.Format("2006-01-02 15:04:05"))
	lines = append(lines, "profile: "+ctx.Plan.Profile)
	lines = append(lines, "os: "+ctx.Plan.Host.OSID+" "+ctx.Plan.Host.OSVersion)
	lines = append(lines, "")
	lines = append(lines, "probed:")
	lines = appendList(lines, ctx.Result.Probed)
	lines = append(lines, "applied:")
	lines = appendList(lines, ctx.Result.Applied)
	lines = append(lines, "verified:")
	lines = appendList(lines, ctx.Result.Verified)
	lines = append(lines, "skipped:")
	lines = appendList(lines, ctx.Result.Skipped)
	lines = append(lines, "failed:")
	lines = appendList(lines, ctx.Result.Failed)
	return lines
}

func (ctx *Context) undoPlanLines() []string {
	return []string{
		"ares undo plan",
		"",
		"This file records manual recovery hints. Review each command before running it.",
		"",
		"SSH:",
		"- Remove /etc/ssh/sshd_config.d/99-ares.conf if SSH hardening must be reverted.",
		"- Restore any /etc/ssh/sshd_config.ares.*.bak backup if needed.",
		"- Validate with sshd -t before reloading SSH.",
		"",
		"Firewall:",
		fmt.Sprintf("- Ensure SSH port %s/tcp remains allowed before changing firewall rules.", ctx.Plan.Host.SSHPort),
		"- Disable UFW with: ufw disable",
		"- For firewalld, remove ares-added ports/services with firewall-cmd before reload.",
		"- For nftables, restore the previous /etc/nftables.conf backup if one exists.",
		"",
		"fail2ban:",
		"- Remove /etc/fail2ban/jail.d/ares-sshd.conf and restart fail2ban.",
		"",
		"Security updates:",
		"- Review /etc/apt/apt.conf.d/20auto-upgrades.",
		"- Review /etc/dnf/automatic.conf and disable dnf-automatic.timer if needed.",
		"",
		"sysctl:",
		"- Remove /etc/sysctl.d/99-ares.conf and run sysctl --system.",
	}
}

func (ctx *Context) artifacts() reports.Artifacts {
	return reports.Artifacts{
		RunReportPath: ctx.Result.ReportPath,
		RunLogPath:    ctx.Result.LogPath,
		UndoPlanPath:  ctx.Result.UndoPlanPath,
	}
}

func appendList(lines []string, values []string) []string {
	if len(values) == 0 {
		return append(lines, "- none")
	}
	for _, value := range values {
		lines = append(lines, "- "+value)
	}
	return lines
}
