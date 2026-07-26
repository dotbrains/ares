package cmd

import (
	"fmt"
	"strings"

	"github.com/dotbrains/ares/internal/plugins"
	"github.com/spf13/cobra"
)

func newPluginsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "plugins",
		Short: "Inspect built-in hardening plugins",
	}

	cmd.AddCommand(&cobra.Command{
		Use:   "list",
		Short: "List built-in plugins",
		Run: func(cmd *cobra.Command, args []string) {
			for _, plugin := range plugins.Builtins() {
				cmd.Printf("%-22s %-8s %s\n", plugin.ID, plugin.Kind, plugin.Summary)
			}
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "show <id>",
		Short: "Show built-in plugin details",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			plugin, ok := plugins.Find(args[0])
			if !ok {
				return fmt.Errorf("plugin %q not found", args[0])
			}
			cmd.Printf("%s (%s)\n", plugin.Name, plugin.ID)
			cmd.Printf("kind: %s\n", plugin.Kind)
			cmd.Printf("summary: %s\n", plugin.Summary)
			if len(plugin.Categories) > 0 {
				cmd.Printf("categories: %s\n", strings.Join(plugin.Categories, ", "))
			}
			if len(plugin.Requires) > 0 {
				cmd.Printf("requires: %s\n", strings.Join(plugin.Requires, ", "))
			}
			if len(plugin.Capabilities) > 0 {
				cmd.Printf("capabilities: %s\n", strings.Join(plugin.Capabilities, ", "))
			}
			if len(plugin.Distros) > 0 {
				cmd.Printf("distros: %s\n", strings.Join(plugin.Distros, ", "))
			}
			if plugin.Probe != "" {
				cmd.Printf("probe: %s\n", plugin.Probe)
			}
			if plugin.Plan != "" {
				cmd.Printf("plan: %s\n", plugin.Plan)
			}
			if plugin.Apply != "" {
				cmd.Printf("apply: %s\n", plugin.Apply)
			}
			if plugin.Verify != "" {
				cmd.Printf("verify: %s\n", plugin.Verify)
			}
			if plugin.Rollback != "" {
				cmd.Printf("rollback: %s\n", plugin.Rollback)
			}
			if plugin.TimeoutSeconds > 0 {
				cmd.Printf("timeout seconds: %d\n", plugin.TimeoutSeconds)
			}
			return nil
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "snippet <id>",
		Short: "Print a config snippet for a built-in plugin",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			plugin, ok := plugins.Find(args[0])
			if !ok {
				return fmt.Errorf("plugin %q not found", args[0])
			}
			if plugin.Config == "" {
				return fmt.Errorf("plugin %q has no config snippet", args[0])
			}
			cmd.Print(plugin.Config)
			return nil
		},
	})

	return cmd
}
