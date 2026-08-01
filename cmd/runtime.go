package cmd

import (
	"github.com/dotbrains/ares/internal/config"
	"github.com/dotbrains/ares/internal/plan"
	"github.com/dotbrains/ares/internal/system"
)

type commandRuntime struct {
	Config *config.Config
	Host   system.Host
	Plan   plan.Plan
}

func buildCommandRuntime(profile string) (commandRuntime, error) {
	cfg, err := config.Load()
	if err != nil {
		return commandRuntime{}, err
	}
	applyFlagOverrides(cfg, profile)
	if err := config.Validate(cfg); err != nil {
		return commandRuntime{}, err
	}
	host, err := system.Detect()
	if err != nil {
		return commandRuntime{}, err
	}
	return commandRuntime{
		Config: cfg,
		Host:   host,
		Plan:   plan.Build(host, cfg),
	}, nil
}
