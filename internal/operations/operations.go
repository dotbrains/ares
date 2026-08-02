package operations

import (
	"fmt"
	"slices"
	"strings"

	"github.com/dotbrains/ares/internal/plan"
	"github.com/dotbrains/ares/internal/plugins"
	"github.com/dotbrains/ares/internal/reports"
)

type Kind string

const (
	WriteFile     Kind = "write_file"
	BackupFile    Kind = "backup_file"
	RunCommand    Kind = "run_command"
	RollbackNote  Kind = "rollback_note"
	CustomCommand Kind = "custom_command"
)

type Operation struct {
	Kind    Kind
	Plugin  string
	Path    string
	Command string
	Args    []string
	Note    string
	Phase   string
}

func Build(hardeningPlan plan.Plan) []Operation {
	var ops []Operation
	for _, plugin := range hardeningPlan.Plugins {
		ops = append(ops, forPlugin(hardeningPlan, plugin)...)
	}
	return ops
}

func SummaryForPlan(hardeningPlan plan.Plan) reports.TransactionSummary {
	return Summary(Build(hardeningPlan))
}

func forPlugin(hardeningPlan plan.Plan, plugin plugins.Plugin) []Operation {
	if slices.Contains(plugin.Categories, "distro") || plugin.Behavior == "provider-advisory" {
		return nil
	}
	var ops []Operation
	for _, path := range plugin.ManagedFiles {
		ops = append(ops, Operation{Kind: WriteFile, Plugin: plugin.ID, Path: path})
	}
	for _, path := range plugin.BackupFiles {
		ops = append(ops, Operation{Kind: BackupFile, Plugin: plugin.ID, Path: path})
	}
	for _, step := range plugin.RollbackSteps {
		ops = append(ops, Operation{Kind: RollbackNote, Plugin: plugin.ID, Note: step})
	}
	switch plugin.Behavior {
	case "ssh-hardening":
		ops = append(ops,
			Operation{Kind: RunCommand, Plugin: plugin.ID, Command: "sshd", Args: []string{"-t"}},
			Operation{Kind: RunCommand, Plugin: plugin.ID, Command: "systemctl", Args: []string{"reload", hardeningPlan.Host.SSHService}},
		)
	case "firewall":
		ops = append(ops, firewallOperations(plugin.ID, hardeningPlan)...)
	case "fail2ban":
		ops = append(ops, installOperation(plugin.ID, hardeningPlan.Host.PackageManager, "fail2ban"))
		ops = append(ops, Operation{Kind: RunCommand, Plugin: plugin.ID, Command: "systemctl", Args: []string{"enable", "--now", "fail2ban"}})
	case "security-updates":
		ops = append(ops, securityUpdateOperations(plugin.ID, hardeningPlan.Host.PackageManager)...)
	case "sysctl":
		ops = append(ops, Operation{Kind: RunCommand, Plugin: plugin.ID, Command: "sysctl", Args: []string{"--system"}})
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

func firewallOperations(pluginID string, hardeningPlan plan.Plan) []Operation {
	switch pluginID {
	case "firewall-ufw":
		ops := []Operation{
			{Kind: RunCommand, Plugin: pluginID, Command: hardeningPlan.Host.PackageManager, Args: []string{"update"}},
			installOperation(pluginID, hardeningPlan.Host.PackageManager, "ufw"),
			{Kind: RunCommand, Plugin: pluginID, Command: "ufw", Args: []string{"allow", hardeningPlan.Host.SSHPort + "/tcp"}},
			{Kind: RunCommand, Plugin: pluginID, Command: "ufw", Args: []string{"default", "deny", "incoming"}},
			{Kind: RunCommand, Plugin: pluginID, Command: "ufw", Args: []string{"default", "allow", "outgoing"}},
			{Kind: RunCommand, Plugin: pluginID, Command: "ufw", Args: []string{"--force", "enable"}},
			{Kind: RunCommand, Plugin: pluginID, Command: "ufw", Args: []string{"status"}},
		}
		return append(ops, Operation{Kind: RollbackNote, Plugin: pluginID, Note: "review firewall rules and run ufw disable if needed"})
	case "firewall-firewalld":
		ops := []Operation{
			installOperation(pluginID, hardeningPlan.Host.PackageManager, "firewalld"),
			{Kind: RunCommand, Plugin: pluginID, Command: "systemctl", Args: []string{"enable", "--now", "firewalld"}},
			{Kind: RunCommand, Plugin: pluginID, Command: "firewall-cmd", Args: []string{"--permanent", "--add-port=" + hardeningPlan.Host.SSHPort + "/tcp"}},
			{Kind: RunCommand, Plugin: pluginID, Command: "firewall-cmd", Args: []string{"--set-default-zone=public"}},
			{Kind: RunCommand, Plugin: pluginID, Command: "firewall-cmd", Args: []string{"--reload"}},
			{Kind: RunCommand, Plugin: pluginID, Command: "firewall-cmd", Args: []string{"--list-all"}},
		}
		return append(ops, Operation{Kind: RollbackNote, Plugin: pluginID, Note: "review firewalld ports/services and reload after manual rollback"})
	case "firewall-nftables":
		return []Operation{
			installOperation(pluginID, hardeningPlan.Host.PackageManager, "nftables"),
			{Kind: RunCommand, Plugin: pluginID, Command: "nft", Args: []string{"-c", "-f", "/etc/nftables.conf"}},
			{Kind: RunCommand, Plugin: pluginID, Command: "systemctl", Args: []string{"enable", "--now", "nftables"}},
			{Kind: RunCommand, Plugin: pluginID, Command: "nft", Args: []string{"list", "ruleset"}},
		}
	}
	return nil
}

func securityUpdateOperations(pluginID string, packageManager string) []Operation {
	switch pluginID {
	case "unattended-upgrades":
		return []Operation{
			{Kind: RunCommand, Plugin: pluginID, Command: packageManager, Args: []string{"update"}},
			installOperation(pluginID, packageManager, "unattended-upgrades"),
		}
	case "dnf-automatic":
		return []Operation{
			installOperation(pluginID, packageManager, "dnf-automatic"),
			{Kind: RunCommand, Plugin: pluginID, Command: "systemctl", Args: []string{"enable", "--now", "dnf-automatic.timer"}},
		}
	case "pacman-upgrade":
		return []Operation{{Kind: RunCommand, Plugin: pluginID, Command: "pacman", Args: []string{"-Syu", "--noconfirm"}}}
	case "zypper-patches":
		return []Operation{{Kind: RunCommand, Plugin: pluginID, Command: "zypper", Args: []string{"--non-interactive", "patch"}}}
	case "apk-upgrade":
		return []Operation{
			{Kind: RunCommand, Plugin: pluginID, Command: "apk", Args: []string{"update"}},
			{Kind: RunCommand, Plugin: pluginID, Command: "apk", Args: []string{"upgrade"}},
		}
	}
	return nil
}

func webOperations(pluginID string, hardeningPlan plan.Plan) []Operation {
	switch hardeningPlan.Host.FirewallBackend {
	case "firewalld":
		return []Operation{
			{Kind: RunCommand, Plugin: pluginID, Command: "firewall-cmd", Args: []string{"--permanent", "--add-service=http"}},
			{Kind: RunCommand, Plugin: pluginID, Command: "firewall-cmd", Args: []string{"--permanent", "--add-service=https"}},
			{Kind: RunCommand, Plugin: pluginID, Command: "firewall-cmd", Args: []string{"--reload"}},
			{Kind: RunCommand, Plugin: pluginID, Command: "firewall-cmd", Args: []string{"--list-all"}},
		}
	case "nftables":
		return []Operation{
			{Kind: WriteFile, Plugin: pluginID, Path: "/etc/nftables.conf"},
			{Kind: BackupFile, Plugin: pluginID, Path: "/etc/nftables.conf"},
			{Kind: RunCommand, Plugin: pluginID, Command: "nft", Args: []string{"-c", "-f", "/etc/nftables.conf"}},
			{Kind: RunCommand, Plugin: pluginID, Command: "nft", Args: []string{"-f", "/etc/nftables.conf"}},
			{Kind: RunCommand, Plugin: pluginID, Command: "nft", Args: []string{"list", "ruleset"}},
		}
	case "ufw":
		return []Operation{
			{Kind: RunCommand, Plugin: pluginID, Command: "ufw", Args: []string{"allow", "80/tcp"}},
			{Kind: RunCommand, Plugin: pluginID, Command: "ufw", Args: []string{"allow", "443/tcp"}},
			{Kind: RunCommand, Plugin: pluginID, Command: "ufw", Args: []string{"status"}},
		}
	}
	return nil
}

func customOperations(plugin plugins.Plugin) []Operation {
	var ops []Operation
	if plugin.Probe != "" {
		ops = append(ops, Operation{Kind: CustomCommand, Plugin: plugin.ID, Phase: "probe", Command: plugin.Probe})
	}
	if plugin.Apply != "" {
		ops = append(ops, Operation{Kind: CustomCommand, Plugin: plugin.ID, Phase: "apply", Command: plugin.Apply})
	}
	if plugin.Verify != "" {
		ops = append(ops, Operation{Kind: CustomCommand, Plugin: plugin.ID, Phase: "verify", Command: plugin.Verify})
	}
	if plugin.Rollback != "" {
		ops = append(ops, Operation{Kind: RollbackNote, Plugin: plugin.ID, Note: "custom " + plugin.ID + " rollback: " + plugin.Rollback})
	}
	return ops
}

func installOperation(pluginID string, packageManager string, packages ...string) Operation {
	name, args, err := installCommand(packageManager, packages...)
	if err != nil {
		return Operation{Kind: RunCommand, Plugin: pluginID, Command: fmt.Sprintf("install %s with unsupported package manager %s", strings.Join(packages, " "), packageManager)}
	}
	return Operation{Kind: RunCommand, Plugin: pluginID, Command: name, Args: args}
}

func installCommand(packageManager string, packages ...string) (string, []string, error) {
	switch packageManager {
	case "apt-get":
		args := append([]string{"install", "-y"}, packages...)
		return packageManager, args, nil
	case "dnf", "yum":
		args := append([]string{"install", "-y"}, packages...)
		return packageManager, args, nil
	case "pacman":
		args := append([]string{"-S", "--needed", "--noconfirm"}, packages...)
		return packageManager, args, nil
	case "zypper":
		args := append([]string{"--non-interactive", "install"}, packages...)
		return packageManager, args, nil
	case "apk":
		args := append([]string{"add"}, packages...)
		return packageManager, args, nil
	default:
		return "", nil, fmt.Errorf("unsupported package manager %q", packageManager)
	}
}

func Summary(ops []Operation) reports.TransactionSummary {
	var summary reports.TransactionSummary
	for _, op := range ops {
		switch op.Kind {
		case WriteFile:
			summary.Files = append(summary.Files, op.Path)
		case BackupFile:
			summary.Backups = append(summary.Backups, op.Path)
		case RunCommand:
			summary.Commands = append(summary.Commands, commandString(op))
		case RollbackNote:
			summary.RollbackSteps = append(summary.RollbackSteps, op.Note)
		case CustomCommand:
			summary.Commands = append(summary.Commands, "custom "+op.Plugin+" "+op.Phase+": "+op.Command)
		}
	}
	summary.Files = uniqueStrings(summary.Files)
	summary.Commands = uniqueStrings(summary.Commands)
	summary.Backups = uniqueStrings(summary.Backups)
	summary.RollbackSteps = uniqueStrings(summary.RollbackSteps)
	return summary
}

func commandString(op Operation) string {
	if len(op.Args) == 0 {
		return op.Command
	}
	return op.Command + " " + strings.Join(op.Args, " ")
}

func RollbackPreview(summary reports.TransactionSummary) []string {
	if len(summary.Files) == 0 && len(summary.Backups) == 0 {
		return nil
	}
	backedUp := map[string]bool{}
	var preview []string
	for _, path := range summary.Backups {
		backedUp[path] = true
		preview = append(preview, "would restore newest backup for "+path)
	}
	for _, path := range summary.Files {
		if backedUp[path] {
			continue
		}
		preview = append(preview, "would remove "+path)
	}
	return preview
}

func uniqueStrings(values []string) []string {
	seen := map[string]bool{}
	result := make([]string, 0, len(values))
	for _, value := range values {
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		result = append(result, value)
	}
	return result
}
