package apply

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/dotbrains/ares/internal/plugins"
)

func TestCustomPluginCommandParsesStructuredOutput(t *testing.T) {
	root := t.TempDir()
	pluginScript := filepath.Join(root, "plugin.sh")
	if err := os.WriteFile(pluginScript, []byte("#!/usr/bin/env sh\nprintf 'applied: custom apply\\nverified: custom side effect\\n'\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	plugin := plugins.Plugin{
		ID:             "custom-hardening",
		Kind:           "custom",
		TimeoutSeconds: 5,
	}
	output, err := runCustomCommand(plugin, pluginScript)
	if err != nil {
		t.Fatal(err)
	}
	ctx := &Context{}
	ctx.appendCustomOutput(plugin.ID, output)
	if !contains(ctx.Result.Applied, "custom-hardening: custom apply") {
		t.Fatalf("missing custom apply output: %+v", ctx.Result)
	}
	if !contains(ctx.Result.Verified, "custom-hardening: custom side effect") {
		t.Fatalf("missing custom verify output: %+v", ctx.Result)
	}
}

func TestCustomPluginProbeFailureSkipsPlugin(t *testing.T) {
	ctx := &Context{}
	plugin := plugins.Plugin{
		ID:    "custom-hardening",
		Kind:  "custom",
		Probe: "printf 'missing dependency' >&2; exit 1",
	}

	if ctx.probePlugin(plugin) {
		t.Fatalf("custom plugin probe passed unexpectedly: %+v", ctx.Result)
	}
	if !contains(ctx.Result.Skipped, "custom-hardening: probe did not pass before apply: missing dependency") {
		t.Fatalf("missing probe failure skip item: %+v", ctx.Result)
	}
}

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

func TestBuiltinPluginProbeFailureDoesNotSkipPlugin(t *testing.T) {
	ctx := &Context{}
	plugin := plugins.Plugin{
		ID:    "fail2ban",
		Kind:  "builtin",
		Probe: "printf 'not installed yet' >&2; exit 1",
	}

	if !ctx.probePlugin(plugin) {
		t.Fatalf("builtin plugin probe gated apply unexpectedly: %+v", ctx.Result)
	}
	if !contains(ctx.Result.Skipped, "fail2ban: probe did not pass before apply: not installed yet") {
		t.Fatalf("missing probe failure skip item: %+v", ctx.Result)
	}
}

func TestVerifyPluginOrErrorReturnsErrorWhenCustomVerificationFails(t *testing.T) {
	root := t.TempDir()
	verifyScript := filepath.Join(root, "verify.sh")
	if err := os.WriteFile(verifyScript, []byte("#!/usr/bin/env sh\nprintf 'failed: custom check failed\\n'\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	ctx := &Context{Plan: testPlan()}

	err := ctx.verifyPluginOrError(plugins.Plugin{
		ID:     "custom-hardening",
		Kind:   "custom",
		Verify: verifyScript,
	})
	if err == nil {
		t.Fatal("expected verification error")
	}
	if !contains(ctx.Result.Failed, "custom-hardening: custom check failed") {
		t.Fatalf("missing custom verification failure: %+v", ctx.Result)
	}
}
