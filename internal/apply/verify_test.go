package apply

import (
	"strings"
	"testing"

	"github.com/dotbrains/ares/internal/plugins"
)

func TestVerifyPluginOrErrorReturnsErrorWhenBuiltinVerificationFails(t *testing.T) {
	root := t.TempDir()
	ctx := &Context{
		Options: Options{Root: root},
		Plan:    testPlan(),
	}

	err := ctx.verifyPluginOrError(plugins.Plugin{
		ID:   "ssh-hardening",
		Kind: "builtin",
	})
	if err == nil {
		t.Fatal("expected verification error")
	}
	if !contains(ctx.Result.Failed, "ssh-hardening: expected /etc/ssh/sshd_config.d/99-ares.conf was not present") {
		t.Fatalf("missing verification failure: %+v", ctx.Result)
	}
}

func TestFirewallVerificationRecordsBackendCommandsInTestRoot(t *testing.T) {
	ctx := &Context{
		Options: Options{Root: t.TempDir()},
		Plan:    testPlan(),
	}

	ctx.verifyUFW("firewall-ufw")

	if !contains(ctx.Result.Verified, "firewall-ufw: would verify with ufw status") {
		t.Fatalf("missing ufw verify command: %+v", ctx.Result)
	}
}

func TestWebProfileNftablesVerificationRecordsExpectedCommands(t *testing.T) {
	ctx := &Context{
		Options: Options{Root: t.TempDir()},
		Plan:    testPlan(),
	}
	ctx.Plan.Host.FirewallBackend = "nftables"

	ctx.verifyWebProfile("web-profile")

	joined := strings.Join(ctx.Result.Verified, "\n")
	if !strings.Contains(joined, "web-profile: would verify with nft list ruleset") {
		t.Fatalf("missing nftables web verify command: %+v", ctx.Result)
	}
}
