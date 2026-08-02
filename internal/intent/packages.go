package intent

import (
	"fmt"
	"strings"

	"github.com/dotbrains/ares/internal/plugins"
)

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
