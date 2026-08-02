package intent

import "github.com/dotbrains/ares/internal/plugins"

var firewallOperationBuilders = map[string]func(Intent) []Operation{
	"ufw":       ufwFirewallOperations,
	"firewalld": firewalldOperations,
	"nftables":  nftablesOperations,
}

var webOperationBuilders = map[string]func(Intent) []Operation{
	"firewalld": firewalldWebOperations,
	"nftables":  nftablesWebOperations,
	"ufw":       ufwWebOperations,
}

func (intent Intent) firewallOperations() []Operation {
	if build, ok := firewallOperationBuilders[plugins.Behavior(intent.Plugin).Variant]; ok {
		return build(intent)
	}
	return nil
}

func (intent Intent) webOperations() []Operation {
	if build, ok := webOperationBuilders[intent.Host.FirewallBackend]; ok {
		return build(intent)
	}
	return nil
}

func ufwFirewallOperations(intent Intent) []Operation {
	pluginID := intent.Plugin.ID
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
}

func firewalldOperations(intent Intent) []Operation {
	pluginID := intent.Plugin.ID
	ops := []Operation{
		installOperation(pluginID, intent.Host.PackageManager, "firewalld"),
		{Kind: RunCommand, Plugin: pluginID, Command: "systemctl", Args: []string{"enable", "--now", "firewalld"}},
		{Kind: RunCommand, Plugin: pluginID, Command: "firewall-cmd", Args: []string{"--permanent", "--add-port=" + intent.Host.SSHPort + "/tcp"}},
		{Kind: RunCommand, Plugin: pluginID, Command: "firewall-cmd", Args: []string{"--set-default-zone=public"}},
		{Kind: RunCommand, Plugin: pluginID, Command: "firewall-cmd", Args: []string{"--reload"}},
		{Kind: RunCommand, Plugin: pluginID, Command: "firewall-cmd", Args: []string{"--list-all"}},
	}
	return append(ops, Operation{Kind: RollbackNote, Plugin: pluginID, Note: "review firewalld ports/services and reload after manual rollback"})
}

func nftablesOperations(intent Intent) []Operation {
	pluginID := intent.Plugin.ID
	return []Operation{
		installOperation(pluginID, intent.Host.PackageManager, "nftables"),
		{Kind: RunCommand, Plugin: pluginID, Command: "nft", Args: []string{"-c", "-f", "/etc/nftables.conf"}},
		{Kind: RunCommand, Plugin: pluginID, Command: "systemctl", Args: []string{"enable", "--now", "nftables"}},
		{Kind: RunCommand, Plugin: pluginID, Command: "nft", Args: []string{"list", "ruleset"}},
	}
}

func firewalldWebOperations(intent Intent) []Operation {
	pluginID := intent.Plugin.ID
	return []Operation{
		{Kind: RunCommand, Plugin: pluginID, Command: "firewall-cmd", Args: []string{"--permanent", "--add-service=http"}},
		{Kind: RunCommand, Plugin: pluginID, Command: "firewall-cmd", Args: []string{"--permanent", "--add-service=https"}},
		{Kind: RunCommand, Plugin: pluginID, Command: "firewall-cmd", Args: []string{"--reload"}},
		{Kind: RunCommand, Plugin: pluginID, Command: "firewall-cmd", Args: []string{"--list-all"}},
	}
}

func nftablesWebOperations(intent Intent) []Operation {
	pluginID := intent.Plugin.ID
	return []Operation{
		{Kind: WriteFile, Plugin: pluginID, Path: "/etc/nftables.conf"},
		{Kind: BackupFile, Plugin: pluginID, Path: "/etc/nftables.conf"},
		{Kind: RunCommand, Plugin: pluginID, Command: "nft", Args: []string{"-c", "-f", "/etc/nftables.conf"}},
		{Kind: RunCommand, Plugin: pluginID, Command: "nft", Args: []string{"-f", "/etc/nftables.conf"}},
		{Kind: RunCommand, Plugin: pluginID, Command: "nft", Args: []string{"list", "ruleset"}},
	}
}

func ufwWebOperations(intent Intent) []Operation {
	pluginID := intent.Plugin.ID
	return []Operation{
		{Kind: RunCommand, Plugin: pluginID, Command: "ufw", Args: []string{"allow", "80/tcp"}},
		{Kind: RunCommand, Plugin: pluginID, Command: "ufw", Args: []string{"allow", "443/tcp"}},
		{Kind: RunCommand, Plugin: pluginID, Command: "ufw", Args: []string{"status"}},
	}
}
