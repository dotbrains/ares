package apply

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestRunReportSchemaIncludesRecoveryFields(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "etc", "ssh"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "etc", "ssh", "sshd_config"), []byte("Port 22\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	result, err := Run(testPlan(), Options{
		Yes:  true,
		Root: root,
		Now:  time.Date(2026, 7, 25, 17, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(result.ReportPath)
	if err != nil {
		t.Fatal(err)
	}
	var report map[string]any
	if err := json.Unmarshal(data, &report); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"schema_version", "profile", "host", "plugins", "warnings", "transaction", "probed", "verified", "applied", "skipped", "failed"} {
		if _, ok := report[key]; !ok {
			t.Fatalf("report missing %q: %s", key, data)
		}
	}
	if report["schema_version"] != "ares.report.v1" {
		t.Fatalf("schema_version = %v", report["schema_version"])
	}
	if !jsonArrayHasObject(report["safety_evidence"], "name", "package_manager") {
		t.Fatalf("safety evidence missing package_manager: %#v", report["safety_evidence"])
	}

	transaction, ok := report["transaction"].(map[string]any)
	if !ok {
		t.Fatalf("transaction has unexpected shape: %#v", report["transaction"])
	}
	for _, key := range []string{"files", "commands", "backups", "rollback_steps"} {
		if _, ok := transaction[key]; !ok {
			t.Fatalf("transaction missing %q: %#v", key, transaction)
		}
	}
	if !jsonArrayContains(transaction["files"], "/etc/ssh/sshd_config.d/99-ares.conf") {
		t.Fatalf("transaction files missing SSH drop-in: %#v", transaction["files"])
	}
	if !jsonArrayContains(transaction["backups"], "/etc/ssh/sshd_config") {
		t.Fatalf("transaction backups missing sshd_config: %#v", transaction["backups"])
	}
}

func jsonArrayHasObject(value any, key string, want string) bool {
	values, ok := value.([]any)
	if !ok {
		return false
	}
	for _, item := range values {
		object, ok := item.(map[string]any)
		if ok && object[key] == want {
			return true
		}
	}
	return false
}

func jsonArrayContains(value any, want string) bool {
	values, ok := value.([]any)
	if !ok {
		return false
	}
	for _, item := range values {
		if item == want {
			return true
		}
	}
	return false
}
