package readiness

import "testing"

func TestRefusalPreservesApplyConfirmationMessage(t *testing.T) {
	err := Refusal(Request{Mode: Apply, Root: "/tmp/ares"})
	if err == nil {
		t.Fatal("expected refusal")
	}
	if err.Error() != "apply mode requires --yes after reviewing the plan" {
		t.Fatalf("refusal = %q", err)
	}
}

func TestRefusalPreservesRollbackRootMessage(t *testing.T) {
	err := Refusal(Request{Mode: Rollback, Yes: true, EffectiveUID: -1})
	if err == nil {
		t.Fatal("expected refusal")
	}
	if err.Error() != "rollback must run as root" {
		t.Fatalf("refusal = %q", err)
	}
}

func TestDryRunDoesNotRequireConfirmationOrRoot(t *testing.T) {
	if err := Refusal(Request{Mode: Apply, DryRun: true, EffectiveUID: -1}); err != nil {
		t.Fatalf("dry-run refused: %v", err)
	}
}
