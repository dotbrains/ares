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

var actionBuilders = map[string]func(Intent) []Action{
	"ssh-hardening":     sshHardeningActions,
	"firewall":          firewallActions,
	"fail2ban":          fail2banActions,
	"security-updates":  securityUpdateActions,
	"sysctl":            sysctlActions,
	"tailscale-ssh":     tailscaleSSHActions,
	"web-profile":       webProfileActions,
	"strict-profile":    strictProfileActions,
	"provider-advisory": providerAdvisoryActions,
}

var operationBuilders = map[string]func(Intent) []Operation{
	"ssh-hardening":    sshHardeningOperations,
	"firewall":         Intent.firewallOperations,
	"fail2ban":         fail2banOperations,
	"security-updates": securityUpdateIntentOperations,
	"sysctl":           sysctlOperations,
	"tailscale-ssh":    tailscaleSSHOperations,
	"web-profile":      Intent.webOperations,
}

func ForPlugin(host system.Host, profile string, plugin plugins.Plugin) Intent {
	return Intent{Host: host, Profile: profile, Plugin: plugin}
}

func (intent Intent) Actions() []Action {
	plugin := intent.Plugin
	behavior := plugins.Behavior(plugin)
	host := intent.Host
	if behavior.IsDistro() || hasCategory(plugin, "distro") {
		return []Action{{
			Plugin: plugin.ID,
			Title:  "Use distro adapter",
			Detail: fmt.Sprintf("Use %s, %s, and SSH service %s for host operations", host.PackageManager, host.InitSystem, host.SSHService),
		}}
	}
	if build, ok := actionBuilders[behavior.Name]; ok {
		return build(intent)
	}
	return fallbackActions(intent)
}

func (intent Intent) Operations() []Operation {
	plugin := intent.Plugin
	behavior := plugins.Behavior(plugin)
	if behavior.IsPlanningOnly() || hasCategory(plugin, "distro") {
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
	if build, ok := operationBuilders[behavior.Name]; ok {
		ops = append(ops, build(intent)...)
	} else if plugin.Kind == "custom" {
		ops = append(ops, customOperations(plugin)...)
	}
	return ops
}

func sshHardeningActions(intent Intent) []Action {
	return []Action{
		{
			Plugin: intent.Plugin.ID,
			Title:  "Back up SSH configuration",
			Detail: "Create timestamped backups before writing a managed sshd drop-in",
		},
		{
			Plugin: intent.Plugin.ID,
			Title:  "Preserve active SSH access",
			Detail: fmt.Sprintf("Keep port %s reachable, validate sshd config, then reload %s", intent.Host.SSHPort, intent.Host.SSHService),
			Risky:  true,
		},
	}
}

func firewallActions(intent Intent) []Action {
	return []Action{{
		Plugin: intent.Plugin.ID,
		Title:  "Configure firewall",
		Detail: fmt.Sprintf("Use %s, allow SSH port %s, deny other inbound traffic, allow outbound traffic", intent.Host.FirewallBackend, intent.Host.SSHPort),
		Risky:  true,
	}}
}

func fail2banActions(intent Intent) []Action {
	return []Action{{
		Plugin: intent.Plugin.ID,
		Title:  "Enable fail2ban",
		Detail: "Install and enable a conservative SSH jail",
	}}
}

func securityUpdateActions(intent Intent) []Action {
	return []Action{{
		Plugin: intent.Plugin.ID,
		Title:  "Configure distro-native updates",
		Detail: "Use the selected distro update mechanism without automatic reboots",
	}}
}

func sysctlActions(intent Intent) []Action {
	return []Action{{
		Plugin: intent.Plugin.ID,
		Title:  "Apply sysctl baseline",
		Detail: "Write conservative network hardening to /etc/sysctl.d/99-ares.conf",
	}}
}

func tailscaleSSHActions(intent Intent) []Action {
	return []Action{
		{
			Plugin: intent.Plugin.ID,
			Title:  "Prepare Tailscale",
			Detail: "Install Tailscale and enable tailscaled without joining a tailnet automatically",
			Risky:  true,
		},
		{
			Plugin: intent.Plugin.ID,
			Title:  "Keep tailnet SSH explicit",
			Detail: "Record manual tailscale up --ssh guidance so authentication and tailnet policy remain operator-controlled",
			Risky:  true,
		},
	}
}

func webProfileActions(intent Intent) []Action {
	return []Action{{
		Plugin: intent.Plugin.ID,
		Title:  "Allow web traffic",
		Detail: "Allow inbound 80/tcp and 443/tcp through the selected firewall backend",
		Risky:  true,
	}}
}

func strictProfileActions(intent Intent) []Action {
	return []Action{{
		Plugin: intent.Plugin.ID,
		Title:  "Apply strict profile",
		Detail: "Use stricter fail2ban defaults and document optional root account lockout steps",
		Risky:  true,
	}}
}

func providerAdvisoryActions(intent Intent) []Action {
	return []Action{{
		Plugin: intent.Plugin.ID,
		Title:  "Record provider advisory",
		Detail: "Add provider-specific recovery and firewall-console reminders to the run report without mutating provider settings",
	}}
}

func fallbackActions(intent Intent) []Action {
	if intent.Plugin.Kind == "custom" {
		return []Action{{
			Plugin: intent.Plugin.ID,
			Title:  "Run custom plugin",
			Detail: fmt.Sprintf("Probe with %q, plan with %q, apply with %q", intent.Plugin.Probe, intent.Plugin.Plan, intent.Plugin.Apply),
			Risky:  true,
		}}
	}
	return []Action{{
		Plugin: intent.Plugin.ID,
		Title:  "Run plugin",
		Detail: fmt.Sprintf("Execute plugin under the %s profile", intent.Profile),
	}}
}

func sshHardeningOperations(intent Intent) []Operation {
	return []Operation{
		{Kind: RunCommand, Plugin: intent.Plugin.ID, Command: "sshd", Args: []string{"-t"}},
		{Kind: RunCommand, Plugin: intent.Plugin.ID, Command: "systemctl", Args: []string{"reload", intent.Host.SSHService}},
	}
}

func fail2banOperations(intent Intent) []Operation {
	return []Operation{
		installOperation(intent.Plugin.ID, intent.Host.PackageManager, "fail2ban"),
		{Kind: RunCommand, Plugin: intent.Plugin.ID, Command: "systemctl", Args: []string{"enable", "--now", "fail2ban"}},
	}
}

func securityUpdateIntentOperations(intent Intent) []Operation {
	return securityUpdateOperations(intent.Plugin, intent.Host.PackageManager)
}

func sysctlOperations(intent Intent) []Operation {
	return []Operation{{Kind: RunCommand, Plugin: intent.Plugin.ID, Command: "sysctl", Args: []string{"--system"}}}
}

func tailscaleSSHOperations(intent Intent) []Operation {
	ops := []Operation{
		installOperation(intent.Plugin.ID, intent.Host.PackageManager, "tailscale"),
		{Kind: RollbackNote, Plugin: intent.Plugin.ID, Note: "disable Tailscale SSH with tailscale up --ssh=false if it was enabled manually"},
	}
	if intent.Host.InitSystem == "systemd" {
		ops = append(ops, Operation{Kind: RunCommand, Plugin: intent.Plugin.ID, Command: "systemctl", Args: []string{"enable", "--now", "tailscaled"}})
	}
	return ops
}
