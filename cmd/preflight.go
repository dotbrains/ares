package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/dotbrains/ares/internal/apply"
	"github.com/dotbrains/ares/internal/config"
	"github.com/dotbrains/ares/internal/plan"
	"github.com/dotbrains/ares/internal/reports"
	"github.com/dotbrains/ares/internal/safety"
	"github.com/dotbrains/ares/internal/system"
	"github.com/spf13/cobra"
)

type preflightCheck = safety.Decision

type preflightReport = reports.PreflightReport

func newPreflightCmd() *cobra.Command {
	var profile string
	var jsonOutput bool

	cmd := &cobra.Command{
		Use:   "preflight",
		Short: "Check apply readiness without changing the host",
		RunE: func(cmd *cobra.Command, args []string) error {
			runtime, err := buildCommandRuntime(config.Overrides{Profile: profile})
			if err != nil {
				return err
			}
			checks := safety.Evaluate(safety.Facts{
				Host:   runtime.Host,
				Plan:   runtime.Plan,
				Config: runtime.Config,
				Root:   os.Getenv("ARES_ROOT"),
			})
			report := buildPreflightReport(runtime.Host, runtime.Plan, checks)
			if jsonOutput {
				if err := printPreflightJSON(cmd, report); err != nil {
					return err
				}
			} else {
				printPreflight(cmd, report)
			}
			if safety.HasFailures(checks) {
				return fmt.Errorf("preflight failed")
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&profile, "profile", "", "hardening profile: basic, web, strict")
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "print preflight report as JSON")
	return cmd
}

func buildPreflightReport(host system.Host, hardeningPlan plan.Plan, checks []preflightCheck) preflightReport {
	ids := make([]string, 0, len(hardeningPlan.Plugins))
	for _, plugin := range hardeningPlan.Plugins {
		ids = append(ids, plugin.ID)
	}
	return preflightReport{
		SchemaVersion: reports.PreflightSchemaVersion,
		Profile:       hardeningPlan.Profile,
		Host:          host,
		Plugins:       ids,
		Checks:        checks,
		Transaction:   apply.BuildTransaction(hardeningPlan),
	}
}

func printPreflight(cmd *cobra.Command, report preflightReport) {
	cmd.Println("preflight:")
	for _, check := range report.Checks {
		cmd.Printf("  - %s: %s (%s)\n", check.Name, check.Status, check.Detail)
	}
	if len(report.Transaction.Files) > 0 || len(report.Transaction.Commands) > 0 || len(report.Transaction.Backups) > 0 {
		cmd.Println("transaction:")
		printNamedList(cmd, "files", report.Transaction.Files)
		printNamedList(cmd, "commands", report.Transaction.Commands)
		printNamedList(cmd, "backups", report.Transaction.Backups)
		printNamedList(cmd, "rollback", report.Transaction.RollbackSteps)
	}
}

func printPreflightJSON(cmd *cobra.Command, report preflightReport) error {
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return err
	}
	cmd.Println(string(data))
	return nil
}
