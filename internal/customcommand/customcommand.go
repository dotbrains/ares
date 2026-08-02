package customcommand

import (
	stdctx "context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/dotbrains/ares/internal/customoutput"
	"github.com/dotbrains/ares/internal/plugins"
)

const DefaultTimeout = 2 * time.Minute

type Command struct {
	PluginID       string
	Phase          string
	Line           string
	TimeoutSeconds int
}

type Result struct {
	Output string
	Err    error
}

func New(plugin plugins.Plugin, phase string, line string) Command {
	return Command{
		PluginID:       plugin.ID,
		Phase:          phase,
		Line:           line,
		TimeoutSeconds: plugin.TimeoutSeconds,
	}
}

func ValidateLine(pluginName string, field string, command string) error {
	trimmed := strings.TrimSpace(command)
	if command != "" && trimmed == "" {
		return fmt.Errorf("custom plugin %q %s command must not be blank", pluginName, field)
	}
	if strings.ContainsAny(trimmed, "\r\n") {
		return fmt.Errorf("custom plugin %q %s command must be a single line", pluginName, field)
	}
	return nil
}

func ValidateLifecycle(pluginName string, apply string, verify string, rollback string) error {
	if strings.TrimSpace(apply) == "" && (strings.TrimSpace(verify) != "" || strings.TrimSpace(rollback) != "") {
		return fmt.Errorf("custom plugin %q apply command is required when verify or rollback is declared", pluginName)
	}
	return nil
}

func CheckExecutable(command string) (string, error) {
	executable := FirstWord(command)
	if executable == "" {
		return "", fmt.Errorf("missing executable")
	}
	if strings.Contains(executable, "/") {
		if _, err := os.Stat(executable); err != nil {
			return executable, fmt.Errorf("%s not found", executable)
		}
		return executable, nil
	}
	path, err := exec.LookPath(executable)
	if err != nil {
		return executable, fmt.Errorf("%s not found on PATH", executable)
	}
	return path, nil
}

func FirstWord(command string) string {
	fields := strings.Fields(command)
	if len(fields) == 0 {
		return ""
	}
	return fields[0]
}

func (command Command) Run() Result {
	timeout := DefaultTimeout
	if command.TimeoutSeconds > 0 {
		timeout = time.Duration(command.TimeoutSeconds) * time.Second
	}
	commandContext, cancel := stdctx.WithTimeout(stdctx.Background(), timeout)
	defer cancel()

	cmd := exec.CommandContext(commandContext, "sh", "-lc", command.Line)
	output, err := cmd.CombinedOutput()
	if commandContext.Err() == stdctx.DeadlineExceeded {
		return Result{Output: string(output), Err: fmt.Errorf("command timed out after %s", timeout)}
	}
	if err != nil {
		return Result{Output: string(output), Err: fmt.Errorf("%s: %w: %s", command.Line, err, strings.TrimSpace(string(output)))}
	}
	return Result{Output: string(output)}
}

func ParseOutput(pluginID string, output string) customoutput.Result {
	return customoutput.Parse(pluginID, output)
}
