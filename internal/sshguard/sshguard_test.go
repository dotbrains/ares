package sshguard

import "testing"

func TestEvaluateRequiresKeyOnActiveSSH(t *testing.T) {
	decision := Evaluate(Facts{RunningOverSSH: true})
	if decision.Allowed {
		t.Fatalf("decision allowed lockout: %+v", decision)
	}
}

func TestEvaluateAllowsExplicitBypass(t *testing.T) {
	decision := Evaluate(Facts{RunningOverSSH: true, AllowPasswordLockout: true})
	if !decision.Allowed || !decision.Bypassed {
		t.Fatalf("decision = %+v", decision)
	}
	if decision.Detail != "password lockout explicitly allowed by config or CLI" {
		t.Fatalf("detail = %q", decision.Detail)
	}
}
