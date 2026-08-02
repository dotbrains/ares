package intent

import "github.com/dotbrains/ares/internal/plugins"

var securityUpdateOperationBuilders = map[string]func(plugins.Plugin, string) []Operation{
	"apt":           aptSecurityUpdateOperations,
	"dnf-automatic": dnfAutomaticOperations,
	"pacman":        pacmanUpgradeOperations,
	"zypper":        zypperPatchOperations,
	"apk":           apkUpgradeOperations,
}

func securityUpdateOperations(plugin plugins.Plugin, packageManager string) []Operation {
	if build, ok := securityUpdateOperationBuilders[plugins.Behavior(plugin).Variant]; ok {
		return build(plugin, packageManager)
	}
	return nil
}

func aptSecurityUpdateOperations(plugin plugins.Plugin, packageManager string) []Operation {
	return []Operation{
		{Kind: RunCommand, Plugin: plugin.ID, Command: packageManager, Args: []string{"update"}},
		installOperation(plugin.ID, packageManager, "unattended-upgrades"),
	}
}

func dnfAutomaticOperations(plugin plugins.Plugin, packageManager string) []Operation {
	return []Operation{
		installOperation(plugin.ID, packageManager, "dnf-automatic"),
		{Kind: RunCommand, Plugin: plugin.ID, Command: "systemctl", Args: []string{"enable", "--now", "dnf-automatic.timer"}},
	}
}

func pacmanUpgradeOperations(plugin plugins.Plugin, _ string) []Operation {
	return []Operation{{Kind: RunCommand, Plugin: plugin.ID, Command: "pacman", Args: []string{"-Syu", "--noconfirm"}}}
}

func zypperPatchOperations(plugin plugins.Plugin, _ string) []Operation {
	return []Operation{{Kind: RunCommand, Plugin: plugin.ID, Command: "zypper", Args: []string{"--non-interactive", "patch"}}}
}

func apkUpgradeOperations(plugin plugins.Plugin, _ string) []Operation {
	return []Operation{
		{Kind: RunCommand, Plugin: plugin.ID, Command: "apk", Args: []string{"update"}},
		{Kind: RunCommand, Plugin: plugin.ID, Command: "apk", Args: []string{"upgrade"}},
	}
}
