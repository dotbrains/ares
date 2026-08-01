package cmd

import (
	"fmt"
	"os"

	"github.com/dotbrains/ares/internal/apply"
	"github.com/dotbrains/ares/internal/config"
	"github.com/dotbrains/ares/internal/plan"
	"github.com/dotbrains/ares/internal/system"
	"github.com/spf13/cobra"
)

func newRootCmd(version string) *cobra.Command {
	var profile string
	var yes bool
	var dryRun bool

	root := &cobra.Command{
		Use:   "ares",
		Short: "Modular VPS hardening runner",
		Long:  "ares hardens fresh Linux VPS instances with a safe, modular plugin-based execution model. It detects the host distro, plans changes, preserves SSH access, and applies provider-agnostic security defaults.",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			applyFlagOverrides(cfg, profile)
			if err := config.Validate(cfg); err != nil {
				return err
			}
			host, err := system.Detect()
			if err != nil {
				return err
			}
			hardeningPlan := plan.Build(host, cfg)
			printBanner(cmd)
			printPlan(cmd, hardeningPlan)
			result, err := apply.Run(hardeningPlan, apply.Options{
				DryRun: dryRun,
				Yes:    yes,
				Root:   os.Getenv("ARES_ROOT"),
			})
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
	root.Flags().BoolVarP(&yes, "yes", "y", false, "answer yes to confirmation prompts")

	// Subcommands
	root.AddCommand(newConfigCmd())
	root.AddCommand(newDetectCmd())
	root.AddCommand(newPlanCmd())
	root.AddCommand(newPluginsCmd())
	root.AddCommand(newRollbackCmd())
	root.AddCommand(newStatusCmd())

	return root
}

func applyFlagOverrides(cfg *config.Config, profile string) {
	if profile != "" {
		cfg.Profile = profile
	}
}

// Execute runs the root command.
func Execute(version string) error {
	return newRootCmd(version).Execute()
}
