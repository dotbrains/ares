package intent

import (
	"fmt"
	"strings"

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

func (intent Intent) firewallOperations() []Operation {
	pluginID := intent.Plugin.ID
	switch plugins.Behavior(intent.Plugin).Variant {
	case "ufw":
		ops := []Operation{
			{Kind: RunCommand, Plugin: pluginID, Command: intent.Host.PackageManager, Args: []string{"update"}},
			installOperation(pluginID, intent.Host.PackageManager, "ufw"),
			{Kind: RunCommand, Plugin: pluginID, Command: "ufw", Args: []string{"allow", intent.Host.SSHPort + "/tcp"}},
			{Kind: RunCommand, Plugin: pluginID, Command: "ufw", Args: []string{"default", "deny", "incoming"}},
			{Kind: RunCommand, Plugin: pluginID, Command: "ufw", Args: []string{"default", "allow", "outgoing"}},
			{Kind: RunCommand, Plugin: pluginID, Command: "ufw", Args: []string{"--force", "enable"}},
			{Kind: RunCommand, Plugin: pluginID, Command: "ufw", Args: []string{"status"}},
		}
		return append(ops, Operation{Kind: RollbackNote, Plugin: pluginID, Note: "review firewall rules and run ufw disable if needed"})
	case "firewalld":
		ops := []Operation{
			installOperation(pluginID, intent.Host.PackageManager, "firewalld"),
			{Kind: RunCommand, Plugin: pluginID, Command: "systemctl", Args: []string{"enable", "--now", "firewalld"}},
			{Kind: RunCommand, Plugin: pluginID, Command: "firewall-cmd", Args: []string{"--permanent", "--add-port=" + intent.Host.SSHPort + "/tcp"}},
			{Kind: RunCommand, Plugin: pluginID, Command: "firewall-cmd", Args: []string{"--set-default-zone=public"}},
			{Kind: RunCommand, Plugin: pluginID, Command: "firewall-cmd", Args: []string{"--reload"}},
			{Kind: RunCommand, Plugin: pluginID, Command: "firewall-cmd", Args: []string{"--list-all"}},
		}
		return append(ops, Operation{Kind: RollbackNote, Plugin: pluginID, Note: "review firewalld ports/services and reload after manual rollback"})
	case "nftables":
		return []Operation{
			installOperation(pluginID, intent.Host.PackageManager, "nftables"),
			{Kind: RunCommand, Plugin: pluginID, Command: "nft", Args: []string{"-c", "-f", "/etc/nftables.conf"}},
			{Kind: RunCommand, Plugin: pluginID, Command: "systemctl", Args: []string{"enable", "--now", "nftables"}},
			{Kind: RunCommand, Plugin: pluginID, Command: "nft", Args: []string{"list", "ruleset"}},
		}
	}
	return nil
}

func (intent Intent) webOperations() []Operation {
	pluginID := intent.Plugin.ID
	switch intent.Host.FirewallBackend {
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

func securityUpdateOperations(plugin plugins.Plugin, packageManager string) []Operation {
	pluginID := plugin.ID
	switch plugins.Behavior(plugin).Variant {
	case "apt":
		return []Operation{
			{Kind: RunCommand, Plugin: pluginID, Command: packageManager, Args: []string{"update"}},
			installOperation(pluginID, packageManager, "unattended-upgrades"),
		}
	case "dnf-automatic":
		return []Operation{
			installOperation(pluginID, packageManager, "dnf-automatic"),
			{Kind: RunCommand, Plugin: pluginID, Command: "systemctl", Args: []string{"enable", "--now", "dnf-automatic.timer"}},
		}
	case "pacman":
		return []Operation{{Kind: RunCommand, Plugin: pluginID, Command: "pacman", Args: []string{"-Syu", "--noconfirm"}}}
	case "zypper":
		return []Operation{{Kind: RunCommand, Plugin: pluginID, Command: "zypper", Args: []string{"--non-interactive", "patch"}}}
	case "apk":
		return []Operation{
			{Kind: RunCommand, Plugin: pluginID, Command: "apk", Args: []string{"update"}},
			{Kind: RunCommand, Plugin: pluginID, Command: "apk", Args: []string{"upgrade"}},
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
	name, args, err := InstallCommand(packageManager, packages...)
	if err != nil {
		return Operation{Kind: RunCommand, Plugin: pluginID, Command: fmt.Sprintf("install %s with unsupported package manager %s", strings.Join(packages, " "), packageManager)}
	}
	return Operation{Kind: RunCommand, Plugin: pluginID, Command: name, Args: args}
}

func InstallCommand(packageManager string, packages ...string) (string, []string, error) {
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

func hasCategory(plugin plugins.Plugin, category string) bool {
	for _, item := range plugin.Categories {
		if item == category {
			return true
		}
	}
	return false
}
