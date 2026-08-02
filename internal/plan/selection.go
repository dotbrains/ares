package plan

import (
	"slices"
	"strconv"

	"github.com/dotbrains/ares/internal/config"
	"github.com/dotbrains/ares/internal/plugins"
	"github.com/dotbrains/ares/internal/system"
)

type Selection struct {
	Host   system.Host
	Config *config.Config
}

func (selection Selection) Plugins() []plugins.Plugin {
	ids := selection.pluginIDs()
	var selected []plugins.Plugin
	for _, id := range ids {
		if plugin, ok := plugins.Find(id); ok {
			selected = append(selected, plugin)
		}
	}
	for _, custom := range selection.Config.Plugins.Custom {
		selected = append(selected, customPlugin(custom))
	}
	return selected
}

func (selection Selection) Warnings() []string {
	var warnings []string
	if _, ok := plugins.DistroAdapter(hostMatcher(selection.Host)); !ok {
		warnings = append(warnings, unsupportedDistroWarning(selection.Host))
	}
	if !selection.Host.RunningOverSSH {
		warnings = append(warnings, "active SSH session was not detected; SSH lockout checks will be less certain")
	}
	return warnings
}

func (selection Selection) pluginIDs() []string {
	ids := resolvePluginIDs(selection.Host, append([]string{}, selection.Config.Plugins.Enabled...))
	if distroPlugin, ok := plugins.DistroAdapter(hostMatcher(selection.Host)); ok && !slices.Contains(ids, distroPlugin.ID) {
		ids = append([]string{distroPlugin.ID}, ids...)
	}
	if providerPlugin, ok := plugins.ProviderAdvisory(hostMatcher(selection.Host)); ok {
		ids = append(ids, providerPlugin.ID)
	}
	switch selection.Config.Profile {
	case "web":
		ids = append(ids, "web-profile")
	case "strict":
		ids = append(ids, "strict-profile")
	}
	return unique(ids)
}

func customPlugin(custom config.CustomPlugin) plugins.Plugin {
	return plugins.Plugin{
		ID:             custom.Name,
		Name:           custom.Name,
		Kind:           "custom",
		Summary:        "Custom local plugin",
		Probe:          custom.Probe,
		Plan:           custom.Plan,
		Apply:          custom.Apply,
		Verify:         custom.Verify,
		Rollback:       custom.Rollback,
		TimeoutSeconds: custom.TimeoutSeconds,
	}
}

func unsupportedDistroWarning(host system.Host) string {
	return "distro " + strconv.Quote(host.OSID) + " is not a first-class target yet"
}
