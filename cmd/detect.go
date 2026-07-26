package cmd

import (
	"github.com/dotbrains/ares/internal/system"
	"github.com/spf13/cobra"
)

func newDetectCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "detect",
		Short: "Detect the current VPS environment",
		RunE: func(cmd *cobra.Command, args []string) error {
			host, err := system.Detect()
			if err != nil {
				return err
			}
			printHost(cmd, host)
			return nil
		},
	}
}
