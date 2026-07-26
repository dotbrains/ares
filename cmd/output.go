package cmd

import (
	"github.com/dotbrains/ares/internal/apply"
	"github.com/dotbrains/ares/internal/plan"
	"github.com/dotbrains/ares/internal/system"
	"github.com/spf13/cobra"
)

func printHost(cmd *cobra.Command, host system.Host) {
	cmd.Printf("os: %s (%s %s)\n", host.OSName, host.OSID, host.OSVersion)
	cmd.Printf("provider: %s\n", host.Provider)
	cmd.Printf("arch: %s\n", host.Architecture)
	cmd.Printf("package manager: %s\n", host.PackageManager)
	cmd.Printf("init system: %s\n", host.InitSystem)
	cmd.Printf("firewall backend: %s\n", host.FirewallBackend)
	cmd.Printf("ssh service: %s\n", host.SSHService)
	cmd.Printf("ssh port: %s\n", host.SSHPort)
	cmd.Printf("running over ssh: %t\n", host.RunningOverSSH)
}

func printPlan(cmd *cobra.Command, hardeningPlan plan.Plan) {
	cmd.Printf("profile: %s\n", hardeningPlan.Profile)
	cmd.Println()
	printHost(cmd, hardeningPlan.Host)

	if len(hardeningPlan.Warnings) > 0 {
		cmd.Println()
		cmd.Println("warnings:")
		for _, warning := range hardeningPlan.Warnings {
			cmd.Printf("  - %s\n", warning)
		}
	}

	cmd.Println()
	cmd.Println("plugins:")
	for _, plugin := range hardeningPlan.Plugins {
		cmd.Printf("  - %s: %s\n", plugin.ID, plugin.Summary)
	}

	cmd.Println()
	cmd.Println("planned actions:")
	for _, action := range hardeningPlan.Actions {
		risk := ""
		if action.Risky {
			risk = " [risky]"
		}
		cmd.Printf("  - %s:%s %s\n", action.Plugin, risk, action.Title)
		cmd.Printf("    %s\n", action.Detail)
	}
}

func printApplyResult(cmd *cobra.Command, result apply.Result) {
	cmd.Println()
	if result.LogPath != "" {
		cmd.Printf("log: %s\n", result.LogPath)
	}
	if result.ReportPath != "" {
		cmd.Printf("report: %s\n", result.ReportPath)
	}
	if result.UndoPlanPath != "" {
		cmd.Printf("undo plan: %s\n", result.UndoPlanPath)
	}
	if len(result.Applied) > 0 {
		cmd.Println("applied:")
		for _, item := range result.Applied {
			cmd.Printf("  - %s\n", item)
		}
	}
	if len(result.Skipped) > 0 {
		cmd.Println("skipped:")
		for _, item := range result.Skipped {
			cmd.Printf("  - %s\n", item)
		}
	}
	if len(result.Failed) > 0 {
		cmd.Println("failed:")
		for _, item := range result.Failed {
			cmd.Printf("  - %s\n", item)
		}
	}
}
