package customcommand

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/dotbrains/ares/internal/plugins"
)

func TestValidateLineRejectsBlankAndMultiline(t *testing.T) {
	if err := ValidateLine("demo", "apply", "   "); err == nil {
		t.Fatal("expected blank command error")
	}
	if err := ValidateLine("demo", "apply", "printf ok\nrm -rf /"); err == nil {
		t.Fatal("expected multiline command error")
	}
}

func TestValidatePolicyRejectsUnsafeLifecycle(t *testing.T) {
	err := ValidatePolicy(PluginPolicy{Name: "tailscale", Verify: "tailscale status"}, nil)
	if err == nil {
		t.Fatal("expected policy error")
	}
}

func TestCheckExecutableSupportsAbsoluteAndPathCommands(t *testing.T) {
	root := t.TempDir()
	script := filepath.Join(root, "plugin.sh")
	if err := os.WriteFile(script, []byte("#!/usr/bin/env sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	if got, err := CheckExecutable(script + " apply"); err != nil || got != script {
		t.Fatalf("CheckExecutable absolute = %q, %v", got, err)
	}
	if got, err := CheckExecutable("sh -c true"); err != nil || got == "" {
		t.Fatalf("CheckExecutable path = %q, %v", got, err)
	}
}

func TestRunParsesStructuredOutput(t *testing.T) {
	command := New(plugins.Plugin{ID: "custom-hardening", Kind: "custom", TimeoutSeconds: 5}, "apply", "printf 'applied: custom apply\\n'")
	result := command.Run()
	if result.Err != nil {
		t.Fatal(result.Err)
	}
	parsed := ParseOutput(command.PluginID, result.Output)
	if len(parsed.Applied) != 1 || parsed.Applied[0] != "custom-hardening: custom apply" {
		t.Fatalf("parsed output = %+v", parsed)
	}
}
