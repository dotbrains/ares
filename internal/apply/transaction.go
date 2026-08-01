package apply

import (
	"fmt"
	"slices"
	"strings"

	"github.com/dotbrains/ares/internal/operations"
	"github.com/dotbrains/ares/internal/plan"
	"github.com/dotbrains/ares/internal/plugins"
	"github.com/dotbrains/ares/internal/reports"
)

type TransactionSummary = reports.TransactionSummary

func BuildTransaction(hardeningPlan plan.Plan) TransactionSummary {
	return operations.Summary(BuildOperations(hardeningPlan))
}

func BuildOperations(hardeningPlan plan.Plan) []operations.Operation {
	var ops []operations.Operation
	for _, plugin := range hardeningPlan.Plugins {
		ops = append(ops, operationsForPlugin(hardeningPlan, plugin)...)
	}
	return ops
}

func operationsForPlugin(hardeningPlan plan.Plan, plugin plugins.Plugin) []operations.Operation {
	if slices.Contains(plugin.Categories, "distro") || strings.HasPrefix(plugin.ID, "provider-") {
		return nil
	}
	var ops []operations.Operation
	for _, path := range plugin.ManagedFiles {
		ops = append(ops, operations.Operation{Kind: operations.WriteFile, Plugin: plugin.ID, Path: path})
	}
	for _, path := range plugin.BackupFiles {
		ops = append(ops, operations.Operation{Kind: operations.BackupFile, Plugin: plugin.ID, Path: path})
	}
	for _, step := range plugin.RollbackSteps {
		ops = append(ops, operations.Operation{Kind: operations.RollbackNote, Plugin: plugin.ID, Note: step})
	}
	switch plugin.ID {
	case "ssh-hardening":
		ops = append(ops,
			operations.Operation{Kind: operations.RunCommand, Plugin: plugin.ID, Command: "sshd", Args: []string{"-t"}},
			operations.Operation{Kind: operations.RunCommand, Plugin: plugin.ID, Command: "systemctl", Args: []string{"reload", hardeningPlan.Host.SSHService}},
		)
	case "firewall-ufw":
		ops = append(ops,
			operations.Operation{Kind: operations.RunCommand, Plugin: plugin.ID, Command: hardeningPlan.Host.PackageManager, Args: []string{"update"}},
			installOperation(plugin.ID, hardeningPlan.Host.PackageManager, "ufw"),
			operations.Operation{Kind: operations.RunCommand, Plugin: plugin.ID, Command: "ufw", Args: []string{"allow", hardeningPlan.Host.SSHPort + "/tcp"}},
			operations.Operation{Kind: operations.RunCommand, Plugin: plugin.ID, Command: "ufw", Args: []string{"default", "deny", "incoming"}},
			operations.Operation{Kind: operations.RunCommand, Plugin: plugin.ID, Command: "ufw", Args: []string{"default", "allow", "outgoing"}},
			operations.Operation{Kind: operations.RunCommand, Plugin: plugin.ID, Command: "ufw", Args: []string{"--force", "enable"}},
			operations.Operation{Kind: operations.RunCommand, Plugin: plugin.ID, Command: "ufw", Args: []string{"status"}},
		)
		ops = append(ops, operations.Operation{Kind: operations.RollbackNote, Plugin: plugin.ID, Note: "review firewall rules and run ufw disable if needed"})
	case "firewall-firewalld":
		ops = append(ops,
			installOperation(plugin.ID, hardeningPlan.Host.PackageManager, "firewalld"),
			operations.Operation{Kind: operations.RunCommand, Plugin: plugin.ID, Command: "systemctl", Args: []string{"enable", "--now", "firewalld"}},
			operations.Operation{Kind: operations.RunCommand, Plugin: plugin.ID, Command: "firewall-cmd", Args: []string{"--permanent", "--add-port=" + hardeningPlan.Host.SSHPort + "/tcp"}},
			operations.Operation{Kind: operations.RunCommand, Plugin: plugin.ID, Command: "firewall-cmd", Args: []string{"--set-default-zone=public"}},
			operations.Operation{Kind: operations.RunCommand, Plugin: plugin.ID, Command: "firewall-cmd", Args: []string{"--reload"}},
			operations.Operation{Kind: operations.RunCommand, Plugin: plugin.ID, Command: "firewall-cmd", Args: []string{"--list-all"}},
		)
		ops = append(ops, operations.Operation{Kind: operations.RollbackNote, Plugin: plugin.ID, Note: "review firewalld ports/services and reload after manual rollback"})
	case "firewall-nftables":
		ops = append(ops, installOperation(plugin.ID, hardeningPlan.Host.PackageManager, "nftables"))
		ops = append(ops,
			operations.Operation{Kind: operations.RunCommand, Plugin: plugin.ID, Command: "nft", Args: []string{"-c", "-f", "/etc/nftables.conf"}},
			operations.Operation{Kind: operations.RunCommand, Plugin: plugin.ID, Command: "systemctl", Args: []string{"enable", "--now", "nftables"}},
			operations.Operation{Kind: operations.RunCommand, Plugin: plugin.ID, Command: "nft", Args: []string{"list", "ruleset"}},
		)
	case "fail2ban":
		ops = append(ops, installOperation(plugin.ID, hardeningPlan.Host.PackageManager, "fail2ban"))
		ops = append(ops, operations.Operation{Kind: operations.RunCommand, Plugin: plugin.ID, Command: "systemctl", Args: []string{"enable", "--now", "fail2ban"}})
	case "unattended-upgrades":
		ops = append(ops,
			operations.Operation{Kind: operations.RunCommand, Plugin: plugin.ID, Command: hardeningPlan.Host.PackageManager, Args: []string{"update"}},
			installOperation(plugin.ID, hardeningPlan.Host.PackageManager, "unattended-upgrades"),
		)
	case "dnf-automatic":
		ops = append(ops, installOperation(plugin.ID, hardeningPlan.Host.PackageManager, "dnf-automatic"))
		ops = append(ops, operations.Operation{Kind: operations.RunCommand, Plugin: plugin.ID, Command: "systemctl", Args: []string{"enable", "--now", "dnf-automatic.timer"}})
	case "pacman-upgrade":
		ops = append(ops, operations.Operation{Kind: operations.RunCommand, Plugin: plugin.ID, Command: "pacman", Args: []string{"-Syu", "--noconfirm"}})
	case "zypper-patches":
		ops = append(ops, operations.Operation{Kind: operations.RunCommand, Plugin: plugin.ID, Command: "zypper", Args: []string{"--non-interactive", "patch"}})
	case "apk-upgrade":
		ops = append(ops,
			operations.Operation{Kind: operations.RunCommand, Plugin: plugin.ID, Command: "apk", Args: []string{"update"}},
			operations.Operation{Kind: operations.RunCommand, Plugin: plugin.ID, Command: "apk", Args: []string{"upgrade"}},
		)
	case "sysctl-baseline":
		ops = append(ops, operations.Operation{Kind: operations.RunCommand, Plugin: plugin.ID, Command: "sysctl", Args: []string{"--system"}})
	case "web-profile":
		ops = append(ops, webOperations(plugin.ID, hardeningPlan)...)
	case "strict-profile":
	default:
		if plugin.Kind == "custom" {
			ops = append(ops, customOperations(plugin)...)
		}
	}
	return ops
}

func webOperations(pluginID string, hardeningPlan plan.Plan) []operations.Operation {
	switch hardeningPlan.Host.FirewallBackend {
	case "firewalld":
		return []operations.Operation{
			{Kind: operations.RunCommand, Plugin: pluginID, Command: "firewall-cmd", Args: []string{"--permanent", "--add-service=http"}},
			{Kind: operations.RunCommand, Plugin: pluginID, Command: "firewall-cmd", Args: []string{"--permanent", "--add-service=https"}},
			{Kind: operations.RunCommand, Plugin: pluginID, Command: "firewall-cmd", Args: []string{"--reload"}},
			{Kind: operations.RunCommand, Plugin: pluginID, Command: "firewall-cmd", Args: []string{"--list-all"}},
		}
	case "nftables":
		return []operations.Operation{
			{Kind: operations.WriteFile, Plugin: pluginID, Path: "/etc/nftables.conf"},
			{Kind: operations.BackupFile, Plugin: pluginID, Path: "/etc/nftables.conf"},
			{Kind: operations.RunCommand, Plugin: pluginID, Command: "nft", Args: []string{"-c", "-f", "/etc/nftables.conf"}},
			{Kind: operations.RunCommand, Plugin: pluginID, Command: "nft", Args: []string{"-f", "/etc/nftables.conf"}},
			{Kind: operations.RunCommand, Plugin: pluginID, Command: "nft", Args: []string{"list", "ruleset"}},
		}
	case "ufw":
		return []operations.Operation{
			{Kind: operations.RunCommand, Plugin: pluginID, Command: "ufw", Args: []string{"allow", "80/tcp"}},
			{Kind: operations.RunCommand, Plugin: pluginID, Command: "ufw", Args: []string{"allow", "443/tcp"}},
			{Kind: operations.RunCommand, Plugin: pluginID, Command: "ufw", Args: []string{"status"}},
		}
	}
	return nil
}

func customOperations(plugin plugins.Plugin) []operations.Operation {
	var ops []operations.Operation
	if plugin.Probe != "" {
		ops = append(ops, operations.Operation{Kind: operations.CustomCommand, Plugin: plugin.ID, Phase: "probe", Command: plugin.Probe})
	}
	if plugin.Apply != "" {
		ops = append(ops, operations.Operation{Kind: operations.CustomCommand, Plugin: plugin.ID, Phase: "apply", Command: plugin.Apply})
	}
	if plugin.Verify != "" {
		ops = append(ops, operations.Operation{Kind: operations.CustomCommand, Plugin: plugin.ID, Phase: "verify", Command: plugin.Verify})
	}
	if plugin.Rollback != "" {
		ops = append(ops, operations.Operation{Kind: operations.RollbackNote, Plugin: plugin.ID, Note: "custom " + plugin.ID + " rollback: " + plugin.Rollback})
	}
	return ops
}

func installOperation(pluginID string, packageManager string, packages ...string) operations.Operation {
	name, args, err := installCommand(packageManager, packages...)
	if err != nil {
		return operations.Operation{Kind: operations.RunCommand, Plugin: pluginID, Command: fmt.Sprintf("install %s with unsupported package manager %s", strings.Join(packages, " "), packageManager)}
	}
	return operations.Operation{Kind: operations.RunCommand, Plugin: pluginID, Command: name, Args: args}
}
