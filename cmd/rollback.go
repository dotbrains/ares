package cmd

import (
	"os"

	"github.com/dotbrains/ares/internal/apply"
	"github.com/spf13/cobra"
)

func newRollbackCmd() *cobra.Command {
	var yes bool
	var dryRun bool

	cmd := &cobra.Command{
		Use:   "rollback",
		Short: "Rollback ares-managed host changes",
	}

	cmd.AddCommand(&cobra.Command{
		Use:   "last",
		Short: "Rollback the latest ares-managed changes",
		RunE: func(cmd *cobra.Command, args []string) error {
			result, err := apply.RollbackLast(apply.RollbackOptions{
				Yes:    yes,
				DryRun: dryRun,
				Root:   os.Getenv("ARES_ROOT"),
			})
			printApplyResult(cmd, result)
			return err
		},
	})
	cmd.PersistentFlags().BoolVarP(&yes, "yes", "y", false, "answer yes to rollback confirmation prompts")
	cmd.PersistentFlags().BoolVar(&dryRun, "dry-run", false, "preview rollback actions without changing files")
	return cmd
}
