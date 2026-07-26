package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/dotbrains/ares/internal/apply"
	"github.com/dotbrains/ares/internal/config"
	"github.com/dotbrains/ares/internal/plan"
	"github.com/dotbrains/ares/internal/plugins"
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
			host, err := system.Detect()
			if err != nil {
				return err
			}
			hardeningPlan := plan.Build(host, cfg)
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
	root.AddCommand(newStatusCmd())

	return root
}

func applyFlagOverrides(cfg *config.Config, profile string) {
	if profile != "" {
		cfg.Profile = profile
	}
}

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
			if plugin.Rollback != "" {
				cmd.Printf("rollback: %s\n", plugin.Rollback)
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

func newConfigCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Manage configuration",
	}

	var force bool

	initCmd := &cobra.Command{
		Use:   "init",
		Short: "Create default config file",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfgPath, err := config.ConfigPath()
			if err != nil {
				return err
			}

			if !force {
				if _, err := os.Stat(cfgPath); err == nil {
					return fmt.Errorf("config already exists at %s (use --force to overwrite)", cfgPath)
				}
			}

			if err := config.Save(config.DefaultConfig()); err != nil {
				return err
			}

			// Shorten the path for display.
			display := cfgPath
			if home, err := os.UserHomeDir(); err == nil {
				if rel, err := filepath.Rel(home, cfgPath); err == nil {
					display = "~/" + rel
				}
			}

			cmd.Printf("✓ Wrote default config to %s\nEdit the file to customize settings.\n", display)
			return nil
		},
	}
	initCmd.Flags().BoolVar(&force, "force", false, "overwrite existing config")

	cmd.AddCommand(initCmd)
	return cmd
}

func printHost(cmd *cobra.Command, host system.Host) {
	cmd.Printf("os: %s (%s %s)\n", host.OSName, host.OSID, host.OSVersion)
	cmd.Printf("provider: %s\n", host.Provider)
	cmd.Printf("arch: %s\n", host.Architecture)
	cmd.Printf("package manager: %s\n", host.PackageManager)
	cmd.Printf("init system: %s\n", host.InitSystem)
	cmd.Printf("firewall backend: %s\n", host.FirewallBackend)
	cmd.Printf("ssh service: %s\n", host.SSHService)
	cmd.Printf("ssh port: %s\n", host.SSHPort)
	cmd.Printf("running over ssh: %t\n", host.RunningOverSSH)
}

func printPlan(cmd *cobra.Command, hardeningPlan plan.Plan) {
	cmd.Printf("profile: %s\n", hardeningPlan.Profile)
	cmd.Println()
	printHost(cmd, hardeningPlan.Host)

	if len(hardeningPlan.Warnings) > 0 {
		cmd.Println()
		cmd.Println("warnings:")
		for _, warning := range hardeningPlan.Warnings {
			cmd.Printf("  - %s\n", warning)
		}
	}

	cmd.Println()
	cmd.Println("plugins:")
	for _, plugin := range hardeningPlan.Plugins {
		cmd.Printf("  - %s: %s\n", plugin.ID, plugin.Summary)
	}

	cmd.Println()
	cmd.Println("planned actions:")
	for _, action := range hardeningPlan.Actions {
		risk := ""
		if action.Risky {
			risk = " [risky]"
		}
		cmd.Printf("  - %s:%s %s\n", action.Plugin, risk, action.Title)
		cmd.Printf("    %s\n", action.Detail)
	}
}

func printApplyResult(cmd *cobra.Command, result apply.Result) {
	cmd.Println()
	if result.LogPath != "" {
		cmd.Printf("log: %s\n", result.LogPath)
	}
	if result.ReportPath != "" {
		cmd.Printf("report: %s\n", result.ReportPath)
	}
	if result.UndoPlanPath != "" {
		cmd.Printf("undo plan: %s\n", result.UndoPlanPath)
	}
	if len(result.Applied) > 0 {
		cmd.Println("applied:")
		for _, item := range result.Applied {
			cmd.Printf("  - %s\n", item)
		}
	}
	if len(result.Skipped) > 0 {
		cmd.Println("skipped:")
		for _, item := range result.Skipped {
			cmd.Printf("  - %s\n", item)
		}
	}
	if len(result.Failed) > 0 {
		cmd.Println("failed:")
		for _, item := range result.Failed {
			cmd.Printf("  - %s\n", item)
		}
	}
}

// Execute runs the root command.
func Execute(version string) error {
	return newRootCmd(version).Execute()
}
