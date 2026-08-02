package cmd

import (
	"fmt"
	"os"

	"github.com/dotbrains/ares/internal/apply"
	"github.com/dotbrains/ares/internal/config"
	"github.com/dotbrains/ares/internal/plan"
	"github.com/dotbrains/ares/internal/reports"
	"github.com/spf13/cobra"
)

func newRootCmd(version string) *cobra.Command {
	var profile string
	var yes bool
	var dryRun bool
	var jsonOutput bool
	var allowPasswordLockout bool

	root := &cobra.Command{
		Use:   "ares",
		Short: "Modular VPS hardening runner",
		Long:  "ares hardens fresh Linux VPS instances with a safe, modular plugin-based execution model. It detects the host distro, plans changes, preserves SSH access, and applies provider-agnostic security defaults.",
		RunE: func(cmd *cobra.Command, args []string) error {
			runtime, err := buildCommandRuntime(config.Overrides{
				Profile:                 profile,
				AllowPasswordLockout:    allowPasswordLockout,
				AllowPasswordLockoutSet: cmd.Flags().Changed("allow-password-lockout"),
			})
			if err != nil {
				return err
			}
			if !jsonOutput {
				printBanner(cmd)
				printPlan(cmd, runtime.Plan)
			}
			result, err := apply.Run(runtime.Plan, apply.Options{
				DryRun:                     dryRun,
				Yes:                        yes,
				Root:                       os.Getenv("ARES_ROOT"),
				AllowPasswordLockout:       runtime.Config.SSH.AllowPasswordLockout,
				AllowPasswordLockoutSource: string(runtime.Effective.Sources["ssh.allow_password_lockout"]),
			})
			if jsonOutput {
				return printRunJSON(cmd, runtime.Plan, result, err)
			}
			printApplyResult(cmd, result)
			return err
		},
		CompletionOptions: cobra.CompletionOptions{
			HiddenDefaultCmd: true,
		},
		Version: version,
	}

	root.SetVersionTemplate(fmt.Sprintf("ares version %s\n", version))
	root.Flags().StringVar(&profile, "profile", "", "hardening profile: basic, web, strict")
	root.Flags().BoolVar(&dryRun, "dry-run", false, "show the hardening plan without applying changes")
	root.Flags().BoolVar(&jsonOutput, "json", false, "print dry-run/apply result as JSON")
	root.Flags().BoolVarP(&yes, "yes", "y", false, "answer yes to confirmation prompts")
	root.Flags().BoolVar(&allowPasswordLockout, "allow-password-lockout", false, "explicitly allow SSH hardening to disable password auth without detected authorized_keys")

	// Subcommands
	root.AddCommand(newConfigCmd())
	root.AddCommand(newDetectCmd())
	root.AddCommand(newPlanCmd())
	root.AddCommand(newPreflightCmd())
	root.AddCommand(newPluginsCmd())
	root.AddCommand(newRollbackCmd())
	root.AddCommand(newStatusCmd())

	return root
}

func printRunJSON(cmd *cobra.Command, hardeningPlan plan.Plan, result apply.Result, runErr error) error {
	data, err := reports.MarshalJSON(reports.NewRunOutput(hardeningPlan, result, runErr))
	if err != nil {
		return err
	}
	cmd.Println(string(data))
	return runErr
}

// Execute runs the root command.
func Execute(version string) error {
	return newRootCmd(version).Execute()
}
