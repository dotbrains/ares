package cmd

import (
	"github.com/dotbrains/ares/internal/config"
	"github.com/dotbrains/ares/internal/plan"
	"github.com/dotbrains/ares/internal/system"
	"github.com/spf13/cobra"
)

func newStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show current host support status",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			host, err := system.Detect()
			if err != nil {
				return err
			}
			hardeningPlan := plan.Build(host, cfg)
			printHost(cmd, host)
			cmd.Println()
			cmd.Printf("profile: %s\n", hardeningPlan.Profile)
			cmd.Printf("plugins: %d selected\n", len(hardeningPlan.Plugins))
			cmd.Printf("warnings: %d\n", len(hardeningPlan.Warnings))
			return nil
		},
	}
}
