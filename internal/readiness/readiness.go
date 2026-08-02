package readiness

import (
	"fmt"
	"os"
)

type Mode string

const (
	Apply    Mode = "apply"
	Rollback Mode = "rollback"
)

type Request struct {
	Mode         Mode
	DryRun       bool
	Yes          bool
	Root         string
	EffectiveUID int
}

type Decision struct {
	Name   string
	Status string
	Detail string
}

func Evaluate(request Request) []Decision {
	if request.EffectiveUID == 0 {
		request.EffectiveUID = os.Geteuid()
	}
	return []Decision{
		dryRunDecision(request),
		confirmationDecision(request),
		privilegeDecision(request),
	}
}

func Refusal(request Request) error {
	for _, decision := range Evaluate(request) {
		if decision.Status == "fail" {
			return fmt.Errorf("%s", decision.Detail)
		}
	}
	return nil
}

func dryRunDecision(request Request) Decision {
	if request.DryRun {
		return Decision{Name: "dry-run", Status: "pass", Detail: string(request.Mode) + " preview requested"}
	}
	return Decision{Name: "dry-run", Status: "pass", Detail: string(request.Mode) + " will mutate host state"}
}

func confirmationDecision(request Request) Decision {
	if request.DryRun || request.Yes {
		return Decision{Name: "confirmation", Status: "pass", Detail: "explicit confirmation satisfied"}
	}
	switch request.Mode {
	case Rollback:
		return Decision{Name: "confirmation", Status: "fail", Detail: "rollback requires --yes after reviewing the undo plan"}
	default:
		return Decision{Name: "confirmation", Status: "fail", Detail: "apply mode requires --yes after reviewing the plan"}
	}
}

func privilegeDecision(request Request) Decision {
	if request.Root != "" {
		return Decision{Name: "privileges", Status: "pass", Detail: "ARES_ROOT test root active"}
	}
	if request.DryRun || request.EffectiveUID == 0 {
		return Decision{Name: "privileges", Status: "pass", Detail: "root privileges satisfied"}
	}
	switch request.Mode {
	case Rollback:
		return Decision{Name: "privileges", Status: "fail", Detail: "rollback must run as root"}
	default:
		return Decision{Name: "privileges", Status: "fail", Detail: "apply mode must run as root"}
	}
}
