package intent

import "github.com/dotbrains/ares/internal/plugins"

func customOperations(plugin plugins.Plugin) []Operation {
	var ops []Operation
	if plugin.Probe != "" {
		ops = append(ops, Operation{Kind: CustomCommand, Plugin: plugin.ID, Phase: "probe", Command: plugin.Probe})
	}
	if plugin.Apply != "" {
		ops = append(ops, Operation{Kind: CustomCommand, Plugin: plugin.ID, Phase: "apply", Command: plugin.Apply})
	}
	if plugin.Verify != "" {
		ops = append(ops, Operation{Kind: CustomCommand, Plugin: plugin.ID, Phase: "verify", Command: plugin.Verify})
	}
	if plugin.Rollback != "" {
		ops = append(ops, Operation{Kind: RollbackNote, Plugin: plugin.ID, Note: "custom " + plugin.ID + " rollback: " + plugin.Rollback})
	}
	return ops
}
