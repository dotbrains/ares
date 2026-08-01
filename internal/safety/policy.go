package safety

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/dotbrains/ares/internal/config"
	"github.com/dotbrains/ares/internal/plan"
	"github.com/dotbrains/ares/internal/reports"
	"github.com/dotbrains/ares/internal/system"
)

type Decision = reports.Decision

type Facts struct {
	Host         system.Host
	Plan         plan.Plan
	Config       *config.Config
	Root         string
	EffectiveUID int
}

func Evaluate(facts Facts) []Decision {
	euid := facts.EffectiveUID
	if euid == 0 {
		euid = os.Geteuid()
	}
	decisions := []Decision{
		rootDecision(facts.Root, euid),
		sshSessionDecision(facts.Host),
		knownValueDecision("package manager", facts.Host.PackageManager),
		knownValueDecision("firewall backend", facts.Host.FirewallBackend),
		knownValueDecision("ssh service", facts.Host.SSHService),
		providerDecision(facts.Host),
		providerAdvisoryDecision(facts.Plan),
		sshPortDecision(facts.Host),
		planWarningsDecision(facts.Plan),
		reportDirectoryDecision(facts.Root),
	}
	decisions = append(decisions, customPluginCommandDecisions(facts.Config)...)
	return decisions
}

func HasFailures(decisions []Decision) bool {
	for _, decision := range decisions {
		if decision.Status == "fail" {
			return true
		}
	}
	return false
}

func SSHLockoutPolicy(root string, allowPasswordLockout bool) string {
	if root != "" {
		return "test root active; SSH lockout guard simulated"
	}
	if allowPasswordLockout {
		return "password lockout explicitly allowed by config or CLI"
	}
	if os.Getenv("SSH_CONNECTION") == "" && os.Getenv("SSH_CLIENT") == "" {
		return "no active SSH session detected"
	}
	if authorizedKeyAvailable() {
		return "authorized key found for a likely login account"
	}
	return "refusing to disable password authentication without a detected authorized key; configure ssh.allow_password_lockout or pass --allow-password-lockout to override"
}

func rootDecision(root string, euid int) Decision {
	if root != "" {
		return Decision{Name: "privileges", Status: "pass", Detail: "ARES_ROOT test root active"}
	}
	if euid == 0 {
		return Decision{Name: "privileges", Status: "pass", Detail: "running as root"}
	}
	return Decision{Name: "privileges", Status: "fail", Detail: "apply mode requires root privileges"}
}

func sshSessionDecision(host system.Host) Decision {
	if host.RunningOverSSH {
		return Decision{Name: "ssh session", Status: "pass", Detail: "active SSH session detected"}
	}
	return Decision{Name: "ssh session", Status: "warn", Detail: "active SSH session was not detected"}
}

func knownValueDecision(name string, value string) Decision {
	if value == "" || value == "unknown" {
		return Decision{Name: name, Status: "fail", Detail: "unknown"}
	}
	return Decision{Name: name, Status: "pass", Detail: value}
}

func providerDecision(host system.Host) Decision {
	if host.Provider == "" || host.Provider == "unknown" {
		return Decision{Name: "provider", Status: "warn", Detail: "unknown provider"}
	}
	return Decision{Name: "provider", Status: "pass", Detail: host.Provider}
}

func providerAdvisoryDecision(hardeningPlan plan.Plan) Decision {
	for _, plugin := range hardeningPlan.Plugins {
		if strings.HasPrefix(plugin.ID, "provider-") {
			return Decision{Name: "provider advisory", Status: "pass", Detail: plugin.ID}
		}
	}
	return Decision{Name: "provider advisory", Status: "warn", Detail: "no provider advisory selected"}
}

func sshPortDecision(host system.Host) Decision {
	if host.SSHPort == "" {
		return Decision{Name: "ssh port", Status: "fail", Detail: "no SSH port detected"}
	}
	return Decision{Name: "ssh port", Status: "pass", Detail: host.SSHPort}
}

func planWarningsDecision(hardeningPlan plan.Plan) Decision {
	if len(hardeningPlan.Warnings) == 0 {
		return Decision{Name: "plan warnings", Status: "pass", Detail: "none"}
	}
	return Decision{Name: "plan warnings", Status: "warn", Detail: fmt.Sprintf("%d warning(s)", len(hardeningPlan.Warnings))}
}

func reportDirectoryDecision(root string) Decision {
	dir := rootedPath(root, "/var/log/ares")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return Decision{Name: "reports", Status: "fail", Detail: err.Error()}
	}
	probe := filepath.Join(dir, ".ares-preflight")
	if err := os.WriteFile(probe, []byte("ok\n"), 0o600); err != nil {
		return Decision{Name: "reports", Status: "fail", Detail: err.Error()}
	}
	if err := os.Remove(probe); err != nil {
		return Decision{Name: "reports", Status: "fail", Detail: err.Error()}
	}
	return Decision{Name: "reports", Status: "pass", Detail: dir}
}

func customPluginCommandDecisions(cfg *config.Config) []Decision {
	if cfg == nil {
		return nil
	}
	var decisions []Decision
	for _, plugin := range cfg.Plugins.Custom {
		for _, command := range []struct {
			name  string
			value string
		}{
			{name: "probe", value: plugin.Probe},
			{name: "apply", value: plugin.Apply},
			{name: "verify", value: plugin.Verify},
			{name: "rollback", value: plugin.Rollback},
		} {
			if strings.TrimSpace(command.value) == "" {
				continue
			}
			decisions = append(decisions, customCommandDecision(plugin.Name, command.name, command.value))
		}
	}
	return decisions
}

func customCommandDecision(pluginName string, phase string, command string) Decision {
	executable := firstCommandWord(command)
	name := "custom " + pluginName + " " + phase
	if executable == "" {
		return Decision{Name: name, Status: "fail", Detail: "missing executable"}
	}
	if strings.Contains(executable, "/") {
		if _, err := os.Stat(executable); err != nil {
			return Decision{Name: name, Status: "fail", Detail: executable + " not found"}
		}
		return Decision{Name: name, Status: "pass", Detail: executable}
	}
	if path, err := exec.LookPath(executable); err == nil {
		return Decision{Name: name, Status: "pass", Detail: path}
	}
	return Decision{Name: name, Status: "fail", Detail: executable + " not found on PATH"}
}

func firstCommandWord(command string) string {
	fields := strings.Fields(command)
	if len(fields) == 0 {
		return ""
	}
	return fields[0]
}

func rootedPath(root string, path string) string {
	if root == "" {
		return path
	}
	return filepath.Join(root, strings.TrimPrefix(path, "/"))
}

func authorizedKeyAvailable() bool {
	for _, dir := range likelyHomeDirs() {
		path := filepath.Join(dir, ".ssh", "authorized_keys")
		data, err := os.ReadFile(path)
		if err == nil && strings.TrimSpace(string(data)) != "" {
			return true
		}
	}
	return false
}

func likelyHomeDirs() []string {
	dirs := []string{}
	for _, name := range []string{"SUDO_USER", "USER", "LOGNAME"} {
		if value := strings.TrimSpace(os.Getenv(name)); value != "" {
			if value == "root" {
				dirs = append(dirs, "/root")
			} else {
				dirs = append(dirs, filepath.Join("/home", value))
			}
		}
	}
	return append(dirs, "/root")
}
