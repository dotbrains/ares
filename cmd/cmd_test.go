package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestExecute_Version(t *testing.T) {
	os.Args = []string{"ares", "--version"}
	err := Execute("0.0.1-test")
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
}

func TestNewRootCmd(t *testing.T) {
	root := newRootCmd("0.1.0")
	if root.Use != "ares" {
		t.Errorf("Use = %q", root.Use)
	}

	// Verify subcommands.
	cmds := make(map[string]bool)
	for _, c := range root.Commands() {
		cmds[c.Name()] = true
	}
	for _, want := range []string{"config"} {
		if !cmds[want] {
			t.Errorf("missing subcommand %q", want)
		}
	}
}

func TestNewRootCmd_Version(t *testing.T) {
	root := newRootCmd("1.2.3")
	if root.Version != "1.2.3" {
		t.Errorf("expected version 1.2.3, got %q", root.Version)
	}
}

func TestRootCmdRejectsUnknownProfileOverride(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)

	root := newRootCmd("test")
	root.SetArgs([]string{"--profile", "unknown", "--dry-run"})

	err := root.Execute()
	if err == nil {
		t.Fatal("expected unknown profile error")
	}
	if !strings.Contains(err.Error(), `unknown profile "unknown"`) {
		t.Fatalf("error = %v, want unknown profile", err)
	}
}

func TestPreflightUsesTestRootAndPrintsTransaction(t *testing.T) {
	rootDir := t.TempDir()
	sshDir := filepath.Join(rootDir, "etc", "ssh")
	if err := os.MkdirAll(sshDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sshDir, "sshd_config"), []byte("Port 2222\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("ARES_ROOT", rootDir)
	t.Setenv("ARES_OS_RELEASE", filepath.Join("..", "tests", "fixtures", "os-release", "ubuntu-24.04"))
	t.Setenv("HOME", t.TempDir())

	root := newRootCmd("test")
	buf := &bytes.Buffer{}
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"preflight"})

	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	output := buf.String()
	for _, want := range []string{
		"preflight:",
		"privileges: pass (ARES_ROOT test root active)",
		"ssh session: warn",
		"provider: warn",
		"provider advisory: warn",
		"reports: pass",
		"transaction:",
		"/etc/ssh/sshd_config.d/99-ares.conf",
		"ufw allow 2222/tcp",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("preflight output missing %q:\n%s", want, output)
		}
	}
}

func TestPreflightJSONIncludesChecksAndTransaction(t *testing.T) {
	rootDir := t.TempDir()
	sshDir := filepath.Join(rootDir, "etc", "ssh")
	if err := os.MkdirAll(sshDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sshDir, "sshd_config"), []byte("Port 2222\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("ARES_ROOT", rootDir)
	t.Setenv("ARES_OS_RELEASE", filepath.Join("..", "tests", "fixtures", "os-release", "ubuntu-24.04"))
	t.Setenv("HOME", t.TempDir())

	root := newRootCmd("test")
	buf := &bytes.Buffer{}
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"preflight", "--json"})

	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	var report struct {
		Profile     string           `json:"profile"`
		Plugins     []string         `json:"plugins"`
		Checks      []preflightCheck `json:"checks"`
		Transaction struct {
			Files []string `json:"files"`
		} `json:"transaction"`
	}
	if err := json.Unmarshal(buf.Bytes(), &report); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, buf.String())
	}
	if report.Profile != "basic" || len(report.Plugins) == 0 || len(report.Checks) == 0 {
		t.Fatalf("incomplete preflight JSON: %+v", report)
	}
	if len(report.Transaction.Files) == 0 {
		t.Fatalf("missing transaction files: %+v", report.Transaction)
	}
}

func TestPreflightFailsWhenCustomCommandExecutableIsMissing(t *testing.T) {
	rootDir := t.TempDir()
	sshDir := filepath.Join(rootDir, "etc", "ssh")
	if err := os.MkdirAll(sshDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sshDir, "sshd_config"), []byte("Port 2222\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	home := t.TempDir()
	configDir := filepath.Join(home, ".config", "ares")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(configDir, "config.yaml"), []byte("plugins:\n  custom:\n    - name: missing-tool\n      apply: missing-ares-tool apply\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("ARES_ROOT", rootDir)
	t.Setenv("ARES_OS_RELEASE", filepath.Join("..", "tests", "fixtures", "os-release", "ubuntu-24.04"))
	t.Setenv("HOME", home)

	root := newRootCmd("test")
	buf := &bytes.Buffer{}
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"preflight"})

	err := root.Execute()
	if err == nil {
		t.Fatal("expected preflight failure")
	}
	if !strings.Contains(buf.String(), "custom missing-tool apply: fail") {
		t.Fatalf("missing custom command failure:\n%s", buf.String())
	}
}

func TestPlanJSONPrintsPlan(t *testing.T) {
	output := runSnapshotCommand(t, "plan", "--json")
	var hardeningPlan struct {
		Profile string `json:"Profile"`
		Actions []any  `json:"Actions"`
	}
	if err := json.Unmarshal([]byte(output), &hardeningPlan); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, output)
	}
	if hardeningPlan.Profile != "basic" || len(hardeningPlan.Actions) == 0 {
		t.Fatalf("unexpected plan JSON: %+v", hardeningPlan)
	}
}

func TestDryRunJSONPrintsPlanAndResult(t *testing.T) {
	t.Setenv("ARES_NO_BANNER", "1")
	output := runSnapshotCommand(t, "--dry-run", "--json")
	var payload struct {
		Plan struct {
			Profile string `json:"Profile"`
		} `json:"plan"`
		Result struct {
			Skipped []string `json:"skipped"`
		} `json:"result"`
	}
	if err := json.Unmarshal([]byte(output), &payload); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, output)
	}
	if payload.Plan.Profile != "basic" || !containsString(payload.Result.Skipped, "dry-run requested; no changes applied") {
		t.Fatalf("unexpected dry-run JSON: %+v", payload)
	}
}

func TestExecute_Help(t *testing.T) {
	root := newRootCmd("dev")
	root.SetArgs([]string{"--help"})
	var out bytes.Buffer
	root.SetOut(&out)

	err := root.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	output := out.String()
	if !strings.Contains(output, "ares") {
		t.Error("expected project name in help output")
	}
	if !strings.Contains(output, "config") {
		t.Error("expected 'config' subcommand in help")
	}
}

func TestRunConfigInit(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)

	root := newRootCmd("test")
	buf := &bytes.Buffer{}
	root.SetOut(buf)
	root.SetArgs([]string{"config", "init"})

	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}

	// Config file should exist.
	configPath := filepath.Join(tmp, ".config", "ares", "config.yaml")
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Error("config file not created")
	}

	if !strings.Contains(buf.String(), "Wrote default config") {
		t.Error("expected success message")
	}
}

func TestRunConfigInit_AlreadyExists(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)

	// Pre-create config.
	configDir := filepath.Join(tmp, ".config", "ares")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(configDir, "config.yaml"), []byte("existing"), 0o644); err != nil {
		t.Fatal(err)
	}

	root := newRootCmd("test")
	buf := &bytes.Buffer{}
	root.SetOut(buf)
	root.SetArgs([]string{"config", "init"})

	err := root.Execute()
	if err == nil {
		t.Fatal("expected error when config exists")
	}
}

func TestRunConfigInit_Force(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)

	// Pre-create config.
	configDir := filepath.Join(tmp, ".config", "ares")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(configDir, "config.yaml"), []byte("existing"), 0o644); err != nil {
		t.Fatal(err)
	}

	root := newRootCmd("test")
	buf := &bytes.Buffer{}
	root.SetOut(buf)
	root.SetArgs([]string{"config", "init", "--force"})

	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(buf.String(), "Wrote default config") {
		t.Error("expected success message with --force")
	}
}

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
