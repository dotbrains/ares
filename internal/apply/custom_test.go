package apply

import (
	"testing"

	"github.com/dotbrains/ares/internal/plugins"
)

func TestCustomPluginProbeTimeoutSkipsPlugin(t *testing.T) {
	ctx := &Context{}
	plugin := plugins.Plugin{
		ID:             "custom-hardening",
		Kind:           "custom",
		Probe:          "sleep 2",
		TimeoutSeconds: 1,
	}

	if ctx.probePlugin(plugin) {
		t.Fatalf("custom plugin probe passed unexpectedly: %+v", ctx.Result)
	}
	if !contains(ctx.Result.Skipped, "custom-hardening: probe did not pass before apply: command timed out after 1s") {
		t.Fatalf("missing probe timeout skip item: %+v", ctx.Result)
	}
}
