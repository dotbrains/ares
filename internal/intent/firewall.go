package intent

import "github.com/dotbrains/ares/internal/plugins"

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
