package intent

import "github.com/dotbrains/ares/internal/plugins"

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
