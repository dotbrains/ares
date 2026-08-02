package reports

import "github.com/dotbrains/ares/internal/plan"

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
