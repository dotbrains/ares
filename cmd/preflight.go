package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/dotbrains/ares/internal/apply"
	"github.com/dotbrains/ares/internal/config"
	"github.com/dotbrains/ares/internal/plan"
	"github.com/dotbrains/ares/internal/system"
	"github.com/spf13/cobra"
)

type preflightCheck struct {
	Name   string
	Status string
	Detail string
}

func newPreflightCmd() *cobra.Command {
	var profile string

	cmd := &cobra.Command{
		Use:   "preflight",
		Short: "Check apply readiness without changing the host",
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
			checks := buildPreflightChecks(host, hardeningPlan, os.Getenv("ARES_ROOT"))
			printPreflight(cmd, checks, apply.BuildTransaction(hardeningPlan))
			if hasFailedPreflight(checks) {
				return fmt.Errorf("preflight failed")
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&profile, "profile", "", "hardening profile: basic, web, strict")
	return cmd
}

func buildPreflightChecks(host system.Host, hardeningPlan plan.Plan, root string) []preflightCheck {
	checks := []preflightCheck{
		rootCheck(root),
		sshSessionCheck(host),
		knownValueCheck("package manager", host.PackageManager),
		knownValueCheck("firewall backend", host.FirewallBackend),
		knownValueCheck("ssh service", host.SSHService),
		providerCheck(host),
		providerAdvisoryCheck(hardeningPlan),
	}
	if host.SSHPort != "" {
		checks = append(checks, preflightCheck{Name: "ssh port", Status: "pass", Detail: host.SSHPort})
	} else {
		checks = append(checks, preflightCheck{Name: "ssh port", Status: "fail", Detail: "no SSH port detected"})
	}
	if len(hardeningPlan.Warnings) == 0 {
		checks = append(checks, preflightCheck{Name: "plan warnings", Status: "pass", Detail: "none"})
	} else {
		checks = append(checks, preflightCheck{Name: "plan warnings", Status: "warn", Detail: fmt.Sprintf("%d warning(s)", len(hardeningPlan.Warnings))})
	}
	checks = append(checks, reportDirectoryCheck(root))
	return checks
}

func rootCheck(root string) preflightCheck {
	if root != "" {
		return preflightCheck{Name: "privileges", Status: "pass", Detail: "ARES_ROOT test root active"}
	}
	if os.Geteuid() == 0 {
		return preflightCheck{Name: "privileges", Status: "pass", Detail: "running as root"}
	}
	return preflightCheck{Name: "privileges", Status: "fail", Detail: "apply mode requires root privileges"}
}

func sshSessionCheck(host system.Host) preflightCheck {
	if host.RunningOverSSH {
		return preflightCheck{Name: "ssh session", Status: "pass", Detail: "active SSH session detected"}
	}
	return preflightCheck{Name: "ssh session", Status: "warn", Detail: "active SSH session was not detected"}
}

func knownValueCheck(name string, value string) preflightCheck {
	if value == "" || value == "unknown" {
		return preflightCheck{Name: name, Status: "fail", Detail: "unknown"}
	}
	return preflightCheck{Name: name, Status: "pass", Detail: value}
}

func providerCheck(host system.Host) preflightCheck {
	if host.Provider == "" || host.Provider == "unknown" {
		return preflightCheck{Name: "provider", Status: "warn", Detail: "unknown provider"}
	}
	return preflightCheck{Name: "provider", Status: "pass", Detail: host.Provider}
}

func providerAdvisoryCheck(hardeningPlan plan.Plan) preflightCheck {
	for _, plugin := range hardeningPlan.Plugins {
		if strings.HasPrefix(plugin.ID, "provider-") {
			return preflightCheck{Name: "provider advisory", Status: "pass", Detail: plugin.ID}
		}
	}
	return preflightCheck{Name: "provider advisory", Status: "warn", Detail: "no provider advisory selected"}
}

func reportDirectoryCheck(root string) preflightCheck {
	dir := rootedPath(root, "/var/log/ares")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return preflightCheck{Name: "reports", Status: "fail", Detail: err.Error()}
	}
	probe := filepath.Join(dir, ".ares-preflight")
	err := os.WriteFile(probe, []byte("ok\n"), 0o600)
	if err != nil {
		return preflightCheck{Name: "reports", Status: "fail", Detail: err.Error()}
	}
	if err := os.Remove(probe); err != nil {
		return preflightCheck{Name: "reports", Status: "fail", Detail: err.Error()}
	}
	return preflightCheck{Name: "reports", Status: "pass", Detail: dir}
}

func rootedPath(root string, path string) string {
	if root == "" {
		return path
	}
	return filepath.Join(root, strings.TrimPrefix(path, "/"))
}

func hasFailedPreflight(checks []preflightCheck) bool {
	for _, check := range checks {
		if check.Status == "fail" {
			return true
		}
	}
	return false
}

func printPreflight(cmd *cobra.Command, checks []preflightCheck, transaction apply.TransactionSummary) {
	cmd.Println("preflight:")
	for _, check := range checks {
		cmd.Printf("  - %s: %s (%s)\n", check.Name, check.Status, check.Detail)
	}
	if len(transaction.Files) > 0 || len(transaction.Commands) > 0 || len(transaction.Backups) > 0 {
		cmd.Println("transaction:")
		printNamedList(cmd, "files", transaction.Files)
		printNamedList(cmd, "commands", transaction.Commands)
		printNamedList(cmd, "backups", transaction.Backups)
		printNamedList(cmd, "rollback", transaction.RollbackSteps)
	}
}
