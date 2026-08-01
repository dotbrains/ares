package cmd

import (
	"encoding/json"

	"github.com/dotbrains/ares/internal/config"
	"github.com/spf13/cobra"
)

func newPlanCmd() *cobra.Command {
	var profile string
	var jsonOutput bool

	cmd := &cobra.Command{
		Use:   "plan",
		Short: "Show the hardening plan",
		RunE: func(cmd *cobra.Command, args []string) error {
			runtime, err := buildCommandRuntime(config.Overrides{Profile: profile})
			if err != nil {
				return err
			}
			if jsonOutput {
				data, err := json.MarshalIndent(runtime.Plan, "", "  ")
				if err != nil {
					return err
				}
				cmd.Println(string(data))
				return nil
			}
			printPlan(cmd, runtime.Plan)
			return nil
		},
	}
	cmd.Flags().StringVar(&profile, "profile", "", "hardening profile: basic, web, strict")
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "print the hardening plan as JSON")
	return cmd
}
