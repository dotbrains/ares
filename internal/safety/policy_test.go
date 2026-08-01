package safety

import (
	"testing"

	"github.com/dotbrains/ares/internal/config"
	"github.com/dotbrains/ares/internal/plan"
	"github.com/dotbrains/ares/internal/system"
)

func TestEvaluateReportsCustomExecutableFailures(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Plugins.Custom = []config.CustomPlugin{{
		Name:  "missing-tool",
		Apply: "missing-ares-tool apply",
	}}
	host := system.Host{
		OSID:            "ubuntu",
		PackageManager:  "apt-get",
		FirewallBackend: "ufw",
		SSHService:      "ssh",
		SSHPort:         "22",
		RunningOverSSH:  true,
	}

	decisions := Evaluate(Facts{
		Host:   host,
		Plan:   plan.Build(host, cfg),
		Config: cfg,
		Root:   t.TempDir(),
	})

	if !HasFailures(decisions) {
		t.Fatalf("expected custom command failure: %+v", decisions)
	}
	if !hasDecision(decisions, "custom missing-tool apply", "fail") {
		t.Fatalf("missing custom command decision: %+v", decisions)
	}
}

func TestSSHLockoutPolicyRequiresExplicitBypass(t *testing.T) {
	t.Setenv("SSH_CONNECTION", "127.0.0.1 1 127.0.0.1 2")
	t.Setenv("USER", "ares-no-key")
	t.Setenv("LOGNAME", "ares-no-key")
	t.Setenv("SUDO_USER", "")

	got := SSHLockoutPolicy("", false)
	if got == "" || got == "password lockout explicitly allowed by config or CLI" {
		t.Fatalf("SSHLockoutPolicy() = %q, want refusal", got)
	}

	if got := SSHLockoutPolicy("", true); got != "password lockout explicitly allowed by config or CLI" {
		t.Fatalf("SSHLockoutPolicy(allow) = %q", got)
	}
}

func hasDecision(decisions []Decision, name string, status string) bool {
	for _, decision := range decisions {
		if decision.Name == name && decision.Status == status {
			return true
		}
	}
	return false
}
