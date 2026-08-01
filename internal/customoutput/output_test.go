package customoutput

import "testing"

func TestParseLineProtocol(t *testing.T) {
	result := Parse("custom-hardening", "applied: wrote file\nverified: ok\nskipped: optional\nfailed: bad\nplain output\n")
	if len(result.Applied) != 2 || result.Applied[0] != "custom-hardening: wrote file" || result.Applied[1] != "custom-hardening: plain output" {
		t.Fatalf("applied = %+v", result.Applied)
	}
	if result.Verified[0] != "custom-hardening: ok" {
		t.Fatalf("verified = %+v", result.Verified)
	}
	if result.Skipped[0] != "custom-hardening: optional" {
		t.Fatalf("skipped = %+v", result.Skipped)
	}
	if result.Failed[0] != "custom-hardening: bad" {
		t.Fatalf("failed = %+v", result.Failed)
	}
}
