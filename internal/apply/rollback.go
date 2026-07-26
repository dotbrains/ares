package apply

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type RollbackOptions struct {
	Yes  bool
	Root string
	Now  time.Time
}

func RollbackLast(opts RollbackOptions) (Result, error) {
	if opts.Now.IsZero() {
		opts.Now = time.Now()
	}
	result := Result{}
	base := rootedPath(opts.Root, "/var/log/ares")
	if err := os.MkdirAll(base, 0o755); err != nil {
		return result, fmt.Errorf("creating report directory: %w", err)
	}
	stamp := opts.Now.Format("20060102-150405")
	result.LogPath = filepath.Join(base, "rollback-"+stamp+".log")
	result.ReportPath = filepath.Join(base, "rollback-latest.json")
	result.UndoPlanPath = filepath.Join(base, "undo-plan.txt")

	if !opts.Yes {
		return finishRollback(result, fmt.Errorf("rollback requires --yes after reviewing the undo plan"))
	}
	if os.Geteuid() != 0 && opts.Root == "" {
		return finishRollback(result, fmt.Errorf("rollback must run as root"))
	}

	rollbackManagedFile(&result, opts.Root, "/etc/ssh/sshd_config.d/99-ares.conf")
	rollbackManagedFile(&result, opts.Root, "/etc/fail2ban/jail.d/ares-sshd.conf")
	rollbackManagedFile(&result, opts.Root, "/etc/apt/apt.conf.d/20auto-upgrades")
	rollbackManagedFile(&result, opts.Root, "/etc/sysctl.d/99-ares.conf")

	restoreNewestBackup(&result, opts.Root, "/etc/ssh/sshd_config")
	restoreNewestBackup(&result, opts.Root, "/etc/nftables.conf")
	restoreNewestBackup(&result, opts.Root, "/etc/dnf/automatic.conf")

	if opts.Root == "" {
		result.Skipped = append(result.Skipped, "service reloads are not automated during rollback; review SSH and firewall access before reloading services")
	}
	return finishRollback(result, nil)
}

func rollbackManagedFile(result *Result, root string, path string) {
	fullPath := rootedPath(root, path)
	if err := os.Remove(fullPath); err != nil {
		if os.IsNotExist(err) {
			result.Skipped = append(result.Skipped, path+": already absent")
			return
		}
		result.Failed = append(result.Failed, path+": "+err.Error())
		return
	}
	result.Applied = append(result.Applied, "removed "+path)
}

func restoreNewestBackup(result *Result, root string, path string) {
	fullPath := rootedPath(root, path)
	matches, err := filepath.Glob(fullPath + ".ares.*.bak")
	if err != nil || len(matches) == 0 {
		result.Skipped = append(result.Skipped, path+": no ares backup found")
		return
	}
	sort.Strings(matches)
	backup := matches[len(matches)-1]
	data, err := os.ReadFile(backup)
	if err != nil {
		result.Failed = append(result.Failed, path+": read backup: "+err.Error())
		return
	}
	if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
		result.Failed = append(result.Failed, path+": create parent: "+err.Error())
		return
	}
	if err := os.WriteFile(fullPath, data, 0o600); err != nil {
		result.Failed = append(result.Failed, path+": restore backup: "+err.Error())
		return
	}
	result.Applied = append(result.Applied, "restored "+path+" from "+strings.TrimPrefix(backup, root))
}

func rootedPath(root string, path string) string {
	if root == "" {
		return path
	}
	return filepath.Join(root, strings.TrimPrefix(path, "/"))
}

func finishRollback(result Result, runErr error) (Result, error) {
	if err := writeRollbackLog(result); err != nil && runErr == nil {
		runErr = err
	}
	return result, runErr
}

func writeRollbackLog(result Result) error {
	var lines []string
	lines = append(lines, "ares rollback")
	lines = append(lines, "")
	lines = append(lines, "applied:")
	lines = appendList(lines, result.Applied)
	lines = append(lines, "skipped:")
	lines = appendList(lines, result.Skipped)
	lines = append(lines, "failed:")
	lines = appendList(lines, result.Failed)
	return os.WriteFile(result.LogPath, []byte(strings.Join(lines, "\n")+"\n"), 0o644)
}
