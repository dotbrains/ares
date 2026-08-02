package apply

import (
	"github.com/dotbrains/ares/internal/operations"
	"github.com/dotbrains/ares/internal/plan"
	"github.com/dotbrains/ares/internal/reports"
)

type TransactionSummary = reports.TransactionSummary

func BuildTransaction(hardeningPlan plan.Plan) TransactionSummary {
	return operations.SummaryForPlan(hardeningPlan)
}

func BuildOperations(hardeningPlan plan.Plan) []operations.Operation {
	return operations.Build(hardeningPlan)
}
