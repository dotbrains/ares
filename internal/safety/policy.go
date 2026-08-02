package safety

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/dotbrains/ares/internal/config"
	"github.com/dotbrains/ares/internal/customcommand"
	"github.com/dotbrains/ares/internal/plan"
	"github.com/dotbrains/ares/internal/reports"
	"github.com/dotbrains/ares/internal/sshguard"
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
		withEvidence(knownValueDecision("package manager", facts.Host.PackageManager), hostEvidence(facts.Host, "package_manager", facts.Host.PackageManager)),
		withEvidence(knownValueDecision("firewall backend", facts.Host.FirewallBackend), hostEvidence(facts.Host, "firewall_backend", facts.Host.FirewallBackend)),
		withEvidence(knownValueDecision("ssh service", facts.Host.SSHService), hostEvidence(facts.Host, "ssh_service", facts.Host.SSHService)),
		providerDecision(facts.Host),
		providerAdvisoryDecision(facts.Plan),
		withEvidence(sshPortDecision(facts.Host), hostEvidence(facts.Host, "ssh_port", facts.Host.SSHPort)),
		planWarningsDecision(facts.Plan),
		reportDirectoryDecision(facts.Root),
	}
	decisions = append(decisions, customPluginCommandDecisions(facts.Config)...)
	return decisions
}

func withEvidence(decision Decision, evidence ...reports.Evidence) Decision {
	decision.Evidence = append(decision.Evidence, evidence...)
	return decision
}

func EvidenceFor(host system.Host, root string, allowPasswordLockout bool, allowPasswordLockoutSource string) []reports.Evidence {
	evidence := []reports.Evidence{
		hostEvidence(host, "os", host.OSID+" "+host.OSVersion),
		hostEvidence(host, "package_manager", host.PackageManager),
		hostEvidence(host, "init_system", host.InitSystem),
		hostEvidence(host, "firewall_backend", host.FirewallBackend),
		hostEvidence(host, "ssh_service", host.SSHService),
		hostEvidence(host, "ssh_port", host.SSHPort),
		hostEvidence(host, "provider", host.Provider),
		hostEvidence(host, "architecture", host.Architecture),
		{
			Name:       "ares_root",
			Value:      root,
			Source:     "env",
			Confidence: confidenceForValue(root),
		},
		{
			Name:       "ssh.allow_password_lockout",
			Value:      fmt.Sprintf("%t", allowPasswordLockout),
			Source:     sourceOrDefault(allowPasswordLockoutSource, "default"),
			Confidence: "high",
		},
	}
	return evidence
}

func hostEvidence(host system.Host, name string, value string) reports.Evidence {
	observed := host.Observed(name)
	if value != "" {
		observed.Value = value
	}
	return reports.Evidence{
		Name:       observed.Name,
		Value:      observed.Value,
		Source:     sourceOrDefault(observed.Source, "unknown"),
		Confidence: sourceOrDefault(observed.Confidence, confidenceForValue(observed.Value)),
	}
}

func sourceOrDefault(value string, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func confidenceForValue(value string) string {
	if strings.TrimSpace(value) == "" || value == "unknown" {
		return "low"
	}
	return "high"
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
	return sshguard.Evaluate(sshguard.Facts{
		Root:                 root,
		RunningOverSSH:       os.Getenv("SSH_CONNECTION") != "" || os.Getenv("SSH_CLIENT") != "",
		AllowPasswordLockout: allowPasswordLockout,
	}).Detail
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
	name := "custom " + pluginName + " " + phase
	path, err := customcommand.CheckExecutable(command)
	if err != nil {
		return Decision{Name: name, Status: "fail", Detail: err.Error()}
	}
	return Decision{Name: name, Status: "pass", Detail: path}
}

func rootedPath(root string, path string) string {
	if root == "" {
		return path
	}
	return filepath.Join(root, strings.TrimPrefix(path, "/"))
}
