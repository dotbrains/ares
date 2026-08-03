package intent_test

import (
	"slices"
	"testing"

	"github.com/dotbrains/ares/internal/config"
	"github.com/dotbrains/ares/internal/intent"
	"github.com/dotbrains/ares/internal/plan"
	"github.com/dotbrains/ares/internal/scenario"
)

func TestIntentProjectsActionsAndOperations(t *testing.T) {
	fixture := scenario.UbuntuBasic()
	hardeningPlan := plan.Build(fixture.Host, config.DefaultConfig())
	var sshPluginFound bool
	for _, plugin := range hardeningPlan.Plugins {
		if plugin.ID != "ssh-hardening" {
			continue
		}
		sshPluginFound = true
		pluginIntent := intent.ForPlugin(hardeningPlan.Host, hardeningPlan.Profile, plugin)
		actions := pluginIntent.Actions()
		if len(actions) != 2 || actions[1].Title != "Preserve active SSH access" {
			t.Fatalf("unexpected actions: %+v", actions)
		}
		var commands []string
		for _, op := range pluginIntent.Operations() {
			if op.Kind == intent.RunCommand {
				commands = append(commands, op.Command+" "+joinArgs(op.Args))
			}
		}
		if !slices.Contains(commands, "sshd -t") {
			t.Fatalf("missing sshd verification command: %v", commands)
		}
	}
	if !sshPluginFound {
		t.Fatalf("missing ssh-hardening plugin: %+v", hardeningPlan.Plugins)
	}
}

func TestTailscaleIntentKeepsTailnetAuthExplicit(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Plugins.Enabled = append(cfg.Plugins.Enabled, "tailscale-ssh")
	hardeningPlan := plan.Build(scenario.UbuntuHost(), cfg)
	var tailscaleFound bool
	for _, plugin := range hardeningPlan.Plugins {
		if plugin.ID != "tailscale-ssh" {
			continue
		}
		tailscaleFound = true
		pluginIntent := intent.ForPlugin(hardeningPlan.Host, hardeningPlan.Profile, plugin)
		actions := pluginIntent.Actions()
		if len(actions) != 2 || actions[1].Title != "Keep tailnet SSH explicit" {
			t.Fatalf("unexpected tailscale actions: %+v", actions)
		}
		var commands []string
		for _, op := range pluginIntent.Operations() {
			if op.Kind == intent.RunCommand {
				commands = append(commands, op.Command+" "+joinArgs(op.Args))
			}
		}
		if slices.Contains(commands, "tailscale up --ssh") {
			t.Fatalf("tailscale auth should stay manual: %v", commands)
		}
		if !slices.Contains(commands, "systemctl enable --now tailscaled") {
			t.Fatalf("missing tailscaled enable command: %v", commands)
		}
	}
	if !tailscaleFound {
		t.Fatalf("missing tailscale plugin: %+v", hardeningPlan.Plugins)
	}
}

func joinArgs(args []string) string {
	if len(args) == 0 {
		return ""
	}
	result := args[0]
	for _, arg := range args[1:] {
		result += " " + arg
	}
	return result
}
