package cmd

import (
	"github.com/dotbrains/ares/internal/config"
	"github.com/dotbrains/ares/internal/plan"
	"github.com/dotbrains/ares/internal/system"
)

type commandRuntime struct {
	Config    *config.Config
	Effective *config.Effective
	Host      system.Host
	Plan      plan.Plan
}

func buildCommandRuntime(overrides config.Overrides) (commandRuntime, error) {
	effective, err := config.LoadEffective(overrides)
	if err != nil {
		return commandRuntime{}, err
	}
	host, err := system.Detect()
	if err != nil {
		return commandRuntime{}, err
	}
	return commandRuntime{
		Config:    effective.Config,
		Effective: effective,
		Host:      host,
		Plan:      plan.Build(host, effective.Config),
	}, nil
}
