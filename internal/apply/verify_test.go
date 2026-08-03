package apply

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/dotbrains/ares/internal/plugins"
)

type fakeRunner struct {
	outputs map[string]string
	err     error
}

func (runner fakeRunner) Run(_ context.Context, name string, args ...string) (string, error) {
	if runner.err != nil {
		return "", runner.err
	}
	return runner.outputs[name+" "+strings.Join(args, " ")], nil
}

func (runner fakeRunner) RunWithStdin(ctx context.Context, stdin string, name string, args ...string) (string, error) {
	return runner.Run(ctx, name, args...)
}

type deadlineRunner struct {
	hasDeadline bool
}

func (runner *deadlineRunner) Run(ctx context.Context, _ string, _ ...string) (string, error) {
	_, runner.hasDeadline = ctx.Deadline()
	return "Status: active\n22/tcp ALLOW Anywhere\n", nil
}

func (runner *deadlineRunner) RunWithStdin(ctx context.Context, _ string, name string, args ...string) (string, error) {
	return runner.Run(ctx, name, args...)
}

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

func TestFirewallVerificationUsesCommandOutput(t *testing.T) {
	ctx := &Context{
		Options: Options{Runner: fakeRunner{outputs: map[string]string{
			"ufw status": "Status: active\n22/tcp ALLOW Anywhere\n",
		}}},
		Plan: testPlan(),
	}

	ctx.verifyUFW("firewall-ufw")

	if !contains(ctx.Result.Verified, "firewall-ufw: verified ufw status") {
		t.Fatalf("missing verified command: %+v", ctx.Result)
	}
}

func TestFirewallVerificationRunsWithDeadline(t *testing.T) {
	runner := &deadlineRunner{}
	ctx := &Context{
		Options: Options{Runner: runner, CommandTimeout: time.Minute},
		Plan:    testPlan(),
	}

	ctx.verifyUFW("firewall-ufw")

	if !runner.hasDeadline {
		t.Fatal("expected verification runner context to have deadline")
	}
}

func TestFirewallVerificationFailsWhenOutputIsMissingRule(t *testing.T) {
	ctx := &Context{
		Options: Options{Runner: fakeRunner{outputs: map[string]string{
			"firewall-cmd --list-all": "public\n  ports:\n",
		}}},
		Plan: testPlan(),
	}

	ctx.verifyFirewalld("firewall-firewalld")

	if !contains(ctx.Result.Failed, "firewall-firewalld: verification output missing 22/tcp") {
		t.Fatalf("missing verification failure: %+v", ctx.Result)
	}
}

func TestFirewallVerificationReportsRunnerError(t *testing.T) {
	ctx := &Context{
		Options: Options{Runner: fakeRunner{err: fmt.Errorf("command failed")}},
		Plan:    testPlan(),
	}

	ctx.verifyUFW("firewall-ufw")

	if !contains(ctx.Result.Failed, "firewall-ufw: command failed") {
		t.Fatalf("missing runner failure: %+v", ctx.Result)
	}
}
