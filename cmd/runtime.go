package cmd

import (
	"os"

	"github.com/dotbrains/ares/internal/apply"
	"github.com/dotbrains/ares/internal/config"
	"github.com/dotbrains/ares/internal/plan"
	"github.com/dotbrains/ares/internal/safety"
	"github.com/dotbrains/ares/internal/system"
)

type commandRuntime struct {
	Config    *config.Config
	Effective *config.Effective
	Host      system.Host
	Plan      plan.Plan
	Root      string
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
		Root:      os.Getenv("ARES_ROOT"),
	}, nil
}

func (runtime commandRuntime) ApplyOptions(dryRun bool, yes bool) apply.Options {
	tailscale := runtime.Config.Tailscale
	return apply.Options{
		DryRun:                     dryRun,
		Yes:                        yes,
		Root:                       runtime.Root,
		AllowPasswordLockout:       runtime.Config.SSH.AllowPasswordLockout,
		AllowPasswordLockoutSource: string(runtime.Effective.Sources["ssh.allow_password_lockout"]),
		Tailscale: apply.TailscaleOptions{
			SSHEnabled:       tailscale.SSHEnabled,
			AuthKeyEnv:       tailscale.AuthKeyEnv,
			AuthKey:          os.Getenv(tailscale.AuthKeyEnv),
			Hostname:         tailscale.Hostname,
			AcceptRoutes:     tailscale.AcceptRoutes,
			LoginServer:      tailscale.LoginServer,
			Tags:             tailscale.Tags,
			ExtraArgs:        tailscale.ExtraArgs,
			SSHEnabledSource: string(runtime.Effective.Sources["tailscale.ssh_enabled"]),
		},
	}
}

func (runtime commandRuntime) SafetyFacts() safety.Facts {
	return safety.Facts{
		Host:   runtime.Host,
		Plan:   runtime.Plan,
		Config: runtime.Config,
		Root:   runtime.Root,
	}
}
