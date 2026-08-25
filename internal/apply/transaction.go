package apply

import (
	"github.com/smeltery/ares/internal/operations"
	"github.com/smeltery/ares/internal/plan"
	"github.com/smeltery/ares/internal/reports"
)

type TransactionSummary = reports.TransactionSummary

func BuildTransaction(hardeningPlan plan.Plan) TransactionSummary {
	return operations.SummaryForPlan(hardeningPlan)
}

func BuildOperations(hardeningPlan plan.Plan) []operations.Operation {
	return operations.Build(hardeningPlan)
}
