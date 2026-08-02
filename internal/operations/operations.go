package operations

import (
	"strings"

	"github.com/dotbrains/ares/internal/intent"
	"github.com/dotbrains/ares/internal/plan"
	"github.com/dotbrains/ares/internal/reports"
)

type Kind string

const (
	WriteFile     Kind = "write_file"
	BackupFile    Kind = "backup_file"
	RunCommand    Kind = "run_command"
	RollbackNote  Kind = "rollback_note"
	CustomCommand Kind = "custom_command"
)

type Operation struct {
	Kind    Kind
	Plugin  string
	Path    string
	Command string
	Args    []string
	Note    string
	Phase   string
}

func fromIntent(op intent.Operation) Operation {
	return Operation{
		Kind:    Kind(op.Kind),
		Plugin:  op.Plugin,
		Path:    op.Path,
		Command: op.Command,
		Args:    op.Args,
		Note:    op.Note,
		Phase:   op.Phase,
	}
}

func Build(hardeningPlan plan.Plan) []Operation {
	var ops []Operation
	for _, plugin := range hardeningPlan.Plugins {
		for _, op := range intent.ForPlugin(hardeningPlan.Host, hardeningPlan.Profile, plugin).Operations() {
			ops = append(ops, fromIntent(op))
		}
	}
	return ops
}

func SummaryForPlan(hardeningPlan plan.Plan) reports.TransactionSummary {
	return Summary(Build(hardeningPlan))
}

func Summary(ops []Operation) reports.TransactionSummary {
	var summary reports.TransactionSummary
	for _, op := range ops {
		switch op.Kind {
		case WriteFile:
			summary.Files = append(summary.Files, op.Path)
		case BackupFile:
			summary.Backups = append(summary.Backups, op.Path)
		case RunCommand:
			summary.Commands = append(summary.Commands, commandString(op))
		case RollbackNote:
			summary.RollbackSteps = append(summary.RollbackSteps, op.Note)
		case CustomCommand:
			summary.Commands = append(summary.Commands, "custom "+op.Plugin+" "+op.Phase+": "+op.Command)
		}
	}
	summary.Files = uniqueStrings(summary.Files)
	summary.Commands = uniqueStrings(summary.Commands)
	summary.Backups = uniqueStrings(summary.Backups)
	summary.RollbackSteps = uniqueStrings(summary.RollbackSteps)
	return summary
}

func commandString(op Operation) string {
	if len(op.Args) == 0 {
		return op.Command
	}
	return op.Command + " " + strings.Join(op.Args, " ")
}

func RollbackPreview(summary reports.TransactionSummary) []string {
	if len(summary.Files) == 0 && len(summary.Backups) == 0 {
		return nil
	}
	backedUp := map[string]bool{}
	var preview []string
	for _, path := range summary.Backups {
		backedUp[path] = true
		preview = append(preview, "would restore newest backup for "+path)
	}
	for _, path := range summary.Files {
		if backedUp[path] {
			continue
		}
		preview = append(preview, "would remove "+path)
	}
	return preview
}

func uniqueStrings(values []string) []string {
	seen := map[string]bool{}
	result := make([]string, 0, len(values))
	for _, value := range values {
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		result = append(result, value)
	}
	return result
}
