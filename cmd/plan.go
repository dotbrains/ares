package cmd

import (
	"github.com/dotbrains/ares/internal/config"
	"github.com/dotbrains/ares/internal/plan"
	"github.com/dotbrains/ares/internal/system"
	"github.com/spf13/cobra"
)

func newPlanCmd() *cobra.Command {
	var profile string

	cmd := &cobra.Command{
		Use:   "plan",
		Short: "Show the hardening plan",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			applyFlagOverrides(cfg, profile)
			host, err := system.Detect()
			if err != nil {
				return err
			}
			printPlan(cmd, plan.Build(host, cfg))
			return nil
		},
	}
	cmd.Flags().StringVar(&profile, "profile", "", "hardening profile: basic, web, strict")
	return cmd
}
