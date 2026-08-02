package reports

import (
	"encoding/json"

	"github.com/dotbrains/ares/internal/plan"
)

func NewRunOutput(hardeningPlan plan.Plan, result any, runErr error) RunOutput {
	return RunOutput{
		SchemaVersion: RunSchemaVersion,
		Plan:          hardeningPlan,
		Result:        result,
		Error:         errorString(runErr),
	}
}

func errorString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

type PreflightInput struct {
	Profile     string
	Host        any
	Plugins     []string
	Checks      []Decision
	Transaction TransactionSummary
}

func NewPreflightReport(input PreflightInput) PreflightReport {
	return PreflightReport{
		SchemaVersion: PreflightSchemaVersion,
		Profile:       input.Profile,
		Host:          input.Host,
		Plugins:       NonNilStrings(input.Plugins),
		Checks:        input.Checks,
		Transaction:   input.Transaction,
	}
}

func MarshalJSON(value any) ([]byte, error) {
	return json.MarshalIndent(value, "", "  ")
}
