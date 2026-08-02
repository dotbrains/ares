package reports

import (
	"os"
	"strings"
)

type Artifacts struct {
	RunReportPath      string
	RunLogPath         string
	UndoPlanPath       string
	RollbackReportPath string
	RollbackLogPath    string
}

type RunArtifactInput struct {
	Report RunReport
	Log    []string
	Undo   []string
}

type RollbackArtifactInput struct {
	Report RollbackReport
	Log    []string
}

func (artifacts Artifacts) FinishRun(input RunArtifactInput) error {
	if err := artifacts.WriteUndoPlan(input.Undo); err != nil {
		return err
	}
	if err := artifacts.WriteRunReport(input.Report); err != nil {
		return err
	}
	return artifacts.WriteRunLog(input.Log)
}

func (artifacts Artifacts) FinishRollback(input RollbackArtifactInput) error {
	if err := artifacts.WriteRollbackReport(input.Report); err != nil {
		return err
	}
	return artifacts.WriteRollbackLog(input.Log)
}

func (artifacts Artifacts) WriteRunReport(report RunReport) error {
	report.SchemaVersion = ReportSchemaVersion
	report.Probed = NonNilStrings(report.Probed)
	report.Verified = NonNilStrings(report.Verified)
	report.Applied = NonNilStrings(report.Applied)
	report.Skipped = NonNilStrings(report.Skipped)
	report.Failed = NonNilStrings(report.Failed)
	return WriteJSON(artifacts.RunReportPath, report)
}

func (artifacts Artifacts) WriteRollbackReport(report RollbackReport) error {
	report.SchemaVersion = RollbackSchemaVersion
	report.Applied = NonNilStrings(report.Applied)
	report.Skipped = NonNilStrings(report.Skipped)
	report.Failed = NonNilStrings(report.Failed)
	return WriteJSON(artifacts.RollbackReportPath, report)
}

func (artifacts Artifacts) WriteRunLog(lines []string) error {
	return writeTextArtifact(artifacts.RunLogPath, lines)
}

func (artifacts Artifacts) WriteRollbackLog(lines []string) error {
	return writeTextArtifact(artifacts.RollbackLogPath, lines)
}

func (artifacts Artifacts) WriteUndoPlan(lines []string) error {
	return writeTextArtifact(artifacts.UndoPlanPath, lines)
}

func NonNilStrings(values []string) []string {
	if values == nil {
		return []string{}
	}
	return values
}

func writeTextArtifact(path string, lines []string) error {
	return os.WriteFile(path, []byte(strings.Join(lines, "\n")+"\n"), 0o644)
}
