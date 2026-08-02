package intent

import (
	"fmt"

	"github.com/dotbrains/ares/internal/plugins"
	"github.com/dotbrains/ares/internal/system"
)

type Action struct {
	Plugin string
	Title  string
	Detail string
	Risky  bool
}

type OperationKind string

const (
	WriteFile     OperationKind = "write_file"
	BackupFile    OperationKind = "backup_file"
	RunCommand    OperationKind = "run_command"
	RollbackNote  OperationKind = "rollback_note"
	CustomCommand OperationKind = "custom_command"
)

type Operation struct {
	Kind    OperationKind
	Plugin  string
	Path    string
	Command string
	Args    []string
	Note    string
	Phase   string
}

type Intent struct {
	Host    system.Host
	Profile string
	Plugin  plugins.Plugin
}

func ForPlugin(host system.Host, profile string, plugin plugins.Plugin) Intent {
	return Intent{Host: host, Profile: profile, Plugin: plugin}
}

func (intent Intent) Actions() []Action {
	plugin := intent.Plugin
	behavior := plugins.Behavior(plugin)
	host := intent.Host
	if behavior.Name == "distro" || hasCategory(plugin, "distro") {
		return []Action{{
			Plugin: plugin.ID,
			Title:  "Use distro adapter",
			Detail: fmt.Sprintf("Use %s, %s, and SSH service %s for host operations", host.PackageManager, host.InitSystem, host.SSHService),
		}}
	}

	switch behavior.Name {
	case "ssh-hardening":
		return []Action{
			{
				Plugin: plugin.ID,
				Title:  "Back up SSH configuration",
				Detail: "Create timestamped backups before writing a managed sshd drop-in",
			},
			{
				Plugin: plugin.ID,
				Title:  "Preserve active SSH access",
				Detail: fmt.Sprintf("Keep port %s reachable, validate sshd config, then reload %s", host.SSHPort, host.SSHService),
				Risky:  true,
			},
		}
	case "firewall":
		return []Action{{
			Plugin: plugin.ID,
			Title:  "Configure firewall",
			Detail: fmt.Sprintf("Use %s, allow SSH port %s, deny other inbound traffic, allow outbound traffic", host.FirewallBackend, host.SSHPort),
			Risky:  true,
		}}
	case "fail2ban":
		return []Action{{
			Plugin: plugin.ID,
			Title:  "Enable fail2ban",
			Detail: "Install and enable a conservative SSH jail",
		}}
	case "security-updates":
		return []Action{{
			Plugin: plugin.ID,
			Title:  "Configure distro-native updates",
			Detail: "Use the selected distro update mechanism without automatic reboots",
		}}
	case "sysctl":
		return []Action{{
			Plugin: plugin.ID,
			Title:  "Apply sysctl baseline",
			Detail: "Write conservative network hardening to /etc/sysctl.d/99-ares.conf",
		}}
	case "web-profile":
		return []Action{{
			Plugin: plugin.ID,
			Title:  "Allow web traffic",
			Detail: "Allow inbound 80/tcp and 443/tcp through the selected firewall backend",
			Risky:  true,
		}}
	case "strict-profile":
		return []Action{{
			Plugin: plugin.ID,
			Title:  "Apply strict profile",
			Detail: "Use stricter fail2ban defaults and document optional root account lockout steps",
			Risky:  true,
		}}
	case "provider-advisory":
		return []Action{{
			Plugin: plugin.ID,
			Title:  "Record provider advisory",
			Detail: "Add provider-specific recovery and firewall-console reminders to the run report without mutating provider settings",
		}}
	default:
		if plugin.Kind == "custom" {
			return []Action{{
				Plugin: plugin.ID,
				Title:  "Run custom plugin",
				Detail: fmt.Sprintf("Probe with %q, plan with %q, apply with %q", plugin.Probe, plugin.Plan, plugin.Apply),
				Risky:  true,
			}}
		}
		return []Action{{
			Plugin: plugin.ID,
			Title:  "Run plugin",
			Detail: fmt.Sprintf("Execute plugin under the %s profile", intent.Profile),
		}}
	}
}

func (intent Intent) Operations() []Operation {
	plugin := intent.Plugin
	behavior := plugins.Behavior(plugin)
	if behavior.Name == "distro" || behavior.Name == "provider-advisory" || hasCategory(plugin, "distro") {
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
	switch behavior.Name {
	case "ssh-hardening":
		ops = append(ops,
			Operation{Kind: RunCommand, Plugin: plugin.ID, Command: "sshd", Args: []string{"-t"}},
			Operation{Kind: RunCommand, Plugin: plugin.ID, Command: "systemctl", Args: []string{"reload", intent.Host.SSHService}},
		)
	case "firewall":
		ops = append(ops, intent.firewallOperations()...)
	case "fail2ban":
		ops = append(ops, installOperation(plugin.ID, intent.Host.PackageManager, "fail2ban"))
		ops = append(ops, Operation{Kind: RunCommand, Plugin: plugin.ID, Command: "systemctl", Args: []string{"enable", "--now", "fail2ban"}})
	case "security-updates":
		ops = append(ops, securityUpdateOperations(plugin, intent.Host.PackageManager)...)
	case "sysctl":
		ops = append(ops, Operation{Kind: RunCommand, Plugin: plugin.ID, Command: "sysctl", Args: []string{"--system"}})
	case "web-profile":
		ops = append(ops, intent.webOperations()...)
	default:
		if plugin.Kind == "custom" {
			ops = append(ops, customOperations(plugin)...)
		}
	}
	return ops
}
