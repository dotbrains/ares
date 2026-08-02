package customcommand

import (
	"fmt"
	"strings"
)

type PluginPolicy struct {
	Name           string
	Probe          string
	Plan           string
	Apply          string
	Verify         string
	Rollback       string
	TimeoutSeconds int
}

func ValidatePolicy(plugin PluginPolicy, reserved func(string) bool) error {
	name := strings.TrimSpace(plugin.Name)
	if name == "" {
		return fmt.Errorf("custom plugin name is required")
	}
	if reserved != nil && reserved(name) {
		return fmt.Errorf("custom plugin name %q conflicts with a reserved plugin selector", name)
	}
	if plugin.TimeoutSeconds < 0 {
		return fmt.Errorf("custom plugin %q timeout_seconds must be non-negative", name)
	}
	for field, command := range map[string]string{
		"probe":    plugin.Probe,
		"plan":     plugin.Plan,
		"apply":    plugin.Apply,
		"verify":   plugin.Verify,
		"rollback": plugin.Rollback,
	} {
		if err := ValidateLine(name, field, command); err != nil {
			return err
		}
	}
	return ValidateLifecycle(name, plugin.Apply, plugin.Verify, plugin.Rollback)
}

func ExecutableDecisions(pluginName string, commands map[string]string, pass func(string, string), fail func(string, string)) {
	for _, phase := range []string{"probe", "apply", "verify", "rollback"} {
		command := commands[phase]
		if strings.TrimSpace(command) == "" {
			continue
		}
		path, err := CheckExecutable(command)
		if err != nil {
			fail(phase, err.Error())
			continue
		}
		pass(phase, path)
	}
}
