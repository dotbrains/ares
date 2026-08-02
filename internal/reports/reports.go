package reports

import (
	"encoding/json"
	"os"

	"github.com/dotbrains/ares/internal/plan"
	"github.com/dotbrains/ares/internal/plugins"
)

const (
	RunSchemaVersion       = "ares.run.v1"
	ReportSchemaVersion    = "ares.report.v1"
	PreflightSchemaVersion = "ares.preflight.v1"
	RollbackSchemaVersion  = "ares.rollback.v1"
)

type TransactionSummary struct {
	Files         []string `json:"files"`
	Commands      []string `json:"commands"`
	Backups       []string `json:"backups"`
	RollbackSteps []string `json:"rollback_steps"`
}

type Evidence struct {
	Name       string `json:"name"`
	Value      string `json:"value"`
	Source     string `json:"source"`
	Confidence string `json:"confidence"`
}

type Decision struct {
	Name     string     `json:"name"`
	Status   string     `json:"status"`
	Detail   string     `json:"detail"`
	Evidence []Evidence `json:"evidence,omitempty"`
}

type RunReport struct {
	SchemaVersion    string             `json:"schema_version"`
	Profile          string             `json:"profile"`
	Host             any                `json:"host"`
	Plugins          []plugins.Plugin   `json:"plugins"`
	Warnings         []string           `json:"warnings"`
	SSHLockoutPolicy string             `json:"ssh_lockout_policy"`
	SafetyEvidence   []Evidence         `json:"safety_evidence,omitempty"`
	Transaction      TransactionSummary `json:"transaction"`
	Probed           []string           `json:"probed"`
	Verified         []string           `json:"verified"`
	Applied          []string           `json:"applied"`
	Skipped          []string           `json:"skipped"`
	Failed           []string           `json:"failed"`
}

type RunOutput struct {
	SchemaVersion string    `json:"schema_version"`
	Plan          plan.Plan `json:"plan"`
	Result        any       `json:"result"`
	Error         string    `json:"error"`
}

type PreflightReport struct {
	SchemaVersion string             `json:"schema_version"`
	Profile       string             `json:"profile"`
	Host          any                `json:"host"`
	Plugins       []string           `json:"plugins"`
	Checks        []Decision         `json:"checks"`
	Transaction   TransactionSummary `json:"transaction"`
}

type RollbackReport struct {
	SchemaVersion string   `json:"schema_version"`
	Applied       []string `json:"applied"`
	Skipped       []string `json:"skipped"`
	Failed        []string `json:"failed"`
}

type CustomPluginReport struct {
	ID             string `json:"ID"`
	Kind           string `json:"Kind"`
	Rollback       string `json:"Rollback"`
	TimeoutSeconds int    `json:"TimeoutSeconds"`
}

type LatestRunReport struct {
	SchemaVersion string               `json:"schema_version"`
	Transaction   TransactionSummary   `json:"transaction"`
	Plugins       []CustomPluginReport `json:"plugins"`
}

func WriteJSON(path string, value any) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(data, '\n'), 0o644)
}

func ReadLatestRun(path string) (LatestRunReport, error) {
	var report LatestRunReport
	data, err := os.ReadFile(path)
	if err != nil {
		return report, err
	}
	if err := json.Unmarshal(data, &report); err != nil {
		return report, err
	}
	return report, nil
}
