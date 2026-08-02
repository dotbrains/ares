package reports

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestArtifactsWriteRollbackReportUsesStableArrays(t *testing.T) {
	path := filepath.Join(t.TempDir(), "rollback.json")
	artifacts := Artifacts{RollbackReportPath: path}
	if err := artifacts.WriteRollbackReport(RollbackReport{}); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var report map[string]any
	if err := json.Unmarshal(data, &report); err != nil {
		t.Fatal(err)
	}
	if report["schema_version"] != RollbackSchemaVersion {
		t.Fatalf("schema_version = %v", report["schema_version"])
	}
	for _, name := range []string{"applied", "skipped", "failed"} {
		if _, ok := report[name].([]any); !ok {
			t.Fatalf("%s = %#v, want array", name, report[name])
		}
	}
}

func TestArtifactsFinishRunWritesAllArtifactsInOrder(t *testing.T) {
	dir := t.TempDir()
	artifacts := Artifacts{
		RunReportPath: filepath.Join(dir, "latest.json"),
		RunLogPath:    filepath.Join(dir, "ares.log"),
		UndoPlanPath:  filepath.Join(dir, "undo.txt"),
	}
	if err := artifacts.FinishRun(RunArtifactInput{
		Report: RunReport{Profile: "basic"},
		Log:    []string{"log"},
		Undo:   []string{"undo"},
	}); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{artifacts.RunReportPath, artifacts.RunLogPath, artifacts.UndoPlanPath} {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("missing artifact %s: %v", path, err)
		}
	}
}
