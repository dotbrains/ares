package config

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/dotbrains/ares/internal/plugins"
	"gopkg.in/yaml.v3"
)

// Config is the top-level configuration.
type Config struct {
	Profile string        `yaml:"profile"`
	Plugins PluginsConfig `yaml:"plugins"`
}

type PluginsConfig struct {
	Enabled []string       `yaml:"enabled"`
	Custom  []CustomPlugin `yaml:"custom,omitempty"`
}

type CustomPlugin struct {
	Name           string `yaml:"name"`
	Probe          string `yaml:"probe"`
	Plan           string `yaml:"plan"`
	Apply          string `yaml:"apply"`
	Verify         string `yaml:"verify,omitempty"`
	Rollback       string `yaml:"rollback,omitempty"`
	TimeoutSeconds int    `yaml:"timeout_seconds,omitempty"`
}

// DefaultConfig returns the built-in default configuration.
func DefaultConfig() *Config {
	return &Config{
		Profile: "basic",
		Plugins: PluginsConfig{
			Enabled: []string{
				"ssh-hardening",
				"firewall-auto",
				"fail2ban",
				"security-updates",
				"sysctl-baseline",
			},
		},
	}
}

// ConfigDir returns the configuration directory path.
func ConfigDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("unable to determine home directory: %w", err)
	}
	return filepath.Join(home, ".config", "ares"), nil
}

// ConfigPath returns the full path to the config file.
func ConfigPath() (string, error) {
	dir, err := ConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "config.yaml"), nil
}

// Load reads the config from disk, falling back to defaults if no file exists.
func Load() (*Config, error) {
	path, err := ConfigPath()
	if err != nil {
		return DefaultConfig(), nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return DefaultConfig(), nil
		}
		return nil, fmt.Errorf("reading config: %w", err)
	}

	cfg := DefaultConfig()
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parsing config %s: %w", path, err)
	}
	if err := Validate(cfg); err != nil {
		return nil, fmt.Errorf("validating config %s: %w", path, err)
	}
	return cfg, nil
}

// LoadFrom reads the config from a specific path.
func LoadFrom(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return DefaultConfig(), nil
		}
		return nil, fmt.Errorf("reading config: %w", err)
	}

	cfg := DefaultConfig()
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parsing config %s: %w", path, err)
	}
	if err := Validate(cfg); err != nil {
		return nil, fmt.Errorf("validating config %s: %w", path, err)
	}
	return cfg, nil
}

// Validate checks loaded configuration before a plan can silently omit behavior.
func Validate(cfg *Config) error {
	if cfg == nil {
		return fmt.Errorf("config is nil")
	}
	if !slices.Contains([]string{"basic", "web", "strict"}, cfg.Profile) {
		return fmt.Errorf("unknown profile %q", cfg.Profile)
	}
	for _, id := range cfg.Plugins.Enabled {
		switch id {
		case "firewall-auto", "security-updates":
			continue
		}
		if _, ok := plugins.Find(id); !ok {
			return fmt.Errorf("unknown enabled plugin %q", id)
		}
	}
	customNames := map[string]bool{}
	for _, plugin := range cfg.Plugins.Custom {
		name := strings.TrimSpace(plugin.Name)
		if name == "" {
			return fmt.Errorf("custom plugin name is required")
		}
		if customNames[name] {
			return fmt.Errorf("duplicate custom plugin name %q", name)
		}
		customNames[name] = true
		switch name {
		case "firewall-auto", "security-updates":
			return fmt.Errorf("custom plugin name %q conflicts with a reserved plugin selector", name)
		}
		if _, ok := plugins.Find(name); ok {
			return fmt.Errorf("custom plugin name %q conflicts with a built-in plugin", name)
		}
		if plugin.TimeoutSeconds < 0 {
			return fmt.Errorf("custom plugin %q timeout_seconds must be non-negative", name)
		}
	}
	return nil
}

// Save writes the config to disk, creating directories as needed.
func Save(cfg *Config) error {
	path, err := ConfigPath()
	if err != nil {
		return err
	}
	return SaveTo(cfg, path)
}

// SaveTo writes the config to a specific path.
func SaveTo(cfg *Config, path string) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("creating config directory: %w", err)
	}

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("marshaling config: %w", err)
	}

	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("writing config: %w", err)
	}
	return nil
}
