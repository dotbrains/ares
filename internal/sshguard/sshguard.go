package sshguard

import (
	"os"
	"os/user"
	"path/filepath"
	"strings"

	"github.com/dotbrains/ares/internal/reports"
)

type Facts struct {
	Root                 string
	RunningOverSSH       bool
	AllowPasswordLockout bool
}

type Decision struct {
	Allowed  bool
	Bypassed bool
	Detail   string
	Evidence []reports.Evidence
}

func Evaluate(facts Facts) Decision {
	if facts.Root != "" {
		return Decision{
			Allowed: true,
			Detail:  "test root active; SSH lockout guard simulated",
			Evidence: []reports.Evidence{{
				Name:       "ares_root",
				Value:      facts.Root,
				Source:     "env",
				Confidence: "high",
			}},
		}
	}
	if facts.AllowPasswordLockout {
		return Decision{
			Allowed:  true,
			Bypassed: true,
			Detail:   "password lockout explicitly allowed by config or CLI",
			Evidence: []reports.Evidence{{
				Name:       "ssh.allow_password_lockout",
				Value:      "true",
				Source:     "config/cli",
				Confidence: "high",
			}},
		}
	}
	if !facts.RunningOverSSH {
		return Decision{Allowed: true, Detail: "no active SSH session detected"}
	}
	if AuthorizedKeyAvailable(facts.Root) {
		return Decision{Allowed: true, Detail: "authorized key found for a likely login account"}
	}
	return Decision{
		Allowed: false,
		Detail:  "refusing to disable password authentication without a detected authorized key; configure ssh.allow_password_lockout or pass --allow-password-lockout to override",
	}
}

func AuthorizedKeyAvailable(root string) bool {
	for _, path := range AuthorizedKeyCandidates() {
		data, err := os.ReadFile(rootedPath(root, path))
		if err == nil && strings.TrimSpace(string(data)) != "" {
			return true
		}
	}
	return false
}

func AuthorizedKeyCandidates() []string {
	candidates := []string{"/root/.ssh/authorized_keys"}
	if home := strings.TrimSpace(os.Getenv("HOME")); home != "" {
		candidates = append(candidates, filepath.Join(home, ".ssh", "authorized_keys"))
	}
	if current, err := user.Current(); err == nil && current.HomeDir != "" {
		candidates = append(candidates, filepath.Join(current.HomeDir, ".ssh", "authorized_keys"))
	}
	return candidates
}

func rootedPath(root string, path string) string {
	if root == "" {
		return path
	}
	return filepath.Join(root, strings.TrimPrefix(path, "/"))
}
