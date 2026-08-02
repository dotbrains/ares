package apply

import (
	"fmt"
	"strings"

	"github.com/dotbrains/ares/internal/customcommand"
	"github.com/dotbrains/ares/internal/plugins"
)

type PluginExecutor struct {
	Context *Context
}

func (executor PluginExecutor) Execute(plugin plugins.Plugin) error {
	ctx := executor.Context
	if !executor.Probe(plugin) {
		return nil
	}
	if err := executor.Apply(plugin); err != nil {
		ctx.Result.Failed = append(ctx.Result.Failed, fmt.Sprintf("%s: %v", plugin.ID, err))
		return err
	}
	return executor.Verify(plugin)
}

func (executor PluginExecutor) Probe(plugin plugins.Plugin) bool {
	ctx := executor.Context
	if plugin.Probe == "" {
		ctx.Result.Probed = append(ctx.Result.Probed, plugin.ID+": no probe declared")
		return true
	}
	if ctx.Options.Root != "" {
		ctx.Result.Probed = append(ctx.Result.Probed, plugin.ID+": would probe with "+plugin.Probe)
		return true
	}
	if plugin.Kind == "custom" {
		output, err := runCustomCommand(plugin, plugin.Probe)
		if err != nil {
			ctx.Result.Skipped = append(ctx.Result.Skipped, plugin.ID+": probe did not pass before apply: "+probeFailureMessage(output, err))
			return false
		}
		ctx.Result.Probed = append(ctx.Result.Probed, plugin.ID+": probe passed")
		return true
	}
	probe := customcommand.Command{PluginID: plugin.ID, Phase: "probe", Line: plugin.Probe}.Run()
	if probe.Err != nil {
		ctx.Result.Skipped = append(ctx.Result.Skipped, plugin.ID+": probe did not pass before apply: "+strings.TrimSpace(probe.Output))
		return plugin.Kind != "custom"
	}
	ctx.Result.Probed = append(ctx.Result.Probed, plugin.ID+": probe passed")
	return true
}

func (executor PluginExecutor) Apply(plugin plugins.Plugin) error {
	ctx := executor.Context
	plugin = pluginWithCatalogMetadata(plugin)
	return behaviorFor(plugin).Apply(ctx, plugin)
}

func (executor PluginExecutor) Verify(plugin plugins.Plugin) error {
	ctx := executor.Context
	plugin = pluginWithCatalogMetadata(plugin)
	failuresBeforeVerify := len(ctx.Result.Failed)
	behaviorFor(plugin).Verify(ctx, plugin)
	if len(ctx.Result.Failed) > failuresBeforeVerify {
		return fmt.Errorf("%s verification failed", plugin.ID)
	}
	return nil
}
