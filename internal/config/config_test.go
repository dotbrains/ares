package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	if cfg == nil {
		t.Fatal("DefaultConfig returned nil")
	}
}

func TestConfigDir(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)

	dir, err := ConfigDir()
	if err != nil {
		t.Fatalf("ConfigDir error: %v", err)
	}
	if dir != filepath.Join(tmp, ".config", "ares") {
		t.Errorf("ConfigDir = %q", dir)
	}
}

func TestConfigPath(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)

	path, err := ConfigPath()
	if err != nil {
		t.Fatalf("ConfigPath error: %v", err)
	}
	expected := filepath.Join(tmp, ".config", "ares", "config.yaml")
	if path != expected {
		t.Errorf("ConfigPath = %q, want %q", path, expected)
	}
}

func TestLoad_NoFile(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load error: %v", err)
	}
	if cfg == nil {
		t.Fatal("Load returned nil for missing file")
	}
}

func TestSaveAndLoad(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)

	cfg := DefaultConfig()
	if err := Save(cfg); err != nil {
		t.Fatalf("Save error: %v", err)
	}

	loaded, err := Load()
	if err != nil {
		t.Fatalf("Load error: %v", err)
	}
	if loaded == nil {
		t.Fatal("Load returned nil after Save")
	}
}

func TestSaveToAndLoadFrom(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "test-config.yaml")

	cfg := DefaultConfig()
	if err := SaveTo(cfg, path); err != nil {
		t.Fatalf("SaveTo error: %v", err)
	}

	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Fatal("SaveTo did not create file")
	}

	loaded, err := LoadFrom(path)
	if err != nil {
		t.Fatalf("LoadFrom error: %v", err)
	}
	if loaded == nil {
		t.Fatal("LoadFrom returned nil")
	}
}

func TestSaveToReplacesExistingConfigAtomically(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "config.yaml")
	if err := os.WriteFile(path, []byte("profile: web\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	if err := SaveTo(DefaultConfig(), path); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := info.Mode().Perm(); got != 0o644 {
		t.Fatalf("mode = %v, want 0644", got)
	}
	matches, err := filepath.Glob(filepath.Join(tmp, ".config.yaml.tmp-*"))
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 0 {
		t.Fatalf("temporary config files left behind: %v", matches)
	}
}

func TestLoadFrom_NoFile(t *testing.T) {
	cfg, err := LoadFrom("/nonexistent/path/config.yaml")
	if err != nil {
		t.Fatalf("LoadFrom error: %v", err)
	}
	if cfg == nil {
		t.Fatal("LoadFrom returned nil for missing file")
	}
}

func TestLoadFrom_MergesWithDefaults(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "profile-only.yaml")
	if err := os.WriteFile(path, []byte("profile: web\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadFrom(path)
	if err != nil {
		t.Fatalf("LoadFrom error: %v", err)
	}
	if cfg.Profile != "web" {
		t.Fatalf("Profile = %q, want web", cfg.Profile)
	}
	if len(cfg.Plugins.Enabled) == 0 {
		t.Fatal("LoadFrom dropped default enabled plugins")
	}
}

func TestLoadFrom_InvalidYAML(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "bad.yaml")
	os.WriteFile(path, []byte("{{invalid yaml"), 0o644)

	_, err := LoadFrom(path)
	if err == nil {
		t.Fatal("expected error for invalid YAML")
	}
}

func TestLoadFrom_RejectsUnknownProfile(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "bad-profile.yaml")
	if err := os.WriteFile(path, []byte("profile: typo\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := LoadFrom(path)
	if err == nil {
		t.Fatal("expected error for unknown profile")
	}
}

func TestLoadFrom_RejectsUnknownEnabledPlugin(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "bad-plugin.yaml")
	if err := os.WriteFile(path, []byte("plugins:\n  enabled:\n    - ssh-hardening\n    - typo-plugin\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := LoadFrom(path)
	if err == nil {
		t.Fatal("expected error for unknown enabled plugin")
	}
}

func TestLoadFrom_AllowsVirtualPluginSelectors(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "virtual-plugins.yaml")
	if err := os.WriteFile(path, []byte("plugins:\n  enabled:\n    - firewall-auto\n    - security-updates\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	if _, err := LoadFrom(path); err != nil {
		t.Fatalf("LoadFrom rejected virtual plugin selectors: %v", err)
	}
}

func TestLoadFrom_RejectsInvalidCustomPlugin(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "bad-custom.yaml")
	if err := os.WriteFile(path, []byte("plugins:\n  custom:\n    - name: ''\n      timeout_seconds: -1\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := LoadFrom(path)
	if err == nil {
		t.Fatal("expected error for invalid custom plugin")
	}
}

func TestLoadFrom_RejectsCustomPluginNameCollision(t *testing.T) {
	cases := []struct {
		name string
		yaml string
	}{
		{
			name: "builtin",
			yaml: "plugins:\n  custom:\n    - name: ssh-hardening\n",
		},
		{
			name: "reserved selector",
			yaml: "plugins:\n  custom:\n    - name: firewall-auto\n",
		},
		{
			name: "duplicate custom",
			yaml: "plugins:\n  custom:\n    - name: local-plugin\n    - name: local-plugin\n",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "bad-custom.yaml")
			if err := os.WriteFile(path, []byte(tc.yaml), 0o644); err != nil {
				t.Fatal(err)
			}

			if _, err := LoadFrom(path); err == nil {
				t.Fatal("expected custom plugin name collision error")
			}
		})
	}
}

func TestLoadFrom_RejectsUnsafeCustomPluginCommands(t *testing.T) {
	cases := []struct {
		name string
		yaml string
	}{
		{
			name: "blank command",
			yaml: "plugins:\n  custom:\n    - name: local-plugin\n      apply: '   '\n",
		},
		{
			name: "multiline command",
			yaml: "plugins:\n  custom:\n    - name: local-plugin\n      apply: |\n        echo one\n        echo two\n",
		},
		{
			name: "verify without apply",
			yaml: "plugins:\n  custom:\n    - name: local-plugin\n      verify: ares-plugin verify\n",
		},
		{
			name: "rollback without apply",
			yaml: "plugins:\n  custom:\n    - name: local-plugin\n      rollback: ares-plugin rollback\n",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "bad-custom.yaml")
			if err := os.WriteFile(path, []byte(tc.yaml), 0o644); err != nil {
				t.Fatal(err)
			}
			if _, err := LoadFrom(path); err == nil {
				t.Fatal("expected unsafe custom command error")
			}
		})
	}
}

func TestLoadFrom_RejectsUnsafeTailscaleConfig(t *testing.T) {
	cases := []struct {
		name string
		yaml string
	}{
		{
			name: "missing auth env",
			yaml: "plugins:\n  enabled:\n    - tailscale-ssh\ntailscale:\n  ssh_enabled: true\n",
		},
		{
			name: "plugin not enabled",
			yaml: "tailscale:\n  ssh_enabled: true\n  auth_key_env: TAILSCALE_AUTHKEY\n",
		},
		{
			name: "up option without ssh enabled",
			yaml: "plugins:\n  enabled:\n    - tailscale-ssh\ntailscale:\n  hostname: web-01\n",
		},
		{
			name: "invalid auth env",
			yaml: "plugins:\n  enabled:\n    - tailscale-ssh\ntailscale:\n  ssh_enabled: true\n  auth_key_env: 'TAILSCALE AUTHKEY'\n",
		},
		{
			name: "auth key in extra args",
			yaml: "plugins:\n  enabled:\n    - tailscale-ssh\ntailscale:\n  ssh_enabled: true\n  auth_key_env: TAILSCALE_AUTHKEY\n  extra_args:\n    - --auth-key=secret\n",
		},
		{
			name: "login server in extra args",
			yaml: "plugins:\n  enabled:\n    - tailscale-ssh\ntailscale:\n  ssh_enabled: true\n  auth_key_env: TAILSCALE_AUTHKEY\n  extra_args:\n    - --login-server=https://headscale.example.com\n",
		},
		{
			name: "advertise tags in extra args",
			yaml: "plugins:\n  enabled:\n    - tailscale-ssh\ntailscale:\n  ssh_enabled: true\n  auth_key_env: TAILSCALE_AUTHKEY\n  extra_args:\n    - --advertise-tags=tag:web\n",
		},
		{
			name: "tag without prefix",
			yaml: "plugins:\n  enabled:\n    - tailscale-ssh\ntailscale:\n  ssh_enabled: true\n  auth_key_env: TAILSCALE_AUTHKEY\n  tags:\n    - web\n",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "bad-tailscale.yaml")
			if err := os.WriteFile(path, []byte(tc.yaml), 0o644); err != nil {
				t.Fatal(err)
			}
			if _, err := LoadFrom(path); err == nil {
				t.Fatal("expected unsafe tailscale config error")
			}
		})
	}
}

func TestEffectiveConfigTracksCLIOverrideSources(t *testing.T) {
	cfg := DefaultConfig()
	effective, err := EffectiveConfig(cfg, true, Overrides{
		Profile:                 "web",
		AllowPasswordLockout:    true,
		AllowPasswordLockoutSet: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if effective.Config.Profile != "web" || !effective.Config.SSH.AllowPasswordLockout {
		t.Fatalf("unexpected effective config: %+v", effective.Config)
	}
	if effective.Sources["profile"] != SourceCLI {
		t.Fatalf("profile source = %q", effective.Sources["profile"])
	}
	if effective.Sources["ssh.allow_password_lockout"] != SourceCLI {
		t.Fatalf("ssh source = %q", effective.Sources["ssh.allow_password_lockout"])
	}
}
