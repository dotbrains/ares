package cmd

import (
	"encoding/json"

	"github.com/dotbrains/ares/internal/config"
	"github.com/dotbrains/ares/internal/plan"
	"github.com/dotbrains/ares/internal/system"
	"github.com/spf13/cobra"
)

func newPlanCmd() *cobra.Command {
	var profile string
	var jsonOutput bool

	cmd := &cobra.Command{
		Use:   "plan",
		Short: "Show the hardening plan",
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
			if jsonOutput {
				data, err := json.MarshalIndent(hardeningPlan, "", "  ")
				if err != nil {
					return err
				}
				cmd.Println(string(data))
				return nil
			}
			printPlan(cmd, hardeningPlan)
			return nil
		},
	}
	cmd.Flags().StringVar(&profile, "profile", "", "hardening profile: basic, web, strict")
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "print the hardening plan as JSON")
	return cmd
}
