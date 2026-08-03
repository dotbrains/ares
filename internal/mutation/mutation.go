package mutation

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/dotbrains/ares/internal/hostfs"
)

const DefaultCommandTimeout = 5 * time.Minute

type Result struct {
	Applied []string
	Skipped []string
	Failed  []string
}

type Operator struct {
	Root           string
	Now            time.Time
	CommandTimeout time.Duration
}

func (operator Operator) FS() hostfs.FS {
	return hostfs.FS{Root: operator.Root, Now: operator.Now}
}

func (operator Operator) Path(path string) string {
	return operator.FS().Path(path)
}

func (operator Operator) Run(name string, args ...string) (Result, error) {
	if operator.Root != "" {
		return Result{Applied: []string{"would run: " + name + " " + strings.Join(args, " ")}}, nil
	}
	timeout := operator.CommandTimeout
	if timeout == 0 {
		timeout = DefaultCommandTimeout
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, name, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return Result{}, fmt.Errorf("%s %s: command timed out after %s: %s", name, strings.Join(args, " "), timeout, strings.TrimSpace(string(output)))
		}
		return Result{}, fmt.Errorf("%s %s: %w: %s", name, strings.Join(args, " "), err, strings.TrimSpace(string(output)))
	}
	if len(output) > 0 {
		return Result{Applied: []string{strings.TrimSpace(string(output))}}, nil
	}
	return Result{}, nil
}

func (operator Operator) Backup(path string) (Result, error) {
	backupPath, created, err := operator.FS().Backup(path)
	if err != nil {
		return Result{}, err
	}
	if backupPath == "" {
		return Result{Skipped: []string{"backup skipped; missing " + path}}, nil
	}
	if !created {
		return Result{Skipped: []string{"backup skipped; existing " + backupPath}}, nil
	}
	return Result{Applied: []string{"backed up " + path + " to " + backupPath}}, nil
}

func (operator Operator) WriteFile(path string, data []byte, perm os.FileMode) (Result, error) {
	if err := operator.FS().WriteFile(path, data, perm); err != nil {
		return Result{}, err
	}
	return Result{Applied: []string{"wrote " + path}}, nil
}

func (operator Operator) Remove(path string) Result {
	if err := operator.FS().Remove(path); err != nil {
		if os.IsNotExist(err) {
			return Result{Skipped: []string{path + ": already absent"}}
		}
		return Result{Failed: []string{path + ": " + err.Error()}}
	}
	return Result{Applied: []string{"removed " + path}}
}

func (operator Operator) RestoreNewestBackup(path string) Result {
	backup, restored, err := operator.FS().RestoreNewestBackup(path)
	if err != nil {
		return Result{Failed: []string{path + ": restore backup: " + err.Error()}}
	}
	if !restored {
		return Result{Skipped: []string{path + ": no ares backup found"}}
	}
	return Result{Applied: []string{"restored " + path + " from " + backup}}
}
