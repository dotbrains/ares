package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	osexec "os/exec"
	"path/filepath"
	"strings"

	"github.com/dotbrains/ares/internal/apply"
	"github.com/dotbrains/ares/internal/config"
	"github.com/dotbrains/ares/internal/plan"
	"github.com/dotbrains/ares/internal/system"
	"github.com/spf13/cobra"
)

type preflightCheck struct {
	Name   string `json:"name"`
	Status string `json:"status"`
	Detail string `json:"detail"`
}

type preflightReport struct {
	Profile     string                   `json:"profile"`
	Host        system.Host              `json:"host"`
	Plugins     []string                 `json:"plugins"`
	Checks      []preflightCheck         `json:"checks"`
	Transaction apply.TransactionSummary `json:"transaction"`
}

func newPreflightCmd() *cobra.Command {
	var profile string
	var jsonOutput bool

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
			checks := buildPreflightChecks(host, hardeningPlan, cfg, os.Getenv("ARES_ROOT"))
			report := buildPreflightReport(host, hardeningPlan, checks)
			if jsonOutput {
				if err := printPreflightJSON(cmd, report); err != nil {
					return err
				}
			} else {
				printPreflight(cmd, report)
			}
			if hasFailedPreflight(checks) {
				return fmt.Errorf("preflight failed")
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&profile, "profile", "", "hardening profile: basic, web, strict")
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "print preflight report as JSON")
	return cmd
}

func buildPreflightChecks(host system.Host, hardeningPlan plan.Plan, cfg *config.Config, root string) []preflightCheck {
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
	checks = append(checks, customPluginCommandChecks(cfg)...)
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

func customPluginCommandChecks(cfg *config.Config) []preflightCheck {
	var checks []preflightCheck
	for _, plugin := range cfg.Plugins.Custom {
		for _, command := range []struct {
			name  string
			value string
		}{
			{name: "probe", value: plugin.Probe},
			{name: "apply", value: plugin.Apply},
			{name: "verify", value: plugin.Verify},
			{name: "rollback", value: plugin.Rollback},
		} {
			if strings.TrimSpace(command.value) == "" {
				continue
			}
			checks = append(checks, customCommandCheck(plugin.Name, command.name, command.value))
		}
	}
	return checks
}

func customCommandCheck(pluginName string, phase string, command string) preflightCheck {
	executable := firstCommandWord(command)
	name := "custom " + pluginName + " " + phase
	if executable == "" {
		return preflightCheck{Name: name, Status: "fail", Detail: "missing executable"}
	}
	if strings.Contains(executable, "/") {
		if _, err := os.Stat(executable); err != nil {
			return preflightCheck{Name: name, Status: "fail", Detail: executable + " not found"}
		}
		return preflightCheck{Name: name, Status: "pass", Detail: executable}
	}
	if path, err := osexec.LookPath(executable); err == nil {
		return preflightCheck{Name: name, Status: "pass", Detail: path}
	}
	return preflightCheck{Name: name, Status: "fail", Detail: executable + " not found on PATH"}
}

func firstCommandWord(command string) string {
	fields := strings.Fields(command)
	if len(fields) == 0 {
		return ""
	}
	return fields[0]
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

func buildPreflightReport(host system.Host, hardeningPlan plan.Plan, checks []preflightCheck) preflightReport {
	ids := make([]string, 0, len(hardeningPlan.Plugins))
	for _, plugin := range hardeningPlan.Plugins {
		ids = append(ids, plugin.ID)
	}
	return preflightReport{
		Profile:     hardeningPlan.Profile,
		Host:        host,
		Plugins:     ids,
		Checks:      checks,
		Transaction: apply.BuildTransaction(hardeningPlan),
	}
}

func printPreflight(cmd *cobra.Command, report preflightReport) {
	cmd.Println("preflight:")
	for _, check := range report.Checks {
		cmd.Printf("  - %s: %s (%s)\n", check.Name, check.Status, check.Detail)
	}
	if len(report.Transaction.Files) > 0 || len(report.Transaction.Commands) > 0 || len(report.Transaction.Backups) > 0 {
		cmd.Println("transaction:")
		printNamedList(cmd, "files", report.Transaction.Files)
		printNamedList(cmd, "commands", report.Transaction.Commands)
		printNamedList(cmd, "backups", report.Transaction.Backups)
		printNamedList(cmd, "rollback", report.Transaction.RollbackSteps)
	}
}

func printPreflightJSON(cmd *cobra.Command, report preflightReport) error {
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return err
	}
	cmd.Println(string(data))
	return nil
}
