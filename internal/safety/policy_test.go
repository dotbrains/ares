package safety

import (
	"testing"

	"github.com/dotbrains/ares/internal/config"
	"github.com/dotbrains/ares/internal/plan"
	"github.com/dotbrains/ares/internal/reports"
	"github.com/dotbrains/ares/internal/scenario"
	"github.com/dotbrains/ares/internal/system"
)

type fakeReportDirectoryChecker struct {
	decision Decision
}

func (checker fakeReportDirectoryChecker) Check(string) Decision {
	return checker.decision
}

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

func TestEvaluateApplyUsesHostSSHAndConfigEvidence(t *testing.T) {
	host := scenario.UbuntuHost()
	host.RunningOverSSH = true

	readiness := EvaluateApply(ApplyReadinessInput{
		Host:                       host,
		DryRun:                     true,
		Yes:                        true,
		AllowPasswordLockout:       true,
		AllowPasswordLockoutSource: "cli",
	})

	if readiness.Refusal != nil {
		t.Fatalf("unexpected refusal: %v", readiness.Refusal)
	}
	if readiness.SSHLockoutPolicy != "password lockout explicitly allowed by config or CLI" {
		t.Fatalf("SSHLockoutPolicy = %q", readiness.SSHLockoutPolicy)
	}
	if !hasEvidence(readiness.SafetyEvidence, "ssh.allow_password_lockout", "true", "cli") {
		t.Fatalf("missing allow-password evidence: %#v", readiness.SafetyEvidence)
	}
}

func TestEvaluateIncludesStructuredHostEvidence(t *testing.T) {
	host := scenario.UbuntuHost()
	decisions := Evaluate(Facts{
		Host: host,
		Plan: plan.Build(host, config.DefaultConfig()),
		Root: t.TempDir(),
	})

	for _, decision := range decisions {
		if decision.Name != "package manager" {
			continue
		}
		if len(decision.Evidence) != 1 {
			t.Fatalf("package manager evidence = %#v", decision.Evidence)
		}
		evidence := decision.Evidence[0]
		if evidence.Name != "package_manager" || evidence.Value != "apt-get" || evidence.Source != "fixture" || evidence.Confidence != "high" {
			t.Fatalf("unexpected package manager evidence: %#v", evidence)
		}
		return
	}
	t.Fatalf("missing package manager decision: %+v", decisions)
}

func TestEvaluateUsesReportDirectoryAdapter(t *testing.T) {
	host := scenario.UbuntuHost()
	decisions := Evaluate(Facts{
		Host: host,
		Plan: plan.Build(host, config.DefaultConfig()),
		ReportDirectories: fakeReportDirectoryChecker{decision: Decision{
			Name:   "reports",
			Status: "warn",
			Detail: "adapter used",
		}},
	})

	if !hasDecision(decisions, "reports", "warn") {
		t.Fatalf("missing adapter report decision: %+v", decisions)
	}
}

func TestEvaluateWarnsForTailscaleOnUnsupportedInit(t *testing.T) {
	host := scenario.DistroHost("alpine", "apk", "nftables")
	host.InitSystem = "openrc"
	cfg := config.DefaultConfig()
	cfg.Plugins.Enabled = append(cfg.Plugins.Enabled, "tailscale-ssh")

	decisions := Evaluate(Facts{
		Host: host,
		Plan: plan.Build(host, cfg),
		Root: t.TempDir(),
	})

	if !hasDecision(decisions, "tailscale service", "warn") {
		t.Fatalf("missing tailscale service warning: %+v", decisions)
	}
}

func TestEvaluateFailsWhenTailscaleAuthKeyEnvIsMissing(t *testing.T) {
	host := scenario.UbuntuHost()
	cfg := config.DefaultConfig()
	cfg.Plugins.Enabled = append(cfg.Plugins.Enabled, "tailscale-ssh")
	cfg.Tailscale.SSHEnabled = true
	cfg.Tailscale.AuthKeyEnv = "TAILSCALE_AUTHKEY"
	t.Setenv("TAILSCALE_AUTHKEY", "")

	decisions := Evaluate(Facts{
		Host:   host,
		Plan:   plan.Build(host, cfg),
		Config: cfg,
		Root:   t.TempDir(),
	})

	if !hasDecision(decisions, "tailscale auth key", "fail") {
		t.Fatalf("missing tailscale auth key failure: %+v", decisions)
	}
	if decision, ok := findDecision(decisions, "tailscale auth key"); !ok || !hasEvidence(decision.Evidence, "tailscale.auth_key_env", "TAILSCALE_AUTHKEY", "config") {
		t.Fatalf("missing tailscale auth key evidence: %+v", decisions)
	}
}

func TestEvaluatePassesWhenTailscaleAuthKeyEnvIsPresent(t *testing.T) {
	host := scenario.UbuntuHost()
	cfg := config.DefaultConfig()
	cfg.Plugins.Enabled = append(cfg.Plugins.Enabled, "tailscale-ssh")
	cfg.Tailscale.SSHEnabled = true
	cfg.Tailscale.AuthKeyEnv = "TAILSCALE_AUTHKEY"
	t.Setenv("TAILSCALE_AUTHKEY", "tskey-secret")

	decisions := Evaluate(Facts{
		Host:   host,
		Plan:   plan.Build(host, cfg),
		Config: cfg,
		Root:   t.TempDir(),
	})

	if !hasDecision(decisions, "tailscale auth key", "pass") {
		t.Fatalf("missing tailscale auth key pass: %+v", decisions)
	}
	for _, decision := range decisions {
		for _, evidence := range decision.Evidence {
			if evidence.Value == "tskey-secret" {
				t.Fatalf("tailscale auth key leaked in evidence: %+v", decisions)
			}
		}
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

func findDecision(decisions []Decision, name string) (Decision, bool) {
	for _, decision := range decisions {
		if decision.Name == name {
			return decision, true
		}
	}
	return Decision{}, false
}

func hasEvidence(evidence []reports.Evidence, name string, value string, source string) bool {
	for _, item := range evidence {
		if item.Name == name && item.Value == value && item.Source == source {
			return true
		}
	}
	return false
}
