package reports

import "testing"

func TestNewPreflightReportUsesStableArrays(t *testing.T) {
	report := NewPreflightReport(PreflightInput{Profile: "basic"})
	if report.SchemaVersion != PreflightSchemaVersion {
		t.Fatalf("schema = %q", report.SchemaVersion)
	}
	if report.Plugins == nil {
		t.Fatal("plugins should be a stable empty array")
	}
}
