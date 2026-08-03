package config

type Source string

const (
	SourceDefault Source = "default"
	SourceFile    Source = "file"
	SourceCLI     Source = "cli"
)

type Effective struct {
	Config  *Config
	Sources map[string]Source
}

type Overrides struct {
	Profile                 string
	AllowPasswordLockout    bool
	AllowPasswordLockoutSet bool
}

func EffectiveConfig(cfg *Config, fileLoaded bool, overrides Overrides) (*Effective, error) {
	if cfg == nil {
		cfg = DefaultConfig()
	}
	sources := map[string]Source{
		"profile":                    SourceDefault,
		"ssh.allow_password_lockout": SourceDefault,
		"tailscale.ssh_enabled":      SourceDefault,
		"plugins.enabled":            SourceDefault,
	}
	if fileLoaded {
		sources["profile"] = SourceFile
		sources["ssh.allow_password_lockout"] = SourceFile
		sources["tailscale.ssh_enabled"] = SourceFile
		sources["plugins.enabled"] = SourceFile
	}
	if overrides.Profile != "" {
		cfg.Profile = overrides.Profile
		sources["profile"] = SourceCLI
	}
	if overrides.AllowPasswordLockoutSet {
		cfg.SSH.AllowPasswordLockout = overrides.AllowPasswordLockout
		sources["ssh.allow_password_lockout"] = SourceCLI
	}
	if err := Validate(cfg); err != nil {
		return nil, err
	}
	return &Effective{Config: cfg, Sources: sources}, nil
}
